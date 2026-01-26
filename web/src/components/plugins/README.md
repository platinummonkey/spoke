# Plugin Marketplace UI

This directory contains the React components for the Spoke Plugin Marketplace, implemented as part of Phase 5 of the Plugin Ecosystem.

## Overview

The Plugin Marketplace provides a comprehensive user interface for discovering, browsing, installing, and reviewing Spoke plugins. It integrates with the backend Plugin Marketplace API (Phase 4) to provide real-time plugin information.

## Architecture

### Technology Stack

- **React 18**: Component framework
- **TypeScript**: Type-safe development
- **React Query**: Server state management and caching
- **React Router**: Client-side routing
- **Axios**: HTTP client
- **CSS**: Custom styling (no CSS-in-JS to keep it simple)

### Component Hierarchy

```
PluginMarketplace (Main Page)
├── PluginFilters (Search and filter controls)
└── PluginCard[] (Grid of plugin cards)
    └── SecurityBadge
    └── StarRating

PluginDetail (Detail Page)
├── SecurityBadge
├── StarRating
└── Tabs
    ├── Overview
    ├── Versions
    │   └── VersionList
    └── Reviews
        ├── ReviewList
        └── Review Form
```

## Components

### PluginMarketplace.tsx

Main marketplace page with search, filters, and plugin grid.

**Features:**
- Search plugins by keyword
- Filter by type (language, validator, generator, runner)
- Filter by security level (official, verified, community)
- Sort by downloads, rating, name, or date
- Pagination support
- Loading and error states

**Props:** None (uses URL query parameters)

**Usage:**
```tsx
<Route path="/plugins" element={<PluginMarketplace />} />
```

### PluginCard.tsx

Displays a single plugin in card format.

**Features:**
- Plugin name, description, author
- Security badge
- Star rating and review count
- Download count
- Latest version
- Click to navigate to detail page

**Props:**
```typescript
interface PluginCardProps {
  plugin: Plugin;
}
```

**Usage:**
```tsx
{plugins.map(plugin => (
  <PluginCard key={plugin.id} plugin={plugin} />
))}
```

### PluginDetail.tsx (Page Component)

Detailed plugin view with tabs for overview, versions, and reviews.

**Features:**
- Complete plugin information
- Install button with version selection
- Three-tab interface:
  - **Overview**: Metadata (author, license, homepage, repository, dates)
  - **Versions**: Version history with download buttons
  - **Reviews**: User reviews with submission form
- Review submission with star rating
- Installation tracking

**Props:** None (uses URL parameter `:id`)

**Usage:**
```tsx
<Route path="/plugins/:id" element={<PluginDetail />} />
```

### PluginFilters.tsx

Filter controls for the marketplace.

**Features:**
- Search input with debouncing
- Type dropdown (all, language, validator, etc.)
- Security level dropdown (all, official, verified, community)
- Sort dropdown (downloads, rating, name, date)

**Props:**
```typescript
interface PluginFiltersProps {
  searchQuery: string;
  setSearchQuery: (query: string) => void;
  selectedType: PluginType | 'all';
  setSelectedType: (type: PluginType | 'all') => void;
  securityLevel: SecurityLevel | 'all';
  setSecurityLevel: (level: SecurityLevel | 'all') => void;
  sortBy: string;
  setSortBy: (sort: string) => void;
}
```

### SecurityBadge.tsx

Displays plugin security level with icon and color coding.

**Security Levels:**
- **Official** (Blue): Built by Spoke team
- **Verified** (Green): Community plugin, code reviewed
- **Community** (Gray): Unverified, user beware

**Props:**
```typescript
interface SecurityBadgeProps {
  level: SecurityLevel;
}
```

**Usage:**
```tsx
<SecurityBadge level={plugin.security_level} />
```

### StarRating.tsx

Displays and collects star ratings.

**Features:**
- Display-only mode (showing existing ratings)
- Interactive mode (for review submission)
- Half-star support
- Three sizes: small, medium, large
- Hover preview in interactive mode

**Props:**
```typescript
interface StarRatingProps {
  rating: number;
  maxRating?: number;  // Default: 5
  size?: 'small' | 'medium' | 'large';
  interactive?: boolean;
  onRatingChange?: (rating: number) => void;
}
```

**Usage:**
```tsx
{/* Display only */}
<StarRating rating={4.5} size="medium" />

{/* Interactive */}
<StarRating
  rating={reviewRating}
  size="large"
  interactive
  onRatingChange={setReviewRating}
/>
```

### VersionList.tsx

Table of plugin versions with download functionality.

**Features:**
- Version number, API version, size, downloads, release date
- Download button for each version
- Byte formatting (KB, MB, GB)
- Click to download plugin archive
- Empty state for plugins with no versions

