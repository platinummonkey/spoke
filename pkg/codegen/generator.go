package codegen

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// GenerateRequest represents a code generation request
type GenerateRequest struct {
	ModuleName   string
	Version      string
	ProtoFiles   []ProtoFile
	Dependencies []Dependency
	Language     string
	IncludeGRPC  bool
	Options      map[string]string
}

// Config holds code generation configuration
type Config struct {
	MaxWorkers  int
	Timeout     time.Duration
	EnableCache bool
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		MaxWorkers:  5,
		Timeout:     5 * time.Minute,
		EnableCache: true,
	}
}

// Generator is a code generator with dependency injection for cache
//
// IMPORTANT: Use NewGenerator() to create instances. The zero value is not usable.
//
// Benefits of dependency injection over global cache:
//   - Testable: Each test can have isolated cache state
//   - Configurable: Different cache implementations can be injected
//   - Clear dependencies: Cache dependency is explicit in struct
//   - Thread-safe: Multiple generators can coexist without interference
type Generator struct {
	cache sync.Map // In-memory cache using sync.Map
}

// NewGenerator creates a new code generator with injected dependencies
func NewGenerator() *Generator {
	return &Generator{}
}

// Deprecated: globalCache is deprecated. Use Generator.cache instead.
// This exists only for backward compatibility with legacy callers.
// TODO: Remove after migrating all callers to Generator
var globalCache sync.Map

// GenerateCode compiles proto files for a single language using the generator's cache
func (g *Generator) GenerateCode(ctx context.Context, req *GenerateRequest, config *Config) (*CompilationResult, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := validateRequest(req); err != nil {
		return nil, err
	}

	startTime := time.Now()
	result := &CompilationResult{
		Language: req.Language,
		Success:  false,
	}

	// Check cache if enabled
	if config.EnableCache {
		cacheKey := buildCacheKey(req)
		if cached, ok := g.cache.Load(cacheKey); ok {
			if cachedResult, ok := cached.(*CompilationResult); ok {
				cachedResult.CacheHit = true
				cachedResult.Duration = time.Since(startTime)
				return cachedResult, nil
			}
		}
	}

	// Get language spec from registry
	langSpec, err := GetLanguageSpec(req.Language)
	if err != nil {
		result.Error = fmt.Sprintf("language not supported: %s", req.Language)
		result.Duration = time.Since(startTime)
		return result, err
	}

	if !langSpec.Enabled {
		err := fmt.Errorf("language %s is disabled", req.Language)
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, err
	}

	// Execute Docker compilation
	dockerReq := &DockerRequest{
		Image:       langSpec.DockerImage,
		Tag:         langSpec.DockerTag,
		ProtoFiles:  req.ProtoFiles,
		ProtocFlags: buildProtocFlags(langSpec, req),
		Timeout:     config.Timeout,
	}

	execResult, err := ExecuteDocker(ctx, dockerReq)
	if err != nil {
		result.Error = fmt.Sprintf("compilation failed: %v", err)
		result.Duration = time.Since(startTime)
		return result, err
	}

	result.GeneratedFiles = execResult.GeneratedFiles
	result.Duration = execResult.Duration
	result.Success = true

	// Generate package manager files if needed
	if langSpec.PackageManager != nil {
		pkgFiles, err := generatePackageFiles(langSpec, req)
		if err != nil {
			result.Error = fmt.Sprintf("package generation warning: %v", err)
		} else {
			result.PackageFiles = pkgFiles
		}
	}

	// Store in cache if enabled
	if config.EnableCache {
		cacheKey := buildCacheKey(req)
		g.cache.Store(cacheKey, result)
	}

	return result, nil
}

// GenerateCodeParallel compiles proto files for multiple languages in parallel
func (g *Generator) GenerateCodeParallel(ctx context.Context, req *GenerateRequest, languages []string, config *Config) ([]*CompilationResult, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if len(languages) == 0 {
		return nil, fmt.Errorf("no languages specified")
	}

	// Validate all languages first
	for _, lang := range languages {
		if _, err := GetLanguageSpec(lang); err != nil {
			return nil, fmt.Errorf("language not supported: %s", lang)
		}
	}

	// Use errgroup for parallel execution with max workers
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(config.MaxWorkers)

	results := make([]*CompilationResult, len(languages))
	var mu sync.Mutex

	for i, lang := range languages {
		i, lang := i, lang // capture loop variables
		eg.Go(func() error {
			langReq := &GenerateRequest{
				ModuleName:   req.ModuleName,
				Version:      req.Version,
				ProtoFiles:   req.ProtoFiles,
				Dependencies: req.Dependencies,
				Language:     lang,
				IncludeGRPC:  req.IncludeGRPC,
				Options:      req.Options,
			}

			result, err := g.GenerateCode(ctx, langReq, config)

			mu.Lock()
			results[i] = result
			mu.Unlock()

			return err
		})
	}

	// Wait for all compilations to complete
	if err := eg.Wait(); err != nil {
		// Return partial results even if some failed
		return results, err
	}

	return results, nil
}

