# GitHub Actions Workflows Setup Guide

This guide will help you set up the comprehensive CI/CD workflows for Spoke Schema Registry.

## Quick Start

Run the workflow generation script:

```bash
cd /Users/cody.lee/go/src/github.com/platinummonkey/spoke
chmod +x scripts/generate-workflows.sh
./scripts/generate-workflows.sh
```

This will create all 5 workflow files in `.github/workflows/`:
- `ci.yml` - Continuous Integration
- `lint.yml` - Code Linting
- `security.yml` - Security Scanning
- `build.yml` - Cross-platform Builds
- `coverage.yml` - Test Coverage

## Files Created

### 1. Configuration Files

#### `.golangci.yml`
Already created. This configures golangci-lint with 18+ linters including:
- govet, errcheck, staticcheck, gosimple, unused, ineffassign
- gofmt, goimports, misspell, gocritic, gocyclo
- dupl, goconst, lll, funlen, godox, typecheck

### 2. Workflow Files (Created by Script)

#### `.github/workflows/ci.yml`
- Tests on Ubuntu, macOS, Windows
- Go versions: 1.22, 1.23, 1.24
- Builds all 3 binaries
- Integration tests with PostgreSQL and Redis
- Coverage reporting

#### `.github/workflows/lint.yml`
- golangci-lint with comprehensive checks
- gofmt formatting check
- go vet analysis
- go mod tidy verification

#### `.github/workflows/security.yml`
- gosec security scanner
- govulncheck vulnerability check
- Dependency review
- Trivy vulnerability scanner
- Weekly scheduled scans

#### `.github/workflows/build.yml`
- Cross-platform builds for:
  - Linux: amd64, arm64
  - macOS: amd64, arm64
  - Windows: amd64
- Automated releases on tags
- SHA256 checksums

#### `.github/workflows/coverage.yml`
- Test coverage analysis
- 70% minimum threshold
- PR comments with coverage report
- Coverage badge generation

### 3. Documentation

#### `docs/github-actions.md`
Already created. Comprehensive documentation covering:
- Workflow details and triggers
- Configuration explanations
- Troubleshooting guide
- Performance metrics
- Integration with existing work

## Setup Steps

### Step 1: Generate Workflow Files

```bash
chmod +x scripts/generate-workflows.sh
./scripts/generate-workflows.sh
```

### Step 2: Review Generated Files

Check the generated workflow files:

```bash
ls -la .github/workflows/
cat .github/workflows/ci.yml
cat .github/workflows/lint.yml
cat .github/workflows/security.yml
cat .github/workflows/build.yml
cat .github/workflows/coverage.yml
```

### Step 3: Configure GitHub Secrets

Add these secrets in your GitHub repository settings (`Settings > Secrets and variables > Actions`):

1. **CODECOV_TOKEN** (optional but recommended)
   - Sign up at https://codecov.io
   - Add your repository
   - Copy the token
   - Add as secret in GitHub

Note: `GITHUB_TOKEN` is automatically provided by GitHub Actions.

### Step 4: Commit and Push

```bash
git add .github/workflows/*.yml
git add .golangci.yml
git add scripts/generate-workflows.sh
git add docs/github-actions.md
git add WORKFLOWS_SETUP.md
git commit -m "Add comprehensive GitHub Actions CI/CD workflows

- CI workflow with matrix testing (Ubuntu, macOS, Windows)
- Lint workflow with golangci-lint and multiple checks
- Security workflow with gosec, govulncheck, trivy
- Build workflow with cross-platform compilation
- Coverage workflow with PR comments and threshold checks

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
git push
```

### Step 5: Verify Workflows

1. Go to your repository on GitHub
2. Click on the "Actions" tab
3. You should see workflows running
4. Monitor the progress and check for any failures

### Step 6: Enable Branch Protection (Recommended)

1. Go to `Settings > Branches`
2. Add a branch protection rule for `main`
3. Enable:
   - Require status checks to pass before merging
   - Select these required checks:
     - `Test`
     - `Build Binaries`
     - `Integration Tests`
     - `golangci-lint`
     - `Go Format Check`
     - `Go Vet`
     - `Go Mod Tidy Check`
     - `Gosec Security Scanner`
     - `Go Vulnerability Check`
     - `Test Coverage`

## Add Status Badges to README

Add these badges to the top of your `README.md`:

