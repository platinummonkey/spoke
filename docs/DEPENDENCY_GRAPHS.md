# Dependency Graph Visualization

## Overview

Spoke provides interactive dependency graph visualization using Cytoscape.js, allowing developers to understand module relationships, identify circular dependencies, and analyze impact of changes.

## Features

### Interactive Graph
- **Multiple Layouts**: Hierarchical (Dagre), Force-Directed (Cola), Breadth-First, Circle, Grid
- **Node Coloring**: Visual distinction between current module (blue), dependencies (green), and dependents (orange)
- **Click Navigation**: Click any node to navigate to that module
- **Zoom and Pan**: Mouse wheel to zoom, drag to pan
- **Export**: Save graph as PNG or JSON

### Graph Types

**1. Dependencies Graph** (default)
```
Current Module → Dependencies → Transitive Dependencies
```
Shows what the current module depends on.

**2. Dependents Graph**
```
Dependents → Current Module
```
Shows what modules depend on the current module.

**3. Both Directions**
```
Dependents → Current Module → Dependencies
```
Full bidirectional dependency view.

## API Endpoints

### Cytoscape.js Format

**Endpoint:** `GET /api/v2/modules/{name}/versions/{version}/graph`

**Query Parameters:**
- `transitive` (boolean) - Include transitive dependencies (default: true)
- `depth` (integer) - Max depth for transitive dependencies (default: unlimited)
- `direction` (string) - "dependencies", "dependents", or "both" (default: "dependencies")

**Example Request:**
```bash
curl "http://localhost:8080/api/v2/modules/user/versions/v1.0.0/graph?transitive=true&direction=both"
```

**Response Format:**
```json
{
  "nodes": [
    {
      "data": {
        "id": "user:v1.0.0",
        "name": "user",
        "version": "v1.0.0",
        "type": "current"
      }
    },
    {
      "data": {
        "id": "common:v1.0.0",
        "name": "common",
        "version": "v1.0.0",
        "type": "dependency"
      }
    },
    {
      "data": {
        "id": "order-service:v1.2.0",
        "name": "order-service",
        "version": "v1.2.0",
        "type": "dependent"
      }
    }
  ],
  "edges": [
    {
      "data": {
        "id": "user:v1.0.0->common:v1.0.0",
        "source": "user:v1.0.0",
        "target": "common:v1.0.0",
        "type": "direct"
      }
    },
    {
      "data": {
        "id": "order-service:v1.2.0->user:v1.0.0",
        "source": "order-service:v1.2.0",
        "target": "user:v1.0.0",
        "type": "depends-on"
      }
    }
  ]
}
```

### Legacy Format

**Endpoint:** `GET /modules/{name}/versions/{version}/graph`

Returns nodes and edges in a simpler format with circular dependency detection.

## Frontend Component

### DependencyGraph Component

**Location:** `web/src/components/DependencyGraph.tsx`

**Props:**
```typescript
interface DependencyGraphProps {
  moduleName: string;
  version: string;
  transitive?: boolean;
  direction?: 'dependencies' | 'dependents' | 'both';
  maxDepth?: number;
  onNodeClick?: (module: string, version: string) => void;
}
```

**Example Usage:**
```tsx
<DependencyGraph
  moduleName="user"
  version="v1.0.0"
  transitive={true}
  direction="both"
  onNodeClick={(module, version) => {
    navigate(`/modules/${module}?version=${version}`);
  }}
/>
```

### Integration in ModuleDetail

The dependency graph is integrated as a tab in the `ModuleDetail` component:

```tsx
<Tabs>
  <TabList>
    <Tab>Overview</Tab>
    <Tab>Types</Tab>
    <Tab>API Explorer</Tab>
    <Tab>Dependencies</Tab>  {/* NEW */}
    <Tab>Usage Examples</Tab>
    <Tab>Migration</Tab>
  </TabList>

  <TabPanels>
    {/* ... other tabs ... */}
    <TabPanel>
      <DependencyGraph
        moduleName={module.name}
        version={selectedVersion}
        transitive={true}
        direction="both"
        onNodeClick={(module, version) => {
          window.location.href = `/modules/${module}?version=${version}`;
        }}
      />
    </TabPanel>
  </TabPanels>
</Tabs>
```

## Layout Options

### 1. Hierarchical (Dagre)
**Best for:** Clear dependency hierarchies
```
        Root
       /    \
    Dep1    Dep2
     |       |
   Dep3    Dep4
```

