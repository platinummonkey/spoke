# Testing Guide

## Overview

This guide covers testing strategies and examples for the Advanced Search & Dependency Visualization features in Spoke.

## Testing Pyramid

```
        /\
       /E2E\         10% - End-to-end tests
      /------\
     /Integration\   30% - Integration tests
    /--------------\
   /   Unit Tests   \ 60% - Unit tests
  /------------------\
```

## Unit Tests

### Backend (Go)

**Location:** `pkg/search/*_test.go`, `pkg/dependencies/*_test.go`

**Query Parser Tests:**
```go
func TestQueryParser_ParseEntityFilter(t *testing.T) {
    parser := NewQueryParser()
    
    tests := []struct {
        name     string
        query    string
        expected []string
    }{
        {
            name:     "single entity filter",
            query:    "user entity:message",
            expected: []string{"message"},
        },
        {
            name:     "multiple entity filters",
            query:    "user entity:message entity:enum",
            expected: []string{"message", "enum"},
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := parser.Parse(tt.query)
            require.NoError(t, err)
            assert.Equal(t, tt.expected, result.EntityTypes)
        })
    }
}
```

**Search Service Tests:**
```go
func TestSearchService_Search(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer db.Close()
    
    // Seed test data
    seedSearchIndex(t, db)
    
    service := NewSearchService(db)
    
    tests := []struct {
        name          string
        query         string
        expectedCount int
    }{
        {
            name:          "search by entity name",
            query:         "UserProfile",
            expectedCount: 1,
        },
        {
            name:          "search with entity filter",
            query:         "user entity:message",
            expectedCount: 3,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := context.Background()
            req := SearchRequest{Query: tt.query, Limit: 50}
            
            response, err := service.Search(ctx, req)
            require.NoError(t, err)
            assert.Equal(t, tt.expectedCount, len(response.Results))
        })
    }
}
```

**Dependency Graph Tests:**
```go
func TestDependencyGraph_GetImpactAnalysis(t *testing.T) {
    graph := setupTestGraph(t)
    
    impact := graph.GetImpactAnalysis("common", "v1.0.0")
    
    assert.Equal(t, "common", impact.Module)
    assert.Equal(t, "v1.0.0", impact.Version)
    assert.Equal(t, 2, len(impact.DirectDependents))
    assert.Equal(t, 1, len(impact.TransitiveDependents))
    assert.Equal(t, 3, impact.TotalImpact)
}
```

**Running Backend Tests:**
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./pkg/search/...

# Run with race detector
go test -race ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Frontend (TypeScript + React)

**Location:** `web/src/**/*.test.tsx`

**Hook Tests (useEnhancedSearch):**
```typescript
import { renderHook, waitFor } from '@testing-library/react';
import { useEnhancedSearch } from '../hooks/useEnhancedSearch';

describe('useEnhancedSearch', () => {
  it('debounces query input', async () => {
    const { result, rerender } = renderHook(() => useEnhancedSearch());
    
    // Change query rapidly
    act(() => {
      result.current.setQuery('u');
    });
    act(() => {
      result.current.setQuery('us');
    });
    act(() => {
      result.current.setQuery('user');
    });
    
    // Should not trigger search immediately
    expect(result.current.loading).toBe(false);
    
    // Wait for debounce (300ms)
    await waitFor(() => {
      expect(result.current.loading).toBe(true);
    }, { timeout: 500 });
  });
  
  it('parses filters correctly', () => {
    const { result } = renderHook(() => useEnhancedSearch());
    
    act(() => {
      result.current.setQuery('user entity:message type:string');
    });
    
    expect(result.current.filters).toHaveLength(2);
    expect(result.current.filters[0]).toMatchObject({
      type: 'entity',
      value: 'message',
    });
    expect(result.current.filters[1]).toMatchObject({
      type: 'field-type',
      value: 'string',
    });
  });
  
  it('cancels in-flight requests', async () => {
    const { result } = renderHook(() => useEnhancedSearch());
    
    // First query
    act(() => {
      result.current.setQuery('user');
    });
    
    await waitFor(() => {
      expect(result.current.loading).toBe(true);
    });
    
    // Second query (should cancel first)
    act(() => {
      result.current.setQuery('order');
    });
    
    // First request should be aborted
    await waitFor(() => {
      expect(result.current.error).toBeNull();
      expect(result.current.results).not.toContain('user');
    });
  });
});
```

