# Impact Analysis

## Overview

The Impact Analysis feature helps developers understand the ripple effects of changes to protobuf schemas by visualizing which modules depend on a given module/version. This enables informed decision-making about breaking changes, deprecations, and version management.

## Purpose

When modifying protobuf schemas, it's critical to understand:
- **How many modules will be affected** by changes
- **Which modules directly depend** on your schema
- **Which modules transitively depend** through other dependencies
- **The severity of impact** based on the number of affected modules

Impact Analysis provides this visibility before making changes, helping prevent accidental breakage across distributed systems.

## Features

### 1. Impact Severity Levels

Visual indicators based on the number of affected modules:

**Success (Green) - 0 modules:**
- No dependencies
- Safe to make breaking changes
- Good opportunity for refactoring

**Info (Blue) - 1-5 modules:**
- Low impact
- Small number of modules affected
- Manageable coordination required

**Warning (Yellow) - 6-10 modules:**
- Medium impact
- Several modules need updates
- Requires careful planning

**Error (Red) - 10+ modules:**
- High impact
- Many modules depend on this
- Breaking changes should be avoided
- Consider major version bump

### 2. Direct Dependents

**Orange Badges** show modules that directly import the analyzed module.

**Characteristics:**
- Immediate impact from any changes
- Must be updated if breaking changes occur
- Require recompilation after schema updates
- Highest priority for coordination

**Example:**
```
If order-service directly imports user@v1.0.0:
  → order-service@v2.1.0 [orange badge]
```

### 3. Transitive Dependents

**Yellow Badges** show modules that depend indirectly through other dependencies.

**Characteristics:**
- May be affected by breaking changes
- Updated when intermediate dependencies update
- Lower priority but still important
- Can have cascading effects

**Example:**
```
If analytics-service imports order-service, which imports user@v1.0.0:
  → analytics-service@v1.5.0 [yellow badge]
```

### 4. Breaking Changes Warning

When dependents exist, a warning box provides guidance:
- Publish new major version instead
- Maintain backward compatibility
- Coordinate with dependent owners
- Provide migration guides

### 5. Best Practices Guide

Built-in recommendations for schema evolution:
- **Additive Changes**: Add new fields (backward compatible)
- **Field Numbers**: Never reuse (breaks binary compatibility)
- **Deprecation**: Mark before removing
- **New Versions**: Create major versions for breaking changes
- **Testing**: Test with all direct dependents

### 6. Collapsible Lists

For modules with many transitive dependents (>10):
- Summary badge shows total count
- Expandable accordion reveals full list
- Improves readability for high-impact modules

## User Interface

### Impact Analysis Tab

**Location:** Module Detail Page → Impact Tab

**Sections:**
1. **Impact Summary Alert**
   - Severity indicator (color-coded)
   - Total impact count
   - Direct/transitive breakdown
   - Severity message

2. **Breaking Changes Warning**
   - Appears when impact > 0
   - Provides guidance for safe changes
   - Lists best practices

3. **Direct Dependents Section**
   - List of modules with orange badges
   - Click to navigate to dependent module
   - Hover effect for visual feedback

4. **Transitive Dependents Section**
   - List of modules with yellow badges
   - Collapsible if >10 modules
   - Click to navigate to dependent module

5. **Safe to Modify Box**
   - Appears when impact = 0
   - Green checkmark icon
   - Encourages refactoring opportunities

6. **Best Practices Box**
   - Always visible when impact > 0
   - Blue background for visibility
   - Structured guidance on schema changes

7. **API Endpoint Reference**
   - Shows the API endpoint used
   - Helpful for automation/scripting

## API Integration

### Backend Endpoint

**URL:** `GET /modules/{name}/versions/{version}/impact`

**Response Format:**
```json
{
  "module": "user",
  "version": "v1.0.0",
  "direct_dependents": [
    {
      "module": "order-service",
      "version": "v2.1.0",
      "type": "direct"
    },
    {
      "module": "auth-service",
      "version": "v1.3.0",
      "type": "direct"
    }
  ],
  "transitive_dependents": [
    {
      "module": "analytics-service",
      "version": "v1.5.0",
      "type": "transitive"
    }
  ],
  "total_impact": 3
}
```