**Configuration:**
- `rankDir`: "TB" (Top to Bottom)
- `nodeSep`: 50px between nodes
- `rankSep`: 100px between ranks
- **Use when:** You want to see clear parent-child relationships

### 2. Force-Directed (Cola)
**Best for:** Natural clustering and relationships
```
    [Node1]---[Node2]
       |    X    |
    [Node3]---[Node4]
```

**Configuration:**
- `nodeSpacing`: 50px minimum
- `edgeLength`: 100px ideal
- `animate`: true (smooth transitions)
- **Use when:** You want to see natural groupings and clusters

### 3. Breadth-First
**Best for:** Level-by-level exploration
```
Level 1:  [Root]
Level 2:  [A] [B] [C]
Level 3:  [D] [E] [F] [G]
```

**Configuration:**
- `directed`: true
- `spacingFactor`: 1.5
- **Use when:** You want to explore dependencies level by level

### 4. Circle
**Best for:** Equal importance, showing connections
```
      [A]
   [F]   [B]
  [E]     [C]
      [D]
```

**Configuration:**
- `spacingFactor`: 1.5
- **Use when:** Modules have equal importance

### 5. Grid
**Best for:** Compact overview
```
[A] [B] [C]
[D] [E] [F]
[G] [H] [I]
```

**Configuration:**
- Auto-calculated rows/cols
- **Use when:** You need a compact, organized view

## Node Types

### Current Module (Blue)
- **Color:** `#3182ce` (blue)
- **Size:** 80x80px (larger)
- **Weight:** Bold
- The module being analyzed

### Dependency (Green)
- **Color:** `#48bb78` (green)
- **Size:** 60x60px
- Modules that the current module depends on

### Dependent (Orange)
- **Color:** `#ed8936` (orange)
- **Size:** 60x60px
- Modules that depend on the current module

## Edge Types

### Direct Dependency
- **Style:** Solid line
- **Width:** 3px
- **Color:** Dark gray (`#2d3748`)
- Direct dependency relationship

### Transitive Dependency
- **Style:** Dashed line
- **Width:** 2px
- **Color:** Gray (`#a0aec0`)
- Indirect dependency (dependency of dependency)

### Depends-On
- **Style:** Solid line with arrow
- Used for dependent relationships
- Shows which module depends on the current module

## Export Capabilities

### Export as PNG

**Usage:**
```tsx
<IconButton
  icon={<DownloadIcon />}
  onClick={exportPNG}
  aria-label="Export PNG"
/>
```

**Features:**
- 2x scale for high-quality images
- Preserves layout and styling
- Automatic filename: `{module}-{version}-dependencies.png`

**Use Cases:**
- Documentation
- Presentations
- Architecture diagrams
- Design reviews

### Export as JSON

**Usage:**
```tsx
<Button onClick={exportJSON}>
  Export JSON
</Button>
```

**Features:**
- Full Cytoscape.js format
- Includes nodes, edges, and metadata
- Can be re-imported or processed

**Use Cases:**
- Data analysis
- Custom visualizations
- Integration with other tools
- Automated dependency checks

## Performance Considerations

### Large Graphs

**Problem:** Graphs with >100 nodes can be slow to render

**Solutions:**
1. **Limit Depth:** Use `maxDepth` parameter
   ```
   ?depth=2
   ```
   Limits transitive dependencies to 2 levels

2. **Filter Direction:** Use `direction=dependencies` instead of `both`
   ```
   ?direction=dependencies
   ```
   Reduces node count by 50%

3. **Non-Transitive:** Use `transitive=false`
   ```
   ?transitive=false
   ```
   Only shows direct dependencies

### Rendering Performance

**Cytoscape.js Optimization:**
- Uses Canvas renderer for graphs >50 nodes
- Lazy loading with React.lazy()
- Debounced layout recalculation
- Memoized style configurations

**Recommendations:**
- Graphs <50 nodes: Any layout works well
- Graphs 50-200 nodes: Use Dagre or Breadthfirst
- Graphs >200 nodes: Limit depth or filter

## Use Cases

### 1. Understanding Dependencies

**Scenario:** New developer joins team, needs to understand module relationships

**Action:**
1. Open module in Spoke web UI
2. Click "Dependencies" tab
3. View graph with `direction=both`
4. Click nodes to explore related modules

**Benefit:** Visual understanding of architecture in minutes

### 2. Impact Analysis

**Scenario:** Planning to make breaking changes to a module

**Action:**
1. Open module in Spoke web UI
2. Click "Dependencies" tab
3. View graph with `direction=dependents`
4. Count affected modules

