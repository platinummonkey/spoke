# GitHub Actions CI/CD Workflows

This document describes the comprehensive GitHub Actions workflows implemented for the Spoke Schema Registry.

## Workflows Overview

### 1. CI Workflow (`.github/workflows/ci.yml`)

**Triggers:**
- Push to `main` or `master` branches
- Pull requests to `main` or `master` branches

**Jobs:**

#### Test Job
- Runs on: `ubuntu-latest`, `macos-latest`, `windows-latest`
- Go versions: `1.22`, `1.23`, `1.24`
- Steps:
  - Checkout code
  - Setup Go with caching
  - Download and verify dependencies
  - Run tests with race detector and coverage (`go test -race -v -coverprofile=coverage.out`)
  - Upload coverage to Codecov (ubuntu-latest, Go 1.24 only)
  - Generate and upload coverage reports

#### Build Job
- Runs on: `ubuntu-latest`
- Depends on: test job
- Steps:
  - Build all three binaries:
    - `spoke-server` (from `./cmd/spoke`)
    - `spoke-cli` (from `./cmd/spoke-cli`)
    - `sprocket` (from `./cmd/sprocket`)
  - Upload binaries as artifacts
  - Test binary execution

#### Integration Test Job
- Runs on: `ubuntu-latest`
- Depends on: build job
- Services:
  - PostgreSQL 16
  - Redis 7
- Steps:
  - Run integration tests with database services

#### Summary Job
- Runs on: `ubuntu-latest`
- Depends on: all previous jobs
- Always runs (even if previous jobs fail)
- Generates CI summary report

**Optimizations:**
- Go module and build caching enabled
- Parallel matrix execution for multiple OS/Go version combinations
- 15-minute timeout to prevent hanging jobs
- Artifact retention for 7 days

---

### 2. Lint Workflow (`.github/workflows/lint.yml`)

**Triggers:**
- Push to `main` or `master` branches
- Pull requests to `main` or `master` branches

**Jobs:**

#### golangci-lint Job
- Runs golangci-lint with comprehensive linter configuration
- Uses `.golangci.yml` configuration file
- Enabled linters:
  - `govet` - Go vet analysis
  - `errcheck` - Check for unchecked errors
  - `staticcheck` - Static analysis
  - `gosimple` - Simplification suggestions
  - `unused` - Unused code detection
  - `ineffassign` - Ineffective assignments
  - `typecheck` - Type checking
  - `gofmt` - Code formatting
  - `goimports` - Import organization
  - `misspell` - Spelling mistakes
  - `gocritic` - Various checks
  - `gocyclo` - Cyclomatic complexity
  - `dupl` - Code duplication
  - `goconst` - Repeated strings
  - `lll` - Line length
  - `funlen` - Function length
  - `godox` - TODO/FIXME comments

#### go-fmt Job
- Checks code formatting with `gofmt -s -l`
- Fails if any files are not properly formatted

#### go-vet Job
- Runs `go vet ./...` for additional static analysis

#### mod-tidy Job
- Verifies `go.mod` and `go.sum` are tidy
- Runs `go mod tidy` and checks for changes

**Optimizations:**
- golangci-lint action has built-in caching
- 5-10 minute timeouts for fast feedback

---

### 3. Security Workflow (`.github/workflows/security.yml`)

**Triggers:**
- Push to `main` or `master` branches
- Pull requests to `main` or `master` branches
- Schedule: Weekly on Mondays at 9:00 UTC

**Jobs:**

#### gosec Job
- Runs gosec security scanner
- Generates SARIF report
- Uploads results to GitHub Security tab

#### govulncheck Job
- Runs Go's official vulnerability checker
- Scans dependencies for known vulnerabilities
- Reports CVEs in Go modules

#### dependency-review Job
- Runs only on pull requests
- Reviews dependency changes
- Fails on moderate or higher severity vulnerabilities

#### trivy Job
- Runs Trivy vulnerability scanner
- Scans filesystem for security issues
- Reports CRITICAL and HIGH severity issues
- Uploads SARIF to GitHub Security tab

**Optimizations:**
- 10-minute timeouts
- SARIF format for GitHub integration
- Scheduled weekly scans for proactive monitoring

---

### 4. Build Workflow (`.github/workflows/build.yml`)

**Triggers:**
- Push to `main` or `master` branches
- Push of tags matching `v*` (e.g., `v1.0.0`)
- Pull requests to `main` or `master` branches

**Jobs:**

#### build-matrix Job
- Cross-platform builds for:
  - **Linux**: amd64, arm64
  - **macOS**: amd64 (Intel), arm64 (Apple Silicon)
  - **Windows**: amd64
- Builds all three binaries with version information
- Generates SHA256 checksums
- Uploads artifacts for each platform

**Build flags:**
- `CGO_ENABLED=0` for static binaries
- `-ldflags="-s -w"` for size optimization
- Version, build date, and git commit embedded

#### release Job
- Runs only on tags
- Downloads all build artifacts
- Creates GitHub Release with:
  - All binaries for all platforms
  - Checksums
  - Auto-generated release notes

**Optimizations:**
- 20-minute timeout for cross-compilation
- 30-day artifact retention
- Automated release creation on tags

---

### 5. Coverage Workflow (`.github/workflows/coverage.yml`)

**Triggers:**
- Push to `main` or `master` branches
- Pull requests to `main` or `master` branches

**Jobs:**