**Component Tests (EnhancedSearchBar):**
```typescript
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { EnhancedSearchBar } from '../components/EnhancedSearchBar';

describe('EnhancedSearchBar', () => {
  it('opens modal on CMD+K', () => {
    render(<EnhancedSearchBar />);
    
    fireEvent.keyDown(document, { key: 'k', metaKey: true });
    
    expect(screen.getByPlaceholderText(/search protobuf/i)).toBeInTheDocument();
  });
  
  it('displays filter chips', async () => {
    render(<EnhancedSearchBar />);
    
    // Open modal
    fireEvent.click(screen.getByText(/advanced search/i));
    
    // Type query with filters
    const input = screen.getByPlaceholderText(/search protobuf/i);
    fireEvent.change(input, { target: { value: 'user entity:message' } });
    
    await waitFor(() => {
      expect(screen.getByText(/entity: message/i)).toBeInTheDocument();
    });
  });
  
  it('removes filter on chip close', async () => {
    render(<EnhancedSearchBar />);
    
    // Open modal and add filter
    fireEvent.click(screen.getByText(/advanced search/i));
    const input = screen.getByPlaceholderText(/search protobuf/i);
    fireEvent.change(input, { target: { value: 'user entity:message' } });
    
    await waitFor(() => {
      expect(screen.getByText(/entity: message/i)).toBeInTheDocument();
    });
    
    // Click X on chip
    const closeButton = screen.getByLabelText(/remove.*filter/i);
    fireEvent.click(closeButton);
    
    expect(screen.queryByText(/entity: message/i)).not.toBeInTheDocument();
  });
});
```

**Running Frontend Tests:**
```bash
# Run all tests
npm test

# Run with coverage
npm test -- --coverage

# Run in watch mode
npm test -- --watch

# Run specific test file
npm test -- EnhancedSearchBar.test.tsx
```

## Integration Tests

### Backend API Tests

**Location:** `tests/integration/`

**Search API Integration:**
```go
func TestSearchAPI_EndToEnd(t *testing.T) {
    // Start test server
    server := setupTestServer(t)
    defer server.Close()
    
    // Push test module
    pushTestModule(t, server, "user", "v1.0.0")
    
    // Wait for indexing
    time.Sleep(100 * time.Millisecond)
    
    // Test search
    resp := httpGet(t, server.URL+"/api/v2/search?q=user")
    assert.Equal(t, 200, resp.StatusCode)
    
    var result SearchResponse
    json.NewDecoder(resp.Body).Decode(&result)
    
    assert.Greater(t, result.TotalCount, 0)
    assert.NotEmpty(t, result.Results)
}

func TestImpactAPI_EndToEnd(t *testing.T) {
    server := setupTestServer(t)
    defer server.Close()
    
    // Push modules with dependencies
    pushTestModule(t, server, "common", "v1.0.0")
    pushTestModule(t, server, "user", "v1.0.0", "common@v1.0.0")
    pushTestModule(t, server, "order", "v1.0.0", "user@v1.0.0")
    
    // Test impact analysis
    resp := httpGet(t, server.URL+"/modules/common/versions/v1.0.0/impact")
    assert.Equal(t, 200, resp.StatusCode)
    
    var impact ImpactAnalysis
    json.NewDecoder(resp.Body).Decode(&impact)
    
    assert.Equal(t, "common", impact.Module)
    assert.Equal(t, 2, len(impact.DirectDependents))
    assert.Equal(t, 1, len(impact.TransitiveDependents))
}
```

**Running Integration Tests:**
```bash
# Run integration tests
go test -tags=integration ./tests/integration/...

# With verbose output
go test -v -tags=integration ./tests/integration/...
```

### Frontend Integration Tests