### Frontend Component

**File:** `web/src/components/ImpactAnalysis.tsx`

**Props:**
```typescript
interface ImpactAnalysisProps {
  moduleName: string;
  version: string;
}
```

**State Management:**
- Fetches data on mount
- Loading spinner during fetch
- Error handling with retry option
- Cached in component state

## Use Cases

### Use Case 1: Planning Breaking Changes

**Scenario:** Developer wants to remove a deprecated field from a message.

**Steps:**
1. Navigate to module detail page
2. Click "Impact" tab
3. Review impact analysis
4. Note: 15 modules affected (high impact)
5. **Decision:** Create new major version (v2.0.0) instead of modifying v1.0.0

**Result:** Avoids breaking 15 existing modules, provides smooth migration path.

### Use Case 2: Safe Refactoring

**Scenario:** Developer wants to refactor internal message structure.

**Steps:**
1. Navigate to module detail page
2. Click "Impact" tab
3. Review impact analysis
4. Note: 0 modules affected (safe)
5. **Decision:** Proceed with refactoring

**Result:** Confident refactoring without coordination overhead.

### Use Case 3: Deprecation Planning

**Scenario:** Team wants to deprecate an old service definition.

**Steps:**
1. Navigate to module detail page
2. Click "Impact" tab
3. Review impact analysis
4. Note: 3 direct dependents
5. **Decision:** Mark as deprecated, coordinate with 3 teams, remove in next major version

**Result:** Coordinated deprecation with affected teams.

### Use Case 4: Coordination Scope

**Scenario:** DevOps planning schema registry update.

**Steps:**
1. Check impact for each module being updated
2. Export list of affected modules
3. Calculate total coordination effort
4. **Decision:** Schedule update during maintenance window, notify all affected teams

**Result:** Smooth rollout with proper communication.

## Integration with Dependency Graph

The Impact Analysis complements the Dependency Graph:

**Dependency Graph (Dependencies Tab):**
- Shows what **this module depends on**
- Visualizes upstream dependencies
- Helps understand import structure

**Impact Analysis (Impact Tab):**
- Shows what **depends on this module**
- Visualizes downstream dependents
- Helps understand change impact

**Together:** Complete picture of module relationships.

## Algorithm Details

### Direct Dependents Calculation

```
For each module M in registry:
  For each version V of M:
    If V.dependencies contains (target_module, target_version):
      Add (M, V) to direct_dependents
```

**Complexity:** O(n*m) where n = modules, m = avg versions per module

### Transitive Dependents Calculation

```
transitive_dependents = {}
queue = [direct_dependents]

while queue not empty:
  current = queue.pop()
  For each module M that depends on current:
    If M not in transitive_dependents:
      Add M to transitive_dependents
      Add M to queue
```

**Complexity:** O(n*d) where n = modules, d = avg dependency depth

### Performance Optimization

**Backend:**
- Dependency graph cached in memory
- Lazy loading (only build when requested)
- Graph traversal uses visited set (prevents cycles)

**Frontend:**
- Single API call on tab open
- Results cached in component state
- No polling or real-time updates (snapshot)

## Best Practices for Schema Changes

### Additive Changes (Safe)

**Examples:**
- Add new fields to messages
- Add new messages to proto file
- Add new services
- Add new methods to services

**Impact:** None (backward compatible)

**Migration:** None required

### Field Number Changes (Dangerous)

**Never:**
- Reuse field numbers
- Change field numbers
- Renumber fields

**Impact:** Breaks binary compatibility

**Migration:** Impossible (corrupted data)

### Deprecation (Recommended)

**Steps:**
1. Mark field as deprecated in proto file
2. Add migration documentation
3. Wait for dependents to migrate (1-2 versions)
4. Remove in next major version

**Example:**
```protobuf
message User {
  string id = 1;
  string name = 2 [deprecated = true]; // Use full_name instead
  string full_name = 3;
}
```

### Major Version Bumps (Breaking Changes)

**When to use:**
- Removing fields
- Changing field types
- Renaming services/methods
- Restructuring messages

**Process:**
1. Create new version (e.g., v2.0.0)
2. Maintain v1.x for compatibility period
3. Provide migration guide
4. Coordinate with dependents
5. Deprecate v1.x after migration window