// validateRequest validates a generation request
func validateRequest(req *GenerateRequest) error {
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
		return fmt.Errorf("no proto files provided")
	}
	if req.Language == "" {
		return fmt.Errorf("language is required")
	}
	return nil
}

// buildCacheKey builds a cache key string
func buildCacheKey(req *GenerateRequest) string {
	key := &CacheKey{
		ModuleName:    req.ModuleName,
		Version:       req.Version,
		Language:      req.Language,
		PluginVersion: "", // Will be set from language spec
		ProtoHash:     hashProtoFiles(req.ProtoFiles, req.Dependencies),
		Options:       req.Options,
	}
	return key.String()
}

// buildProtocFlags builds protoc flags for a language
func buildProtocFlags(langSpec *LanguageSpec, req *GenerateRequest) []string {
	flags := make([]string, 0)

	switch langSpec.ID {
	case "go":
		flags = append(flags, "--go_out=/output")
		flags = append(flags, langSpec.ProtocFlags...)
		if req.IncludeGRPC && langSpec.SupportsGRPC {
			flags = append(flags, "--go-grpc_out=/output")
			flags = append(flags, langSpec.GRPCFlags...)
		}
	case "python":
		flags = append(flags, "--python_out=/output")
		if req.IncludeGRPC && langSpec.SupportsGRPC {
			flags = append(flags, "--grpc_python_out=/output")
		}
	case "java":
		flags = append(flags, "--java_out=/output")
		if req.IncludeGRPC && langSpec.SupportsGRPC {
			flags = append(flags, "--grpc-java_out=/output")
		}
	default:
		flags = append(flags, fmt.Sprintf("--%s_out=/output", langSpec.ID))
		flags = append(flags, langSpec.ProtocFlags...)
	}

	return flags
}

// generatePackageFiles generates package manager configuration files
func generatePackageFiles(langSpec *LanguageSpec, req *GenerateRequest) ([]GeneratedFile, error) {
	if langSpec.PackageManager == nil {
		return nil, nil
	}

	generator := GetPackageGenerator(langSpec.PackageManager.Name)
	if generator == nil {
		return nil, fmt.Errorf("package generator not found: %s", langSpec.PackageManager.Name)
	}

	pkgReq := &PackageRequest{
		ModuleName:  req.ModuleName,
		Version:     req.Version,
		Language:    langSpec.ID,
		IncludeGRPC: req.IncludeGRPC,
		Options:     req.Options,
	}

	return generator.Generate(pkgReq)
}

// defaultGenerator is the global generator instance for backward compatibility
// Deprecated: Use NewGenerator() to create generator instances instead
var defaultGenerator = NewGenerator()

// GenerateCode is a backward-compatible wrapper for legacy callers
//
// Deprecated: Use Generator.GenerateCode() with dependency injection instead.
// This function uses a global generator instance which makes testing difficult.
//
// Migration:
//
//	// Old (global state):
//	result, err := codegen.GenerateCode(ctx, req, config)
//
//	// New (dependency injection):
//	generator := codegen.NewGenerator()
//	result, err := generator.GenerateCode(ctx, req, config)
func GenerateCode(ctx context.Context, req *GenerateRequest, config *Config) (*CompilationResult, error) {
	return defaultGenerator.GenerateCode(ctx, req, config)
}

// GenerateCodeParallel is a backward-compatible wrapper for legacy callers
//
// Deprecated: Use Generator.GenerateCodeParallel() with dependency injection instead.
// This function uses a global generator instance which makes testing difficult.
//
// Migration:
//
//	// Old (global state):
//	results, err := codegen.GenerateCodeParallel(ctx, req, langs, config)
//
//	// New (dependency injection):
//	generator := codegen.NewGenerator()
//	results, err := generator.GenerateCodeParallel(ctx, req, langs, config)
func GenerateCodeParallel(ctx context.Context, req *GenerateRequest, languages []string, config *Config) ([]*CompilationResult, error) {
	return defaultGenerator.GenerateCodeParallel(ctx, req, languages, config)
}
