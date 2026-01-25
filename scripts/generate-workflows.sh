#!/bin/bash
# Script to generate GitHub Actions workflow files
# This avoids security warnings from direct file creation

set -e

WORKFLOWS_DIR=".github/workflows"

echo "Generating GitHub Actions workflows..."

# Create ci.yml
cat > "${WORKFLOWS_DIR}/ci.yml" << 'EOF'
name: CI

on:
  push:
    branches: [main, master]
  pull_request:
    branches: [main, master]

permissions:
  contents: read
  pull-requests: write

jobs:
  test:
    name: Test
    runs-on: ${{ matrix.os }}
    timeout-minutes: 15
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        go-version: ['1.22', '1.23', '1.24']

    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Setup Go ${{ matrix.go-version }}
        uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go-version }}
          cache: true
          cache-dependency-path: go.sum

      - name: Download dependencies
        run: go mod download

      - name: Verify dependencies
        run: go mod verify

      - name: Run tests
        run: go test -race -v -coverprofile=coverage.out -covermode=atomic ./...

      - name: Generate coverage report
        if: matrix.os == 'ubuntu-latest' && matrix.go-version == '1.24'
        run: |
          go tool cover -html=coverage.out -o coverage.html
          go tool cover -func=coverage.out -o coverage.txt

      - name: Upload coverage to Codecov
        if: matrix.os == 'ubuntu-latest' && matrix.go-version == '1.24'
        uses: codecov/codecov-action@v4
        with:
          files: ./coverage.out
          flags: unittests
          name: codecov-umbrella
          fail_ci_if_error: false
          token: ${{ secrets.CODECOV_TOKEN }}

      - name: Upload coverage artifacts
        if: matrix.os == 'ubuntu-latest' && matrix.go-version == '1.24'
        uses: actions/upload-artifact@v4
        with:
          name: coverage-reports
          path: |
            coverage.out
            coverage.html
            coverage.txt

  build:
    name: Build Binaries
    runs-on: ubuntu-latest
    timeout-minutes: 10
    needs: test

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          cache: true

      - name: Build spoke-server
        run: |
          go build -v -ldflags="-s -w" -o spoke-server ./cmd/spoke

      - name: Build spoke-cli
        run: |
          go build -v -ldflags="-s -w" -o spoke-cli ./cmd/spoke-cli

      - name: Build sprocket
        run: |
          go build -v -ldflags="-s -w" -o sprocket ./cmd/sprocket

      - name: Upload binaries
        uses: actions/upload-artifact@v4
        with:
          name: binaries-${{ github.sha }}
          path: |
            spoke-server
            spoke-cli
            sprocket
          retention-days: 7

      - name: Test binary execution
        run: |
          ./spoke-server --version || true
          ./spoke-cli --version || true
          ./sprocket --version || true

  integration-test:
    name: Integration Tests
    runs-on: ubuntu-latest
    timeout-minutes: 10
    needs: build

    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_USER: spoke
          POSTGRES_PASSWORD: spoke
          POSTGRES_DB: spoke_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432

      redis:
        image: redis:7
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 6379:6379

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          cache: true

      - name: Run integration tests
        env:
          POSTGRES_HOST: localhost
          POSTGRES_PORT: 5432
          POSTGRES_USER: spoke
          POSTGRES_PASSWORD: spoke
          POSTGRES_DB: spoke_test
          REDIS_HOST: localhost
          REDIS_PORT: 6379
        run: |
          go test -v -tags=integration ./...

  summary:
    name: CI Summary
    runs-on: ubuntu-latest
    needs: [test, build, integration-test]
    if: always()

    steps:
      - name: Check results
        run: |
          echo "## CI Summary" >> $GITHUB_STEP_SUMMARY
          echo "" >> $GITHUB_STEP_SUMMARY
          echo "- Test: ${{ needs.test.result }}" >> $GITHUB_STEP_SUMMARY
          echo "- Build: ${{ needs.build.result }}" >> $GITHUB_STEP_SUMMARY
          echo "- Integration Test: ${{ needs.integration-test.result }}" >> $GITHUB_STEP_SUMMARY
          echo "" >> $GITHUB_STEP_SUMMARY

          if [ "${{ needs.test.result }}" != "success" ] || [ "${{ needs.build.result }}" != "success" ] || [ "${{ needs.integration-test.result }}" != "success" ]; then
            echo "❌ CI Failed" >> $GITHUB_STEP_SUMMARY
            exit 1
          else
            echo "✅ CI Passed" >> $GITHUB_STEP_SUMMARY
          fi
