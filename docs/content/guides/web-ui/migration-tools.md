---
title: "Migration Tools"
weight: 4
---

# Version Comparison & Migration Tools

Spoke provides powerful tools for understanding schema changes between versions and planning safe upgrades with automated diff detection and manual migration guides.

## Overview

The **Migration** tab (5th tab in Module Detail) offers two complementary tools:

1. **Schema Diff**: Automated breaking change detection
2. **Migration Guide**: Human-authored upgrade instructions

Together, these tools help you:
- Identify breaking changes automatically
- Understand the impact of upgrades
- Plan migration strategies
- Minimize downtime during upgrades

## Accessing Migration Tools

1. Navigate to any module
2. Click the **Migration** tab (5th tab)
3. Select versions to compare:
   - **From Version**: Older version (baseline)
   - **To Version**: Newer version (target)
4. Choose sub-tab: **Schema Diff** or **Migration Guide**

## Version Selector

### Default Behavior

By default, the tool compares:
- **From**: Second-newest version
- **To**: Newest version

This shows what changed in the latest release.

### Custom Comparison

Select any two versions:
- Compare current production → latest
- Compare any historical versions
- Skip versions (v1.0 → v3.0)

**Example:**
```
From: v1.5.0  →  To: v2.0.0
```

### Warning

If you select the same version for both:
```
⚠️ Please select two different versions to compare.
```

## Schema Diff

The Schema Diff tab provides automated analysis of changes between proto definitions.

### Statistics Dashboard

At the top, a summary shows:

```
┌────────────────────────────────────────┐
│  📊 Changes Summary                    │
├────────────────────────────────────────┤
│  Breaking:      3                      │
│  Non-Breaking:  5                      │
│  Warnings:      1                      │
│  Total:         9                      │
└────────────────────────────────────────┘
```

**Color Coding:**
- 🔴 **Breaking** (Red): Changes that break backward compatibility
- 🟢 **Non-Breaking** (Green): Safe, backward-compatible changes
- 🟡 **Warnings** (Yellow): Potential compatibility concerns

### Change Types

The diff engine detects 16 types of changes:

#### Breaking Changes

**Field Removed:**
```
❌ Breaking: field_removed
Location: user.proto:message User:field phone_number
Old: string phone_number = 3
New: (removed)
Description: Field 'phone_number' was removed from message 'User'
Migration Tip: Update all code referencing User.phone_number
```

**Type Changed:**
```
❌ Breaking: type_changed
Location: user.proto:message User:field age
Old: string age = 4
New: int32 age = 4
Description: Field 'age' type changed from string to int32
Migration Tip: Update serialization code to handle int32 instead of string
```

**Field Number Changed:**
```
❌ Breaking: field_number_changed
Location: user.proto:message User:field email
Old: string email = 2
New: string email = 5
Description: Field 'email' number changed from 2 to 5
Migration Tip: CRITICAL: Do not deploy without migrating all data
```

**Message Removed:**
```
❌ Breaking: message_removed
Location: user.proto:message LegacyUser
Old: message LegacyUser { ... }
New: (removed)
Description: Message 'LegacyUser' was removed
Migration Tip: Update code to use new User message type
```

**Enum Value Removed:**
```
❌ Breaking: enum_value_removed
Location: user.proto:enum Status:value PENDING
Old: PENDING = 1
New: (removed)
Description: Enum value 'PENDING' was removed from Status
Migration Tip: Replace PENDING usage with new status values
```

**Method Removed:**
```
❌ Breaking: method_removed
Location: user.proto:service UserService:method DeleteUser
Old: rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse)
New: (removed)
Description: Method 'DeleteUser' was removed from service UserService
Migration Tip: Remove all client calls to DeleteUser
```

#### Non-Breaking Changes

**Field Added (Optional):**
```
✓ Non-Breaking: field_added
Location: user.proto:message User:field middle_name
Old: (not present)
New: string middle_name = 6
Description: Optional field 'middle_name' was added to message 'User'
Migration Tip: Field is optional and backward compatible
```

**Message Added:**
```
✓ Non-Breaking: message_added
Location: user.proto:message UserPreferences
Old: (not present)
New: message UserPreferences { ... }
Description: New message 'UserPreferences' was added
Migration Tip: New functionality available, no migration required
```

**Enum Value Added:**
```
✓ Non-Breaking: enum_value_added
Location: user.proto:enum Status:value ARCHIVED
Old: (not present)
New: ARCHIVED = 4
Description: Enum value 'ARCHIVED' was added to Status
Migration Tip: New status available for use
```

**Method Added:**
```
✓ Non-Breaking: method_added
Location: user.proto:service UserService:method BatchGetUsers
Old: (not present)
New: rpc BatchGetUsers(BatchGetUsersRequest) returns (BatchGetUsersResponse)
Description: Method 'BatchGetUsers' was added to service UserService
Migration Tip: New method available for batch operations
```

