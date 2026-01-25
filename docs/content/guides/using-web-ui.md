---
title: "Using the Web UI"
weight: 50
---

# Using the Spoke Web UI

The Spoke web interface provides a comprehensive, interactive documentation platform for exploring protobuf schemas, testing APIs, comparing versions, and generating code examples.

## Overview

Access the web UI at `http://localhost:8080` (or your deployment URL). The interface includes:

- **Module Browser**: Browse available modules and versions
- **API Explorer**: Interactive service and method browser
- **Code Examples**: Auto-generated usage examples for 15 languages
- **Schema Comparison**: Visual diff tool for version changes
- **Migration Guides**: Step-by-step upgrade instructions
- **Full-Text Search**: Fast search across modules, messages, services, and fields

## Module Browser

### Viewing Modules

The home page displays all available modules with:
- Module name and description
- Number of versions
- Latest version badge
- Repository information
- Commit SHA and branch

**Features:**
- Search modules by name or description
- Click any module to view details
- Sort by name, version count, or date

### Module Details

Click on a module to view:
- Complete version history
- Source repository links
- Dependencies (clickable links to dependent modules)
- Version-specific metadata

**Tabs Available:**
1. **Overview** - Version list and metadata
2. **Types** - Message and enum definitions
3. **API Explorer** - Interactive service browser
4. **Usage Examples** - Code snippets
5. **Migration** - Version comparison and guides

## API Explorer

Navigate to the **API Explorer** tab to interactively explore your protobuf services.

### Browsing Services

The API Explorer shows:
- All services defined in your proto files
- Method count for each service
- Expandable service list (accordion style)

**How to use:**
1. Click a service to expand and view its methods
2. Click a method to see details
3. View request and response message schemas

### Viewing Methods

When you select a method, you'll see:
- Method name and signature
- Request message schema (left side)
- Response message schema (right side)
- Streaming indicators (client/server/bidirectional)
- Field details: name, type, number, label (required/optional/repeated)

**Streaming Method Types:**
- **Unary**: Single request, single response
- **Client Stream**: Stream of requests, single response
- **Server Stream**: Single request, stream of responses
- **Bidirectional Stream**: Stream of requests and responses

### Message Schemas

Message viewer displays:
- Field name, type, and field number
- Required/optional/repeated labels
- Color-coded type badges
- Nested message support (expandable)
- Enum values
- Field comments from proto files

**Type Color Coding:**
- Purple: String, bytes
- Blue: Numeric types (int32, int64, float, double)
- Green: Boolean
- Orange: Message references
- Teal: Enum types

## Try It Out (Playground)

The **Try It Out** tab in Method Detail allows you to:

### Build Requests

1. Auto-generated sample JSON appears based on message schema
2. Edit the JSON in the text editor
3. Validation highlights errors instantly

**JSON Validation:**
- Type checking (string, int, bool, etc.)
- Required field verification
- Enum value validation
- Nested message validation

### View Responses

- Copy request/response JSON to clipboard
- Format validation errors clearly
- Mock responses for demo purposes

## Code Examples

The **Usage Examples** tab provides language-specific code snippets.

### Viewing Examples

1. Navigate to any module version
2. Click **Usage Examples** tab
3. Select your language from dropdown (15 options)
4. Copy code to clipboard

**Available Languages:**
- Go, Python, Java, C++, C#
- Rust, TypeScript, JavaScript
- Dart, Swift, Kotlin
- Objective-C, Ruby, PHP, Scala

### Example Content

Generated examples include:
- gRPC client setup and connection
- Service client instantiation
- Sample method calls with realistic data
- Error handling
- Proper imports and package setup

**Example (Go):**
```go
package main

import (
    "context"
    "log"
    pb "github.com/company/user-service"
    "google.golang.org/grpc"
)

func main() {
    conn, err := grpc.Dial("localhost:50051", ...)
    client := pb.NewUserServiceClient(conn)

    resp, err := client.CreateUser(ctx, &pb.CreateUserRequest{
        Email: "user@example.com",
        Name: "John Doe",
    })
}
```

## Version Comparison & Migration

The **Migration** tab helps you understand changes between versions.

### Comparing Versions

1. Navigate to **Migration** tab
2. Select **From Version** (older)
3. Select **To Version** (newer)
4. View changes in two sub-tabs: Schema Diff and Migration Guide

### Schema Diff

The Schema Diff displays:

**Statistics Dashboard:**
- Total changes count
- Breaking changes (red)
- Non-breaking changes (green)
- Warnings (yellow)

**Change List:**
- Expandable accordion for each change
- Severity badge (breaking/non-breaking/warning)
- Location in proto file
- Old vs new values
- Migration tip

**Change Types Detected:**
- Field added/removed
- Type changed
- Message removed
- Enum value removed
- Service method changes
- Import changes

**Example:**
```
Breaking Change: field_removed
Location: user.proto:message User:field phone_number
Description: Field 'phone_number' was removed from message 'User'
Migration Tip: Update code referencing User.phone_number
```

### Migration Guides

