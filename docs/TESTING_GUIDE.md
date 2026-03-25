# Testing Guide

This document provides comprehensive guidance for running unit tests, integration tests, and end-to-end tests in SereniBase.

## Table of Contents

1. [Test Types](#test-types)
2. [Running Tests](#running-tests)
3. [Test Coverage](#test-coverage)
4. [Writing Tests](#writing-tests)
5. [CI/CD Testing](#cicd-testing)
6. [Troubleshooting](#troubleshooting)

---

## Test Types

### Unit Tests

**Purpose:** Test individual functions and components in isolation

**Location:** `tests/*/test.go` and `internal/*/test.go`

**Characteristics:**
- Fast execution (< 1 second typically)
- No external dependencies (use mocks)
- High coverage target (90%+)
- Run on every commit

**Example:**
```bash
go test -v -race ./tests/...
```

### Integration Tests

**Purpose:** Test interactions between components and systems

**Location:** `tests/integration_test.go`

**Characteristics:**
- Requires running services (PostgreSQL, email, storage)
- Tests workflows involving multiple components
- Slower execution (5-30 seconds)
- Validates data persistence and isolation
- Security-focused (RBAC, multi-tenancy)

**Test Categories:**
- User workflows
- Multi-tenant isolation
- RBAC enforcement
- Database transactions
- API workflows
- Audit logging
- Error recovery

**Example:**
```bash
go test -v -tags=integration ./tests/integration_test.go
```

### End-to-End (E2E) Tests

**Purpose:** Test complete user workflows through the full application stack

**Requirements:**
- All services running (API, database, email, storage)
- Frontend application running
- Valid environment configuration

**Example:**
```bash
# Start application
make up

# Run E2E tests in separate terminal
make test-e2e
```

---

## Running Tests

### Quick Test

Run all tests excluding integration tests:

```bash
make test
```

### Full Test Suite with Coverage

Run all tests including integration tests with coverage report:

```bash
make test-coverage
```

### Specific Test Categories

#### Run only unit tests
```bash
go test -v ./tests/...
```

#### Run only integration tests
```bash
go test -v -tags=integration ./tests/integration_test.go
```

#### Run specific test function
```bash
go test -v -run TestUserWorkflowIntegration ./tests/...
```

#### Run specific test case
```bash
go test -v -run TestUserWorkflowIntegration/User_SignUp_Login_CreateWorkspace ./tests/...
```

### With Race Detection

Detect data races during concurrent execution:

```bash
go test -race -v ./tests/...
```

### With Verbose Output

See detailed test execution logs:

```bash
go test -v ./tests/...
```

### With Short Timeout

For CI/CD with strict timeouts:

```bash
go test -timeout=5m ./tests/...
```

---

## Test Coverage

### Coverage Report

Generate and view coverage report:

```bash
# Generate coverage.out file
make test-coverage

# View in browser (on macOS)
go tool cover -html=coverage.out -o coverage.html
open coverage.html

# View in browser (on Windows)
go tool cover -html=coverage.out -o coverage.html
start coverage.html

# View in terminal
go tool cover -func=coverage.out | head -20
```

### Coverage Distribution

Current coverage by package:

| Package | Coverage |
|---------|----------|
| internal/handlers | 88% |
| internal/services | 91% |
| internal/utils | 94% |
| internal/middleware | 85% |
| internal/config | 92% |
| internal/dto | 89% |
| **Overall** | **90.1%** |

### Coverage Goals

| Category | Target |
|----------|--------|
| Unit Tests | 90%+ |
| Integration Tests | Critical paths covered |
| E2E Tests | Happy path + main workflows |

---

## Writing Tests

### Unit Test Template

```go
package handlers

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserHandler_GetUser(t *testing.T) {
	// ARRANGE: Set up test data and mocks
	mockUserService := &MockUserService{}
	handler := NewUserHandler(mockUserService)

	// ACT: Execute the code under test
	result, err := handler.GetUser("user-123")

	// ASSERT: Verify results
	require.NoError(t, err)
	assert.Equal(t, "user-123", result.ID)
	assert.Equal(t, "John Doe", result.Name)
}
```

### Integration Test Template

```go
func TestWorkflowIntegration_CreateWorkspace(t *testing.T) {
	t.Run("Happy_Path_Create_Workspace", func(t *testing.T) {
		// SETUP: Initialize application with real database
		app, db := setupTestApp(t)
		defer teardownTestApp(db)

		// TEST: Verify workspace creation workflow
		// 1. Create user
		// 2. Create workspace
		// 3. Verify database state

		// CLEANUP: Handled by defer

		t.Log("✓ Workspace creation workflow verified")
	})
}
```

### Best Practices

1. **Use Table-Driven Tests**
   ```go
   tests := []struct {
       name    string
       input   string
       expect  string
   }{
       {"Empty", "", ""},
       {"Valid", "test", "TEST"},
   }
   for _, tt := range tests {
       t.Run(tt.name, func(t *testing.T) {
           result := ToUpper(tt.input)
           assert.Equal(t, tt.expect, result)
       })
   }
   ```

2. **Use Subtests with `t.Run()`**
   ```go
   t.Run("User_Login", func(t *testing.T) {
       // Test cases
   })
   ```

3. **Use Testify for Assertions**
   ```go
   assert.Equal(t, expected, actual)
   require.NoError(t, err) // Fail test if error
   ```

4. **Mock External Dependencies**
   ```go
   type MockAuthService struct {
       mock.Mock
   }
   ```

5. **Test Edge Cases**
   - Empty inputs
   - Null/nil values
   - Boundary values
   - Concurrent access
   - Error conditions

---

## CI/CD Testing

### GitHub Actions Workflow

Tests run automatically on:
- Every push to `main` or `develop`
- Every pull request

### Test Execution Steps

1. **Checkout Code** → Pull latest code
2. **Setup Go** → Configure Go 1.24 environment
3. **Go Vet** → Static analysis
4. **Unit Tests** → Run all unit tests with coverage
5. **Docker Build** → Verify Docker image builds
6. **SonarQube** → Quality gate analysis (on main/develop only)

### SonarQube Quality Gate

Tests must pass SonarQube quality gate to merge:

- Code coverage ≥ 80%
- No critical security hotspots
- No critical bugs
- No critical code smells

View dashboard: https://sonar.aptlogica.com/

---

## Troubleshooting

### Tests Fail: Connection Refused

**Problem:** Database connection fails

**Solution:**
```bash
# Start PostgreSQL
docker-compose up -d postgres

# Verify connection
psql -h localhost -U postgres -d serenibase

# Check .env file
cat .env | grep DATABASE_
```

### Tests Fail: Timeout

**Problem:** Tests take too long

**Solution:**
```bash
# Increase timeout
go test -timeout=10m ./tests/...

# Run only fast tests
go test -v -short ./tests/...
```

### Coverage Report Missing

**Problem:** `coverage.out` not generated

**Solution:**
```bash
# Create coverage directory
mkdir -p coverage

# Run with coverage
go test -v -coverprofile=coverage/coverage.out ./tests/...
```

### Mock Not Working

**Problem:** Mock methods not being called

**Solution:**
```go
// Verify mock was called
mock.AssertExpectations(t)

// Verify call arguments
mock.AssertCalledWith(t, "expected", "args")
```

### Race Condition Detected

**Problem:** `-race` flag detects concurrent access issue

**Solution:**
1. Identify the failing test
2. Add synchronization (mutex, channels)
3. Verify with: `go test -race -v ./tests/...`

---

## Test Development Workflow

### 1. Write Failing Test

```bash
# Write test in tests/foo_test.go
# Run test - it should fail
go test -v -run TestFoo ./tests/...
```

### 2. Implement Feature

```bash
# Add code to make test pass
# Edit internal/foo/foo.go
```

### 3. Verify Test Passes

```bash
# Ensure test passes
go test -v -run TestFoo ./tests/...

# Check coverage improved
go test -coverprofile=coverage.out ./tests/...
```

### 4. Check Overall Coverage

```bash
go tool cover -func=coverage.out | grep total
```

### 5. Commit

```bash
git add tests/foo_test.go internal/foo/foo.go
git commit -m "feat: add foo feature with tests"
```

---

## Test Maintenance

### Regular Tasks

| Task | Frequency |
|------|-----------|
| Review coverage reports | Weekly |
| Update mocks | When APIs change |
| Add regression tests | When bugs fixed |
| Refactor slow tests | Monthly |
| Archive old tests | Quarterly |

### Adding New Tests

1. Identify untested code
2. Write test following template
3. Run test locally
4. Commit with feature
5. Monitor in CI/CD

### Keeping Tests Fast

- Use mocks instead of real services
- Parallelize independent tests: `go test -parallel 4 ./...`
- Clean up resources in `defer`
- Use table-driven tests for variations
- Skip slow tests in `-short` mode

---

## References

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Go Vet](https://pkg.go.dev/cmd/vet)
- [Coverage Tools](https://go.dev/blog/cover)

## Support

For test-related questions:
- Check existing tests in `tests/` directory
- Review [Contributing Guide](../CONTRIBUTING.md)
- Open an issue or contact maintainers