#### Warnings

**Field Added (Required):**
```
⚠ Warning: field_added
Location: user.proto:message CreateUserRequest:field tenant_id
Old: (not present)
New: string tenant_id = 5
Description: Required field 'tenant_id' was added
Migration Tip: Requires code changes to populate new field
```

### Change List

Below the dashboard, an accordion shows all changes:

```
▼ Breaking: Field Removed - user.proto:message User:field phone_number
    Old Value: string phone_number = 3
    New Value: (removed)

    📝 Migration Tip:
    Update all code that references User.phone_number. Consider
    using the new ContactInfo message for phone numbers.

▼ Non-Breaking: Field Added - user.proto:message User:field bio
    Old Value: (not present)
    New Value: string bio = 7

    📝 Migration Tip:
    Optional field is backward compatible. Older clients will
    ignore this field.
```

**Interaction:**
- Click to expand/collapse details
- Multiple changes can be expanded simultaneously
- Scroll through long lists

### Severity Classification

Changes are automatically classified by severity:

| Severity | Risk Level | Examples |
|----------|-----------|----------|
| Breaking | High | Field removed, type changed, number changed |
| Non-Breaking | None | Optional field added, new message/enum/method |
| Warning | Medium | Required field added, deprecated usage |

### Using Schema Diff

#### Before Upgrading

1. Navigate to Migration tab
2. Select current production version → new version
3. Review breaking changes carefully
4. Count breaking changes (aim for zero in minor versions)
5. Check warnings for hidden issues

#### Planning Migration

1. Export or screenshot the diff
2. Share with team for review
3. Create tickets for each breaking change
4. Estimate migration effort
5. Plan rollout strategy (canary, blue-green, etc.)

#### During Code Review

1. Compare feature branch proto → main
2. Verify no accidental breaking changes
3. Ensure appropriate version bump (major vs minor)
4. Document changes in CHANGELOG

## Migration Guide

The **Migration Guide** tab shows human-authored upgrade documentation.

### Guide Structure

A typical migration guide includes:

```markdown
# Migration Guide: user-service v1.0.0 → v1.1.0

## Overview
- 2 breaking changes
- 3 new features
- 1 deprecation

## Breaking Changes

### 1. Field Removed: User.phone_number

**What changed:**
The `phone_number` field was removed from the `User` message.

**Before:**
```protobuf
message User {
  string user_id = 1;
  string email = 2;
  string phone_number = 3;  // REMOVED
}
```

**After:**
```protobuf
message User {
  string user_id = 1;
  string email = 2;
  ContactInfo contact_info = 3;  // NEW
}

message ContactInfo {
  string phone = 1;
  string address = 2;
}
```

**Migration Steps:**
1. Search codebase for `user.phone_number`
2. Replace with `user.contact_info.phone`
3. Update database schema if storing User protos
4. Run integration tests

## New Features

### 1. Batch Operations
New methods for bulk operations:
- `BatchGetUsers` - Fetch multiple users at once
- `BatchUpdateUsers` - Update multiple users

**Example:**
```go
resp, err := client.BatchGetUsers(ctx, &pb.BatchGetUsersRequest{
    UserIds: []string{"123", "456", "789"},
})
```

## Testing Recommendations

1. Run existing test suite against new version
2. Test all user-related API calls
3. Verify batch operations work correctly
4. Load test with production-like data

## Rollback Plan

If issues occur:
1. **Before deploying**: Tag current deployment
2. **If problems**: Revert to v1.0.0
3. **Database**: No schema changes, safe to rollback
```

### Guide Availability

**When guides exist:**
- Markdown content renders with proper formatting
- Code blocks are syntax highlighted
- Tables and lists display correctly
- Links are clickable

**When guides don't exist:**
```
ℹ️ No manual migration guide available.

Use the Schema Diff tab for automated change detection.

Want to contribute a guide?
1. Create file: migrations/{module}/v{from}-to-v{to}.md
2. Follow the template structure
3. Submit a pull request
```

### Guide Location

Migration guides are stored as markdown files:

```
docs/content/migrations/
├── user-service/
│   ├── v1.0.0-to-v1.1.0.md
│   ├── v1.1.0-to-v2.0.0.md
│   └── v2.0.0-to-v2.1.0.md
└── order-service/
    └── v1.0.0-to-v1.1.0.md
```

Accessible via:
```
/migrations/{module}/v{from}-to-v{to}.md
```

## Migration Workflow

### 1. Review Phase

**Using Schema Diff:**
1. Select versions to compare
2. Review statistics (breaking/non-breaking/warnings)
3. Expand and read each breaking change
4. Note migration tips

**Using Migration Guide:**
1. Read overview section
2. Review all breaking changes
3. Check new features
4. Note deprecations

### 2. Planning Phase

