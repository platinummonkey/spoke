package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/platinummonkey/spoke/pkg/api"
	"github.com/platinummonkey/spoke/pkg/storage"
)

var tracer = otel.Tracer("spoke/storage/postgres")

// PostgresStorage implements storage.Storage using PostgreSQL + S3 + Redis
type PostgresStorage struct {
	connManager *ConnectionManager
	db          *sql.DB // Deprecated: use connManager.Primary() instead
	s3Client    *S3Client
	redisClient *RedisClient
	config      storage.Config
}

// NewPostgresStorage creates a new PostgreSQL-backed storage
func NewPostgresStorage(config storage.Config) (*PostgresStorage, error) {
	// Initialize connection manager with primary and replicas
	connConfig := ConnectionConfig{
		PrimaryURL:  config.PostgresURL,
		ReplicaURLs: ParseReplicaURLs(config.PostgresReplicaURLs),
		MaxConns:    config.PostgresMaxConns,
		MinConns:    config.PostgresMinConns,
		Timeout:     config.PostgresTimeout,
		MaxLifetime: 1 * time.Hour,
		MaxIdleTime: 10 * time.Minute,
	}

	connManager, err := NewConnectionManager(connConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	// Get primary connection for backward compatibility
	db := connManager.Primary()

	// TODO: Initialize S3 client
	var s3Client *S3Client
	if config.S3Endpoint != "" {
		s3Client, err = NewS3Client(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create s3 client: %w", err)
		}
	}

	// TODO: Initialize Redis client
	var redisClient *RedisClient
	if config.CacheEnabled && config.RedisURL != "" {
		redisClient, err = NewRedisClient(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create redis client: %w", err)
		}
	}

	return &PostgresStorage{
		connManager: connManager,
		db:          db,
		s3Client:    s3Client,
		redisClient: redisClient,
		config:      config,
	}, nil
}

// Backward-compatible methods that delegate to context-aware versions

func (s *PostgresStorage) CreateModule(module *api.Module) error {
	return s.CreateModuleContext(context.Background(), module)
}

func (s *PostgresStorage) GetModule(name string) (*api.Module, error) {
	return s.GetModuleContext(context.Background(), name)
}

func (s *PostgresStorage) ListModules() ([]*api.Module, error) {
	return s.ListModulesContext(context.Background())
}

func (s *PostgresStorage) CreateVersion(version *api.Version) error {
	return s.CreateVersionContext(context.Background(), version)
}

func (s *PostgresStorage) GetVersion(moduleName, version string) (*api.Version, error) {
	return s.GetVersionContext(context.Background(), moduleName, version)
}

func (s *PostgresStorage) ListVersions(moduleName string) ([]*api.Version, error) {
	return s.ListVersionsContext(context.Background(), moduleName)
}

func (s *PostgresStorage) UpdateVersion(version *api.Version) error {
	return s.UpdateVersionContext(context.Background(), version)
}

func (s *PostgresStorage) GetFile(moduleName, version, path string) (*api.File, error) {
	return s.GetFileContext(context.Background(), moduleName, version, path)
}

// Context-aware implementations

func (s *PostgresStorage) CreateModuleContext(ctx context.Context, module *api.Module) error {
	ctx, span := tracer.Start(ctx, "CreateModule",
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.operation", "INSERT"),
			attribute.String("db.table", "modules"),
			attribute.String("module.name", module.Name),
		),
	)
	defer span.End()

	query := `
		INSERT INTO modules (name, description, metadata)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRowContext(ctx, query,
		module.Name,
		module.Description,
		"{}",
	).Scan(&module.CreatedAt, &module.UpdatedAt)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create module")
		return fmt.Errorf("failed to create module: %w", err)
	}

	// Invalidate cache
	if s.redisClient != nil {
		s.redisClient.InvalidateModule(ctx, module.Name)
	}

	span.SetStatus(codes.Ok, "module created successfully")
	return nil
}

func (s *PostgresStorage) GetModuleContext(ctx context.Context, name string) (*api.Module, error) {
	ctx, span := tracer.Start(ctx, "GetModule",
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "modules"),
			attribute.String("module.name", name),
		),
	)
	defer span.End()

	// Check cache first
	if s.redisClient != nil {
		if module, err := s.redisClient.GetModule(ctx, name); err == nil && module != nil {
			span.SetAttributes(attribute.Bool("cache.hit", true))
			span.SetStatus(codes.Ok, "module retrieved from cache")
			return module, nil
		}
	}
	span.SetAttributes(attribute.Bool("cache.hit", false))

	query := `
		SELECT name, description, created_at, updated_at
		FROM modules
		WHERE name = $1
	`

	var module api.Module
	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&module.Name,
		&module.Description,
		&module.CreatedAt,
		&module.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		span.SetStatus(codes.Error, "module not found")
		return nil, fmt.Errorf("module not found: %s", name)
	} else if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get module")
		return nil, fmt.Errorf("failed to get module: %w", err)
	}

	// Cache result
	if s.redisClient != nil {
		s.redisClient.SetModule(ctx, &module)
	}

	span.SetStatus(codes.Ok, "module retrieved from database")
	return &module, nil
}

func (s *PostgresStorage) ListModulesContext(ctx context.Context) ([]*api.Module, error) {
	modules, _, err := s.ListModulesPaginated(ctx, 1000, 0)
	return modules, err
}

func (s *PostgresStorage) ListModulesPaginated(ctx context.Context, limit, offset int) ([]*api.Module, int64, error) {
	// Count total
	var total int64
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM modules").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count modules: %w", err)
	}

	// Query page
	query := `
		SELECT name, description, created_at, updated_at
		FROM modules
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list modules: %w", err)
	}
	defer rows.Close()

	var modules []*api.Module
	for rows.Next() {
		var m api.Module
		err := rows.Scan(&m.Name, &m.Description, &m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan module: %w", err)
		}
		modules = append(modules, &m)
	}

	return modules, total, nil
}

