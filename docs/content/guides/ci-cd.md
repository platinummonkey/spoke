---
title: "CI/CD Integration"
weight: 7
---

# CI/CD Integration

Integrate Spoke into your CI/CD pipelines for automated schema management.

## Overview

Spoke can be integrated into:
- **GitHub Actions**
- **GitLab CI**
- **Jenkins**
- **CircleCI**
- **Travis CI**
- **Azure Pipelines**

## GitHub Actions

### Basic Workflow

Create `.github/workflows/spoke-push.yml`:

```yaml
name: Push Protobuf Schemas to Spoke

on:
  push:
    tags:
      - 'v*'  # Trigger on version tags

jobs:
  push-schemas:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v3

      - name: Install Spoke CLI
        run: |
          wget https://github.com/platinummonkey/spoke/releases/latest/download/spoke-cli-linux-amd64
          chmod +x spoke-cli-linux-amd64
          sudo mv spoke-cli-linux-amd64 /usr/local/bin/spoke-cli

      - name: Push schemas to Spoke
        env:
          SPOKE_TOKEN: ${{ secrets.SPOKE_TOKEN }}
          SPOKE_REGISTRY: ${{ secrets.SPOKE_REGISTRY }}
        run: |
          VERSION=${GITHUB_REF#refs/tags/}
          spoke-cli push \
            -module ${{ github.event.repository.name }} \
            -version $VERSION \
            -dir ./proto \
            -registry $SPOKE_REGISTRY \
            -description "Release $VERSION"
```

### Advanced Workflow with Validation

```yaml
name: Spoke Schema CI

on:
  pull_request:
    paths:
      - 'proto/**'
  push:
    branches:
      - main
    tags:
      - 'v*'

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install Spoke CLI
        run: |
          wget https://github.com/platinummonkey/spoke/releases/latest/download/spoke-cli-linux-amd64
          chmod +x spoke-cli-linux-amd64
          sudo mv spoke-cli-linux-amd64 /usr/local/bin/spoke-cli

      - name: Validate protobuf files
        run: spoke-cli validate -dir ./proto

      - name: Check compatibility
        if: github.event_name == 'pull_request'
        env:
          SPOKE_REGISTRY: ${{ secrets.SPOKE_REGISTRY }}
        run: |
          # Get current version from main branch
          CURRENT_VERSION=$(git describe --tags --abbrev=0 main)

          # Check compatibility
          spoke-cli compatibility-check \
            -module ${{ github.event.repository.name }} \
            -version $CURRENT_VERSION \
            -dir ./proto \
            -registry $SPOKE_REGISTRY

  push:
    needs: validate
    if: github.event_name == 'push' && startsWith(github.ref, 'refs/tags/')
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install Spoke CLI
        run: |
          wget https://github.com/platinummonkey/spoke/releases/latest/download/spoke-cli-linux-amd64
          chmod +x spoke-cli-linux-amd64
          sudo mv spoke-cli-linux-amd64 /usr/local/bin/spoke-cli

      - name: Push to Spoke
        env:
          SPOKE_TOKEN: ${{ secrets.SPOKE_TOKEN }}
          SPOKE_REGISTRY: ${{ secrets.SPOKE_REGISTRY }}
        run: |
          VERSION=${GITHUB_REF#refs/tags/}
          spoke-cli push \
            -module ${{ github.event.repository.name }} \
            -version $VERSION \
            -dir ./proto \
            -registry $SPOKE_REGISTRY \
            -description "Release $VERSION from commit ${{ github.sha }}"
```

### Secrets Configuration

Add secrets in GitHub repository settings:

- `SPOKE_TOKEN`: API token from Spoke
- `SPOKE_REGISTRY`: URL of your Spoke registry (e.g., `https://spoke.company.com`)

## GitLab CI

### Basic Configuration

Create `.gitlab-ci.yml`:

```yaml
stages:
  - validate
  - push

variables:
  SPOKE_CLI_VERSION: "latest"
  MODULE_NAME: ${CI_PROJECT_NAME}

before_script:
  - wget https://github.com/platinummonkey/spoke/releases/latest/download/spoke-cli-linux-amd64
  - chmod +x spoke-cli-linux-amd64
  - mv spoke-cli-linux-amd64 /usr/local/bin/spoke-cli

validate-schemas:
  stage: validate
  script:
    - spoke-cli validate -dir ./proto
  only:
    - merge_requests
    - main

push-to-spoke:
  stage: push
  script:
    - |
      if [[ "$CI_COMMIT_TAG" ]]; then
        VERSION=$CI_COMMIT_TAG
      else
        VERSION="dev-$CI_COMMIT_SHORT_SHA"
      fi

      spoke-cli push \
        -module $MODULE_NAME \
        -version $VERSION \
        -dir ./proto \
        -registry $SPOKE_REGISTRY \
        -description "GitLab CI build $CI_PIPELINE_ID"
  only:
    - tags
    - main
  variables:
    SPOKE_TOKEN: $SPOKE_TOKEN
    SPOKE_REGISTRY: $SPOKE_REGISTRY
```