Migration guides are manually authored markdown documents providing:
- Overview of changes
- Breaking change details with examples
- New feature highlights
- Step-by-step upgrade instructions
- Testing recommendations
- Rollback plan

**Location:** `/migrations/{module}/v{from}-to-v{to}.md`

**Example Guide Structure:**
```markdown
# Migration Guide: user-service v1.0.0 → v1.1.0

## Overview
- 2 breaking changes
- 3 new features
- 1 deprecation

## Breaking Changes
### 1. Field Removed: User.phone_number
**Before:**
`User.phone_number` (string)

**After:**
Use `User.contact_info.phone` instead

**Migration:**
1. Search code for `user.phone_number`
2. Replace with `user.contact_info.phone`
```

## Search

Press **CMD+K** (Mac) or **CTRL+K** (Windows/Linux) to open the global search modal.

### Search Features

**What you can search:**
- Module names
- Message types
- Service names
- Method names
- Field names
- Enum types
- Descriptions

**Search UI:**
- Instant results as you type (300ms debounce)
- Highlighted matching terms (yellow background)
- Top 10 most relevant results
- Preview of matched content
- Color-coded badges showing what matched

**Keyboard Shortcuts:**
- **CMD+K / CTRL+K**: Open search
- **ESC**: Close search
- **Enter**: Navigate to selected result

**Result Display:**
- Module name and version
- Description with highlighted matches
- "Matched:" badges (name, messages, services, methods)
- Preview of matched messages/services/methods
- Click to navigate to module detail page

### Search Index

The search is powered by client-side Lunr.js with:
- **Field Boosting**: Name matches rank higher than description
- **Fast Search**: <100ms response time
- **Offline Capable**: Works without backend connection
- **Comprehensive Index**: All modules, versions, and types

**Boost Values:**
- Name: 10x
- Description: 5x
- Messages: 3x
- Services: 3x
- Methods: 2x
- Enums: 2x
- Fields: 1x

## Accessibility Features

The Spoke web UI is built with accessibility in mind:

### Keyboard Navigation
- All interactive elements are keyboard accessible
- Tab navigation through forms and buttons
- Enter to activate buttons and links
- Arrow keys in dropdown menus

### Screen Reader Support
- ARIA labels on all interactive elements
- Semantic HTML structure
- Role attributes for custom components
- Descriptive alt text for icons

### Visual Accessibility
- High contrast color scheme (WCAG 2.1 AA compliant)
- Color-coded badges with text labels
- Clear focus indicators
- Responsive font sizes

## Mobile Support

The interface is fully responsive and optimized for mobile devices:
- Touch-friendly UI elements
- Mobile-optimized layouts
- Responsive breakpoints for tablet/phone
- Hamburger menu for navigation (when implemented)

## Performance Optimizations

### Code Splitting
- ModuleList and ModuleDetail load on demand
- Separate chunks reduce initial bundle size
- Lazy loading for better perceived performance

### Loading States
- Skeleton screens during data fetching
- Progressive loading for large datasets
- Spinners for async operations

### Caching
- Search index cached client-side
- Proto definitions cached per module
- 15-minute cache for web fetches

## Tips & Best Practices

### Efficient Module Exploration
1. Use search to quickly find modules
2. Bookmark frequently accessed modules
3. Use browser back/forward for navigation
4. Copy URLs to share specific versions

### API Testing Workflow
1. Explore service in API Explorer
2. Build request in Try It Out tab
3. Copy JSON to your API client
4. Validate with generated code examples

### Version Upgrade Planning
1. Compare versions in Migration tab
2. Review breaking changes first
3. Check migration guide for detailed steps
4. Download examples for updated code
5. Test in staging environment

### Code Generation
1. Browse API in API Explorer
2. Copy example from Usage Examples tab
3. Download compiled artifacts via Overview tab
4. Integrate into your project
5. Refer to CODE_GENERATION_GUIDE.md for details

## Troubleshooting

### Search Not Working
- **Issue**: Search returns no results
- **Fix**: Ensure search index is loaded (check browser console for errors)
- **Rebuild**: Run `search-indexer` CLI tool to regenerate index

### Examples Not Loading
- **Issue**: Code examples show "Loading..." indefinitely
- **Fix**: Check that backend is running and accessible
- **Check**: GET `/api/v1/modules/{name}/versions/{version}/examples/{lang}`

### Diff Not Comparing
- **Issue**: Schema diff shows "Failed to load"
- **Fix**: Ensure both versions exist in the registry
- **Verify**: Both versions have valid proto files

### Module Not Found
- **Issue**: 404 error when viewing module
- **Fix**: Push module to registry using `spoke push` CLI
- **Verify**: Check `/modules` API endpoint

## What's Next?

- **API Reference**: Detailed REST API documentation
- **CLI Guide**: Command-line interface usage
- **Code Generation Guide**: Multi-language compilation
- **Migration Guide Template**: Creating migration documents
- **Deployment Guide**: Hosting Spoke in production

## Feedback & Support

Found a bug or have a feature request?
- GitHub Issues: https://github.com/platinummonkey/spoke/issues
- Documentation: https://spoke.dev/docs
- Community: https://spoke.dev/community
