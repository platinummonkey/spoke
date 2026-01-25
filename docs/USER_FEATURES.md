# User Features: Saved Searches & Bookmarks

## Overview

The User Features system provides saved searches and bookmarks functionality, enabling users to save frequently-used queries and bookmark important modules/entities for quick access. The system supports both authenticated users (stored in database) and anonymous users (stored in localStorage).

## Features

### 1. Saved Searches

Save and reuse complex search queries with filters.

**Capabilities:**
- **Save search queries** - Store queries with names and descriptions
- **Execute saved searches** - One-click to run a previously saved search
- **Edit search metadata** - Update names, queries, and descriptions
- **Delete searches** - Remove searches no longer needed
- **Local storage fallback** - Anonymous users can save searches locally

**Use Cases:**
- Developers frequently searching for specific message types
- Team leads monitoring service definitions across modules
- QA engineers searching for test-related entities
- Documentation writers finding commented entities

### 2. Bookmarks

Bookmark modules and specific entities for quick access.

**Capabilities:**
- **Bookmark modules** - Save references to entire modules
- **Bookmark entities** - Save references to specific messages, services, etc.
- **Add notes and tags** - Organize bookmarks with metadata
- **Navigate quickly** - One-click to open bookmarked modules
- **Local storage fallback** - Anonymous users can bookmark locally

**Use Cases:**
- Frequently accessed modules during development
- Key service definitions for API documentation
- Important message types for data modeling
- Breaking change tracking across versions

## Architecture

### Backend (Go)

**File:** `pkg/api/user_features_handlers.go`

Provides RESTful API endpoints for managing saved searches and bookmarks.

**Endpoints:**

**Saved Searches:**
- `GET /api/v2/saved-searches` - List all saved searches
- `POST /api/v2/saved-searches` - Create new saved search
- `GET /api/v2/saved-searches/{id}` - Get saved search details
- `PUT /api/v2/saved-searches/{id}` - Update saved search
- `DELETE /api/v2/saved-searches/{id}` - Delete saved search

**Bookmarks:**
- `GET /api/v2/bookmarks` - List all bookmarks
- `POST /api/v2/bookmarks` - Create new bookmark
- `GET /api/v2/bookmarks/{id}` - Get bookmark details
- `PUT /api/v2/bookmarks/{id}` - Update bookmark (notes, tags)
- `DELETE /api/v2/bookmarks/{id}` - Delete bookmark

**Data Models:**

```go
type SavedSearch struct {
    ID          int64                  `json:"id"`
    UserID      *int64                 `json:"user_id,omitempty"`
    Name        string                 `json:"name"`
    Query       string                 `json:"query"`
    Filters     map[string]interface{} `json:"filters,omitempty"`
    Description string                 `json:"description,omitempty"`
    IsShared    bool                   `json:"is_shared"`
    CreatedAt   string                 `json:"created_at"`
    UpdatedAt   string                 `json:"updated_at"`
}

type Bookmark struct {
    ID         int64    `json:"id"`
    UserID     *int64   `json:"user_id,omitempty"`
    ModuleName string   `json:"module_name"`
    Version    string   `json:"version"`
    EntityPath string   `json:"entity_path,omitempty"`
    EntityType string   `json:"entity_type,omitempty"`
    Notes      string   `json:"notes,omitempty"`
    Tags       []string `json:"tags,omitempty"`
    CreatedAt  string   `json:"created_at"`
    UpdatedAt  string   `json:"updated_at"`
}
```

### Frontend (React + TypeScript)

**Hooks:**

**1. useSavedSearches Hook**

`web/src/hooks/useSavedSearches.ts`

Manages saved searches with localStorage fallback for anonymous users.

```typescript
const {
  searches,           // SavedSearch[]
  loading,            // boolean
  error,              // string | null
  createSearch,       // (search: Partial<SavedSearch>) => Promise<SavedSearch>
  updateSearch,       // (id: number, updates: Partial<SavedSearch>) => Promise<void>
  deleteSearch,       // (id: number) => Promise<void>
  refresh,            // () => Promise<void>
} = useSavedSearches();
```

**Features:**
- Dual storage (API + localStorage)
- Automatic fallback when API unavailable
- Merge strategy for local and remote data
- Error handling and loading states

**2. useBookmarks Hook**

`web/src/hooks/useBookmarks.ts`

Manages bookmarks with localStorage fallback for anonymous users.

```typescript
const {
  bookmarks,          // Bookmark[]
  loading,            // boolean
  error,              // string | null
  isBookmarked,       // (moduleName, version, entityPath?) => boolean
  createBookmark,     // (bookmark: Partial<Bookmark>) => Promise<Bookmark>
  updateBookmark,     // (id: number, updates: Partial<Bookmark>) => Promise<void>
  deleteBookmark,     // (id: number) => Promise<void>
  toggleBookmark,     // (moduleName, version, entityPath?) => Promise<void>
  refresh,            // () => Promise<void>
} = useBookmarks();
```

**Features:**
- Dual storage (API + localStorage)
- Bookmark existence checking
- Toggle functionality (add/remove)
- Tag and note management

