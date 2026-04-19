# Testing Guide

## Overview

SkillHub requires **80% minimum test coverage** across all packages. This document explains how to run tests, add new tests, and maintain coverage standards.

---

## Running Tests

### Run all tests
```bash
go test ./...
```

### Run with coverage
```bash
go test -cover ./...
```

### Run with race detection
```bash
go test -race ./...
```

### Run integration tests
```bash
# Requires PostgreSQL running on localhost:5432
export DATABASE_URL="postgres://skillhub:skillhub_dev@localhost:5432/skillhub_test?sslmode=disable"
go test -v ./internal/handler/...
```

### Run eval tests
```bash
go test -v -tags=eval ./internal/eval
```

---

## Test Types

### 1. Unit Tests

Test individual functions and methods in isolation.

**Location**: `*_test.go` files next to source files

**Example**:
```go
// internal/crypto/crypto_test.go
func TestHashToken(t *testing.T) {
    token := "test-token-123"
    hash := HashToken(token)

    if hash == "" {
        t.Error("Expected non-empty hash")
    }
    if hash == token {
        t.Error("Hash should not equal plaintext token")
    }
}
```

### 2. Integration Tests

Test multiple components working together, including database interactions.

**Location**: `internal/handler/*_test.go`

**Requirements**:
- PostgreSQL test database running
- Migrations applied
- Test data cleanup after each test

**Example**:
```go
// internal/handler/skill_submit_test.go
func TestSubmit_ValidSkill_ReturnsCreated(t *testing.T) {
    pool, cleanup := setupTestDB(t)
    defer cleanup()

    // Test full submission flow
    // ...
}
```

### 3. Eval Tests

Test AI review quality and accuracy.

**Location**: `internal/eval/`

**Run with**: `go test -tags=eval ./internal/eval`

**Purpose**:
- Measure false positive/negative rates
- Ensure consistent review quality
- Catch regressions in security detection

---

## Coverage Requirements

### Per-Package Minimums

| Package | Minimum Coverage | Notes |
|---------|------------------|-------|
| `internal/handler` | 80% | Core API handlers |
| `internal/review` | 90% | Security-critical |
| `internal/crypto` | 90% | Security-critical |
| `internal/db` | 70% | Database layer |
| `internal/models` | 50% | Mostly structs |
| `cmd/api` | 60% | Main entry point |

### Checking Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View in terminal
go tool cover -func=coverage.out

# View in browser
go tool cover -html=coverage.out
```

### CI Enforcement

The CI pipeline fails if:
- Total coverage < 80%
- Any critical package < minimum threshold
- Eval accuracy < 90%

---

## Writing Good Tests

### Table-Driven Tests

Preferred pattern for testing multiple cases:

```go
func TestValidateSemVer(t *testing.T) {
    tests := []struct {
        name    string
        version string
        wantErr bool
    }{
        {"valid semver", "1.0.0", false},
        {"invalid format", "1.0", true},
        {"with v prefix", "v1.0.0", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateSemVer(tt.version)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateSemVer() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Test Helpers

Use `t.Helper()` for setup functions:

```go
func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
    t.Helper()

    pool, err := pgxpool.New(context.Background(), testDBURL)
    if err != nil {
        t.Skipf("Cannot connect to test DB: %v", err)
    }

    cleanup := func() {
        pool.Exec(context.Background(), "TRUNCATE TABLE skills CASCADE")
        pool.Close()
    }

    return pool, cleanup
}
```

### Testing Error Cases

Always test both success and failure paths:

```go
func TestSubmit_EmptyContent_ReturnsBadRequest(t *testing.T) {
    // Test that empty content is rejected
    // ...

    if w.Code != http.StatusBadRequest {
        t.Errorf("Expected 400, got %d", w.Code)
    }
}
```

---

## Test Database Setup

### Local Development

```bash
# Start PostgreSQL with Docker
docker run -d \
  --name skillhub-test \
  -e POSTGRES_DB=skillhub_test \
  -e POSTGRES_USER=skillhub \
  -e POSTGRES_PASSWORD=skillhub_dev \
  -p 5432:5432 \
  pgvector/pgvector:pg17

# Run migrations
export DATABASE_URL="postgres://skillhub:skillhub_dev@localhost:5432/skillhub_test?sslmode=disable"
goose -dir migrations postgres "$DATABASE_URL" up

# Run tests
go test ./...
```

### CI Environment

Tests run automatically in GitHub Actions with:
- PostgreSQL service container
- Automatic migration application
- Coverage reporting to Codecov

---

## Adding New Tests

### Checklist

When adding a new feature:

- [ ] Write unit tests for new functions
- [ ] Write integration tests for new endpoints
- [ ] Add eval cases if touching review logic
- [ ] Test edge cases (empty input, invalid data, etc.)
- [ ] Test error handling (timeouts, DB failures, etc.)
- [ ] Verify coverage meets threshold

### Example Workflow

1. **Write the test first** (TDD):
   ```go
   func TestNewFeature(t *testing.T) {
       // Test should fail initially
   }
   ```

2. **Implement the feature**:
   ```go
   func NewFeature() {
       // Implementation
   }
   ```

3. **Run tests**:
   ```bash
   go test -v ./internal/handler
   ```

4. **Check coverage**:
   ```bash
   go test -cover ./internal/handler
   ```

5. **Add edge cases** until coverage ≥ 80%

---

## Common Patterns

### Testing HTTP Handlers

```go
func TestHandler(t *testing.T) {
    req := httptest.NewRequest("POST", "/v1/skills", body)
    w := httptest.NewRecorder()

    handler.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("Expected 200, got %d", w.Code)
    }
}
```

### Testing Database Operations

```go
func TestInsertSkill(t *testing.T) {
    pool, cleanup := setupTestDB(t)
    defer cleanup()

    // Insert test data
    _, err := pool.Exec(ctx, "INSERT INTO skills ...")
    if err != nil {
        t.Fatalf("Insert failed: %v", err)
    }

    // Verify insertion
    var count int
    pool.QueryRow(ctx, "SELECT COUNT(*) FROM skills").Scan(&count)
    if count != 1 {
        t.Errorf("Expected 1 skill, got %d", count)
    }
}
```

### Testing Async Operations

```go
func TestAsyncReview(t *testing.T) {
    // Enqueue job
    _, err := riverClient.Insert(ctx, ReviewJobArgs{...})
    if err != nil {
        t.Fatalf("Failed to enqueue: %v", err)
    }

    // Wait for completion (with timeout)
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    // Poll for result
    for {
        select {
        case <-ctx.Done():
            t.Fatal("Timeout waiting for review")
        default:
            // Check if review completed
            // ...
        }
    }
}
```

---

## Troubleshooting

### Tests fail with "cannot connect to database"

Ensure PostgreSQL is running:
```bash
docker ps | grep skillhub-test
```

### Coverage report shows 0% for a package

Check if tests are in the same package:
```go
// ✅ Correct
package handler

func TestSubmit(t *testing.T) { ... }

// ❌ Wrong
package handler_test  // External test package

func TestSubmit(t *testing.T) { ... }
```

### Integration tests are slow

Use `testing.Short()` to skip in short mode:
```go
func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    // ...
}
```

Run fast tests only:
```bash
go test -short ./...
```

---

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Table-Driven Tests](https://go.dev/wiki/TableDrivenTests)
- [Testify Assertions](https://github.com/stretchr/testify)
- [httptest Package](https://pkg.go.dev/net/http/httptest)
