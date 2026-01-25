package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/platinummonkey/spoke/pkg/codegen"
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

	return &DefaultOrchestrator{
		config:           config,
		languageRegistry: langRegistry,
		dockerRunner:     dockerRunner,
		packageRegistry:  pkgRegistry,
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

	// TODO: Check cache (Phase 5)

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

	// TODO: Upload to S3 (Phase 5)
	// TODO: Store in cache (Phase 5)

	result.Success = true
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
	if o.dockerRunner != nil {
		return o.dockerRunner.Close()
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