**Components:**

**1. SavedSearches Component**

`web/src/components/SavedSearches.tsx`

Displays and manages saved searches.

**Features:**
- List saved searches with metadata
- Create/edit/delete operations
- Execute search (opens search modal with query)
- Modal for creating/editing searches
- Empty state messaging

**Props:**
```typescript
interface SavedSearchesProps {
  onSelectSearch?: (query: string) => void;
}
```

**2. Bookmarks Component**

`web/src/components/Bookmarks.tsx`

Displays and manages bookmarks.

**Features:**
- List bookmarks with module/entity info
- Edit notes and tags
- Delete bookmarks
- Navigate to bookmarked modules
- Visual distinction with star icons
- Tag management (add/remove)

**3. UserFeatures Page**

`web/src/components/UserFeatures.tsx`

Combined page displaying saved searches and bookmarks side-by-side.

**Route:** `/library`

**Features:**
- Breadcrumb navigation
- Two-column layout (desktop) or stacked (mobile)
- Integrated saved searches and bookmarks

## Database Schema

### saved_searches Table

```sql
CREATE TABLE IF NOT EXISTS saved_searches (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    query TEXT NOT NULL,
    filters JSONB DEFAULT '{}',
    description TEXT,
    is_shared BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

**Indexes:**
- `idx_saved_searches_user_id` - Filter by user
- `idx_saved_searches_is_shared` - Filter shared searches
- `idx_saved_searches_created_at` - Sort by date

**Trigger:**
- `update_saved_searches_updated_at` - Auto-update timestamp on changes

### bookmarks Table

```sql
CREATE TABLE IF NOT EXISTS bookmarks (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    module_name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    entity_path TEXT,
    entity_type VARCHAR(50),
    notes TEXT,
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, module_name, version, entity_path)
);
```

**Indexes:**
- `idx_bookmarks_user_id` - Filter by user
- `idx_bookmarks_module_version` - Lookup bookmarks by module
- `idx_bookmarks_entity_type` - Filter by entity type
- `idx_bookmarks_tags` - GIN index for tag searches
- `idx_bookmarks_created_at` - Sort by date

**Constraint:**
- Unique constraint prevents duplicate bookmarks per user

**Trigger:**
- `update_bookmarks_updated_at` - Auto-update timestamp on changes

## User Workflows

### Workflow 1: Save a Search

**Scenario:** User performs a complex search and wants to reuse it later.

**Steps:**
1. Execute search with filters (e.g., "user entity:message type:string")
2. Navigate to "My Library" page
3. Click "+" button in Saved Searches section
4. Fill in form:
   - **Name:** "User String Fields"
   - **Query:** "user entity:message type:string"
   - **Description:** "All string fields in user messages"
5. Click "Save"
6. Search appears in saved searches list

**Result:** Search is saved and can be executed later with one click.

### Workflow 2: Execute Saved Search

**Scenario:** User wants to run a previously saved search.

**Steps:**
1. Navigate to "My Library" page
2. Find desired saved search
3. Click dropdown menu (⌄)
4. Click "Execute Search"
5. Search modal opens with query pre-populated
6. Results displayed automatically

**Result:** Search executes without manually typing the query.

### Workflow 3: Bookmark a Module

**Scenario:** User frequently accesses a module and wants quick access.

**Steps:**
1. Navigate to module detail page
2. Click star icon (⭐) in module header
3. Module is bookmarked
4. Optionally add notes/tags from "My Library" page

**Result:** Module appears in bookmarks list for quick access.

### Workflow 4: Organize Bookmarks with Tags

**Scenario:** User has many bookmarks and wants to organize them.

**Steps:**
1. Navigate to "My Library" page
2. Find bookmark in list
3. Click dropdown menu (⌄)
4. Click "Edit Notes & Tags"
5. Add tags (e.g., "auth", "v2", "deprecated")
6. Add notes (e.g., "Used in login flow")
7. Click "Save"

**Result:** Bookmark has metadata for easier organization.

### Workflow 5: Delete Unused Items

**Scenario:** User wants to clean up saved searches and bookmarks.

**Steps:**
1. Navigate to "My Library" page
2. For saved searches or bookmarks:
   - Click dropdown menu (⌄)
   - Click "Delete" (red option)
   - Confirm deletion
3. Item removed from list

**Result:** Unused items removed from library.

## Local Storage Format

For anonymous users, data is stored in browser localStorage.

### Saved Searches

**Key:** `spoke_saved_searches`

**Format:**
```json
[
  {
    "id": 1643234567890,
    "name": "User String Fields",
    "query": "user entity:message type:string",
    "description": "All string fields in user messages",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
]
```

### Bookmarks

**Key:** `spoke_bookmarks`

**Format:**
```json
[
  {
    "id": 1643234567891,
    "module_name": "user",
    "version": "v1.0.0",
    "entity_path": "UserService.CreateUser",
    "entity_type": "method",
    "notes": "Primary user creation endpoint",
    "tags": ["auth", "v2"],
    "created_at": "2024-01-15T11:00:00Z",
    "updated_at": "2024-01-15T11:00:00Z"
  }
]
```

## API Examples

### Create Saved Search

**Request:**
```http
POST /api/v2/saved-searches
Content-Type: application/json

{
  "name": "User String Fields",
  "query": "user entity:message type:string",
  "description": "All string fields in user messages"
}
```

**Response:**
```json
{
  "id": 1,
  "name": "User String Fields",
  "query": "user entity:message type:string",
  "description": "All string fields in user messages",
  "is_shared": false,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### Create Bookmark

**Request:**
```http
POST /api/v2/bookmarks
Content-Type: application/json

{
  "module_name": "user",
  "version": "v1.0.0",
  "entity_path": "UserService.CreateUser",
  "entity_type": "method",
  "notes": "Primary user creation endpoint",
  "tags": ["auth", "v2"]
}
```

**Response:**
```json
{
  "id": 1,
  "module_name": "user",
  "version": "v1.0.0",
  "entity_path": "UserService.CreateUser",
  "entity_type": "method",
  "notes": "Primary user creation endpoint",
  "tags": ["auth", "v2"],
  "created_at": "2024-01-15T11:00:00Z",
  "updated_at": "2024-01-15T11:00:00Z"
}
```

### List Bookmarks

**Request:**
```http
GET /api/v2/bookmarks
```

**Response:**
```json
{
  "bookmarks": [
    {
      "id": 1,
      "module_name": "user",
      "version": "v1.0.0",
      "entity_path": "UserService.CreateUser",
      "entity_type": "method",
      "notes": "Primary user creation endpoint",
      "tags": ["auth", "v2"],
      "created_at": "2024-01-15T11:00:00Z",
      "updated_at": "2024-01-15T11:00:00Z"
    }
  ],
  "count": 1
}
```

## Integration Points

### Search Modal Integration

When a saved search is executed, the query is passed to the EnhancedSearchBar component:

```typescript
<SavedSearches
  onSelectSearch={(query) => {
    // Open search modal with query
    setSearchQuery(query);
    openSearchModal();
  }}
/>
```

### Module Detail Integration

(Future enhancement) Add bookmark button to ModuleDetail component:

```typescript
const { toggleBookmark, isBookmarked } = useBookmarks();

<IconButton
  icon={<StarIcon />}
  colorScheme={isBookmarked(moduleName, version) ? 'orange' : 'gray'}
  onClick={() => toggleBookmark(moduleName, version)}
  aria-label="Toggle bookmark"
/>
```

## Future Enhancements

### Shared Searches (Team Feature)

Allow users to share saved searches with team members:
- `is_shared` flag already exists in database
- Add UI toggle for sharing searches
- Team search library page
- Permissions management

### Smart Suggestions

Suggest searches based on usage patterns:
- Track most executed searches
- Suggest related searches
- Personalized search recommendations

### Export/Import

Allow users to export and import their library:
- Export as JSON
- Import from JSON
- Sync across devices

### Search Folders

Organize saved searches into folders:
- Create folders/categories
- Drag-and-drop organization
- Nested folder support

### Bookmark Collections

Group bookmarks into collections:
- Project-specific collections
- Version-specific collections
- Share collections with team

## Accessibility

### Keyboard Navigation

- Tab through saved searches and bookmarks
- Enter to execute search or open bookmark
- Delete key to remove items
- Arrow keys to navigate lists

### Screen Reader Support

- ARIA labels for all interactive elements
- Semantic HTML structure
- Status announcements for actions
- Role attributes for custom components

### Focus Management

- Focus returns to trigger after modal close
- Focus indication on interactive elements
- Skip links for long lists

## Performance Considerations

### Local Storage Size

- Maximum ~5-10MB per origin (browser dependent)
- Typical usage: <100KB for ~100 saved items
- No pagination needed for local storage

### API Response Time

- List endpoints: <50ms (indexed queries)
- Create/update: <100ms (single row operations)
- No complex joins or aggregations

### Frontend Rendering

- Lists virtualized if >100 items (future enhancement)
- Lazy loading of bookmarks on scroll (future enhancement)
- Debounced search input (already implemented)

## Troubleshooting

### Saved Searches Not Appearing

**Cause:** localStorage full or disabled

**Solution:**
- Clear browser storage
- Check browser settings for storage permissions
- Use API backend (when authenticated)

### Bookmarks Lost After Browser Clear

**Cause:** localStorage cleared by user

**Solution:**
- Use authenticated mode (stores in database)
- Export bookmarks before clearing browser data
- Implement sync feature (future enhancement)

### Can't Delete Saved Search

**Cause:** API endpoint not responding

**Solution:**
- Check network tab for errors
- Verify backend is running
- Check database permissions
- Fallback to local storage operations

## Summary

The User Features system provides saved searches and bookmarks with:
- **Dual storage** - API (authenticated) and localStorage (anonymous)
- **Rich metadata** - Names, descriptions, notes, tags
- **Quick access** - One-click execution and navigation
- **Organization** - Tags and categories for bookmarks
- **Seamless fallback** - Works without authentication

All features work locally first, syncing with backend when available for authenticated users.