## Troubleshooting

### Impact Shows 0 but Module is Used

**Possible Causes:**
1. Dependencies not tracked in registry
2. Module used but not declared as dependency
3. Dependency graph not built yet

**Solutions:**
- Ensure all modules push dependencies to registry
- Check dependency declarations in proto files
- Rebuild dependency graph if stale

### Transitive Dependents Missing

**Possible Causes:**
1. Incomplete dependency graph
2. Circular dependencies breaking traversal
3. Missing intermediate modules

**Solutions:**
- Verify all intermediate modules exist in registry
- Check for circular dependency warnings
- Use Dependency Graph tab to visualize structure

### High Impact Unexpected

**Possible Causes:**
1. Module widely used (core infrastructure)
2. Transitive dependencies amplify impact
3. Old versions still in use

**Solutions:**
- Review dependency graph to understand usage
- Consider providing adapter/shim layer
- Coordinate breaking changes carefully

### API Endpoint Not Responding

**Possible Causes:**
1. Module/version doesn't exist
2. Dependency resolver error
3. Database query timeout

**Solutions:**
- Verify module exists: `GET /modules/{name}/versions/{version}`
- Check server logs for errors
- Reduce dependency graph size (prune old versions)

## Future Enhancements

### Impact Severity Customization

Allow teams to configure severity thresholds:
```json
{
  "low_impact": 3,
  "medium_impact": 8,
  "high_impact": 15
}
```

### Critical Service Flagging

Mark certain modules as "critical":
- Highlight in impact analysis
- Block breaking changes without approval
- Require extra testing

### Impact Diff

Compare impact between versions:
```
v1.0.0: 5 dependents
v1.1.0: 8 dependents (↑3)
```

### Export Impact Report

Generate reports for:
- Breaking change proposals
- Deprecation planning
- Quarterly reviews

**Formats:**
- Markdown
- CSV
- JSON
- PDF

### Notifications

Alert dependent module owners:
- Email notifications
- Slack/Teams integration
- In-app notifications
- GitHub issues

### Impact Timeline

Show historical impact:
```
Chart showing dependents over time:
v1.0.0 (Jan): 2 dependents
v1.1.0 (Feb): 5 dependents
v1.2.0 (Mar): 8 dependents
```

### Dependency Health Score

Calculate module health:
- Impact factor (how many depend on it)
- Stability (breaking change frequency)
- Maintenance (update frequency)
- Documentation quality

## Accessibility

### Keyboard Navigation

- Tab through dependent badges
- Enter to navigate to module
- Escape to close accordions

### Screen Reader Support

- ARIA labels for severity levels
- Semantic HTML for alerts
- Role attributes for badges
- Status announcements for loading states

### Color Contrast

All severity colors meet WCAG 2.1 AA standards:
- Green: #38A169 (success)
- Blue: #3182CE (info)
- Yellow: #D69E2E (warning)
- Red: #E53E3E (error)

### Focus Indicators

- Clear focus outlines on badges
- Visible hover states
- Consistent focus ring style

## Performance Metrics

### Backend Performance

- Impact calculation: <100ms (p95)
- Graph traversal: O(n) with memoization
- API response size: ~1-10KB typically

### Frontend Performance

- Initial render: <50ms
- API fetch: <200ms (p95)
- Badge rendering: Virtual scrolling if >100 items
- Accordion animation: 200ms transition

### Scaling Considerations

**Small registries (<100 modules):**
- No performance issues
- All features work smoothly

**Medium registries (100-1000 modules):**
- Graph traversal may take 100-500ms
- Consider caching impact results
- Virtual scrolling for large lists

**Large registries (>1000 modules):**
- Implement background workers for graph building
- Cache impact results with TTL
- Pagination for transitive dependents
- Consider distributed graph storage

## Summary

Impact Analysis provides critical visibility into schema change effects:

- **Visual severity indicators** - Quickly assess impact level
- **Direct and transitive dependents** - Complete dependency picture
- **Breaking change warnings** - Prevent accidental breakage
- **Best practices guidance** - In-context recommendations
- **Integrated with module details** - One-click access

Use Impact Analysis before making schema changes to ensure smooth coordination and prevent system-wide breakage.
