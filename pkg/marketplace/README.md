# Plugin Marketplace Package

The `marketplace` package provides a complete REST API and service layer for the Spoke Plugin Marketplace.

## Features

- **Plugin Discovery** - Search, filter, and browse plugins
- **Version Management** - Track multiple versions of plugins
- **Reviews & Ratings** - Community feedback system
- **Installation Tracking** - Monitor plugin adoption
- **Statistics** - Download and usage analytics
- **Plugin Submission** - API for publishing plugins

## Architecture

```
┌─────────────────────────────────────────────────┐
│                  HTTP Layer                     │
│              (handlers.go)                      │
│  GET /plugins, POST /plugins/{id}/reviews, etc │
└─────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────┐
│                Business Logic                   │
│               (service.go)                      │
│  ListPlugins, CreateReview, RecordDownload, etc│
└─────────────────────────────────────────────────┘
                      ↓
┌────────────────────┬────────────────────────────┐
│    Database        │      Storage               │
│  (PostgreSQL)      │  (Filesystem/S3)          │
│  - plugins         │  - plugin archives         │
│  - versions        │  - manifests               │
│  - reviews         │                            │
└────────────────────┴────────────────────────────┘
```

## Usage

### Basic Setup

```go
import (
    "database/sql"
    "github.com/gorilla/mux"
    "github.com/platinummonkey/spoke/pkg/marketplace"
)

// Initialize database
db, err := sql.Open("postgres", "postgres://user:pass@localhost:5432/spoke?sslmode=disable")
if err != nil {
    log.Fatal(err)
}

// Create storage backend
storage, err := marketplace.NewFileSystemStorage("/var/spoke/plugins", "https://cdn.spoke.dev")
if err != nil {
    log.Fatal(err)
}

// Create service
service := marketplace.NewService(db, storage)

// Create HTTP handlers
handlers := marketplace.NewHandlers(service)

// Register routes
router := mux.NewRouter()
handlers.RegisterRoutes(router)

// Start server
http.ListenAndServe(":8080", router)
```

### Service Layer

The service layer provides business logic:

```go
// List plugins with filters
plugins, err := service.ListPlugins(ctx, &marketplace.PluginListRequest{
    Type:      "language",
    SortBy:    "downloads",
    SortOrder: "desc",
    Limit:     20,
})

// Get plugin details
plugin, err := service.GetPlugin(ctx, "rust-language")

// Create a review
err = service.CreateReview(ctx, &marketplace.PluginReview{
    PluginID: "rust-language",
    UserID:   "user123",
    Rating:   5,
    Review:   "Excellent plugin!",
})

// Record download
err = service.RecordDownload(ctx, "rust-language", "1.2.0")
```

### Storage Interface

Implement custom storage backends:

```go
type MyStorage struct {
    // your storage implementation
}

func (s *MyStorage) StorePluginArchive(ctx context.Context, pluginID, version string, data io.Reader) (string, error) {
    // Upload to your storage
    url := "https://storage.example.com/" + pluginID + "/" + version + ".tar.gz"
    return url, nil
}

func (s *MyStorage) GetPluginArchive(ctx context.Context, pluginID, version string) (io.ReadCloser, error) {
    // Download from your storage
    return reader, nil
}

// Implement other Storage interface methods...
```

## Database Schema

Run migrations to create required tables:

```bash
# Up migration
psql -U user -d spoke -f migrations/010_plugin_marketplace.up.sql

# Down migration (rollback)
psql -U user -d spoke -f migrations/010_plugin_marketplace.down.sql
```

Tables created:
- `plugins` - Plugin registry
- `plugin_versions` - Version history
- `plugin_reviews` - User reviews and ratings
- `plugin_installations` - Installation tracking
- `plugin_stats_daily` - Aggregated statistics
- `plugin_dependencies` - Plugin dependencies
- `plugin_tags` - Tags for discovery

## API Endpoints

### Public Endpoints (No Auth Required)

```
GET    /api/v1/plugins                          - List plugins
GET    /api/v1/plugins/search?q=rust            - Search plugins
GET    /api/v1/plugins/trending                 - Trending plugins
GET    /api/v1/plugins/{id}                     - Get plugin details
GET    /api/v1/plugins/{id}/versions            - List versions
GET    /api/v1/plugins/{id}/versions/{version}  - Get version details
GET    /api/v1/plugins/{id}/versions/{version}/download - Download plugin
GET    /api/v1/plugins/{id}/reviews             - List reviews
GET    /api/v1/plugins/{id}/stats               - Get statistics
```

### Authenticated Endpoints

```
POST   /api/v1/plugins                          - Submit new plugin
POST   /api/v1/plugins/{id}/versions            - Submit new version
POST   /api/v1/plugins/{id}/reviews             - Create/update review
POST   /api/v1/plugins/{id}/install             - Record installation
POST   /api/v1/plugins/{id}/uninstall           - Record uninstallation
```