**React Testing Library:**
```typescript
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import App from '../App';

describe('Search Flow Integration', () => {
  it('performs full search workflow', async () => {
    // Mock API
    global.fetch = jest.fn((url) => {
      if (url.includes('/api/v2/search')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            results: [
              {
                id: 1,
                entity_type: 'message',
                entity_name: 'UserProfile',
                full_path: 'user.v1.UserProfile',
                module_name: 'user',
                version: 'v1.0.0',
                rank: 0.85,
              },
            ],
            total_count: 1,
            query: 'user entity:message',
          }),
        });
      }
    });
    
    render(
      <BrowserRouter>
        <App />
      </BrowserRouter>
    );
    
    // Open search modal
    const user = userEvent.setup();
    await user.keyboard('{Meta>}k{/Meta}');
    
    // Type query
    const input = screen.getByPlaceholderText(/search protobuf/i);
    await user.type(input, 'user entity:message');
    
    // Wait for results
    await waitFor(() => {
      expect(screen.getByText('UserProfile')).toBeInTheDocument();
    });
    
    // Verify result details
    expect(screen.getByText('user.v1.UserProfile')).toBeInTheDocument();
    expect(screen.getByText('user@v1.0.0')).toBeInTheDocument();
    
    // Click result
    await user.click(screen.getByText('UserProfile'));
    
    // Should navigate to module detail
    await waitFor(() => {
      expect(window.location.pathname).toContain('/modules/user');
    });
  });
});
```

## End-to-End Tests

### Cypress Tests

**Location:** `web/cypress/e2e/`

**Setup:**
```bash
npm install --save-dev cypress
npx cypress open
```

**cypress.config.ts:**
```typescript
import { defineConfig } from 'cypress';

export default defineConfig({
  e2e: {
    baseUrl: 'http://localhost:3000',
    setupNodeEvents(on, config) {
      // implement node event listeners here
    },
  },
});
```

**Search E2E Test:**
```typescript
// cypress/e2e/search.cy.ts
describe('Advanced Search', () => {
  beforeEach(() => {
    cy.visit('/');
  });
  
  it('searches with filters', () => {
    // Open search modal
    cy.get('[aria-label*="search"]').click();
    
    // Type query with filters
    cy.get('input[placeholder*="Search"]')
      .type('user entity:message');
    
    // Verify filter chips appear
    cy.contains('Entity: message').should('be.visible');
    
    // Wait for results
    cy.contains(/found \d+ result/i, { timeout: 5000 })
      .should('be.visible');
    
    // Verify result appears
    cy.contains('UserProfile').should('be.visible');
    
    // Click result
    cy.contains('UserProfile').click();
    
    // Verify navigation
    cy.url().should('include', '/modules/user');
  });
  
  it('uses saved search', () => {
    cy.visit('/library');
    
    // Create saved search
    cy.contains('Saved Searches').parent().within(() => {
      cy.get('[aria-label*="Save new"]').click();
    });
    
    cy.get('input[placeholder*="name"]').type('User Messages');
    cy.get('input[placeholder*="query"]').type('user entity:message');
    cy.contains('button', 'Save').click();
    
    // Verify saved
    cy.contains('User Messages').should('be.visible');
    
    // Execute saved search
    cy.contains('User Messages').parent().within(() => {
      cy.get('[aria-label*="options"]').click();
    });
    cy.contains('Execute Search').click();
    
    // Verify search modal opens with query
    cy.get('input[placeholder*="Search"]')
      .should('have.value', 'user entity:message');
  });
});
```

**Dependency Graph E2E Test:**
```typescript
describe('Dependency Graph', () => {
  it('renders and navigates graph', () => {
    cy.visit('/modules/common?version=v1.0.0');
    
    // Click Dependencies tab
    cy.contains('Dependencies').click();
    
    // Wait for graph to render
    cy.get('[data-id="cytoscape-container"]', { timeout: 5000 })
      .should('be.visible');
    
    // Verify nodes exist
    cy.contains('common').should('be.visible');
    
    // Change layout
    cy.get('select').select('Force-Directed (Cola)');
    
    // Verify layout changed (graph re-rendered)
    cy.wait(1000);
    
    // Export as PNG
    cy.get('[aria-label="Export PNG"]').click();
    
    // Verify toast notification
    cy.contains('Graph exported').should('be.visible');
  });
});
```

