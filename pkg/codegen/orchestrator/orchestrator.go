package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/platinummonkey/spoke/pkg/codegen"
	"github.com/platinummonkey/spoke/pkg/codegen/artifacts"
	"github.com/platinummonkey/spoke/pkg/codegen/cache"
	"github.com/platinummonkey/spoke/pkg/codegen/docker"
	"github.com/platinummonkey/spoke/pkg/codegen/languages"
	"github.com/platinummonkey/spoke/pkg/codegen/packages"
)

// DefaultOrchestrator implements the Orchestrator interface
type DefaultOrchestrator struct {
	config           *Config
	languageRegistry *languages.Registry
	dockerRunner     docker.Runner
	packageRegistry  *packages.Registry
	cache            cache.Cache
	artifactsManager artifacts.Manager
	jobs             map[string]*codegen.CompilationJob
	jobsMu           sync.RWMutex
}

// NewOrchestrator creates a new compilation orchestrator
func NewOrchestrator(config *Config) (*DefaultOrchestrator, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Initialize language registry
	langRegistry := languages.InitializeDefaultRegistry()

	// Initialize Docker runner
	dockerRunner, err := docker.NewDockerRunner()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Docker runner: %v", err)
	}

	// Initialize package generator registry
	pkgRegistry := packages.NewRegistry()
	// Register default package generators
	// Generators will be registered as we implement them
	// For now, the registry is empty but ready to accept registrations

	// Initialize cache (optional)
	var cacheInstance cache.Cache
	if config.EnableCache {
		cacheConfig := &cache.Config{
			EnableL1:    true,
			L1MaxSize:   10 * 1024 * 1024, // 10MB
			L1TTL:       5 * time.Minute,
			EnableL2:    config.RedisAddr != "",
			L2Addr:      config.RedisAddr,
			L2Password:  config.RedisPassword,
			L2DB:        config.RedisDB,
			L2TTL:       24 * time.Hour,
			L2KeyPrefix: "spoke:compiled:",
		}
		cacheInstance, err = cache.NewCache(cacheConfig)
		if err != nil {
			// Log error but continue without cache
			fmt.Printf("Warning: failed to initialize cache: %v\n", err)
		}
	}

	// Initialize artifacts manager (optional)
	var artifactsManagerInstance artifacts.Manager
	if config.S3Bucket != "" {
		artifactsConfig := &artifacts.Config{
			S3Bucket:          config.S3Bucket,
			S3Prefix:          config.S3Prefix,
			S3Region:          config.S3Region,
			CompressionFormat: "tar.gz",
			EnableChecksum:    true,
		}
		artifactsManagerInstance, err = artifacts.NewS3Manager(artifactsConfig)
		if err != nil {
			// Log error but continue without S3
			fmt.Printf("Warning: failed to initialize artifacts manager: %v\n", err)
		}
	}

	return &DefaultOrchestrator{
		config:           config,
		languageRegistry: langRegistry,
		dockerRunner:     dockerRunner,
		packageRegistry:  pkgRegistry,
		cache:            cacheInstance,
		artifactsManager: artifactsManagerInstance,
		jobs:             make(map[string]*codegen.CompilationJob),
	}, nil
}