// Placeholder implementations for remaining methods

func (s *PostgresStorage) CreateVersionContext(ctx context.Context, version *api.Version) error {
	ctx, span := tracer.Start(ctx, "CreateVersion",
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.operation", "INSERT"),
			attribute.String("db.table", "versions"),
			attribute.String("module.name", version.ModuleName),
			attribute.String("version", version.Version),
			attribute.Int("file.count", len(version.Files)),
		),
	)
	defer span.End()

	if s.s3Client == nil {
		span.SetStatus(codes.Error, "s3 client not initialized")
		return fmt.Errorf("s3 client not initialized")
	}

	// Get module ID
	var moduleID int64
	err := s.db.QueryRowContext(ctx, "SELECT id FROM modules WHERE name = $1", version.ModuleName).Scan(&moduleID)
	if err == sql.ErrNoRows {
		span.SetStatus(codes.Error, "module not found")
		return fmt.Errorf("module not found: %s", version.ModuleName)
	} else if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get module")
		return fmt.Errorf("failed to get module: %w", err)
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to start transaction")
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert version
	var versionID int64
	versionQuery := `
		INSERT INTO versions (module_id, version, dependencies, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	now := time.Now()
	if version.CreatedAt.IsZero() {
		version.CreatedAt = now
	}

	// Convert dependencies to JSON array
	depsJSON := "[]"
	if len(version.Dependencies) > 0 {
		depsJSON = fmt.Sprintf(`["%s"]`, version.Dependencies[0])
		for i := 1; i < len(version.Dependencies); i++ {
			depsJSON = depsJSON[:len(depsJSON)-1] + fmt.Sprintf(`, "%s"]`, version.Dependencies[i])
		}
	}

	err = tx.QueryRowContext(ctx, versionQuery,
		moduleID,
		version.Version,
		depsJSON,
		version.CreatedAt,
		now,
	).Scan(&versionID)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to insert version")
		return fmt.Errorf("failed to insert version: %w", err)
	}

	// Upload files to S3 and insert metadata
	fileQuery := `
		INSERT INTO proto_files (version_id, file_path, content_hash, object_key, file_size, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, file := range version.Files {
		// Upload to S3 using content-addressable storage
		contentBytes := []byte(file.Content)
		hash, err := s.s3Client.PutObjectWithHash(ctx, contentBytes, "application/x-protobuf")
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to upload file to s3")
			return fmt.Errorf("failed to upload file %s to s3: %w", file.Path, err)
		}

		// Construct S3 object key
		objectKey := fmt.Sprintf("proto-files/sha256/%s/%s", hash[:2], hash[2:])

		// Insert file metadata
		_, err = tx.ExecContext(ctx, fileQuery,
			versionID,
			file.Path,
			hash,
			objectKey,
			len(contentBytes),
			now,
		)

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to insert file metadata")
			return fmt.Errorf("failed to insert file metadata for %s: %w", file.Path, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to commit transaction")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate cache
	if s.redisClient != nil {
		s.redisClient.InvalidateVersion(ctx, version.ModuleName, version.Version)
	}

	span.SetStatus(codes.Ok, "version created successfully")
	return nil
}

func (s *PostgresStorage) GetVersionContext(ctx context.Context, moduleName, version string) (*api.Version, error) {
	ctx, span := tracer.Start(ctx, "GetVersion",
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "versions"),
			attribute.String("module.name", moduleName),
			attribute.String("version", version),
		),
	)
	defer span.End()

	if s.s3Client == nil {
		span.SetStatus(codes.Error, "s3 client not initialized")
		return nil, fmt.Errorf("s3 client not initialized")
	}

	// Check cache first
	if s.redisClient != nil {
		if cachedVersion, err := s.redisClient.GetVersion(ctx, moduleName, version); err == nil && cachedVersion != nil {
			span.SetAttributes(attribute.Bool("cache.hit", true))
			span.SetStatus(codes.Ok, "version retrieved from cache")
			return cachedVersion, nil
		}
	}
	span.SetAttributes(attribute.Bool("cache.hit", false))

	// Get version metadata
	query := `
		SELECT v.id, v.version, v.dependencies, v.created_at, v.updated_at
		FROM versions v
		JOIN modules m ON v.module_id = m.id
		WHERE m.name = $1 AND v.version = $2
	`

	var versionID int64
	var depsJSON string
	var createdAt, updatedAt time.Time
	var versionStr string

	err := s.db.QueryRowContext(ctx, query, moduleName, version).Scan(
		&versionID,
		&versionStr,
		&depsJSON,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		span.SetStatus(codes.Error, "version not found")
		return nil, fmt.Errorf("version not found: %s@%s", moduleName, version)
	} else if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get version")
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	// Parse dependencies (simple JSON array parsing)
	var dependencies []string
	// TODO: Use proper JSON parsing for dependencies

	// Get file metadata
	fileQuery := `
		SELECT file_path, content_hash, object_key
		FROM proto_files
		WHERE version_id = $1
		ORDER BY file_path
	`

	rows, err := s.db.QueryContext(ctx, fileQuery, versionID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to query files")
		return nil, fmt.Errorf("failed to query files: %w", err)
	}
	defer rows.Close()

	var files []api.File
	for rows.Next() {
		var filePath, contentHash, objectKey string
		if err := rows.Scan(&filePath, &contentHash, &objectKey); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to scan file metadata")
			return nil, fmt.Errorf("failed to scan file metadata: %w", err)
		}

		// Download file content from S3
		reader, err := s.s3Client.GetObject(ctx, objectKey)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to download file from s3")
			return nil, fmt.Errorf("failed to download file %s from s3: %w", filePath, err)
		}

		contentBytes, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to read file content")
			return nil, fmt.Errorf("failed to read file %s content: %w", filePath, err)
		}

		files = append(files, api.File{
			Path:    filePath,
			Content: string(contentBytes),
		})
	}

	if err := rows.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "error iterating files")
		return nil, fmt.Errorf("error iterating files: %w", err)
	}

	result := &api.Version{
		ModuleName:   moduleName,
		Version:      versionStr,
		Files:        files,
		CreatedAt:    createdAt,
		Dependencies: dependencies,
	}

	// Cache result
	if s.redisClient != nil {
		s.redisClient.SetVersion(ctx, result)
	}

	span.SetAttributes(attribute.Int("file.count", len(files)))
	span.SetStatus(codes.Ok, "version retrieved from database")
	return result, nil
}

