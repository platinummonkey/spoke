---
title: "Search"
weight: 5
---

# Full-Text Search

Spoke's search feature provides fast, comprehensive searching across all modules, messages, services, methods, fields, and enums using client-side full-text indexing.

## Quick Start

### Opening Search

Press **CMD+K** (Mac) or **CTRL+K** (Windows/Linux) anywhere in the web UI to open the search modal.

### Closing Search

- Press **ESC** to close
- Click outside the modal
- Click a search result (navigates and closes)

### Basic Search

1. Press **CMD+K**
2. Type your query (e.g., "User")
3. Results appear instantly
4. Click a result to navigate to that module

## Search Interface

### Search Modal

```
┌─────────────────────────────────────────────┐
│  🔍  Search...                       ⌘K     │
├─────────────────────────────────────────────┤
│                                             │
│  Found 5 results for "user"                 │
│                                             │
│  📦 user-service v1.0.0                     │
│     User management service                 │
│     Matched: name, services, messages       │
│     Messages: User, CreateUserRequest...    │
│                                             │
│  📦 order-service v1.0.0                    │
│     Order processing service                │
│     Matched: fields                         │
│     Fields: user_id, user_email...          │
│                                             │
├─────────────────────────────────────────────┤
│  Press ESC to close • Click to view details │
└─────────────────────────────────────────────┘
```

### Search Bar

The search bar includes:
- **Search icon** (🔍): Visual indicator
- **Input field**: Type your query
- **Keyboard hint** (⌘K): Shows shortcut
- **Loading spinner**: Appears during search

### Results List

Each result shows:
- **Module name**: Bold, clickable
- **Version**: Badge on the right
- **Description**: First line of module description
- **Matched fields**: Color-coded badges
- **Preview**: Snippet of matched content

## What Can You Search?

### Module Names

```
Query: "user"
Matches:
- user-service
- user-management
- multi-user-system
```

### Descriptions

```
Query: "authentication"
Matches:
- user-service: "User management and authentication"
- auth-service: "Authentication and authorization"
```

### Message Types

```
Query: "CreateUserRequest"
Matches:
- user-service: messages include CreateUserRequest
- admin-service: messages include CreateUserRequest
```

### Service Names

```
Query: "UserService"
Matches:
- user-service: services include UserService
- api-gateway: services include UserService
```

### Methods

```
Query: "GetUser"
Matches:
- user-service: methods include GetUser
- cache-service: methods include GetUser
```

### Field Names

```
Query: "email"
Matches:
- user-service: fields include email, user_email
- notification-service: fields include recipient_email
```

### Enum Types

```
Query: "Status"
Matches:
- user-service: enums include UserStatus
- order-service: enums include OrderStatus
```

## Search Features

### Fuzzy Matching

Search uses fuzzy matching for typos and variations:

```
Query: "usr"
Matches: user, users, UserService
```

```
Query: "servce"
Matches: service, UserService, OrderService
```

### Case-Insensitive

All searches are case-insensitive:

```
Query: "user"     → Matches: User, USER, user
Query: "API"      → Matches: api, Api, API
Query: "GetUser"  → Matches: getUser, GETUSER
```

### Partial Matching

Match parts of words:

```
Query: "order"
Matches:
- order-service
- user-order-history
- OrderStatus
- CreateOrderRequest
```

### Multi-Word Search

Search multiple terms:

```
Query: "user create"
Matches:
- CreateUserRequest
- user-service (create methods)
- CreateOrderRequest (user_id field)
```

### Field-Specific Results

Results are organized by where matches occurred:

```
📦 user-service v1.0.0
   User management and authentication

   Matched: name, services, messages

   Messages: User, CreateUserRequest, GetUserRequest
   Services: UserService, AuthService
```

**Matched badges show:**
- **name** (blue): Module name matched
- **description** (green): Description matched
- **messages** (purple): Message types matched
- **services** (orange): Service names matched
- **methods** (pink): RPC methods matched
- **enums** (teal): Enum types matched
- **fields** (cyan): Field names matched