// CompileSingle compiles proto files for a single language
func (o *DefaultOrchestrator) CompileSingle(ctx context.Context, req *CompileRequest) (*codegen.CompilationResult, error) {
	// Validate request
	if err := o.validateRequest(req); err != nil {
		return nil, err
	}

	// Get language spec
	langSpec, err := o.languageRegistry.Get(req.Language)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrLanguageNotSupported, req.Language)
	}

	if !langSpec.Enabled {
		return nil, fmt.Errorf("language %s is disabled", req.Language)
	}

	startTime := time.Now()
	result := &codegen.CompilationResult{
		Language: req.Language,
		Success:  false,
	}

	// Check cache if enabled
	if o.cache != nil && o.config.EnableCache {
		cacheKey := cache.GenerateCacheKey(
			req.ModuleName,
			req.Version,
			req.Language,
			langSpec.PluginVersion,
			req.ProtoFiles,
			req.Dependencies,
			req.Options,
		)

		cachedResult, err := o.cache.Get(ctx, cacheKey)
		if err == nil && cachedResult != nil {
			// Cache hit!
			cachedResult.CacheHit = true
			cachedResult.Duration = time.Since(startTime)
			return cachedResult, nil
		}
		// Cache miss - continue with compilation
	}

	// Build Docker execution request
	dockerReq := &docker.ExecutionRequest{
		Image:       langSpec.DockerImage,
		Tag:         langSpec.DockerTag,
		ProtoFiles:  req.ProtoFiles,
		ProtocFlags: o.buildProtocFlags(langSpec, req),
		Timeout:     time.Duration(o.config.CompilationTimeout) * time.Second,
	}

	// Execute compilation in Docker
	execResult, err := o.dockerRunner.Execute(ctx, dockerReq)
	if err != nil {
		result.Error = fmt.Errorf("compilation failed: %v", err).Error()
		result.Duration = time.Since(startTime)
		return result, err
	}

	result.GeneratedFiles = execResult.GeneratedFiles
	result.Duration = execResult.Duration

	// Generate package manager files
	if langSpec.PackageManager != nil {
		pkgFiles, err := o.generatePackageFiles(langSpec, req)
		if err != nil {
			// Non-fatal error - log but continue
			result.Error = fmt.Sprintf("package generation warning: %v", err)
		} else {
			result.PackageFiles = pkgFiles
		}
	}

	result.Success = true

	// Upload to S3 if configured
	if o.artifactsManager != nil {
		storeReq := &artifacts.StoreRequest{
			ModuleName:        req.ModuleName,
			Version:           req.Version,
			Language:          req.Language,
			Files:             result.GeneratedFiles,
			Metadata:          make(map[string]string),
			CompressionFormat: "tar.gz",
		}
		storeReq.Metadata["plugin_version"] = langSpec.PluginVersion
		storeReq.Metadata["include_grpc"] = fmt.Sprintf("%v", req.IncludeGRPC)

		storeResult, err := o.artifactsManager.Store(ctx, storeReq)
		if err != nil {
			// Log error but don't fail the compilation
			fmt.Printf("Warning: failed to upload artifacts to S3: %v\n", err)
		} else {
			result.S3Key = storeResult.S3Key
			result.S3Bucket = storeResult.S3Bucket
			result.ArtifactHash = storeResult.Hash
		}
	}

	// Store in cache if enabled
	if o.cache != nil && o.config.EnableCache {
		cacheKey := cache.GenerateCacheKey(
			req.ModuleName,
			req.Version,
			req.Language,
			langSpec.PluginVersion,
			req.ProtoFiles,
			req.Dependencies,
			req.Options,
		)

		if err := o.cache.Set(ctx, cacheKey, result, 24*time.Hour); err != nil {
			// Log error but don't fail the compilation
			fmt.Printf("Warning: failed to store result in cache: %v\n", err)
		}
	}

	return result, nil
}

// CompileAll compiles proto files for multiple languages in parallel
func (o *DefaultOrchestrator) CompileAll(ctx context.Context, req *CompileRequest, languageIDs []string) ([]*codegen.CompilationResult, error) {
	if len(languageIDs) == 0 {
		return nil, fmt.Errorf("no languages specified")
	}

	// Validate all languages exist and are enabled
	for _, langID := range languageIDs {
		langSpec, err := o.languageRegistry.Get(langID)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrLanguageNotSupported, langID)
		}
		if !langSpec.Enabled {
			return nil, fmt.Errorf("language %s is disabled", langID)
		}
	}

	// Create worker pool
	maxWorkers := o.config.MaxParallelWorkers
	if maxWorkers <= 0 {
		maxWorkers = 5
	}
	if maxWorkers > len(languageIDs) {
		maxWorkers = len(languageIDs)
	}

	// Channel for work distribution
	type workItem struct {
		language string
		index    int
	}

	workCh := make(chan workItem, len(languageIDs))
	resultCh := make(chan struct {
		result *codegen.CompilationResult
		index  int
		err    error
	}, len(languageIDs))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range workCh {
				// Create language-specific request
				langReq := &CompileRequest{
					ModuleName:   req.ModuleName,
					Version:      req.Version,
					VersionID:    req.VersionID,
					ProtoFiles:   req.ProtoFiles,
					Dependencies: req.Dependencies,
					IncludeGRPC:  req.IncludeGRPC,
					Options:      req.Options,
					StorageDir:   req.StorageDir,
					S3Bucket:     req.S3Bucket,
					Language:     work.language,
				}

				result, err := o.CompileSingle(ctx, langReq)
				resultCh <- struct {
					result *codegen.CompilationResult
					index  int
					err    error
				}{result: result, index: work.index, err: err}
			}
		}()
	}

	// Distribute work
	for i, langID := range languageIDs {
		workCh <- workItem{language: langID, index: i}
	}
	close(workCh)

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results in order
	results := make([]*codegen.CompilationResult, len(languageIDs))
	var errs []error

	for item := range resultCh {
		results[item.index] = item.result
		if item.err != nil {
			errs = append(errs, item.err)
		}
	}

	// If any compilations failed, return aggregated error
	if len(errs) > 0 {
		return results, fmt.Errorf("compilation failed for %d languages: %v", len(errs), errs[0])
	}

	return results, nil
}

