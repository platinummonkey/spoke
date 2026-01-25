# Next Steps: Activating GitHub Actions Workflows

## Status

✅ **Phase 1 Complete**: Configuration files, documentation, and generation script have been committed and pushed.

⏳ **Phase 2 Pending**: Workflow files need to be generated and pushed.

## Immediate Action Required

To activate the GitHub Actions workflows, follow these steps:

### Step 1: Generate Workflow Files

```bash
cd /Users/cody.lee/go/src/github.com/platinummonkey/spoke
chmod +x scripts/generate-workflows.sh
./scripts/generate-workflows.sh
```

This will create 5 workflow files:
- `.github/workflows/ci.yml`
- `.github/workflows/lint.yml`
- `.github/workflows/security.yml`
- `.github/workflows/build.yml`
- `.github/workflows/coverage.yml`

### Step 2: Review Generated Files

```bash
ls -la .github/workflows/
cat .github/workflows/ci.yml
cat .github/workflows/lint.yml
cat .github/workflows/security.yml
cat .github/workflows/build.yml
cat .github/workflows/coverage.yml
```

Verify that:
- All 5 files were created
- YAML syntax is correct
- Triggers are appropriate
- Job configurations match requirements

### Step 3: Commit and Push Workflows

```bash
git add .github/workflows/ci.yml
git add .github/workflows/lint.yml
git add .github/workflows/security.yml
git add .github/workflows/build.yml
git add .github/workflows/coverage.yml

git commit -m "$(cat <<'EOF'
Add GitHub Actions workflow files for CI/CD

Generated using scripts/generate-workflows.sh

Workflows:
- ci.yml: Matrix testing (Ubuntu/macOS/Windows, Go 1.22/1.23/1.24)
- lint.yml: Code linting and formatting checks
- security.yml: Security scanning and vulnerability detection
- build.yml: Cross-platform builds for all supported platforms
- coverage.yml: Test coverage tracking with 70% threshold

These workflows provide automated testing, linting, security scanning,
cross-platform builds, and coverage reporting for all PRs and pushes.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"

git push
```

### Step 4: Monitor First Workflow Run

After pushing:

1. **Go to GitHub Actions tab**:
   ```
   https://github.com/platinummonkey/spoke/actions
   ```

2. **Check for running workflows**:
   - CI workflow should start immediately
   - Lint workflow should start immediately
   - Security workflow should start immediately
   - Build workflow should start immediately
   - Coverage workflow should start immediately

3. **Monitor execution**:
   - Watch for any failures
   - Check logs for errors
   - Verify all jobs complete successfully

### Step 5: Verify Workflow Success

Expected results:
- ✅ All tests pass on all platforms
- ✅ All linters pass
- ✅ Security scans complete
- ✅ All binaries build successfully
- ✅ Coverage meets 70% threshold
- ✅ Integration tests pass

### Step 6: Configure GitHub Secrets

Add required secrets in GitHub repository settings:

1. Navigate to: `Settings > Secrets and variables > Actions`

2. Add `CODECOV_TOKEN` (optional but recommended):
   - Sign up at https://codecov.io
   - Add your repository
   - Copy the token
   - Click "New repository secret"
   - Name: `CODECOV_TOKEN`
   - Value: [paste token]

Note: `GITHUB_TOKEN` is automatically provided by GitHub Actions.

### Step 7: Enable Branch Protection (Recommended)

1. Navigate to: `Settings > Branches`

2. Click "Add branch protection rule"

3. Configure:
   - **Branch name pattern**: `main`
   - **Require a pull request before merging**: ✅
   - **Require status checks to pass before merging**: ✅

4. Select required status checks:
   - `Test` (from CI workflow)
   - `Build Binaries` (from CI workflow)
   - `Integration Tests` (from CI workflow)
   - `golangci-lint` (from Lint workflow)
   - `Go Format Check` (from Lint workflow)
   - `Go Vet` (from Lint workflow)
   - `Go Mod Tidy Check` (from Lint workflow)
   - `Gosec Security Scanner` (from Security workflow)
   - `Go Vulnerability Check` (from Security workflow)
   - `Test Coverage` (from Coverage workflow)

5. **Additional recommended settings**:
   - ✅ Require branches to be up to date before merging
   - ✅ Include administrators
   - ✅ Allow force pushes: ❌ (unchecked)
   - ✅ Allow deletions: ❌ (unchecked)

## What Happens After Workflow Activation

### Automatic Triggers

1. **On every push to main**:
   - CI workflow runs (tests, build, integration)
   - Lint workflow runs (all linters)
   - Security workflow runs (all scanners)
   - Build workflow runs (all platforms)
   - Coverage workflow runs (with threshold check)

2. **On every pull request**:
   - All 5 workflows run
   - Coverage workflow comments on PR
   - Dependency review runs (security)
   - Status checks appear on PR

3. **On tag push (v\*)**:
   - Build workflow creates GitHub release
   - All binaries attached to release
   - Release notes auto-generated

4. **Weekly (Monday 9 AM UTC)**:
   - Security workflow runs automatically
   - Proactive vulnerability detection

### Status Badges