```markdown
[![CI](https://github.com/platinummonkey/spoke/actions/workflows/ci.yml/badge.svg)](https://github.com/platinummonkey/spoke/actions/workflows/ci.yml)
[![Lint](https://github.com/platinummonkey/spoke/actions/workflows/lint.yml/badge.svg)](https://github.com/platinummonkey/spoke/actions/workflows/lint.yml)
[![Security](https://github.com/platinummonkey/spoke/actions/workflows/security.yml/badge.svg)](https://github.com/platinummonkey/spoke/actions/workflows/security.yml)
[![Coverage](https://github.com/platinummonkey/spoke/actions/workflows/coverage.yml/badge.svg)](https://github.com/platinummonkey/spoke/actions/workflows/coverage.yml)
[![Build](https://github.com/platinummonkey/spoke/actions/workflows/build.yml/badge.svg)](https://github.com/platinummonkey/spoke/actions/workflows/build.yml)
```

## Workflow Details

### CI Workflow
- **Trigger**: Push to main/master, PRs
- **Duration**: ~10-15 minutes
- **Jobs**: Test (matrix), Build, Integration Test, Summary

### Lint Workflow
- **Trigger**: Push to main/master, PRs
- **Duration**: ~3-5 minutes
- **Jobs**: golangci-lint, go-fmt, go-vet, mod-tidy

### Security Workflow
- **Trigger**: Push to main/master, PRs, Weekly schedule
- **Duration**: ~5-8 minutes
- **Jobs**: gosec, govulncheck, dependency-review, trivy

### Build Workflow
- **Trigger**: Push to main/master, tags, PRs
- **Duration**: ~15-20 minutes
- **Jobs**: build-matrix, release (tags only)

### Coverage Workflow
- **Trigger**: Push to main/master, PRs
- **Duration**: ~8-10 minutes
- **Jobs**: coverage with PR comments

## Testing Workflows Locally

Install `act` to test workflows locally:

```bash
# macOS
brew install act

# Linux
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash
```

Run workflows:

```bash
# Test CI workflow
act -j test

# Test lint workflow
act -j golangci-lint

# Test security workflow
act -j gosec

# List all jobs
act -l
```

## Troubleshooting

### Workflows Not Appearing

If workflows don't appear in the Actions tab:
1. Check that files are in `.github/workflows/` directory
2. Verify YAML syntax: `yamllint .github/workflows/*.yml`
3. Check that workflows have valid triggers

### Tests Failing

Common issues:
1. **Windows tests fail**: Check for path separator issues (`/` vs `\`)
2. **Integration tests fail**: Verify service configurations
3. **Coverage below threshold**: Add tests or adjust threshold

### Lint Failures

Fix common lint issues:
```bash
# Format code
gofmt -s -w .

# Tidy modules
go mod tidy

# Run linters locally
golangci-lint run
```

### Security Alerts

Address security findings:
```bash
# Update dependencies
go get -u ./...

# Check for vulnerabilities
govulncheck ./...
```

## Workflow Optimization Tips

1. **Use caching**: Already enabled for Go modules
2. **Fail fast**: Disabled to see all results
3. **Parallel execution**: Matrix strategy for tests
4. **Timeouts**: Set to prevent hanging jobs

## Integration with Existing Workflows

These workflows are designed to coexist with:
- `spoke-push.yml` - Schema push on proto changes
- `spoke-validate.yml` - Schema validation
- Any Hugo docs workflows

## Next Steps

1. ✅ Generate workflow files (run script)
2. ✅ Review and understand workflows
3. ✅ Commit and push changes
4. ⏳ Monitor first workflow runs
5. ⏳ Configure secrets (CODECOV_TOKEN)
6. ⏳ Enable branch protection
7. ⏳ Add status badges to README
8. ⏳ Set up notifications (optional)

## Support

For detailed information, see:
- `docs/github-actions.md` - Complete workflow documentation
- `.golangci.yml` - Linter configuration
- `scripts/generate-workflows.sh` - Workflow generation script

## Quality Gates Summary

All PRs must pass:
- ✅ Tests on all platforms (Ubuntu, macOS, Windows)
- ✅ Tests on all Go versions (1.22, 1.23, 1.24)
- ✅ All linters (18+ enabled)
- ✅ Security scans (no critical/high vulnerabilities)
- ✅ 70% minimum test coverage
- ✅ Successful builds for all platforms
- ✅ Integration tests pass

This ensures high code quality and reliability for Spoke Schema Registry.