**Create Migration Plan:**
1. List all breaking changes
2. Estimate effort for each change
3. Identify affected services/teams
4. Create migration tickets/tasks
5. Set timeline and milestones

**Example Plan:**

```
Sprint 1:
- Remove User.phone_number references
- Update to ContactInfo structure
- Write unit tests

Sprint 2:
- Deploy to staging
- Integration testing
- Performance testing

Sprint 3:
- Canary deployment (10%)
- Monitor metrics
- Full rollout if stable
```

### 3. Implementation Phase

**Code Changes:**
1. Update proto imports
2. Modify affected code
3. Run local tests
4. Commit with descriptive message

**Testing:**
1. Unit tests for changed code
2. Integration tests end-to-end
3. Regression tests for unchanged features
4. Load tests if performance-critical

### 4. Deployment Phase

**Staged Rollout:**
1. Deploy to dev/test environment
2. Smoke tests and validation
3. Deploy to staging
4. Full integration test suite
5. Deploy to production (canary or blue-green)

**Monitoring:**
1. Watch error rates
2. Monitor latency metrics
3. Check logs for unexpected errors
4. Have rollback plan ready

### 5. Post-Deployment

**Validation:**
1. Verify all services healthy
2. Check dependent services
3. Monitor for 24-48 hours
4. Document any issues

**Cleanup:**
1. Remove deprecated code after grace period
2. Update documentation
3. Close migration tickets
4. Retrospective meeting

## Best Practices

### Schema Evolution

**Do:**
- ✅ Add optional fields (non-breaking)
- ✅ Add new messages and enums
- ✅ Add new service methods
- ✅ Reserve removed field numbers
- ✅ Deprecate before removing

**Don't:**
- ❌ Change field types
- ❌ Change field numbers
- ❌ Remove required fields without deprecation
- ❌ Reuse field numbers
- ❌ Make optional fields required

### Version Numbering

Follow semantic versioning:
- **Major (v2.0.0)**: Breaking changes
- **Minor (v1.1.0)**: New features, backward compatible
- **Patch (v1.0.1)**: Bug fixes, no API changes

### Deprecation Strategy

Before removing a field:

```protobuf
message User {
  string user_id = 1;
  string email = 2;
  string phone_number = 3 [deprecated = true];  // Deprecated in v1.1
  ContactInfo contact_info = 4;  // Use this instead
}
```

1. Mark as deprecated
2. Document replacement
3. Give grace period (2-3 versions)
4. Remove in major version bump

### Communication

**Announce changes:**
1. Send migration guide to stakeholders
2. Post in team channels
3. Update API documentation
4. Include in release notes
5. Provide support during migration

## Troubleshooting

### Diff shows no changes

**Symptoms**: Both versions appear identical
**Possible causes**:
- Selected same version twice
- Proto files not different
- Comparison not refreshed

**Fix**: Verify different versions selected, refresh page

### Too many changes to review

**Symptoms**: 50+ changes listed
**Strategy**:
- Filter by breaking changes first
- Group by proto file
- Create spreadsheet for tracking
- Split review across team

### Migration guide not found

**Symptoms**: "No manual migration guide available"
**Reasons**:
- Guide not created yet
- File naming doesn't match pattern
- File not in correct location

**Fix**: Use Schema Diff, or contribute a guide

### Unexpected breaking change

**Symptoms**: Change marked breaking but seems safe
**Check**:
- Field removal (always breaking)
- Type change (always breaking)
- Required field addition (breaking)
- Re-read migration tip

## Creating Migration Guides

Want to contribute a migration guide?

### Template

Use this structure:

```markdown
# Migration Guide: {module} v{from} → v{to}

## Overview
- X breaking changes
- Y new features
- Z deprecations

## Breaking Changes
### 1. {Change Title}
**What changed:** ...
**Before:** ...
**After:** ...
**Migration Steps:** ...

## New Features
### 1. {Feature Title}
**Description:** ...
**Example:** ...

## Deprecations
- {Field/Method} deprecated, use {Replacement}

## Testing Recommendations
1. ...

## Rollback Plan
- ...

## Support
- ...
```

### Best Practices for Guides

- Be specific with code examples
- Include before/after comparisons
- Provide step-by-step instructions
- Add test recommendations
- Document rollback procedures
- Link to related documentation

## What's Next?

- [**API Explorer**](api-explorer) - Understand new methods
- [**Code Examples**](code-examples) - Get updated client code
- [**CLI Reference**](../cli-reference) - Download specific versions
- [**Version Management**](../../examples/versioning) - Best practices

## Related Documentation

- [Semantic Versioning](https://semver.org/) - Version numbering standard
- [Protobuf Evolution](https://protobuf.dev/programming-guides/proto3/#updating) - Official guide
- [API Evolution Patterns](../../examples/api-evolution) - Safe change patterns
