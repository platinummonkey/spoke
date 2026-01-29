# Test Coverage Improvement Plan

## Current Situation

The CI is failing due to coverage thresholds defined in `.testcoverage.yml`:
- **File threshold**: 60%
- **Package threshold**: 65%
- **Total threshold**: 65%

Current total coverage: **32.1%** (target: 65%)

## Files Requiring Coverage

CI identified **70+ files with 0% coverage** and many more below 60%.

## Completed Improvements

### ✅ pkg/cli/compile.go
- **Before**: 53.2%
- **After**: ~80%+ (runCompile: 78.4%, other functions: 100%)
- **Tests Added**: 15+ new test cases covering all languages, flags, error paths

### ✅ pkg/linter/registry.go
- **Before**: 53.3%
- **After**: 100% (all functions)
- **Impact**: Linter package now at 67.2% (above 65% threshold)

### ✅ pkg/analytics/alerts.go
- **Before**: 52.5%
- **After**: ~85%+ (all functions above 80%)
- **Critical**: CheckAllAlerts (0% → 88.9%), SendAlert (0% → 100%)

## Packages Meeting Thresholds

- ✅ **pkg/linter**: 67.2% (above 65%)
- ✅ **pkg/linter/rules**: 79.0%
- ✅ **pkg/codegen/languages**: 70.7%
- ✅ **pkg/dependencies**: 72.3%
- ✅ **pkg/compatibility**: 69.4%
- ✅ **pkg/async**: 86.2%
- ✅ **pkg/codegen/packages/gomod**: 81.8%

## Remaining Challenges

### High-Priority Packages (Need Work)
1. **pkg/api**: Core API handlers - many at 0-10% coverage
2. **pkg/codegen**: Core compilation features - 0% in many files
3. **pkg/middleware**: Security/auth middleware - 21.5%
4. **pkg/storage/postgres**: Database layer - 0.9%
5. **pkg/observability**: Monitoring - 19.0%
6. **pkg/cli**: CLI commands - 30.8%

### Files With 0% Coverage (Examples)
- `pkg/api/server.go`
- `pkg/api/languages.go`
- `pkg/api/search_adapter.go`
- `pkg/codegen/artifacts/manager.go`
- `pkg/codegen/cache/cache.go`
- `pkg/config/config.go`
- `pkg/middleware/auth.go`
- `pkg/middleware/quota.go`
- And 60+ more...

## Recommendations

### Option 1: Adjust Coverage Thresholds (Pragmatic)
Lower thresholds to achievable levels while improving incrementally:

```yaml
# .testcoverage.yml
threshold:
  file: 40      # was 60
  package: 50   # was 65
  total: 50     # was 65
```

**Rationale**:
- Current 32.1% → 50% is achievable with focused effort
- 60-65% would require comprehensive testing of entire codebase
- Allows incremental improvement without blocking development

### Option 2: Exclude Low-Priority Packages
Exclude packages that aren't critical:

```yaml
exclude:
  paths:
    - \.pb\.go$
    - \.pb\.gw\.go$
    - _gen\.go$
    - ^cmd/
    - ^examples/
    - /testdata/
    - _test\.go$
    # Add these:
    - ^pkg/marketplace/     # Feature in development
    - ^pkg/billing/         # External integration
    - ^pkg/sso/             # Optional feature
    - ^pkg/webhooks/        # Optional feature
    - ^pkg/docs/            # Documentation tools
```

### Option 3: Incremental Improvement Plan
Target packages incrementally:

**Phase 1** (Target: 45% total):
- ✅ pkg/linter (done: 67.2%)
- pkg/api core handlers
- pkg/codegen/orchestrator

**Phase 2** (Target: 55% total):
- pkg/middleware
- pkg/storage/postgres
- pkg/cli commands

**Phase 3** (Target: 65% total):
- pkg/observability
- pkg/plugins
- Remaining packages

## Quick Wins Still Available

Files close to 60% that need minor work:
- `pkg/plugins/buf/adapter.go`: 58.1% → need 2% more
- `pkg/api/protobuf/scanner.go`: 52.6% → need 7-8% more
- `pkg/api/protobuf/ast.go`: 53.1% → need 7% more
- `pkg/middleware/ratelimit.go`: 46.6% → need 13-14% more

## Next Steps

1. **Decide on approach** (Option 1, 2, or 3 above)
2. **Update `.testcoverage.yml`** if lowering thresholds
3. **Continue adding tests** for high-priority packages
4. **Enable coverage trend tracking** to prevent regression

## Testing Best Practices

For adding tests:
1. Use `sqlmock` for database operations
2. Use `httptest` for HTTP handlers
3. Focus on business logic, not just coverage percentage
4. Test error paths and edge cases
5. Mock external dependencies (Redis, S3, etc.)

## Coverage Analysis Commands

```bash
# Run tests with coverage
go test -coverprofile=coverage.out -covermode=atomic ./...

# View total coverage
go tool cover -func=coverage.out | grep total

# View per-file coverage
go tool cover -func=coverage.out | less

# HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

## Conclusion

Achieving 65% coverage for this codebase requires significant effort. The tests added in this session demonstrate feasible patterns, but scaling to 65% would require:

- **Estimated effort**: 40-60 hours for 65% total coverage
- **Alternative**: Lower thresholds to 40-50% (achievable in 10-15 hours)
- **Recommended**: Start with Option 1 (lower thresholds), improve incrementally

The foundation is in place. The question is: do we need 65% coverage now, or can we achieve it incrementally?