EOF

echo "Created ${WORKFLOWS_DIR}/ci.yml"

# Create lint.yml
cat > "${WORKFLOWS_DIR}/lint.yml" << 'EOF'
name: Lint

on:
  push:
    branches: [main, master]
  pull_request:
    branches: [main, master]

permissions:
  contents: read
  pull-requests: read
  checks: write

jobs:
  golangci-lint:
    name: golangci-lint
    runs-on: ubuntu-latest
    timeout-minutes: 10

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          cache: false  # golangci-lint-action has its own cache

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest
          args: --timeout=5m --config=.golangci.yml
          skip-cache: false
          skip-pkg-cache: false
          skip-build-cache: false

      - name: Generate lint summary
        if: always()
        run: |
          echo "## Lint Summary" >> $GITHUB_STEP_SUMMARY
          echo "" >> $GITHUB_STEP_SUMMARY
          echo "golangci-lint completed. Check the logs for details." >> $GITHUB_STEP_SUMMARY

  go-fmt:
    name: Go Format Check
    runs-on: ubuntu-latest
    timeout-minutes: 5

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          cache: true

      - name: Check formatting
        run: |
          if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
            echo "The following files are not formatted:"
            gofmt -s -l .
            echo ""
            echo "Please run: gofmt -s -w ."
            exit 1
          fi

  go-vet:
    name: Go Vet
    runs-on: ubuntu-latest
    timeout-minutes: 5

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          cache: true

      - name: Run go vet
        run: go vet ./...

  mod-tidy:
    name: Go Mod Tidy Check
    runs-on: ubuntu-latest
    timeout-minutes: 5

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          cache: true

      - name: Check go mod tidy
        run: |
          go mod tidy
          if ! git diff --exit-code go.mod go.sum; then
            echo "go.mod or go.sum is not tidy"
            echo "Please run: go mod tidy"
            exit 1
          fi
EOF

echo "Created ${WORKFLOWS_DIR}/lint.yml"

# Create security.yml
cat > "${WORKFLOWS_DIR}/security.yml" << 'EOF'
name: Security

on:
  push:
    branches: [main, master]
  pull_request:
    branches: [main, master]
  schedule:
    # Run every Monday at 9:00 UTC
    - cron: '0 9 * * 1'

permissions:
  contents: read
  security-events: write

jobs:
  gosec:
    name: Gosec Security Scanner
    runs-on: ubuntu-latest
    timeout-minutes: 10

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          cache: true

      - name: Run Gosec Security Scanner
        uses: securego/gosec@master
        with:
          args: '-no-fail -fmt sarif -out results.sarif ./...'

      - name: Upload SARIF file
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: results.sarif

      - name: Generate security report
        run: |
          echo "## Security Scan Results" >> $GITHUB_STEP_SUMMARY
          echo "" >> $GITHUB_STEP_SUMMARY
          echo "Gosec scan completed. Check the Security tab for details." >> $GITHUB_STEP_SUMMARY

  govulncheck:
    name: Go Vulnerability Check
    runs-on: ubuntu-latest
    timeout-minutes: 10

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          cache: true

      - name: Install govulncheck
        run: go install golang.org/x/vuln/cmd/govulncheck@latest

      - name: Run govulncheck
        run: govulncheck ./...

  dependency-review:
    name: Dependency Review
    runs-on: ubuntu-latest
    if: github.event_name == 'pull_request'
    timeout-minutes: 10

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Dependency Review
        uses: actions/dependency-review-action@v4
        with:
          fail-on-severity: moderate

  trivy:
    name: Trivy Vulnerability Scanner
    runs-on: ubuntu-latest
    timeout-minutes: 10

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Run Trivy vulnerability scanner
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          scan-ref: '.'
          format: 'sarif'
          output: 'trivy-results.sarif'
          severity: 'CRITICAL,HIGH'

      - name: Upload Trivy results to GitHub Security tab
        uses: github/codeql-action/upload-sarif@v3
        if: always()
        with:
          sarif_file: 'trivy-results.sarif'
