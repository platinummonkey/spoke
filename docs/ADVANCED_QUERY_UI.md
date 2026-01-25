# Advanced Query UI

## Overview

The Advanced Query UI provides a powerful, user-friendly interface for searching protobuf schemas with advanced filters, autocomplete suggestions, and real-time feedback. It integrates with the PostgreSQL-backed search service to deliver fast, accurate results.

## Features

### 1. Enhanced Search Bar

**Location:** `web/src/components/EnhancedSearchBar.tsx`

The enhanced search bar offers:
- **Modal-based interface** - Full-screen search experience (CMD/CTRL+K to open)
- **Real-time autocomplete** - Suggestions from search history (debounced 300ms)
- **Filter chips** - Visual representation of active filters
- **Advanced filters popover** - Quick access to filter syntax and entity type buttons
- **Keyboard shortcuts** - ESC to close, CMD+K to open

**Example Usage:**
```typescript
import { EnhancedSearchBar } from './components/EnhancedSearchBar';

<EnhancedSearchBar />
```

### 2. Filter Chips

Active filters are displayed as colored chips that can be removed individually:

- **Entity Type** (Purple) - `entity:message`, `entity:field`
- **Field Type** (Blue) - `type:string`, `type:int32`
- **Module** (Green) - `module:user`
- **Version** (Orange) - `version:v1.0.0`
- **Has Comment** (Cyan) - `has-comment:true`

**Interactions:**
- Click X on chip to remove that filter
- Click clear button to remove all filters
- Filters automatically parsed from query string

### 3. Autocomplete Suggestions

Suggestions appear after typing 2+ characters, based on recent search history:

**Features:**
- Debounced API calls (300ms) to avoid excessive requests
- Top 5 most popular/recent queries
- Click suggestion to populate search box
- Automatically hidden when not needed

**Backend Endpoint:**
```
GET /api/v2/search/suggestions?prefix={text}&limit=5
```

**Response:**
```json
{
  "prefix": "user",
  "suggestions": [
    "user entity:message",
    "UserProfile",
    "user type:string",
    "user module:auth"
  ]
}
```

### 4. Advanced Filters Popover

Quick access panel providing:
- **Entity type buttons** - One-click filters for message, enum, service, method, field
- **Query syntax examples** - Visual guide with code blocks showing filter syntax
- **Interactive filter insertion** - Click buttons to add filters to query

**Example Syntax Shown:**
- `entity:message` - Find messages
- `type:string` - Find string fields
- `module:user` - Search in user module
- `has-comment:true` - Only entities with comments
- `user entity:message type:string` - Combined filters

### 5. Enhanced Search Results

**Location:** `web/src/components/EnhancedSearchResults.tsx`

Rich result display featuring:

#### Entity Type Badges
Color-coded badges for each entity type:
- **Message** - Purple
- **Enum** - Teal
- **Service** - Orange
- **Method** - Pink
- **Field** - Cyan

#### Relevance Scoring
- Displays PostgreSQL ts_rank score as percentage
- Formatted as "85.3%" for easy understanding
- Helps users identify most relevant results

#### Full Path Display
- Shows complete qualified path (e.g., `user.v1.UserProfile.email`)
- Syntax-highlighted with code formatting
- Hover tooltip for long paths

#### Parent Path
- Displays parent entity if different from full path
- Useful for understanding context
- Example: Field's parent is the message

#### Module and Version Info
- Module name as clickable blue link
- Version badge with gray scheme
- Proto file path when available

#### Descriptions and Comments
- Entity descriptions shown if available
- Comments displayed in gray box
- Text highlighting for matching query terms
- Maximum 2 lines (expandable on module detail page)

#### Field-Specific Details
For field entities, displays:
- **Field type** - `string`, `int32`, `repeated`, etc.
- **Field number** - Field position in message
- **Modifiers** - `repeated`, `optional` tags

**Example:**
```
field_type: string
Field #3
[repeated] [optional]
```

#### Method-Specific Details
For method entities, displays:
- **Input type** - Request message type
- **Output type** - Response message type

**Example:**
```
Input: CreateUserRequest
Output: CreateUserResponse
```

## User Workflows

### Basic Search

**Scenario:** Search for all messages containing "user"

**Steps:**
1. Press CMD+K (or click search bar)
2. Type "user"
3. View results with entity type badges
4. Click result to navigate to module detail

**Query:** `user`

**Results:**
- UserProfile (message)
- UserService (service)
- CreateUser (method)
- user_id (field)

### Filtered Search

**Scenario:** Find all string fields in user module

**Steps:**
1. Press CMD+K
2. Type "user"
3. Click Advanced Filters (info icon)
4. Click "field" entity type button
5. Add filter: `type:string`
6. View filtered results

**Query:** `user entity:field type:string`

**Results:**
- user_name (field, string)
- user_email (field, string)
- display_name (field, string)

### Using Suggestions

**Scenario:** Repeat a previous search