## Examples

### List Language Plugins

```bash
curl "http://localhost:8080/api/v1/plugins?type=language&limit=10"
```

Response:
```json
{
  "plugins": [
    {
      "id": "rust-language",
      "name": "Rust Language Plugin",
      "type": "language",
      "security_level": "verified",
      "download_count": 1523,
      "latest_version": "1.2.0",
      "avg_rating": 4.7
    }
  ],
  "total": 42,
  "limit": 10,
  "offset": 0
}
```

### Search Plugins

```bash
curl "http://localhost:8080/api/v1/plugins/search?q=rust"
```

### Get Plugin Details

```bash
curl "http://localhost:8080/api/v1/plugins/rust-language"
```

### Download Plugin

```bash
curl -L "http://localhost:8080/api/v1/plugins/rust-language/versions/1.2.0/download" -o plugin.tar.gz
```

### Submit Review

```bash
curl -X POST "http://localhost:8080/api/v1/plugins/rust-language/reviews" \
  -H "Authorization: Bearer token" \
  -H "Content-Type: application/json" \
  -d '{"rating": 5, "review": "Great plugin!"}'
```

## Testing

Run tests:

```bash
go test -v github.com/platinummonkey/spoke/pkg/marketplace
```

Run with coverage:

```bash
go test -cover github.com/platinummonkey/spoke/pkg/marketplace
```

## Configuration

Environment variables:

```bash
# Database
DATABASE_URL=postgres://user:pass@localhost:5432/spoke?sslmode=disable

# Storage
PLUGIN_STORAGE_TYPE=filesystem  # or "s3"
PLUGIN_STORAGE_PATH=/var/spoke/plugins
PLUGIN_STORAGE_URL=https://cdn.spoke.dev

# S3 (if using S3 storage)
AWS_S3_BUCKET=spoke-plugins
AWS_S3_REGION=us-east-1
AWS_S3_PREFIX=plugins/
```

## Security

- **Input Validation** - All inputs validated before database operations
- **SQL Injection Protection** - Parameterized queries used throughout
- **Rate Limiting** - Should be implemented at HTTP layer
- **Authentication** - Required for write operations
- **Plugin Verification** - Security levels: official, verified, community

## Monitoring

Key metrics to monitor:

- Total plugins registered
- Download rate (downloads/hour)
- Installation rate
- Average rating across plugins
- API response times
- Database query performance

Example queries:

```sql
-- Top 10 most downloaded plugins
SELECT id, name, download_count
FROM plugins
ORDER BY download_count DESC
LIMIT 10;

-- Plugins by security level
SELECT security_level, COUNT(*)
FROM plugins
GROUP BY security_level;

-- Average rating by plugin type
SELECT type, AVG(avg_rating)
FROM (
    SELECT p.id, p.type, AVG(r.rating) as avg_rating
    FROM plugins p
    LEFT JOIN plugin_reviews r ON p.id = r.plugin_id
    GROUP BY p.id
) subq
GROUP BY type;
```

## Performance

Optimization tips:

1. **Indexes** - Database indexes on frequently queried columns
2. **Caching** - Cache popular plugin metadata
3. **CDN** - Serve plugin archives from CDN
4. **Pagination** - Always use pagination for lists
5. **Query Optimization** - Use aggregates and joins efficiently

## Integration

### With Plugin Loader

```go
// Marketplace-aware plugin loader
loader := plugins.NewLoader(dirs, log)

// Fetch plugin from marketplace if not found locally
plugin, err := loader.LoadPlugin(ctx, "rust-language")
if err != nil {
    // Download from marketplace
    archive, err := marketplaceClient.DownloadPlugin(ctx, "rust-language", "1.2.0")
    if err != nil {
        return err
    }

    // Extract to local plugins directory
    err = extractArchive(archive, "~/.spoke/plugins/rust-language")
    if err != nil {
        return err
    }

    // Retry loading
    plugin, err = loader.LoadPlugin(ctx, "rust-language")
}
```

### With CLI

```bash
# Spoke CLI integration
spoke plugin search rust
spoke plugin install rust-language
spoke plugin list --installed
spoke plugin review rust-language --rating 5 --review "Excellent!"
```

## Roadmap

Future enhancements:

- [ ] Full-text search with Elasticsearch
- [ ] Webhooks for plugin events
- [ ] GraphQL API
- [ ] Plugin recommendations
- [ ] Dependency resolution
- [ ] Automated security scanning
- [ ] Plugin marketplace website

## See Also

- [Plugin API Documentation](../../docs/PLUGIN_API.md)
- [Database Schema](../../migrations/010_plugin_marketplace.up.sql)
- [Plugin Development Guide](../../docs/PLUGIN_DEVELOPMENT.md)