func (s *PostgresStorage) ListVersionsContext(ctx context.Context, moduleName string) ([]*api.Version, error) {
	// Use replica for read-only query
	query := `
		SELECT v.version, v.dependencies, v.created_at
		FROM versions v
		JOIN modules m ON v.module_id = m.id
		WHERE m.name = $1
		ORDER BY v.created_at DESC
	`

	rows, err := s.replica().QueryContext(ctx, query, moduleName)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}
	defer rows.Close()

	var versions []*api.Version
	for rows.Next() {
		var v api.Version
		var depsJSON string

		err := rows.Scan(&v.Version, &depsJSON, &v.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}

		v.ModuleName = moduleName
		// TODO: Parse dependencies JSON properly
		versions = append(versions, &v)
	}

	return versions, nil
}

func (s *PostgresStorage) UpdateVersionContext(ctx context.Context, version *api.Version) error {
	// TODO: Implement version update
	return fmt.Errorf("not implemented")
}

func (s *PostgresStorage) GetFileContext(ctx context.Context, moduleName, version, path string) (*api.File, error) {
	if s.s3Client == nil {
		return nil, fmt.Errorf("s3 client not initialized")
	}

	// Query for file metadata
	query := `
		SELECT pf.content_hash, pf.object_key
		FROM proto_files pf
		JOIN versions v ON pf.version_id = v.id
		JOIN modules m ON v.module_id = m.id
		WHERE m.name = $1 AND v.version = $2 AND pf.file_path = $3
	`

	var contentHash, objectKey string
	err := s.db.QueryRowContext(ctx, query, moduleName, version, path).Scan(&contentHash, &objectKey)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("file not found: %s@%s:%s", moduleName, version, path)
	} else if err != nil {
		return nil, fmt.Errorf("failed to query file: %w", err)
	}

	// Download from S3
	reader, err := s.s3Client.GetObject(ctx, objectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to download file from s3: %w", err)
	}
	defer reader.Close()

	contentBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	return &api.File{
		Path:    path,
		Content: string(contentBytes),
	}, nil
}