**Steps:**
1. Press CMD+K
2. Start typing previous query (e.g., "user entity:")
3. View suggestions below search bar
4. Click suggestion to populate full query
5. View results

**Query:** `user entity:message type:string` (from suggestion)

### Removing Filters

**Scenario:** Narrow search results by removing filters

**Steps:**
1. Execute search with multiple filters
2. View filter chips below search bar
3. Click X on individual chip to remove that filter
4. Or click clear button to remove all filters
5. Results update automatically

**Before:** `user entity:message type:string module:auth`

**After:** `user entity:message` (removed type and module filters)

## Frontend Architecture

### useEnhancedSearch Hook

**Location:** `web/src/hooks/useEnhancedSearch.ts`

Custom React hook providing:
- **State management** - Query, results, loading, error states
- **Debouncing** - Configurable debounce (default 300ms)
- **Filter parsing** - Extracts filters from query string
- **API integration** - Calls `/api/v2/search` and `/api/v2/search/suggestions`
- **Request cancellation** - AbortController for in-flight requests

**Example Usage:**
```typescript
const {
  query,
  setQuery,
  results,
  totalCount,
  loading,
  error,
  suggestions,
  fetchSuggestions,
  filters,
  removeFilter,
  addFilter,
  clear,
} = useEnhancedSearch({ debounceMs: 300, limit: 50 });
```

**Interface:**
```typescript
interface UseEnhancedSearchOptions {
  debounceMs?: number;
  limit?: number;
}

interface EnhancedSearchResult {
  id: number;
  entity_type: string;
  entity_name: string;
  full_path: string;
  parent_path?: string;
  module_name: string;
  version: string;
  description?: string;
  comments?: string;
  field_type?: string;
  field_number?: number;
  method_input_type?: string;
  method_output_type?: string;
  rank: number;
}

interface SearchFilter {
  type: 'entity' | 'field-type' | 'module' | 'version' | 'has-comment';
  value: string;
  display: string;
}
```

### Component Hierarchy

```
EnhancedSearchBar
├── Modal (Chakra UI)
│   ├── InputGroup
│   │   ├── SearchIcon
│   │   ├── Input (query)
│   │   └── Spinner (loading)
│   ├── Filter Chips
│   │   ├── Tag (entity:message)
│   │   ├── Tag (type:string)
│   │   └── IconButton (clear all)
│   ├── Advanced Filters Popover
│   │   ├── Entity Type Buttons
│   │   └── Query Syntax Examples
│   ├── Suggestions
│   │   └── Button[] (recent queries)
│   └── EnhancedSearchResults
│       └── ResultItem[]
│           ├── Entity Type Badge
│           ├── Entity Name
│           ├── Relevance Score
│           ├── Full Path
│           ├── Module/Version
│           ├── Description
│           ├── Comments
│           └── Type-Specific Details
```

## API Integration

### Search Endpoint

**URL:** `GET /api/v2/search`

**Query Parameters:**
- `q` (required) - Search query with filters
- `limit` (optional) - Max results (default: 50, max: 1000)
- `offset` (optional) - Pagination offset (default: 0)

**Example Request:**
```http
GET /api/v2/search?q=user%20entity:message&limit=20&offset=0
```

**Response:**
```json
{
  "results": [
    {
      "id": 123,
      "entity_type": "message",
      "entity_name": "UserProfile",
      "full_path": "user.v1.UserProfile",
      "module_name": "user",
      "version": "v1.0.0",
      "description": "User profile information",
      "comments": "// User profile with contact details",
      "rank": 0.853,
      "proto_file_path": "user/v1/user.proto",
      "line_number": 42
    }
  ],
  "total_count": 15,
  "query": "user entity:message"
}
```

### Suggestions Endpoint

**URL:** `GET /api/v2/search/suggestions`

**Query Parameters:**
- `prefix` (required) - Query prefix (minimum 2 characters)
- `limit` (optional) - Max suggestions (default: 5, max: 20)

**Example Request:**
```http
GET /api/v2/search/suggestions?prefix=user&limit=5
```

**Response:**
```json
{
  "prefix": "user",
  "suggestions": [
    "user entity:message",
    "UserProfile",
    "user type:string",
    "user module:auth",
    "UserService entity:service"
  ]
}
```

## Keyboard Shortcuts

**Global:**
- `CMD+K` or `CTRL+K` - Open search modal
- `ESC` - Close search modal and clear query

**Future Enhancements (not yet implemented):**
- `↑` / `↓` - Navigate results
- `ENTER` - Open selected result
- `TAB` - Cycle through filter suggestions

## Performance Considerations

### Debouncing

Search requests are debounced by 300ms to avoid excessive API calls:
- User types "user" → waits 300ms → triggers search
- User types "user entity:message" quickly → only 1 search after 300ms pause

### Request Cancellation

In-flight requests are cancelled when new query is issued:
- Prevents race conditions
- Reduces server load
- Uses AbortController API