**Props:**
```typescript
interface VersionListProps {
  versions: PluginVersion[];
  pluginId: string;
}
```

**Usage:**
```tsx
<VersionList versions={versions} pluginId={pluginId} />
```

### ReviewList.tsx

Displays list of plugin reviews.

**Features:**
- Review author name
- Star rating
- Review text
- Created date
- "Edited" label for updated reviews
- Empty state for plugins with no reviews

**Props:**
```typescript
interface ReviewListProps {
  reviews: PluginReview[];
}
```

**Usage:**
```tsx
<ReviewList reviews={reviews} />
```

## Services

### pluginService.ts

Axios-based API client for plugin operations.

**Methods:**

**Discovery:**
- `listPlugins(params?)`: List plugins with filters
- `searchPlugins(query, params?)`: Search plugins by keyword
- `getTrendingPlugins(limit?)`: Get trending plugins
- `getPlugin(id)`: Get plugin details

**Versions:**
- `listVersions(id)`: List plugin versions
- `getVersion(id, version)`: Get specific version
- `downloadPlugin(id, version)`: Get download URL

**Reviews:**
- `listReviews(id, params?)`: List plugin reviews
- `createReview(id, review)`: Submit review

**Installation:**
- `recordInstallation(id, version)`: Track installation
- `recordUninstallation(id, version)`: Track uninstallation

**Stats:**
- `getPluginStats(id, period?)`: Get plugin analytics

**Authentication:**
- `setAuthToken(token)`: Set auth header
- `setUserId(userId)`: Set user ID header

**Usage:**
```typescript
import pluginService from '../services/pluginService';

// Set authentication
pluginService.setAuthToken('your-jwt-token');
pluginService.setUserId('user-123');

// List plugins
const response = await pluginService.listPlugins({
  type: 'language',
  security_level: 'verified',
  sort_by: 'downloads',
  limit: 20,
});
```

## Hooks

### usePlugins.ts

React Query hooks for plugin operations with automatic caching and invalidation.

**Query Hooks:**

**Discovery:**
- `usePlugins(params?)`: List plugins
- `useSearchPlugins(query, params?)`: Search plugins
- `useTrendingPlugins(limit?)`: Trending plugins
- `usePlugin(id)`: Plugin details

**Versions:**
- `usePluginVersions(id)`: List versions
- `usePluginVersion(id, version)`: Specific version

**Reviews:**
- `usePluginReviews(id, params?)`: List reviews

**Stats:**
- `usePluginStats(id, period?)`: Plugin analytics

**Mutation Hooks:**

- `useCreateReview(id)`: Submit review
- `useRecordInstallation()`: Track installation
- `useRecordUninstallation()`: Track uninstallation

**Query Key Management:**

All hooks use a centralized `pluginKeys` factory for cache management:

```typescript
const pluginKeys = {
  all: ['plugins'],
  lists: () => [...pluginKeys.all, 'list'],
  list: (params?: PluginListRequest) => [...pluginKeys.lists(), params],
  details: () => [...pluginKeys.all, 'detail'],
  detail: (id: string) => [...pluginKeys.details(), id],
  versions: (id: string) => [...pluginKeys.detail(id), 'versions'],
  reviews: (id: string) => [...pluginKeys.detail(id), 'reviews'],
};
```

**Usage:**
```typescript
import { usePlugins, usePlugin, useCreateReview } from '../hooks/usePlugins';

function MyComponent() {
  // Query: auto-fetches and caches
  const { data: plugins, isLoading, error } = usePlugins({
    type: 'language',
    limit: 10,
  });

  // Mutation: invalidates cache on success
  const createReviewMutation = useCreateReview('rust-language');

  const handleSubmit = () => {
    createReviewMutation.mutate({
      rating: 5,
      review: 'Excellent plugin!',
    });
  };

  // ...
}
```

## Types

### plugin.ts

TypeScript interfaces matching backend Go types.

**Core Types:**
- `Plugin`: Complete plugin information
- `PluginVersion`: Version metadata and downloads
- `PluginReview`: User review with rating
- `PluginInstallation`: Installation tracking
- `PluginStats`: Analytics data

**Request/Response Types:**
- `PluginListRequest`: Filters for listing plugins
- `PluginListResponse`: Paginated plugin list
- `PluginSearchRequest`: Search parameters
- `PluginReviewRequest`: Review submission

**Enums:**
```typescript
export type PluginType = 'language' | 'validator' | 'generator' | 'runner' | 'transform';
export type SecurityLevel = 'official' | 'verified' | 'community';
export type SortBy = 'name' | 'downloads' | 'rating' | 'created_at';
```

## Styling

Each component has a corresponding CSS file with the same name:

