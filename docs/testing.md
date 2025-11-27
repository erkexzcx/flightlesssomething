# Testing Guide

## Running Tests

### Backend Unit Tests

Run all Go tests:
```bash
go test ./...
```

Run with coverage:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Run specific test:
```bash
go test -v ./internal/app -run TestHandleListBenchmarks
```

### Backend Integration Tests

Comprehensive API endpoint testing:
```bash
./backend_test.sh
```

Requirements:
- `jq` command-line JSON processor
- Go toolchain
- Test data in `testdata/` directory

Tests 20 scenarios including:
- Health check
- Authentication (admin login)
- Benchmark CRUD operations
- Data upload/download
- API tokens
- Audit logs
- Search functionality

### Frontend Unit Tests

Run JavaScript unit tests:
```bash
cd web
npm run test:unit
```

Or directly:
```bash
node web/tests/dateFormatter.test.js
```

### Frontend E2E Tests

Run Playwright tests:
```bash
cd web

# First time setup
npm install
npx playwright install --with-deps chromium

# Run tests
npm test

# Run with UI
npm run test:ui

# Run in headed mode (visible browser)
npm run test:headed
```

### Linting

Backend (Go):
```bash
golangci-lint run --timeout=5m
```

Frontend (JavaScript/Vue):
```bash
cd web
npm run lint
npm run lint:fix  # Auto-fix issues
```

## Test Structure

### Backend Tests

**Location**: `internal/app/*_test.go`

**Helper functions** (`test_helpers.go`):
- `setupTestDB(t)` - Create test database
- `cleanupTestDB(t, db)` - Clean up test database
- `createTestUser(db, username, isAdmin)` - Create test users
- `setupTestRouter(db)` - Create test router

**Example test**:
```go
func TestYourFeature(t *testing.T) {
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    t.Run("test case name", func(t *testing.T) {
        // Your test code
    })
}
```

### Frontend Tests

**Unit tests**: `web/tests/*.test.js`

Simple Node.js runner pattern:
```javascript
#!/usr/bin/env node

function test(description, fn) {
  try {
    fn();
    console.log(`✓ ${description}`);
  } catch (error) {
    console.error(`✗ ${description}`);
    console.error(`  ${error.message}`);
  }
}

test('your test', () => {
  const result = yourFunction(input);
  if (result !== expected) {
    throw new Error(`Expected ${expected} but got ${result}`);
  }
});
```

**E2E tests**: `web/tests/*.spec.js`

Playwright tests:
```javascript
import { test, expect } from '@playwright/test';

test('your test', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('selector')).toBeVisible();
});
```

## CI/CD Testing

Tests run automatically on every push via GitHub Actions.

**Workflow**: `.github/workflows/test.yml`

**Jobs**:
1. `lint-go` - Go linting
2. `lint-frontend` - JavaScript/Vue linting
3. `unit-tests` - Go unit tests with coverage
4. `build` - Build server binary (shared as artifact)
5. `backend-integration-test` - API testing
6. `e2e-tests` - Playwright E2E tests

All jobs must pass for PR merge.

## Writing Tests

### Backend Unit Test

1. Create/edit `internal/app/*_test.go`
2. Use `setupTestDB()` and `cleanupTestDB()`
3. Use table-driven tests for multiple cases
4. Always clean up resources

### Backend Integration Test

Edit `backend_test.sh`:
1. Add test section with incrementing number
2. Use `curl` for requests
3. Use `jq` for JSON validation
4. Check status codes and content
5. Add descriptive log messages

### Frontend Unit Test

Create `web/tests/*.test.js`:
1. Import function to test
2. Use `test()` helper
3. Throw error if assertion fails
4. Keep tests simple and focused

### Frontend E2E Test

Create `web/tests/*.spec.js`:
1. Import Playwright test utilities
2. Use `page.goto()` for navigation
3. Use `expect()` for assertions
4. Test user interactions

## Test Data

Test files in `testdata/benchmark1/`:
- CSV files from MangoHud
- Used by unit and integration tests
- Do not modify or remove

## Best Practices

1. **Test naming** - Use descriptive names
2. **Test isolation** - Independent tests
3. **Cleanup** - Always clean up resources
4. **Assertions** - Check success and failure
5. **Coverage** - Aim for high coverage on critical paths
6. **Documentation** - Comment complex test setups

## Troubleshooting

### Tests fail locally but pass in CI
- Check Go/Node.js versions match CI
- Ensure dependencies installed
- Check for race conditions

### Database errors
- Verify cleanup working properly
- Check file permissions
- Use unique database names for parallel tests

### E2E tests timeout
- Increase timeout in `playwright.config.js`
- Check server starting properly
- Verify baseURL correct

## Coverage

Current backend coverage: ~37.5%

View coverage:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Pre-commit Checks

Before committing:
```bash
# Run linters
golangci-lint run
cd web && npm run lint

# Run tests
go test ./...
./backend_test.sh
cd web && npm test
```