### Result Limits

Default limit of 50 results to balance performance and usability:
- Prevents overwhelming UI
- Reduces payload size
- Future enhancement: pagination for more results

### Lazy Loading

Suggestions fetched only when needed:
- Minimum 2 characters before fetching
- Cancelled if query changes
- Cached briefly on backend

## Accessibility

### ARIA Labels

All interactive elements have proper ARIA labels:
- Search input: "Search protobuf schemas"
- Filter buttons: "entity:message filter", "type:string filter"
- Remove filter: "Remove entity filter"
- Advanced filters: "Open advanced filters"

### Keyboard Navigation

Modal supports full keyboard navigation:
- Tab through interactive elements
- ESC to close
- Focus management (input auto-focuses on open)

### Screen Reader Support

- Search status announced ("Loading results", "5 results found")
- Filter changes announced
- Error messages announced

## Error Handling

### API Errors

**Search Errors:**
- Display user-friendly error message
- Suggest checking query syntax
- Fallback to empty results

**Suggestions Errors:**
- Silently fail (hide suggestions)
- Log error to console
- Don't block main search

### Network Errors

- Retry logic (not yet implemented)
- Offline detection (not yet implemented)
- Timeout handling (AbortController with timeout)

### Query Parsing Errors

- Invalid filter syntax ignored
- Partial query support (e.g., `entity:` without value)
- Graceful degradation to keyword search

## Future Enhancements

### Search History

- Local storage for anonymous users
- Database-backed for authenticated users
- Clear history option
- Privacy controls

### Saved Searches

- Save frequently used queries
- Name and organize searches
- Share searches with team (future)

### Bookmarks

- Bookmark specific entities
- Quick access from search
- Notes/annotations

### Query Builder

- Visual query builder (no syntax required)
- Drag-and-drop filters
- Preview results as you build

### Result Sorting

- Sort by relevance (default)
- Sort by name, module, type
- Recent/popular results

### Result Grouping

- Group by module
- Group by entity type
- Collapsible groups

### Export Results

- Export as CSV
- Export as JSON
- Copy as markdown

## Testing

### Unit Tests

**Hook Tests:**
```typescript
describe('useEnhancedSearch', () => {
  it('debounces query', async () => { ... });
  it('parses filters correctly', () => { ... });
  it('cancels in-flight requests', () => { ... });
  it('fetches suggestions', async () => { ... });
});
```

**Component Tests:**
```typescript
describe('EnhancedSearchBar', () => {
  it('opens on CMD+K', () => { ... });
  it('displays filter chips', () => { ... });
  it('removes filter on chip close', () => { ... });
  it('adds filter from popover', () => { ... });
});
```

### Integration Tests

```typescript
describe('Search Flow', () => {
  it('performs full search workflow', async () => {
    // Open modal
    // Type query
    // Add filters
    // View results
    // Click result
    // Navigate to module
  });
});
```

### E2E Tests

```bash
# Cypress tests
describe('Advanced Search', () => {
  it('searches with filters', () => {
    cy.visit('/');
    cy.get('[data-testid="search-button"]').click();
    cy.get('[data-testid="search-input"]').type('user entity:message');
    cy.get('[data-testid="search-result"]').should('have.length.greaterThan', 0);
  });
});
```

## Troubleshooting

### No Results Found

**Possible Causes:**
1. Typo in search query
2. No entities match filters
3. Search index not yet built

**Solutions:**
- Check query spelling
- Remove some filters to broaden search
- Verify search index exists: `SELECT COUNT(*) FROM proto_search_index;`

### Suggestions Not Appearing

**Possible Causes:**
1. Less than 2 characters typed
2. No search history yet
3. API endpoint not available

**Solutions:**
- Type at least 2 characters
- Perform some searches first
- Check browser console for API errors

### Slow Search

**Possible Causes:**
1. Large result set
2. Complex query with many filters
3. Database not optimized

**Solutions:**
- Add more filters to narrow results
- Check PostgreSQL query performance: `EXPLAIN ANALYZE`
- Verify GIN indexes exist: `\d proto_search_index`

### Filter Chips Not Appearing

**Possible Causes:**
1. Invalid filter syntax
2. JavaScript error
3. React rendering issue

**Solutions:**
- Check query syntax matches pattern: `filter:value`
- Check browser console for errors
- Clear cache and reload

## Summary

The Advanced Query UI provides a powerful, intuitive interface for searching protobuf schemas with:
- **Real-time feedback** - See results as you type (debounced 300ms)
- **Visual filters** - Filter chips show active filters
- **Autocomplete** - Suggestions from search history
- **Rich results** - Entity types, paths, descriptions, comments
- **Keyboard shortcuts** - CMD+K, ESC for efficient navigation
- **Accessibility** - ARIA labels, keyboard navigation, screen reader support

All powered by PostgreSQL Full-Text Search for fast, accurate results.