### GitLab CI Variables

Add variables in GitLab project settings:

- `SPOKE_TOKEN`: Protected, masked
- `SPOKE_REGISTRY`: URL of your Spoke registry

## Jenkins

### Jenkinsfile

```groovy
pipeline {
    agent any

    environment {
        SPOKE_CLI = '/usr/local/bin/spoke-cli'
        SPOKE_REGISTRY = credentials('spoke-registry-url')
        SPOKE_TOKEN = credentials('spoke-api-token')
        MODULE_NAME = "${env.JOB_NAME}"
    }

    stages {
        stage('Install Spoke CLI') {
            steps {
                sh '''
                    wget https://github.com/platinummonkey/spoke/releases/latest/download/spoke-cli-linux-amd64
                    chmod +x spoke-cli-linux-amd64
                    sudo mv spoke-cli-linux-amd64 /usr/local/bin/spoke-cli
                '''
            }
        }

        stage('Validate Schemas') {
            steps {
                sh 'spoke-cli validate -dir ./proto'
            }
        }

        stage('Push to Spoke') {
            when {
                tag "v*"
            }
            steps {
                sh '''
                    VERSION=${GIT_TAG}
                    spoke-cli push \
                        -module ${MODULE_NAME} \
                        -version ${VERSION} \
                        -dir ./proto \
                        -registry ${SPOKE_REGISTRY} \
                        -description "Jenkins build ${BUILD_NUMBER}"
                '''
            }
        }
    }

    post {
        success {
            echo 'Schemas pushed successfully to Spoke'
        }
        failure {
            echo 'Failed to push schemas to Spoke'
        }
    }
}
```

## CircleCI

### .circleci/config.yml

```yaml
version: 2.1

executors:
  spoke-executor:
    docker:
      - image: cimg/base:stable

jobs:
  validate:
    executor: spoke-executor
    steps:
      - checkout
      - run:
          name: Install Spoke CLI
          command: |
            wget https://github.com/platinummonkey/spoke/releases/latest/download/spoke-cli-linux-amd64
            chmod +x spoke-cli-linux-amd64
            sudo mv spoke-cli-linux-amd64 /usr/local/bin/spoke-cli
      - run:
          name: Validate protobuf files
          command: spoke-cli validate -dir ./proto

  push:
    executor: spoke-executor
    steps:
      - checkout
      - run:
          name: Install Spoke CLI
          command: |
            wget https://github.com/platinummonkey/spoke/releases/latest/download/spoke-cli-linux-amd64
            chmod +x spoke-cli-linux-amd64
            sudo mv spoke-cli-linux-amd64 /usr/local/bin/spoke-cli
      - run:
          name: Push to Spoke
          command: |
            VERSION=${CIRCLE_TAG:-dev-${CIRCLE_SHA1:0:8}}
            spoke-cli push \
              -module ${CIRCLE_PROJECT_REPONAME} \
              -version $VERSION \
              -dir ./proto \
              -registry $SPOKE_REGISTRY \
              -description "CircleCI build ${CIRCLE_BUILD_NUM}"

workflows:
  version: 2
  validate-and-push:
    jobs:
      - validate:
          filters:
            branches:
              only: /.*/
      - push:
          requires:
            - validate
          filters:
            tags:
              only: /^v.*/
            branches:
              ignore: /.*/
```

## Docker-based CI

### Dockerfile for CI

```dockerfile
FROM golang:1.20-alpine AS builder

RUN apk add --no-cache git wget

# Install protoc
RUN wget https://github.com/protocolbuffers/protobuf/releases/download/v21.12/protoc-21.12-linux-x86_64.zip \
    && unzip protoc-21.12-linux-x86_64.zip -d /usr/local \
    && rm protoc-21.12-linux-x86_64.zip

# Install Spoke CLI
RUN wget https://github.com/platinummonkey/spoke/releases/latest/download/spoke-cli-linux-amd64 \
    && chmod +x spoke-cli-linux-amd64 \
    && mv spoke-cli-linux-amd64 /usr/local/bin/spoke-cli

# Install Go plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

WORKDIR /workspace

ENTRYPOINT ["/usr/local/bin/spoke-cli"]
```