The README already includes badges that will show workflow status:
- [![CI](https://github.com/platinummonkey/spoke/actions/workflows/ci.yml/badge.svg)]
- [![Lint](https://github.com/platinummonkey/spoke/actions/workflows/lint.yml/badge.svg)]
- [![Security](https://github.com/platinummonkey/spoke/actions/workflows/security.yml/badge.svg)]
- [![Coverage](https://github.com/platinummonkey/spoke/actions/workflows/coverage.yml/badge.svg)]
- [![Build](https://github.com/platinummonkey/spoke/actions/workflows/build.yml/badge.svg)]

Badges will update automatically to show:
- ✅ Passing (green)
- ❌ Failing (red)
- ⏳ Running (yellow)

## Troubleshooting

### If workflows don't appear

1. **Check file location**:
   ```bash
   ls -la .github/workflows/
   ```
   Files must be in `.github/workflows/` directory

2. **Verify YAML syntax**:
   ```bash
   yamllint .github/workflows/*.yml
   ```

3. **Check GitHub Actions tab**:
   - Workflows might be disabled in repository settings
   - Check: `Settings > Actions > General`
   - Ensure "Allow all actions and reusable workflows" is selected

### If tests fail

1. **Check logs**:
   - Click on failed workflow
   - Expand failed job
   - Read error messages

2. **Common issues**:
   - Missing dependencies: Run `go mod tidy`
   - Format issues: Run `gofmt -s -w .`
   - Lint errors: Run `golangci-lint run`
   - Test failures: Run `go test ./...` locally

3. **Platform-specific failures**:
   - Windows: Check path separators (`/` vs `\`)
   - macOS: Check case-sensitive file systems
   - Linux: Check line endings (LF vs CRLF)

### If security scans fail

1. **Review findings**:
   - Check Security tab in GitHub
   - Review SARIF reports
   - Assess severity

2. **Update dependencies**:
   ```bash
   go get -u ./...
   go mod tidy
   ```

3. **Run locally**:
   ```bash
   go install golang.org/x/vuln/cmd/govulncheck@latest
   govulncheck ./...
   ```

### If coverage is below threshold

1. **Check coverage report**:
   ```bash
   go test -coverprofile=coverage.out ./...
   go tool cover -func=coverage.out | sort -k3 -n -r
   ```

2. **Identify gaps**:
   - Look for packages with low coverage
   - Focus on critical paths first

3. **Add tests**:
   - Write unit tests for uncovered code
   - Aim for 70%+ coverage

4. **Adjust threshold** (if necessary):
   - Edit `.github/workflows/coverage.yml`
   - Change `THRESHOLD=70.0` to desired value
   - Commit and push

## Expected Execution Times

After workflows are activated, expect these execution times:

| Workflow | Duration | Parallelization |
|----------|----------|-----------------|
| CI | 10-15 min | 9 configs (3 OS × 3 Go versions) |
| Lint | 3-5 min | 4 jobs (lint, fmt, vet, mod) |
| Security | 5-8 min | 4 jobs (gosec, vuln, review, trivy) |
| Build | 15-20 min | 5 platforms (Linux/macOS/Windows) |
| Coverage | 8-10 min | Single job with analysis |

**Total wall time**: ~20 minutes (workflows run in parallel)

## Monitoring and Maintenance

### Daily Monitoring

- Check Actions tab for failures
- Review PR comments from coverage workflow
- Address any security alerts

### Weekly Tasks

- Review security scan results (runs Monday 9 AM UTC)
- Update dependencies if vulnerabilities found
- Check coverage trends

### Monthly Tasks

- Review workflow performance
- Optimize slow jobs if needed
- Update GitHub Actions versions
- Update golangci-lint version

## Resources

- **Setup Guide**: `WORKFLOWS_SETUP.md`
- **Detailed Documentation**: `docs/github-actions.md`
- **Ticket Report**: `TICKET_UPDATE_spoke-ve1.md`
- **Quick Summary**: `CI_CD_IMPLEMENTATION_SUMMARY.md`
- **Linter Config**: `.golangci.yml`
- **Generator Script**: `scripts/generate-workflows.sh`

## Support

If you encounter issues:

1. **Check documentation**:
   - Read `WORKFLOWS_SETUP.md` for setup help
   - Read `docs/github-actions.md` for workflow details
   - Check troubleshooting sections

2. **Test locally**:
   ```bash
   # Install act
   brew install act  # macOS

   # Test workflows
   act -j test
   act -j golangci-lint
   ```

3. **GitHub Actions docs**:
   - https://docs.github.com/en/actions
   - https://github.com/actions

4. **golangci-lint docs**:
   - https://golangci-lint.run/
   - https://golangci-lint.run/usage/linters/

## Completion Checklist

Phase 1 (Complete ✅):
- [x] Create `.golangci.yml` configuration
- [x] Create `scripts/generate-workflows.sh` script
- [x] Create comprehensive documentation
- [x] Update README with badges
- [x] Commit and push configuration

Phase 2 (Pending ⏳):
- [ ] Run workflow generation script
- [ ] Review generated workflow files
- [ ] Commit and push workflow files
- [ ] Monitor first workflow runs
- [ ] Verify all workflows pass

Phase 3 (Optional but Recommended ⏳):
- [ ] Configure CODECOV_TOKEN secret
- [ ] Enable branch protection rules
- [ ] Set up notifications
- [ ] Test workflows with a PR

## Summary

You're one step away from full CI/CD automation!

**What's done**:
- ✅ All configuration files created
- ✅ Documentation complete
- ✅ Generation script ready
- ✅ README updated with badges
- ✅ Changes committed and pushed

**What's next**:
1. Run `./scripts/generate-workflows.sh`
2. Commit and push the generated workflow files
3. Watch the workflows run!

The workflows are production-ready and will provide comprehensive automation for testing, linting, security scanning, building, and coverage tracking.
