---
title: "Web UI Documentation"
weight: 50
---

# Spoke Web UI

The Spoke web interface provides a comprehensive, interactive documentation platform for exploring protobuf schemas, testing APIs, comparing versions, and generating code examples.

## Quick Access

Access the web UI at `http://localhost:8080` (or your deployment URL).

## Overview

The Spoke Web UI transforms protobuf schema exploration from static documentation into an interactive experience. Key features include:

### 🔍 Interactive API Explorer
Browse your protobuf services with an intuitive, expandable interface that shows:
- Service definitions and method signatures
- Request and response message schemas
- Field-level details with types and labels
- Nested message structures
- Color-coded type system

[Learn more about the API Explorer →](api-explorer)

### 💻 Auto-Generated Code Examples
View working code examples in 15+ programming languages:
- Go, Python, Java, C++, C#
- Rust, TypeScript, JavaScript
- Dart, Swift, Kotlin, Objective-C
- Ruby, PHP, Scala

Each example includes gRPC client setup, service instantiation, and method calls with realistic sample data.

[View code examples →](code-examples)

### 🎮 Request/Response Playground
Build and validate JSON requests interactively:
- Auto-generate sample requests from schemas
- JSON syntax highlighting
- Real-time validation against proto definitions
- Copy request/response payloads

[Try the playground →](playground)

### 🔄 Version Comparison Tools
Understand schema evolution with visual diff tools:
- Automated breaking change detection
- Side-by-side version comparison
- Migration tips and recommendations
- Manual migration guide support

[Compare versions →](migration-tools)

### 🔎 Full-Text Search
Fast, client-side search across all schema elements:
- Press **CMD+K** (Mac) or **CTRL+K** (Windows/Linux)
- Search modules, messages, services, methods, fields
- Instant results with highlighted matches
- Keyboard-friendly navigation

[Learn about search →](search)

## Getting Started

1. **Browse Modules**: Start at the home page to see all available modules
2. **Explore a Module**: Click any module to view versions and details
3. **Navigate Tabs**: Use the tab interface to switch between views
4. **Search Anytime**: Press CMD+K to quickly find what you need
5. **Copy Examples**: Use the code examples tab to get started quickly

## Navigation Structure

```
Home (Module List)
└── Module Detail
    ├── Overview (versions, downloads)
    ├── Types (messages, enums)
    ├── API Explorer (services, methods)
    ├── Usage Examples (code snippets)
    └── Migration (version comparison)
```

## Key Features at a Glance

| Feature | Description | Access |
|---------|-------------|--------|
| Module Browser | Browse all registered modules | Home page |
| Version History | View all versions of a module | Overview tab |
| Type Explorer | Inspect message and enum definitions | Types tab |
| API Explorer | Interactive service browser | API Explorer tab |
| Code Examples | Multi-language code generation | Usage Examples tab |
| Schema Diff | Compare versions side-by-side | Migration tab |
| Search | Fast full-text search | CMD+K / CTRL+K |
| Downloads | Get compiled artifacts | Overview tab |

## Accessibility

The Spoke Web UI is built with accessibility as a priority:
- **Keyboard Navigation**: All features accessible via keyboard
- **Screen Reader Support**: ARIA labels and semantic HTML
- **High Contrast**: WCAG 2.1 AA compliant color scheme
- **Focus Management**: Clear focus indicators
- **Responsive Design**: Works on desktop, tablet, and mobile

## Performance

The UI is optimized for speed:
- **Code Splitting**: Lazy-loaded components reduce initial load
- **Client-Side Search**: Fast search without server round-trips
- **Caching**: Smart caching reduces redundant requests
- **Skeleton Screens**: Instant feedback during loading

## Browser Support

The Spoke Web UI supports:
- Chrome/Edge (latest 2 versions)
- Firefox (latest 2 versions)
- Safari (latest 2 versions)
- Mobile browsers (iOS Safari, Chrome Mobile)

## Next Steps

Choose a topic to dive deeper:

- [**API Explorer**](api-explorer) - Learn to browse services and methods interactively
- [**Code Examples**](code-examples) - See examples in your favorite language
- [**Request Playground**](playground) - Build and test requests
- [**Version Comparison**](migration-tools) - Understand schema changes
- [**Search**](search) - Master the search features

## Feedback

Found an issue or have a suggestion?
- GitHub Issues: [platinummonkey/spoke/issues](https://github.com/platinummonkey/spoke/issues)
- Documentation: [spoke.dev/docs](https://spoke.dev/docs)