### Result Previews

For specific matches, previews show:

**Messages matched:**
```
Messages: User, CreateUserRequest, GetUserRequest...
```

**Services matched:**
```
Services: UserService, AuthService...
```

**Methods matched:**
```
Methods: CreateUser, GetUser, UpdateUser...
```

Shows up to 3 items, then "..." for more.

### Result Highlighting

Matching terms are highlighted in yellow:

```
📦 user-service v1.0.0
   ^^^^
   User management service
   ^^^^
```

The highlight makes it easy to see why a result matched.

## Search Performance

### Speed

- **Response time**: <100ms for most queries
- **Index size**: ~5-10MB for typical registries
- **Indexing**: Done once on page load
- **Re-indexing**: Automatic on page refresh

### Result Ranking

Results are ranked by relevance using field boosting:

| Field | Boost | Priority |
|-------|-------|----------|
| Module name | 10x | Highest |
| Description | 5x | High |
| Messages | 3x | Medium-High |
| Services | 3x | Medium-High |
| Methods | 2x | Medium |
| Enums | 2x | Medium |
| Fields | 1x | Low |

**Example:**

Query: "user"

1. **user-service** (name match, 10x boost)
2. **user-management** (name match, 10x boost)
3. **auth-service** (description contains "user", 5x boost)
4. **order-service** (has User message, 3x boost)
5. **api-gateway** (has user_id field, 1x boost)

### Result Limits

- **Top 10 results** shown
- Most relevant matches first
- More results available via scrolling
- Use more specific queries to narrow results

## Advanced Search Techniques

### Exact Phrases

Use quotes for exact matches:

```
Query: "UserService"
Matches: Only exact "UserService", not "User" or "Service"
```

(Note: Not yet implemented, coming soon)

### Wildcards

Use * for wildcard matching:

```
Query: "User*"
Matches: User, UserService, UserManager, UserProfile
```

(Note: Not yet implemented, coming soon)

### Field-Specific Search

Target specific fields:

```
Query: "service:User"
Matches: Only services named User*
```

(Note: Not yet implemented, coming soon)

## Common Search Patterns

### Finding a Specific Service

**Goal**: Find where UserService is defined

```
1. Press CMD+K
2. Type "UserService"
3. Look for results with "services" badge
4. Click to navigate
```

### Finding All Uses of a Message

**Goal**: See which modules use the User message

```
1. Press CMD+K
2. Type "User message"
3. Review all results
4. Check "messages" badges
```

### Finding Methods by Name

**Goal**: Find all CreateUser methods across modules

```
1. Press CMD+K
2. Type "CreateUser"
3. Look for "methods" badges
4. Compare implementations
```

### Finding Fields Across Schemas

**Goal**: See which messages have an "email" field

```
1. Press CMD+K
2. Type "email"
3. Review "fields" matches
4. Check consistency
```

### Discovering Related Modules

**Goal**: Find all authentication-related modules

```
1. Press CMD+K
2. Type "auth"
3. Review descriptions
4. Explore related services
```

## Integration with Other Features

### Search to API Explorer

1. Search for a service name
2. Click result to open module
3. Navigate to API Explorer tab
4. Explore methods in detail

### Search to Code Examples

1. Search for a module
2. Click result
3. Switch to Usage Examples tab
4. Generate code for your language

### Search to Migration

1. Search for module with multiple versions
2. Click result
3. Go to Migration tab
4. Compare versions

## Keyboard Shortcuts

### Global

- **CMD+K / CTRL+K**: Open search
- **ESC**: Close search

### Within Search

- **Arrow Up/Down**: Navigate results (coming soon)
- **Enter**: Open selected result (coming soon)
- **Tab**: Navigate UI elements

## Tips & Best Practices

### Efficient Searching

**Start broad, then narrow:**
```
1. Search "user" (100 results)
2. Refine to "user service" (20 results)
3. Refine to "UserService create" (5 results)
```

