# Search Query Syntax

## Overview

Spoke's advanced search supports a powerful query syntax that combines free-text search with specialized filters. The query parser recognizes filters in the format `key:value` and applies PostgreSQL Full-Text Search for relevance ranking.

## Basic Search

**Free-text search:**
```
user
```
Searches for "user" in all indexed fields (entity names, paths, descriptions, comments).

**Multiple terms (AND by default):**
```
user email
```
Searches for documents containing both "user" AND "email".

## Boolean Operators

### AND (default)
```
user AND email
```
Explicit AND operator (same as space-separated terms).

### OR
```
user OR email
```
Searches for documents containing either "user" OR "email".

### NOT
```
user NOT deleted
```
Searches for documents containing "user" but NOT "deleted".

**Complex expressions:**
```
(user OR admin) AND active NOT deleted
```
Boolean operators can be combined for complex queries.

## Filter Syntax

Filters use the format `key:value` or `key:"quoted value"` for values with spaces.

### Entity Type Filter

**Filter by proto entity type:**

```
entity:message
```
Searches only message definitions.

**Supported entity types:**
- `message` - Proto message definitions
- `field` - Message fields
- `enum` - Enum definitions
- `enum_value` - Enum values
- `service` - gRPC service definitions
- `method` - RPC methods

**Examples:**
```
User entity:message          # Find messages named "User"
Status entity:enum           # Find enums named "Status"
CreateUser entity:method     # Find methods named "CreateUser"
email entity:field           # Find fields named "email"
```

### Field Type Filter

**Filter by protobuf field type:**

```
type:string
```
Searches only string fields.

**Common field types:**
- Scalars: `string`, `int32`, `int64`, `uint32`, `uint64`, `bool`, `float`, `double`, `bytes`
- Well-known types: `google.protobuf.Timestamp`, `google.protobuf.Duration`, etc.
- Custom message types: `User`, `Order`, etc.

**Examples:**
```
id type:string               # Find string fields named "id"
timestamp type:int64          # Find int64 fields named "timestamp"
active type:bool             # Find boolean fields named "active"
```

### Module Filter

**Filter by module name:**

```
module:user
```
Searches only in the "user" module.

**Wildcard support:**
```
module:common.*              # Modules starting with "common"
module:*-service             # Modules ending with "-service"
```

**Examples:**
```
User module:user             # Find "User" in user module
Status module:common         # Find "Status" in common module
```

### Version Filter

**Filter by exact version:**

```
version:v1.0.0
```
Searches only in version v1.0.0.

**Version constraints (coming soon):**
```
version:>=1.0.0              # Versions >= 1.0.0
version:~1.2.0               # Versions compatible with 1.2.0
version:<2.0.0               # Versions < 2.0.0
```

**Examples:**
```
User version:v1.0.0          # Find "User" in v1.0.0
CreateUser version:>=1.5.0   # Find "CreateUser" in v1.5.0+
```

### Import Filter (coming soon)

**Filter by proto file imports:**

```
imports:common.proto
```
Finds entities in files that import "common.proto".

**Examples:**
```
User imports:timestamp.proto  # Find "User" in files importing timestamp.proto
```

### Dependency Filter (coming soon)

**Filter by module dependencies:**

```
depends-on:common
```
Finds entities in modules that depend on the "common" module.

**Examples:**
```
User depends-on:common       # Find "User" in modules depending on common
Status depends-on:types      # Find "Status" in modules depending on types
```

### Comment Filter

**Filter by comment presence:**

```
has-comment:true
```
Finds only entities with comments/documentation.

**Examples:**
```
deprecated has-comment:true   # Find documented entities mentioning "deprecated"
TODO has-comment:true         # Find entities with TODO comments
```

## Combined Filters

Combine multiple filters for precise searches:

```
email entity:field type:string module:user
```
Finds string fields named "email" in the user module.

```
CreateUser entity:method version:>=1.0.0
```
Finds methods named "CreateUser" in versions >= 1.0.0.

```
Status entity:enum module:common.* has-comment:true
```
Finds documented enums named "Status" in modules starting with "common".

## Quoted Values

Use quotes for values containing spaces or special characters:

```
module:"user-service"
description:"user profile"
```

## API Endpoints

### Advanced Search

**Endpoint:** `GET /api/v2/search`

**Query Parameters:**
- `q` (required) - Search query with filters
- `limit` (optional) - Max results (default: 50, max: 1000)
- `offset` (optional) - Pagination offset (default: 0)

**Example Request:**
```bash
curl "http://localhost:8080/api/v2/search?q=email+entity:field+type:string&limit=20"
```

