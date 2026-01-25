package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/platinummonkey/spoke/pkg/api"
	"github.com/platinummonkey/spoke/pkg/storage"
)

// PostgresStorage implements StorageV2 using PostgreSQL + S3 + Redis
type PostgresStorage struct {
	db          *sql.DB
	s3Client    *S3Client
	redisClient *RedisClient
	config      storage.Config
}

// NewPostgresStorage creates a new PostgreSQL-backed storage
func NewPostgresStorage(config storage.Config) (*PostgresStorage, error) {
	// Connect to PostgreSQL
	db, err := sql.Open("postgres", config.PostgresURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.PostgresMaxConns)
	db.SetMaxIdleConns(config.PostgresMinConns)
	db.SetConnMaxLifetime(1 * time.Hour)
	db.SetConnMaxIdleTime(10 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), config.PostgresTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

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
		return fmt.Errorf("failed to create module: %w", err)
	}

	// Invalidate cache
	if s.redisClient != nil {
		s.redisClient.InvalidateModule(ctx, module.Name)
	}

	return nil
}

func (s *PostgresStorage) GetModuleContext(ctx context.Context, name string) (*api.Module, error) {
	// Check cache first
	if s.redisClient != nil {
		if module, err := s.redisClient.GetModule(ctx, name); err == nil && module != nil {
			return module, nil
		}
	}

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
		return nil, fmt.Errorf("module not found: %s", name)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get module: %w", err)
	}

	// Cache result
	if s.redisClient != nil {
		s.redisClient.SetModule(ctx, &module)
	}

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
	// TODO: Implement version creation with file upload to S3
	return fmt.Errorf("not implemented")
}

func (s *PostgresStorage) GetVersionContext(ctx context.Context, moduleName, version string) (*api.Version, error) {
	// TODO: Implement version retrieval with file download from S3
	return nil, fmt.Errorf("not implemented")
}

func (s *PostgresStorage) ListVersionsContext(ctx context.Context, moduleName string) ([]*api.Version, error) {
	// TODO: Implement version listing
	return nil, fmt.Errorf("not implemented")
}

func (s *PostgresStorage) UpdateVersionContext(ctx context.Context, version *api.Version) error {
	// TODO: Implement version update
	return fmt.Errorf("not implemented")
}

func (s *PostgresStorage) GetFileContext(ctx context.Context, moduleName, version, path string) (*api.File, error) {
	// TODO: Implement file retrieval from S3
	return nil, fmt.Errorf("not implemented")
}

func (s *PostgresStorage) ListVersionsPaginated(ctx context.Context, moduleName string, limit, offset int) ([]*api.Version, int64, error) {
	// TODO: Implement
	return nil, 0, fmt.Errorf("not implemented")
}

func (s *PostgresStorage) GetFileContent(ctx context.Context, hash string) (io.ReadCloser, error) {
	// TODO: Implement S3 retrieval by hash
	return nil, fmt.Errorf("not implemented")
}

func (s *PostgresStorage) PutFileContent(ctx context.Context, content io.Reader, contentType string) (hash string, err error) {
	// TODO: Implement S3 upload with hash calculation
	return "", fmt.Errorf("not implemented")
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