**Use distinctive terms:**
```
Instead of: "get"
Try: "GetUserById"
```

**Search by unique identifiers:**
```
Field numbers: "field 42"
Versions: "v2.1.0"
Packages: "com.example.user"
```

### Understanding Results

**Check matched badges first:**
```
If searching for a service:
- Look for "services" badge
- Ignore "fields" only results
```

**Use previews:**
```
Preview shows which messages matched
Helps confirm it's the right result
```

**Check multiple results:**
```
Same message might exist in multiple modules
Compare implementations
```

### Search Strategy

**For exploration:**
```
1. Search broad terms ("auth", "order")
2. Browse top results
3. Discover related modules
```

**For specific lookup:**
```
1. Search exact name ("CreateUserRequest")
2. First result usually correct
3. Verify version if needed
```

**For troubleshooting:**
```
1. Search error message terms
2. Find related services
3. Check method signatures
```

## Troubleshooting

### No results found

**Symptoms**: "No results found for '{query}'"

**Possible causes:**
- Typo in search term
- Module not in registry
- Search index not loaded

**Solutions:**
1. Check spelling
2. Try broader search ("usr" → "user")
3. Verify module exists in list
4. Refresh page to reload index

### Search not opening

**Symptoms**: CMD+K doesn't work

**Possible causes:**
- Browser intercepted shortcut
- JavaScript error
- Page not fully loaded

**Solutions:**
1. Try CTRL+K instead
2. Check browser console for errors
3. Refresh page
4. Disable conflicting extensions

### Slow search results

**Symptoms**: Spinner shows for >1 second

**Possible causes:**
- Large registry (1000+ modules)
- Complex query
- Browser performance

**Solutions:**
1. Close other tabs
2. Clear browser cache
3. Use more specific queries
4. Consider server-side search (future)

### Wrong results appearing

**Symptoms**: Results don't seem relevant

**Possible causes:**
- Fuzzy matching too permissive
- Field names vs module names confusion

**Understanding:**
- Results ranked by boosted relevance
- Field matches have low priority
- Try more specific terms

## Search Index

### How It Works

1. **Build Time**: Index generated by CLI tool
   ```bash
   search-indexer --storage-dir ./storage --output web/public/search-index.json
   ```

2. **Load Time**: Browser fetches index on page load
   ```
   GET /search-index.json
   ```

3. **Search Time**: Lunr.js performs client-side search
   ```javascript
   index.search(query)
   ```

### Index Structure

```json
{
  "modules": [
    {
      "id": "user-service-v1.0.0",
      "name": "user-service",
      "version": "v1.0.0",
      "description": "User management service",
      "messages": ["User", "CreateUserRequest"],
      "services": ["UserService"],
      "methods": ["CreateUser", "GetUser"],
      "fields": ["user_id", "email", "name"],
      "enums": ["UserStatus", "Role"]
    }
  ]
}
```

### Rebuilding Index

After adding new modules:

```bash
# Regenerate search index
cd /path/to/spoke
./bin/search-indexer --storage-dir ./data/storage --output web/public/search-index.json

# Restart web server
# Index will be available on next page load
```

### Index Size

Typical sizes:
- 10 modules: ~100 KB
- 100 modules: ~1 MB
- 1000 modules: ~10 MB

Recommended: Keep under 10 MB for good performance.

## Future Enhancements

Planned features:
- Server-side search for large registries
- Advanced query syntax (field:value, wildcards)
- Search history and suggestions
- Saved searches and filters
- Search within specific modules
- Export search results

## What's Next?

- [**API Explorer**](api-explorer) - Explore search results in detail
- [**Code Examples**](code-examples) - Generate code after finding modules
- [**Module Browser**](..) - Browse all modules systematically
- [**CLI Search**](../cli-reference#search) - Command-line search tools

## Related Documentation

- [Lunr.js Documentation](https://lunrjs.com/) - Search library used
- [Search Index CLI](../cli-reference#search-indexer) - Rebuilding the index
- [API Performance](../../architecture/performance) - Search optimization