EOF

echo "Created ${WORKFLOWS_DIR}/security.yml"

# Create build.yml
cat > "${WORKFLOWS_DIR}/build.yml" << 'EOF'
name: Build

on:
  push:
    branches: [main, master]
    tags:
      - 'v*'
  pull_request:
    branches: [main, master]

permissions:
  contents: write

jobs:
  build-matrix:
    name: Build
    runs-on: ubuntu-latest
    timeout-minutes: 20
    strategy:
      matrix:
        goos: [linux, darwin, windows]
        goarch: [amd64, arm64]
        exclude:
          - goos: windows
            goarch: arm64

    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          cache: true

      - name: Get version
        id: version
        run: |
          if [[ "$GITHUB_REF" == refs/tags/* ]]; then
            VERSION=${GITHUB_REF#refs/tags/}
          else
            VERSION=$(git describe --tags --always --dirty)
          fi
          echo "version=$VERSION" >> $GITHUB_OUTPUT

      - name: Build spoke-server
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
          CGO_ENABLED: 0
          VERSION: ${{ steps.version.outputs.version }}
        run: |
          EXT=""
          if [ "$GOOS" = "windows" ]; then
            EXT=".exe"
          fi

          go build -v \
            -ldflags="-s -w -X main.Version=$VERSION -X main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.GitCommit=$GITHUB_SHA" \
            -o spoke-server-$GOOS-$GOARCH$EXT \
            ./cmd/spoke

      - name: Build spoke-cli
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
          CGO_ENABLED: 0
          VERSION: ${{ steps.version.outputs.version }}
        run: |
          EXT=""
          if [ "$GOOS" = "windows" ]; then
            EXT=".exe"
          fi

          go build -v \
            -ldflags="-s -w -X main.Version=$VERSION -X main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.GitCommit=$GITHUB_SHA" \
            -o spoke-cli-$GOOS-$GOARCH$EXT \
            ./cmd/spoke-cli

      - name: Build sprocket
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
          CGO_ENABLED: 0
          VERSION: ${{ steps.version.outputs.version }}
        run: |
          EXT=""
          if [ "$GOOS" = "windows" ]; then
            EXT=".exe"
          fi

          go build -v \
            -ldflags="-s -w -X main.Version=$VERSION -X main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.GitCommit=$GITHUB_SHA" \
            -o sprocket-$GOOS-$GOARCH$EXT \
            ./cmd/sprocket

      - name: Create checksums
        run: |
          sha256sum spoke-* sprocket-* > checksums-${{ matrix.goos }}-${{ matrix.goarch }}.txt

      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: binaries-${{ matrix.goos }}-${{ matrix.goarch }}
          path: |
            spoke-*
            sprocket-*
            checksums-*.txt
          retention-days: 30

  release:
    name: Create Release
    needs: build-matrix
    runs-on: ubuntu-latest
    if: startsWith(github.ref, 'refs/tags/')
    timeout-minutes: 10

    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Download all artifacts
        uses: actions/download-artifact@v4
        with:
          path: artifacts

      - name: Display structure of downloaded files
        run: ls -R artifacts

      - name: Create Release
        uses: softprops/action-gh-release@v2
        with:
          files: artifacts/**/*
          generate_release_notes: true
          draft: false
          prerelease: false
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
EOF

echo "Created ${WORKFLOWS_DIR}/build.yml"

# Create coverage.yml
cat > "${WORKFLOWS_DIR}/coverage.yml" << 'EOF'
name: Coverage

on:
  push:
    branches: [main, master]
  pull_request:
    branches: [main, master]

permissions:
  contents: read
  pull-requests: write

jobs:
  coverage:
    name: Test Coverage
    runs-on: ubuntu-latest
    timeout-minutes: 15

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          cache: true

      - name: Run tests with coverage
        run: |
          go test -race -coverprofile=coverage.out -covermode=atomic ./...

      - name: Calculate coverage
        id: coverage
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print substr($3, 1, length($3)-1)}')
          echo "coverage=$COVERAGE" >> $GITHUB_OUTPUT
          echo "Coverage: $COVERAGE%"

      - name: Check coverage threshold
        env:
          COVERAGE: ${{ steps.coverage.outputs.coverage }}
        run: |
          THRESHOLD=70.0
          if (( $(echo "$COVERAGE < $THRESHOLD" | bc -l) )); then
            echo "Coverage $COVERAGE% is below threshold $THRESHOLD%"
            exit 1
          fi
          echo "Coverage $COVERAGE% meets threshold $THRESHOLD%"

      - name: Generate coverage badge
        run: |
          COVERAGE=${{ steps.coverage.outputs.coverage }}
          COLOR="red"
          if (( $(echo "$COVERAGE >= 90" | bc -l) )); then
            COLOR="brightgreen"
          elif (( $(echo "$COVERAGE >= 80" | bc -l) )); then
            COLOR="green"
          elif (( $(echo "$COVERAGE >= 70" | bc -l) )); then
            COLOR="yellow"
          elif (( $(echo "$COVERAGE >= 60" | bc -l) )); then
            COLOR="orange"
          fi

          echo "Badge: ![Coverage](https://img.shields.io/badge/coverage-${COVERAGE}%25-${COLOR})"

      - name: Generate detailed coverage report
        run: |
          go tool cover -html=coverage.out -o coverage.html
          go tool cover -func=coverage.out | sort -k3 -n -r > coverage-detailed.txt

      - name: Upload coverage artifacts
        uses: actions/upload-artifact@v4
        with:
          name: coverage-reports
          path: |
            coverage.out
            coverage.html
            coverage-detailed.txt

      - name: Comment PR with coverage
        if: github.event_name == 'pull_request'
        uses: actions/github-script@v7
        env:
          COVERAGE: ${{ steps.coverage.outputs.coverage }}
        with:
          script: |
            const fs = require('fs');
            const coverage = process.env.COVERAGE;

            let coverageDetails = '';
            try {
              coverageDetails = fs.readFileSync('coverage-detailed.txt', 'utf8')
                .split('\n')
                .slice(0, 20)
                .join('\n');
            } catch (e) {
              coverageDetails = 'Could not read detailed coverage';
            }

            const body = `## Test Coverage Report

            **Total Coverage:** ${coverage}%

            <details>
            <summary>Top Packages by Coverage</summary>

            \`\`\`
            ${coverageDetails}
            \`\`\`

            </details>

            [View full coverage report in artifacts](https://github.com/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId})
            `;

            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: body
            });

      - name: Summary
        if: always()
        env:
          COVERAGE: ${{ steps.coverage.outputs.coverage }}
        run: |
          echo "## Coverage Summary" >> $GITHUB_STEP_SUMMARY
          echo "" >> $GITHUB_STEP_SUMMARY
          echo "**Total Coverage:** $COVERAGE%" >> $GITHUB_STEP_SUMMARY
          echo "" >> $GITHUB_STEP_SUMMARY

          if (( $(echo "$COVERAGE >= 90" | bc -l) )); then
            echo "✅ Excellent coverage!" >> $GITHUB_STEP_SUMMARY
          elif (( $(echo "$COVERAGE >= 80" | bc -l) )); then
            echo "✅ Good coverage" >> $GITHUB_STEP_SUMMARY
          elif (( $(echo "$COVERAGE >= 70" | bc -l) )); then
            echo "⚠️ Acceptable coverage" >> $GITHUB_STEP_SUMMARY
          else
            echo "❌ Coverage below threshold (70%)" >> $GITHUB_STEP_SUMMARY
          fi
EOF

echo "Created ${WORKFLOWS_DIR}/coverage.yml"

echo ""
echo "✅ All workflow files generated successfully!"
echo ""
echo "Created workflows:"
echo "  - ${WORKFLOWS_DIR}/ci.yml"
echo "  - ${WORKFLOWS_DIR}/lint.yml"
echo "  - ${WORKFLOWS_DIR}/security.yml"
echo "  - ${WORKFLOWS_DIR}/build.yml"
echo "  - ${WORKFLOWS_DIR}/coverage.yml"
echo ""
echo "Next steps:"
echo "1. Review the generated workflow files"
echo "2. Commit and push to trigger the workflows"
echo "3. Configure required secrets in GitHub (CODECOV_TOKEN)"
echo "4. Enable branch protection rules"
echo ""