func (s *PostgresStorage) ListVersionsPaginated(ctx context.Context, moduleName string, limit, offset int) ([]*api.Version, int64, error) {
	// TODO: Implement
	return nil, 0, fmt.Errorf("not implemented")
}

func (s *PostgresStorage) GetFileContent(ctx context.Context, hash string) (io.ReadCloser, error) {
	if s.s3Client == nil {
		return nil, fmt.Errorf("s3 client not initialized")
	}

	// Construct S3 key from hash (content-addressable)
	key := fmt.Sprintf("proto-files/sha256/%s/%s", hash[:2], hash[2:])

	return s.s3Client.GetObject(ctx, key)
}

func (s *PostgresStorage) PutFileContent(ctx context.Context, content io.Reader, contentType string) (hash string, err error) {
	if s.s3Client == nil {
		return "", fmt.Errorf("s3 client not initialized")
	}

	// Read content to calculate hash
	contentBytes, err := io.ReadAll(content)
	if err != nil {
		return "", fmt.Errorf("failed to read content: %w", err)
	}

	// Upload using content-addressable storage
	return s.s3Client.PutObjectWithHash(ctx, contentBytes, contentType)
}

func (s *PostgresStorage) GetCompiledArtifact(ctx context.Context, moduleName, version, language string) (io.ReadCloser, error) {
	// TODO: Implement
	return nil, fmt.Errorf("not implemented")
}

func (s *PostgresStorage) PutCompiledArtifact(ctx context.Context, moduleName, version, language string, artifact io.Reader) error {
	// TODO: Implement
	return fmt.Errorf("not implemented")
}

func (s *PostgresStorage) InvalidateCache(ctx context.Context, patterns ...string) error {
	if s.redisClient == nil {
		return nil
	}
	return s.redisClient.InvalidatePatterns(ctx, patterns...)
}

func (s *PostgresStorage) HealthCheck(ctx context.Context) error {
	// Check PostgreSQL
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("postgres unhealthy: %w", err)
	}

	// Check S3
	if s.s3Client != nil {
		if err := s.s3Client.HealthCheck(ctx); err != nil {
			return fmt.Errorf("s3 unhealthy: %w", err)
		}
	}

	// Check Redis
	if s.redisClient != nil {
		if err := s.redisClient.Ping(ctx); err != nil {
			return fmt.Errorf("redis unhealthy: %w", err)
		}
	}

	return nil
}

// GetDB returns the primary database connection for health checks
func (s *PostgresStorage) GetDB() *sql.DB {
	return s.db
}

// GetRedis returns the Redis client (may be nil if not configured)
func (s *PostgresStorage) GetRedis() *RedisClient {
	return s.redisClient
}

// GetConnectionManager returns the connection manager
func (s *PostgresStorage) GetConnectionManager() *ConnectionManager {
	return s.connManager
}

// primary returns the primary database connection (for writes)
func (s *PostgresStorage) primary() *sql.DB {
	return s.connManager.Primary()
}

// replica returns a read replica connection (for reads)
// Falls back to primary if no replicas available
func (s *PostgresStorage) replica() *sql.DB {
	return s.connManager.Replica()
}

// Close closes all connections
func (s *PostgresStorage) Close() error {
	if s.db != nil {
		s.db.Close()
	}
	if s.redisClient != nil {
		s.redisClient.Close()
	}
	return nil
}

// Verify that PostgresStorage implements storage.Storage at compile time
var _ storage.Storage = (*PostgresStorage)(nil)