// GetStatus returns the status of a compilation job
func (o *DefaultOrchestrator) GetStatus(ctx context.Context, jobID string) (*codegen.CompilationJob, error) {
	o.jobsMu.RLock()
	defer o.jobsMu.RUnlock()

	job, exists := o.jobs[jobID]
	if !exists {
		return nil, ErrJobNotFound
	}

	return job, nil
}

// Close releases resources
func (o *DefaultOrchestrator) Close() error {
	var errs []error

	if o.dockerRunner != nil {
		if err := o.dockerRunner.Close(); err != nil {
			errs = append(errs, fmt.Errorf("docker runner: %w", err))
		}
	}

	if o.cache != nil {
		if err := o.cache.Close(); err != nil {
			errs = append(errs, fmt.Errorf("cache: %w", err))
		}
	}

	if o.artifactsManager != nil {
		if err := o.artifactsManager.Close(); err != nil {
			errs = append(errs, fmt.Errorf("artifacts manager: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}

	return nil
}

// validateRequest validates a compilation request
func (o *DefaultOrchestrator) validateRequest(req *CompileRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.ModuleName == "" {
		return fmt.Errorf("module name is required")
	}

	if req.Version == "" {
		return fmt.Errorf("version is required")
	}

	if len(req.ProtoFiles) == 0 {
		return ErrNoProtoFiles
	}

	// Validate language if specified in request
	if req.Language != "" {
		if _, err := o.languageRegistry.Get(req.Language); err != nil {
			return fmt.Errorf("%w: %s", ErrLanguageNotSupported, req.Language)
		}
	}

	return nil
}

// buildProtocFlags builds the protoc flags for a language
func (o *DefaultOrchestrator) buildProtocFlags(langSpec *languages.LanguageSpec, req *CompileRequest) []string {
	flags := make([]string, 0)

	// Add language-specific flags
	switch langSpec.ID {
	case languages.LanguageGo:
		flags = append(flags, "--go_out=/output")
		flags = append(flags, langSpec.ProtocFlags...)
		if req.IncludeGRPC && langSpec.SupportsGRPC {
			flags = append(flags, "--go-grpc_out=/output")
			flags = append(flags, langSpec.GRPCFlags...)
		}
	case languages.LanguagePython:
		flags = append(flags, "--python_out=/output")
		if req.IncludeGRPC && langSpec.SupportsGRPC {
			flags = append(flags, "--grpc_python_out=/output")
		}
	case languages.LanguageJava:
		flags = append(flags, "--java_out=/output")
		if req.IncludeGRPC && langSpec.SupportsGRPC {
			flags = append(flags, "--grpc-java_out=/output")
		}
	default:
		// Generic flags for other languages
		flags = append(flags, fmt.Sprintf("--%s_out=/output", langSpec.ID))
		flags = append(flags, langSpec.ProtocFlags...)
	}

	return flags
}

// generatePackageFiles generates package manager configuration files
func (o *DefaultOrchestrator) generatePackageFiles(langSpec *languages.LanguageSpec, req *CompileRequest) ([]codegen.GeneratedFile, error) {
	if langSpec.PackageManager == nil {
		return nil, nil
	}

	generator, exists := o.packageRegistry.Get(langSpec.PackageManager.Name)
	if !exists {
		return nil, fmt.Errorf("package generator not found: %s", langSpec.PackageManager.Name)
	}

	pkgReq := &packages.GenerateRequest{
		ModuleName:  req.ModuleName,
		Version:     req.Version,
		Language:    langSpec.ID,
		IncludeGRPC: req.IncludeGRPC,
		Options:     req.Options,
	}

	return generator.Generate(pkgReq)
}

// createJob creates a new compilation job
func (o *DefaultOrchestrator) createJob(versionID int64, language string) *codegen.CompilationJob {
	now := time.Now()
	job := &codegen.CompilationJob{
		ID:        uuid.New().String(),
		VersionID: versionID,
		Language:  language,
		Status:    codegen.JobStatusPending,
		StartedAt: &now,
	}

	o.jobsMu.Lock()
	o.jobs[job.ID] = job
	o.jobsMu.Unlock()

	return job
}

// updateJob updates a compilation job
func (o *DefaultOrchestrator) updateJob(jobID string, status codegen.JobStatus, result *codegen.CompilationResult, err error) {
	o.jobsMu.Lock()
	defer o.jobsMu.Unlock()

	job, exists := o.jobs[jobID]
	if !exists {
		return
	}

	now := time.Now()
	job.Status = status
	job.CompletedAt = &now
	job.Result = result

	if err != nil {
		job.Error = err.Error()
	}
}