**Response:**
```json
{
  "results": [
    {
      "id": 1,
      "entity_type": "field",
      "entity_name": "email",
      "full_path": "user.v1.User.email",
      "parent_path": "user.v1.User",
      "module_name": "user",
      "version": "v1.0.0",
      "proto_file_path": "user.proto",
      "line_number": 15,
      "description": "Email address",
      "field_type": "string",
      "field_number": 2,
      "rank": 0.5
    }
  ],
  "total_count": 5,
  "query": "email entity:field type:string",
  "parsed_query": {
    "terms": ["email"],
    "entity_types": ["field"],
    "field_types": ["string"]
  }
}
```

### Search Suggestions

**Endpoint:** `GET /api/v2/search/suggestions`

**Query Parameters:**
- `prefix` (required) - Query prefix for autocomplete
- `limit` (optional) - Max suggestions (default: 5, max: 20)

**Example Request:**
```bash
curl "http://localhost:8080/api/v2/search/suggestions?prefix=user&limit=5"
```

**Response:**
```json
{
  "prefix": "user",
  "suggestions": [
    "user entity:message",
    "user email",
    "user service"
  ]
}
```

## Query Examples

### By Entity Type

**Find all messages:**
```
entity:message
```

**Find messages named "User":**
```
User entity:message
```

**Find all services:**
```
entity:service
```

### By Field Type

**Find all string fields:**
```
entity:field type:string
```

**Find ID fields (any type):**
```
id entity:field
```

**Find timestamp fields:**
```
timestamp entity:field type:int64
```

### By Module

**Find everything in user module:**
```
module:user
```

**Find User messages in common modules:**
```
User entity:message module:common.*
```

### Complex Queries

**Find string fields in user module:**
```
entity:field type:string module:user
```

**Find documented services:**
```
entity:service has-comment:true
```

**Find CreateUser methods in v1+ versions:**
```
CreateUser entity:method version:>=1.0.0
```

**Find email OR username fields:**
```
(email OR username) entity:field
```

## Relevance Ranking

Search results are ranked by relevance using PostgreSQL's `ts_rank` function:

- **Weight A (highest)**: Entity name
- **Weight B**: Full path
- **Weight C**: Description, field types, method types
- **Weight D (lowest)**: Comments

Example: Searching for "user" will rank entities named "user" higher than entities mentioning "user" in comments.

## Best Practices

### 1. Start Broad, Then Refine

Start with a simple search:
```
user
```

Add filters to narrow results:
```
user entity:message
user entity:message module:common
```

### 2. Use Entity Type Filters

If you know what you're looking for:
```
entity:field             # Instead of searching all entities
entity:service           # Narrows to just services
```

### 3. Combine Filters for Precision

Multiple filters work together (AND logic):
```
email entity:field type:string module:user version:v1.0.0
```

### 4. Use Wildcards for Module Patterns

Find entities across related modules:
```
User module:common.*     # All common-* modules
Status module:*-api      # All *-api modules
```

### 5. Leverage Suggestions

Use the suggestions endpoint for autocomplete:
- Suggests popular queries based on search history
- Helps discover common search patterns
- Shows frequently accessed entities

## Advanced Tips

### Prefix Matching

All search terms use prefix matching (`user` matches `users`, `username`, etc.). This is implemented with PostgreSQL's `:*` suffix.

### Case Insensitivity

All searches are case-insensitive (PostgreSQL's `english` configuration).

### Performance

- **Simple queries**: <50ms
- **Complex queries with multiple filters**: <200ms (p95)
- **Pagination**: Use `limit` and `offset` for large result sets
- **Suggestions**: Cached in materialized view, <10ms

### Debugging

Check the `parsed_query` field in the response to see how your query was interpreted:

```json
{
  "parsed_query": {
    "terms": ["user", "email"],
    "entity_types": ["field"],
    "field_types": ["string"],
    "module_pattern": "common.*"
  }
}
```

## Limitations

- **No regex**: Use wildcard patterns (`*`) instead
- **No fuzzy matching**: Exact prefix matching only
- **No cross-module joins**: Search one module at a time or use wildcards
- **Version constraints**: Currently exact match only (coming soon: >=, ~, <)

## Future Enhancements

- Semantic version constraint parsing (`>=1.0.0`, `~1.2.0`)
- Import and dependency filters
- Fuzzy matching for typo tolerance
- Cross-field ranking (boost recent entities, popular entities)
- Search highlighting in results
- Saved search queries
- Search analytics and trending queries

## References

- [PostgreSQL Full-Text Search](https://www.postgresql.org/docs/current/textsearch.html)
- [Query Parser Implementation](../pkg/search/query_parser.go)
- [Search Service Implementation](../pkg/search/search_service.go)
- [Search API Handlers](../pkg/api/search_handlers.go)