**Benefit:** Understand blast radius before making changes

### 3. Circular Dependency Detection

**Scenario:** Suspecting circular dependencies

**Action:**
1. Use graph endpoint: `GET /modules/{name}/versions/{version}/graph`
2. Check response for `has_circular_dependency` field
3. Examine `circular_path` array

**Benefit:** Identify and fix circular dependencies

### 4. Documentation

**Scenario:** Creating architecture documentation

**Action:**
1. Open module in Spoke web UI
2. Select "Hierarchical (Dagre)" layout
3. Export as PNG
4. Include in documentation

**Benefit:** Always up-to-date dependency diagrams

### 5. Dependency Audits

**Scenario:** Security audit requires understanding all dependencies

**Action:**
1. Export graph as JSON
2. Parse dependencies programmatically
3. Cross-reference with vulnerability databases

**Benefit:** Automated security analysis

## Keyboard Shortcuts

**Note:** Browser zoom controls apply to entire page

- **Zoom In:** Mouse wheel up or `Ctrl` + `+`
- **Zoom Out:** Mouse wheel down or `Ctrl` + `-`
- **Reset Zoom:** `Ctrl` + `0`
- **Pan:** Click and drag background

## Troubleshooting

### Graph Not Loading

**Symptom:** "Failed to load dependency graph" error

**Causes:**
1. Module/version doesn't exist
2. Backend API down
3. No dependencies to show

**Solutions:**
1. Verify module and version exist: `GET /modules/{name}/versions/{version}`
2. Check browser console for API errors
3. Try legacy endpoint: `GET /modules/{name}/versions/{version}/dependencies`

### Graph Too Large

**Symptom:** Browser becomes slow or unresponsive

**Solutions:**
1. Limit depth: `?depth=2`
2. Change direction: `?direction=dependencies`
3. Disable transitive: `?transitive=false`
4. Use simpler layout: Select "Grid" layout

### Layout Looks Wrong

**Symptom:** Nodes overlap or graph is messy

**Solutions:**
1. Try different layout (Dagre usually works best)
2. Refresh graph with refresh button
3. Resize browser window and refresh
4. Export and re-import

### Nodes Not Clickable

**Symptom:** Clicking nodes doesn't navigate

**Solutions:**
1. Check browser console for JavaScript errors
2. Ensure `onNodeClick` handler is provided
3. Try right-click → "Open in new tab" as fallback

## Advanced Features

### Custom Styling

Modify Cytoscape.js styles in `DependencyGraph.tsx`:

```tsx
style: [
  {
    selector: 'node[type="current"]',
    style: {
      'background-color': '#3182ce', // Change color
      width: '100px',                // Change size
      'font-size': '14px',           // Change font
    },
  },
  // ... more styles
]
```

### Custom Layouts

Add new layout configurations:

```tsx
case 'custom':
  return {
    name: 'cose', // Compound Spring Embedder
    idealEdgeLength: 100,
    nodeOverlap: 20,
    animate: true,
  };
```

### Backend Filtering

Extend `graph_visualization.go` to support additional filters:

```go
// Filter by dependency type
if depType := r.URL.Query().Get("type"); depType != "" {
    // Filter edges by type
}

// Filter by module pattern
if pattern := r.URL.Query().Get("module_pattern"); pattern != "" {
    // Filter nodes by pattern
}
```

## Integration with Other Tools

### CI/CD Pipelines

**Example: Detect new dependencies**
```bash
# Before change
curl "/api/v2/modules/user/versions/v1.0.0/graph" > before.json

# After change
curl "/api/v2/modules/user/versions/v1.1.0/graph" > after.json

# Compare
diff before.json after.json
```

### Grafana Dashboards

**Example: Track dependency count over time**
```sql
SELECT
  COUNT(*) as dependency_count,
  module_name,
  version
FROM dependencies
GROUP BY module_name, version
ORDER BY created_at DESC
```

### Prometheus Alerts

**Example: Alert on circular dependencies**
```yaml
- alert: CircularDependencyDetected
  expr: circular_dependencies > 0
  for: 5m
  annotations:
    summary: "Circular dependency detected in {{ $labels.module }}"
```

## References

- [Cytoscape.js Documentation](https://js.cytoscape.org/)
- [Dagre Layout](https://github.com/cytoscape/cytoscape.js-dagre)
- [Cola Layout](https://github.com/cytoscape/cytoscape.js-cola)
- [Backend Implementation](../pkg/dependencies/graph_visualization.go)
- [Frontend Component](../web/src/components/DependencyGraph.tsx)