#### coverage Job
- Runs comprehensive test coverage analysis
- Calculates total coverage percentage
- Enforces 70% minimum coverage threshold
- Generates coverage badge with color coding:
  - Green (≥90%)
  - Yellow (≥80%)
  - Orange (≥70%)
  - Red (<70%)
- Creates detailed coverage reports:
  - HTML report
  - Text report with per-package breakdown
- Comments on PRs with coverage summary
- Top 20 packages by coverage included

**Optimizations:**
- 15-minute timeout
- Coverage artifacts uploaded
- PR comments for visibility

---

## Configuration Files

### `.golangci.yml`

Comprehensive linter configuration with:
- 18+ enabled linters
- Customized settings for each linter
- Test file exclusions for certain rules
- 5-minute timeout
- Line length limit: 140 characters
- Cyclomatic complexity limit: 15
- Function length limits: 100 lines or 50 statements

**Key Settings:**
```yaml
linters:
  enable:
    - govet
    - errcheck
    - staticcheck
    - gosimple
    - unused
    - ineffassign
    # ... and more

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - gocyclo
        - funlen
        - dupl
```

---

## Usage

### Running Workflows Locally

To test workflows before pushing:

```bash
# Install act (GitHub Actions local runner)
brew install act  # macOS
# or
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash  # Linux

# Run specific workflow
act -j test
act -j lint
act -j security
```

### Monitoring Workflow Status

All workflows can be monitored at:
```
https://github.com/platinummonkey/spoke/actions
```

### Quality Gates

Before a PR can be merged, the following must pass:
1. All tests pass on all platforms and Go versions
2. No linter errors
3. No security vulnerabilities
4. Code coverage meets 70% threshold
5. All binaries build successfully

### Badges

Add these badges to your README.md:

```markdown
[![CI](https://github.com/platinummonkey/spoke/actions/workflows/ci.yml/badge.svg)](https://github.com/platinummonkey/spoke/actions/workflows/ci.yml)
[![Lint](https://github.com/platinummonkey/spoke/actions/workflows/lint.yml/badge.svg)](https://github.com/platinummonkey/spoke/actions/workflows/lint.yml)
[![Security](https://github.com/platinummonkey/spoke/actions/workflows/security.yml/badge.svg)](https://github.com/platinummonkey/spoke/actions/workflows/security.yml)
[![Coverage](https://github.com/platinummonkey/spoke/actions/workflows/coverage.yml/badge.svg)](https://github.com/platinummonkey/spoke/actions/workflows/coverage.yml)
[![Build](https://github.com/platinummonkey/spoke/actions/workflows/build.yml/badge.svg)](https://github.com/platinummonkey/spoke/actions/workflows/build.yml)
```

---

## Secrets Required

Configure these secrets in GitHub repository settings:

1. **CODECOV_TOKEN** (optional)
   - For uploading coverage to Codecov
   - Get from: https://codecov.io

2. **GITHUB_TOKEN** (automatic)
   - Automatically provided by GitHub Actions
   - Used for creating releases and SARIF uploads

---

## Troubleshooting

### Tests Failing on Windows

If tests fail on Windows but pass on Linux/macOS:
- Check for path separator issues (`/` vs `\`)
- Verify line ending handling (CRLF vs LF)
- Test with `GOOS=windows` locally

### golangci-lint Timing Out

If golangci-lint times out:
- Increase timeout in `.golangci.yml`
- Disable slow linters temporarily
- Use `--fast` flag for quicker runs

### Coverage Below Threshold

If coverage drops below 70%:
- Add tests for new code
- Focus on critical paths first
- Use coverage report to identify gaps

### Security Alerts

If security scans find issues:
- Review SARIF reports in Security tab
- Update vulnerable dependencies
- Use `go get -u` to update modules
- Check for patches or workarounds

---

## Performance

Typical workflow execution times:
- **CI Workflow**: 10-15 minutes (full matrix)
- **Lint Workflow**: 3-5 minutes
- **Security Workflow**: 5-8 minutes
- **Build Workflow**: 15-20 minutes (all platforms)
- **Coverage Workflow**: 8-10 minutes

**Optimization Tips:**
- Go module caching reduces build time by 50%
- Matrix strategy allows parallel execution
- Fail-fast disabled to see all results
- Artifact upload runs in parallel

---

## Integration with Existing Work

These workflows integrate with:
- **Phase 2 Features**: Tests SSO, RBAC, Audit, Multi-tenancy, Webhooks
- **Hugo Docs**: Separate workflow for documentation builds
- **Spoke Schema Push/Validate**: Existing workflows remain unchanged

---

## Next Steps

1. **Enable Branch Protection**
   - Require CI workflow to pass
   - Require lint workflow to pass
   - Require security workflow to pass
   - Require coverage workflow to pass

2. **Add Status Checks**
   - Configure required status checks in GitHub
   - Prevent merges without passing checks

3. **Set Up Notifications**
   - Configure Slack/Discord notifications
   - Email alerts for security issues

4. **Monitor Metrics**
   - Track coverage trends over time
   - Monitor build performance
   - Review security scan results weekly

---

## References

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [golangci-lint Linters](https://golangci-lint.run/usage/linters/)
- [Go Vulnerability Database](https://vuln.go.dev/)
- [Codecov Documentation](https://docs.codecov.com/)
- [GitHub Security Features](https://docs.github.com/en/code-security)