- `PluginMarketplace.css`: Main page layout, grid, pagination
- `PluginCard.css`: Card styling with hover effects
- `PluginFilters.css`: Filter controls layout
- `PluginDetail.css`: Detail page with tabs
- `SecurityBadge.css`: Badge colors for security levels
- `StarRating.css`: Star icons and sizes
- `VersionList.css`: Table styling
- `ReviewList.css`: Review item layout

**Design System:**

Colors (matching Chakra UI):
- Primary: `#3182ce` (blue)
- Success: `#48bb78` (green)
- Warning: `#ecc94b` (yellow)
- Gray scale: `#2d3748`, `#4a5568`, `#718096`, `#a0aec0`, `#cbd5e0`, `#e2e8f0`, `#f7fafc`

Typography:
- Font family: System default
- Headings: 600-700 weight
- Body: 400 weight
- Monospace: Monaco, Menlo for version numbers

Spacing:
- 0.25rem, 0.5rem, 0.75rem, 1rem, 1.5rem, 2rem

Border radius:
- Small: 4px
- Medium: 6px
- Large: 8px
- Pill: 9999px

## Integration with App.tsx

The plugin marketplace is integrated into the main application via React Router:

```tsx
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 60 * 1000, // 1 minute
      retry: 1,
    },
  },
});

function App() {
  return (
    <ChakraProvider>
      <QueryClientProvider client={queryClient}>
        <Router>
          <Routes>
            <Route path="/plugins" element={<PluginMarketplace />} />
            <Route path="/plugins/:id" element={<PluginDetail />} />
            {/* ... other routes ... */}
          </Routes>
        </Router>
      </QueryClientProvider>
    </ChakraProvider>
  );
}
```

## API Endpoints Used

The UI consumes the following backend endpoints (see `docs/PLUGIN_API.md`):

- `GET /api/v1/plugins` - List plugins
- `GET /api/v1/plugins/search?q=` - Search plugins
- `GET /api/v1/plugins/trending` - Trending plugins
- `GET /api/v1/plugins/{id}` - Plugin details
- `GET /api/v1/plugins/{id}/versions` - List versions
- `GET /api/v1/plugins/{id}/versions/{version}` - Version details
- `GET /api/v1/plugins/{id}/versions/{version}/download` - Download plugin
- `GET /api/v1/plugins/{id}/reviews` - List reviews
- `POST /api/v1/plugins/{id}/reviews` - Create review
- `POST /api/v1/plugins/{id}/install` - Record installation
- `POST /api/v1/plugins/{id}/uninstall` - Record uninstallation
- `GET /api/v1/plugins/{id}/stats` - Plugin statistics

## Error Handling

All components include comprehensive error handling:

**Network Errors:**
- React Query automatically retries failed requests
- Error boundaries catch component errors
- User-friendly error messages displayed

**Validation Errors:**
- Form validation on review submission
- Required fields enforced
- Rating must be 1-5

**Loading States:**
- Skeleton loaders during data fetch
- Spinner for long operations
- Disabled buttons during mutations

**Empty States:**
- "No plugins found" message
- "No versions available"
- "No reviews yet. Be the first to review!"

## Performance Optimizations

**Code Splitting:**
- Components lazy-loaded via `React.lazy()`
- Routes loaded on demand

**Caching:**
- React Query caches all API responses
- 1-minute stale time reduces API calls
- Automatic cache invalidation on mutations

**Pagination:**
- Only load 20 plugins at a time
- Offset-based pagination for large datasets

**Debouncing:**
- Search input debounced to reduce API calls

**Optimistic Updates:**
- Review submission updates cache immediately
- Rollback on error

## Testing Considerations

**Unit Tests:**
- Test component rendering
- Test user interactions
- Test error states
- Mock API responses

**Integration Tests:**
- Test full user flows (search → view → install)
- Test form submission
- Test navigation

**E2E Tests:**
- Test marketplace browsing
- Test plugin installation
- Test review submission

## Future Enhancements

**Phase 6 Integration:**
- Display verification status from security scanning
- Show security scan results
- Plugin submission form

**Additional Features:**
- Plugin comparison view
- My installed plugins page
- Plugin update notifications
- Plugin categories/tags
- Advanced search with filters
- Sort by relevance for search results

## Development

**Run Development Server:**
```bash
cd web
npm install
npm run dev
```

**Build for Production:**
```bash
npm run build
```

**Type Check:**
```bash
npx tsc --noEmit
```

## Related Documentation

- [Plugin Manifest Specification](../../../docs/PLUGIN_MANIFEST.md)
- [Plugin API Documentation](../../../docs/PLUGIN_API.md)
- [Plugin Development Guide](../../../docs/PLUGIN_DEVELOPMENT.md)
- [Phase 5 Implementation Plan](../../../PLAN.md#phase-5-plugin-marketplace-ui-week-7)
