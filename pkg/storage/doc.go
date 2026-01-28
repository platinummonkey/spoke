// Package storage provides pluggable persistence backends for the Spoke protobuf schema registry.
//
// # Overview
//
// This package defines the storage abstraction layer for Spoke, enabling multiple backend
// implementations (filesystem, PostgreSQL, Redis, S3) while providing a unified interface
// for the API layer. It manages protobuf modules, versions, files, and compiled artifacts.
//
// # Architecture
//
// The storage layer uses interface segregation to compose focused capabilities:
//
//   - ModuleReader: Read operations for modules (GetModule, ListModules)
//   - ModuleWriter: Write operations for modules (CreateModule)
//   - VersionReader: Read operations for versions (GetVersion, ListVersions, GetFile)
//   - VersionWriter: Write operations for versions (CreateVersion, UpdateVersion)
//   - FileStorage: Content-addressed file storage (GetFileContent, PutFileContent)
//   - ArtifactStorage: Compiled artifact storage (GetCompiledArtifact, PutCompiledArtifact)
//   - CacheManager: Cache invalidation (InvalidateCache)
//   - HealthChecker: Backend health monitoring (HealthCheck)
//
// These interfaces compose into the unified Storage interface that provides all capabilities.
//
// # Storage Interface
//
// The canonical storage.Storage interface supersedes both api.Storage and storage.StorageV2.
// It provides context-aware operations with proper cancellation, timeouts, and tracing support:
//
//	type Storage interface {
//		ModuleReader
//		ModuleWriter
//		VersionReader
//		VersionWriter
//		FileStorage
//		ArtifactStorage
//		CacheManager
//		HealthChecker
//	}
//
// All methods accept context.Context as the first parameter, enabling:
//   - Request cancellation propagation from HTTP handlers
//   - Timeout enforcement to prevent hanging operations
//   - Distributed tracing through OpenTelemetry
//   - Proper resource cleanup on cancellation
//
// # Backend Implementations
//
// FileSystemStorage: Stores data as JSON files and directories on disk.
// Best for development, single-node deployments, and simple use cases.
//
//	storage, err := storage.NewFileSystemStorage("/var/spoke/data")
//
// PostgresStorage: Stores metadata in PostgreSQL with optional S3 for file content.
// Best for production, multi-node deployments, ACID requirements, and advanced features
// like full-text search, analytics, and audit logging.
//
//	config := storage.Config{
//		Type:            "postgres",
//		PostgresURL:     "postgres://localhost/spoke",
//		PostgresMaxConns: 20,
//	}
//	storage, err := storage.NewPostgresStorage(config)
//
// RedisStorage: Caching layer for frequently accessed data (planned).
// Reduces load on primary storage for hot data like latest versions and popular modules.
//
// S3Storage: Content-addressed storage for large proto files and artifacts (planned).
// Offloads large binary data from PostgreSQL, reducing costs and improving performance.
//
// HybridStorage: Combines PostgreSQL (metadata) + S3 (content) + Redis (cache).
// Best for high-scale production deployments with traffic patterns requiring aggressive caching.
//
// # Migration from api.Storage
//
// The api.Storage interface is DEPRECATED and will be removed in v2.0.0 (12 months from v1.8.0).
// Migrate to storage.Storage to gain context support and modern Go practices.
//
// Migration Timeline:
//   - v1.8.0: storage.Storage available, api.Storage deprecated
//   - v1.9.0: Runtime deprecation warnings added
//   - v1.10.0: Breaking change notices in logs
//   - v2.0.0: api.Storage removed completely
//
// Why migrate?
//   - api.Storage lacks context support (can't cancel or timeout operations)
//   - api.Storage is monolithic (must implement all methods even for read-only use)
//   - api.Storage creates import cycles when composing with other packages
//   - api.Storage lacks pagination (can't handle large datasets efficiently)
//
// Before (api.Storage):
//
//	func (s *MyStorage) GetModule(name string) (*api.Module, error) {
//		// No way to cancel or timeout this operation
//		return s.queryModule(name)
//	}
//
// After (storage.Storage):
//
//	func (s *MyStorage) GetModuleContext(ctx context.Context, name string) (*api.Module, error) {
//		// Context enables cancellation and timeouts
//		select {
//		case <-ctx.Done():
//			return nil, ctx.Err()
//		default:
//			return s.queryModule(ctx, name)
//		}
//	}
//
// See pkg/storage/DEPRECATION.md for complete migration guide.
//
// # StorageV2 Deprecation
//
// StorageV2 is also DEPRECATED in favor of the cleaner storage.Storage interface.
// StorageV2 embedded api.Storage for backward compatibility, creating unnecessary coupling.
// The new storage.Storage interface uses composition instead, following Go best practices.
//
// Migrate from StorageV2 to Storage by removing the api.Storage embedding and implementing
// only the focused sub-interfaces you need. See pkg/storage/MIGRATION.md for details.
//
// # Configuration
//
// Storage backends are configured through the Config struct:
//
//	config := storage.DefaultConfig()
//	config.Type = "postgres"
//	config.PostgresURL = "postgres://localhost/spoke"
//	config.PostgresMaxConns = 20
//	config.PostgresMinConns = 2
//	config.PostgresTimeout = 10 * time.Second
//
//	// Optional S3 for file content
//	config.S3Endpoint = "s3.amazonaws.com"
//	config.S3Region = "us-east-1"
//	config.S3Bucket = "spoke-proto-files"
//	config.S3AccessKey = "..."
//	config.S3SecretKey = "..."
//
//	// Optional Redis for caching
//	config.RedisURL = "redis://localhost:6379"
//	config.RedisPoolSize = 10
//	config.CacheEnabled = true
//	config.CacheTTL = map[string]time.Duration{
//		"module": 1 * time.Hour,
//		"version": 1 * time.Hour,
//	}
//
// # Usage Examples
//
// Basic filesystem storage:
//
//	// Create filesystem backend
//	storage, err := storage.NewFileSystemStorage("/var/spoke/data")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Create a module
//	ctx := context.Background()
//	module := &api.Module{
//		Name:        "user-service",
//		Description: "User management API",
//	}
//	err = storage.CreateModuleContext(ctx, module)
//
//	// List all modules
//	modules, err := storage.ListModulesContext(ctx)
//
//	// Get a specific module
//	module, err := storage.GetModuleContext(ctx, "user-service")
//
// Context with timeout:
//
//	// Enforce 5-second timeout on database query
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	version, err := storage.GetVersionContext(ctx, "user-service", "v1.0.0")
//	if err == context.DeadlineExceeded {
//		log.Println("Query timed out after 5 seconds")
//	}
//
// Pagination for large datasets:
//
//	// List modules 100 at a time
//	limit := 100
//	offset := 0
//	for {
//		modules, total, err := storage.ListModulesPaginated(ctx, limit, offset)
//		if err != nil {
//			return err
//		}
//
//		for _, module := range modules {
//			fmt.Println(module.Name)
//		}
//
//		offset += len(modules)
//		if offset >= int(total) {
//			break
//		}
//	}
//
// Content-addressed file storage:
//
//	// Store proto file content
//	content := strings.NewReader("syntax = \"proto3\"; message User {...}")
//	hash, err := storage.PutFileContent(ctx, content, "application/x-protobuf")
//	// hash is SHA256 of content, used for deduplication
//
//	// Retrieve proto file content
//	reader, err := storage.GetFileContent(ctx, hash)
//	defer reader.Close()
//	content, err := io.ReadAll(reader)
//
// Compiled artifact storage:
//
//	// Store compiled Go code
//	artifact := bytes.NewReader(compiledGoCode)
//	err = storage.PutCompiledArtifact(ctx, "user-service", "v1.0.0", "go", artifact)
//
//	// Retrieve compiled artifact
//	reader, err := storage.GetCompiledArtifact(ctx, "user-service", "v1.0.0", "go")
//	defer reader.Close()
//
// Health check:
//
//	// Check if storage backend is healthy
//	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
//	defer cancel()
//
//	if err := storage.HealthCheck(ctx); err != nil {
//		log.Printf("Storage unhealthy: %v", err)
//	}
//
// # Design Decisions
//
// Interface Segregation: Instead of one monolithic Storage interface, we compose
// focused sub-interfaces (ModuleReader, ModuleWriter, etc.). This follows the Interface
// Segregation Principle (SOLID), making it easier to:
//   - Mock only the methods needed for testing
//   - Implement read-only or write-only storage backends
//   - Compose functionality through embedding
//
// Context-First Design: All methods accept context.Context as the first parameter,
// following Go best practices established in the standard library (database/sql, net/http).
// This enables cancellation, timeouts, and tracing without breaking API compatibility.
//
// Content-Addressed Storage: Proto files are stored by SHA256 hash, enabling:
//   - Automatic deduplication (identical files stored once)
//   - Immutable content (hash changes if content changes)
//   - Efficient caching and CDN integration
//   - Verification of file integrity
//
// Separation of Metadata and Content: Module/version metadata is stored in the primary
// backend (PostgreSQL), while large content (proto files, compiled artifacts) can be
// offloaded to S3. This reduces database size and improves performance.
//
// Backend-Agnostic API: The API layer depends only on the Storage interface, not concrete
// implementations. This enables:
//   - Swapping backends without changing API code
//   - Testing with in-memory mock storage
//   - Gradual migration (filesystem → PostgreSQL → hybrid)
//
// # Performance Considerations
//
// Connection Pooling: PostgresStorage uses connection pooling with configurable min/max
// connections. Tune these based on concurrent request load:
//
//	config.PostgresMaxConns = 50  // Max concurrent queries
//	config.PostgresMinConns = 5   // Keep connections warm
//
// Caching: Enable Redis caching to reduce load on primary storage:
//
//	config.CacheEnabled = true
//	config.CacheTTL["version"] = 1 * time.Hour  // Cache versions for 1 hour
//
// Pagination: Always use paginated methods for list operations in production:
//
//	// Bad: Loads all modules into memory
//	modules, err := storage.ListModulesContext(ctx)
//
//	// Good: Processes modules in batches
//	modules, total, err := storage.ListModulesPaginated(ctx, 100, 0)
//
// Read Replicas: Configure PostgreSQL read replicas for high-read workloads:
//
//	config.PostgresReplicaURLs = "postgres://replica1:5432/spoke,postgres://replica2:5432/spoke"
//
// Timeouts: Always use context with timeout for production code:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//	module, err := storage.GetModuleContext(ctx, name)
//
// # File Organization
//
// The package is organized by concern:
//
//   - interfaces.go: Storage interface definitions and Config
//   - filesystem.go: FileSystemStorage implementation
//   - postgres/: PostgreSQL implementation (separate subpackage)
//   - redis/: Redis caching implementation (planned)
//   - s3/: S3 content storage implementation (planned)
//   - DEPRECATION.md: api.Storage deprecation timeline and migration guide
//   - MIGRATION.md: StorageV2 to Storage migration guide
//
// # Related Packages
//
//   - pkg/api: HTTP API layer that consumes storage.Storage
//   - pkg/codegen: Code generation that stores artifacts via ArtifactStorage
//   - pkg/compatibility: Schema validation that reads versions via VersionReader
//   - pkg/search: Search indexing that reads modules via ModuleReader
//   - pkg/analytics: Usage tracking that queries via custom SQL (for performance)
//
// # Testing
//
// Use interface mocking for unit tests:
//
//	type mockModuleReader struct {
//		storage.ModuleReader
//		getModuleFunc func(context.Context, string) (*api.Module, error)
//	}
//
//	func (m *mockModuleReader) GetModuleContext(ctx context.Context, name string) (*api.Module, error) {
//		return m.getModuleFunc(ctx, name)
//	}
//
//	// In test
//	mock := &mockModuleReader{
//		getModuleFunc: func(ctx context.Context, name string) (*api.Module, error) {
//			return &api.Module{Name: name}, nil
//		},
//	}
//
// For integration tests, use testcontainers to spin up real PostgreSQL:
//
//	postgres, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
//		ContainerRequest: testcontainers.ContainerRequest{
//			Image: "postgres:15",
//			Env:   map[string]string{"POSTGRES_PASSWORD": "test"},
//		},
//		Started: true,
//	})
package storage