### Usage in CI

```yaml
# GitHub Actions
- name: Push schemas
  uses: docker://your-registry/spoke-ci:latest
  with:
    args: push -module mymodule -version v1.0.0 -dir ./proto
  env:
    SPOKE_TOKEN: ${{ secrets.SPOKE_TOKEN }}
```

## Webhook Integration

### Trigger CI on Schema Updates

Configure Spoke webhook to trigger CI when schemas are updated:

```yaml
# Spoke webhook configuration
webhooks:
  - url: https://api.github.com/repos/org/repo/dispatches
    events:
      - module.created
      - version.published
    headers:
      Authorization: "Bearer ${GITHUB_TOKEN}"
      Accept: "application/vnd.github.v3+json"
    payload:
      event_type: "spoke-schema-updated"
      client_payload:
        module: "{{ .Module }}"
        version: "{{ .Version }}"
```

### GitHub Workflow for Webhook

```yaml
name: Schema Updated

on:
  repository_dispatch:
    types: [spoke-schema-updated]

jobs:
  update-schemas:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Pull updated schemas
        env:
          SPOKE_REGISTRY: ${{ secrets.SPOKE_REGISTRY }}
        run: |
          MODULE=${{ github.event.client_payload.module }}
          VERSION=${{ github.event.client_payload.version }}

          spoke-cli pull \
            -module $MODULE \
            -version $VERSION \
            -dir ./proto \
            -registry $SPOKE_REGISTRY \
            -recursive

      - name: Regenerate code
        run: |
          spoke-cli compile \
            -dir ./proto \
            -out ./generated \
            -lang go

      - name: Create Pull Request
        uses: peter-evans/create-pull-request@v4
        with:
          title: "Update schemas from Spoke"
          body: "Automated schema update from Spoke"
          branch: "auto/schema-update"
```

## Multi-Module Repositories

### Push Multiple Modules

```yaml
# GitHub Actions
jobs:
  push-all-modules:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        module:
          - name: common
            dir: proto/common
          - name: user
            dir: proto/user
          - name: order
            dir: proto/order

    steps:
      - uses: actions/checkout@v3

      - name: Push ${{ matrix.module.name }}
        env:
          SPOKE_TOKEN: ${{ secrets.SPOKE_TOKEN }}
          SPOKE_REGISTRY: ${{ secrets.SPOKE_REGISTRY }}
        run: |
          VERSION=${GITHUB_REF#refs/tags/}
          spoke-cli push \
            -module ${{ matrix.module.name }} \
            -version $VERSION \
            -dir ${{ matrix.module.dir }} \
            -registry $SPOKE_REGISTRY
```

## Best Practices

### 1. Validate Before Push

Always validate schemas before pushing:

```yaml
- name: Validate
  run: spoke-cli validate -dir ./proto

- name: Push
  if: success()
  run: spoke-cli push ...
```

### 2. Use Semantic Versioning

Tag releases with semantic versions:

```bash
git tag v1.0.0
git push --tags
```

### 3. Add Descriptive Messages

Include build metadata:

```bash
spoke-cli push \
  -module mymodule \
  -version v1.0.0 \
  -description "Release v1.0.0 from commit ${GIT_SHA} by ${GIT_AUTHOR}"
```

### 4. Cache CLI Binary

Cache Spoke CLI to speed up builds:

```yaml
# GitHub Actions
- name: Cache Spoke CLI
  uses: actions/cache@v3
  with:
    path: /usr/local/bin/spoke-cli
    key: spoke-cli-${{ runner.os }}-latest
```

### 5. Use Protected Branches

Require validation before merging:

```yaml
# GitHub branch protection
- Require status checks: validate-schemas
- Require review from code owners
```

## Troubleshooting

### Authentication Fails

**Check token:**

```bash
# Test token
curl -H "Authorization: Bearer $SPOKE_TOKEN" \
  http://spoke.company.com/health
```

### Network Issues

**Use retry logic:**

```bash
for i in {1..3}; do
  spoke-cli push ... && break
  sleep 5
done
```

### Version Conflicts

**Handle existing versions:**

```bash
# Check if version exists
if spoke-cli list -module mymodule | grep -q "v1.0.0"; then
  echo "Version already exists, skipping"
  exit 0
fi

# Or use force flag (not recommended)
spoke-cli push --force ...
```

## Next Steps

- [Webhooks Guide](/guides/webhooks/) - Webhook configuration
- [API Reference](/guides/api-reference/) - REST API documentation
- [CLI Reference](/guides/cli-reference/) - Complete CLI commands