**Running E2E Tests:**
```bash
# Open Cypress UI
npx cypress open

# Run headless
npx cypress run

# Run specific test
npx cypress run --spec "cypress/e2e/search.cy.ts"
```

## Performance Testing

### Load Testing with Apache Bench

```bash
# Test search endpoint
ab -n 1000 -c 10 \
  http://localhost:8080/api/v2/search?q=user

# Expected results:
# Requests per second: >100
# Mean response time: <50ms
# 95th percentile: <200ms
```

### k6 Load Testing

```javascript
// load-test.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '1m', target: 50 },  // Ramp up to 50 users
    { duration: '3m', target: 50 },  // Stay at 50 users
    { duration: '1m', target: 0 },   // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<200'], // 95% of requests under 200ms
    http_req_failed: ['rate<0.01'],   // Less than 1% errors
  },
};

export default function () {
  const res = http.get('http://localhost:8080/api/v2/search?q=user');
  
  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 200ms': (r) => r.timings.duration < 200,
  });
  
  sleep(1);
}
```

**Run k6:**
```bash
k6 run load-test.js
```

## Test Coverage Goals

### Backend
- **Unit Tests:** >80% code coverage
- **Integration Tests:** All API endpoints
- **Critical Paths:** 100% coverage for search, dependencies

### Frontend
- **Components:** >70% coverage
- **Hooks:** >80% coverage
- **Integration:** All user workflows

### Overall
- **Critical Bugs:** 0 in production
- **Test Execution Time:** <5 minutes for full suite
- **Flaky Tests:** <2% failure rate

## Continuous Integration

### GitHub Actions Workflow

```yaml
name: Test Suite

on: [push, pull_request]

jobs:
  backend-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run tests
        run: go test -race -coverprofile=coverage.out ./...
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
  
  frontend-tests:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Node
        uses: actions/setup-node@v3
        with:
          node-version: '18'
      
      - name: Install dependencies
        run: cd web && npm ci
      
      - name: Run tests
        run: cd web && npm test -- --coverage
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./web/coverage/lcov.info
  
  e2e-tests:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up environment
        run: |
          docker-compose up -d
          npm install -g wait-on
          wait-on http://localhost:8080
      
      - name: Run Cypress
        uses: cypress-io/github-action@v5
        with:
          working-directory: web
          start: npm start
          wait-on: 'http://localhost:3000'
```

## Best Practices

### Test Organization
- Keep tests close to source code
- Use descriptive test names
- Follow AAA pattern (Arrange, Act, Assert)
- One assertion per test (when possible)
- Mock external dependencies

### Test Data
- Use factories for test data generation
- Clean up after each test
- Isolate tests (no shared state)
- Use realistic test data

### Assertions
- Be specific (avoid generic matchers)
- Test both success and failure cases
- Test edge cases and boundary conditions
- Verify error messages

### Maintenance
- Remove flaky tests immediately
- Update tests when requirements change
- Refactor tests along with code
- Run tests locally before pushing

## Debugging Tests

### Backend
```bash
# Run with verbose output
go test -v ./pkg/search/

# Run specific test
go test -run TestQueryParser_ParseEntityFilter

# Enable test cache
go test -count=1 ./...

# Print test output
go test -v ./... | grep -A 10 "FAIL"
```

### Frontend
```bash
# Debug mode
npm test -- --no-coverage --verbose

# Run single test
npm test -- --testNamePattern="debounces query"

# Update snapshots
npm test -- -u
```

### Cypress
```bash
# Open Cypress UI for debugging
npx cypress open

# Screenshot on failure
npx cypress run --config screenshotOnRunFailure=true

# Video recording
npx cypress run --config video=true
```

## Summary

Comprehensive testing strategy:
- **Unit Tests** (60%): Fast, isolated, high coverage
- **Integration Tests** (30%): API endpoints, hooks, components
- **E2E Tests** (10%): Critical user workflows
- **Performance Tests**: Load testing, benchmarks
- **CI/CD**: Automated testing on every push

Target: >80% backend coverage, >70% frontend coverage, 0 critical bugs in production.
