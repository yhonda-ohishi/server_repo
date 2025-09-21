# Quick Start: Achieving 100% Test Coverage

## Prerequisites

- Go 1.21 or higher
- Make installed
- Docker (for integration tests)
- Redis (for cache tests)

## Quick Commands

### Run All Tests with Coverage
```bash
# Run all tests and generate coverage report
make test-coverage

# Or manually:
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

### Check Current Coverage
```bash
# See coverage percentage
go test -cover ./...

# Detailed coverage by function
go tool cover -func=coverage.out

# Coverage by package
go test -coverpkg=./... -coverprofile=coverage.out ./...
```

### Run Specific Test Categories

```bash
# Unit tests only
go test -tags=unit ./...

# Integration tests only
go test -tags=integration ./tests/integration

# Contract tests
go test ./tests/contract

# Performance tests
go test -tags=performance -bench=. ./tests/performance
```

## Step-by-Step Coverage Improvement

### Step 1: Generate Mocks
```bash
# Install mockgen
go install github.com/golang/mock/mockgen@latest

# Generate all mocks
make mocks

# Or generate specific mock
mockgen -source=src/services/cache/cache.go -destination=tests/mocks/cache_mock.go
```

### Step 2: Run Coverage Analysis
```bash
# Find uncovered code
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# List uncovered files
go list -f '{{if lt .Coverage 100}}{{.ImportPath}}: {{.Coverage}}%{{end}}' ./...
```

### Step 3: Write Missing Tests
```bash
# Create test file if it doesn't exist
touch src/services/registry/registry_test.go

# Run tests for specific package
go test -v -cover ./src/services/registry

# Run with race detection
go test -race ./src/services/registry
```

### Step 4: Validate Coverage
```bash
# Check if coverage meets target
./scripts/check-coverage.sh 100

# Generate coverage badge
go test -coverprofile=coverage.out ./...
gocov convert coverage.out | gocov-xml > coverage.xml
```

## Test Development Workflow

### 1. Identify Uncovered Code
```bash
# Generate HTML report
go tool cover -html=coverage.out -o coverage.html

# Open in browser and look for red (uncovered) lines
open coverage.html
```

### 2. Write Test Cases
```go
// Example test structure
func TestServiceRegistry_Register(t *testing.T) {
    tests := []struct {
        name    string
        service *models.ServiceRegistry
        wantErr bool
    }{
        {
            name:    "valid service",
            service: testdata.ValidService(),
            wantErr: false,
        },
        {
            name:    "duplicate service",
            service: testdata.DuplicateService(),
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### 3. Run and Verify
```bash
# Run new tests
go test -v -run TestServiceRegistry_Register ./src/services/registry

# Check coverage increased
go test -cover ./src/services/registry
```

## CI/CD Integration

### GitHub Actions Setup
```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run tests
        run: |
          go test -race -coverprofile=coverage.out ./...
          go tool cover -func=coverage.out

      - name: Check coverage
        run: |
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print substr($3, 1, length($3)-1)}')
          echo "Coverage: $coverage%"
          if (( $(echo "$coverage < 100" | bc -l) )); then
            echo "Coverage is below 100%"
            exit 1
          fi

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
```

### Local Pre-commit Hook
```bash
# .git/hooks/pre-commit
#!/bin/sh
echo "Running tests..."
go test -cover ./...
if [ $? -ne 0 ]; then
    echo "Tests failed. Commit aborted."
    exit 1
fi

coverage=$(go test -cover ./... 2>&1 | grep -E "^ok" | awk '{print $5}' | sed 's/[^0-9.]//g' | awk '{sum+=$1; count++} END {print sum/count}')
if (( $(echo "$coverage < 100" | bc -l) )); then
    echo "Coverage is ${coverage}%, must be 100%"
    exit 1
fi
```

## Troubleshooting

### Common Issues

#### Tests Not Running
```bash
# Check for build errors
go build ./...

# Run verbose mode
go test -v ./...

# Check test tags
go test -tags=integration ./...
```

#### Coverage Not Increasing
```bash
# Check if tests are actually executing code
go test -v -covermode=atomic ./...

# Look for skipped tests
go test -v ./... | grep SKIP

# Check for build constraints
grep -r "// +build" tests/
```

#### Flaky Tests
```bash
# Run tests multiple times
go test -count=10 ./...

# Run with race detection
go test -race ./...

# Run tests in isolation
go test -parallel=1 ./...
```

### Performance Issues

#### Slow Tests
```bash
# Profile test execution
go test -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof

# Run only quick tests
go test -short ./...

# Skip slow tests
go test -run '!Slow' ./...
```

#### Memory Issues
```bash
# Check for memory leaks
go test -memprofile=mem.prof ./...
go tool pprof mem.prof

# Run with memory limit
GOGC=50 go test ./...
```

## Best Practices

### Test Organization
```
tests/
├── unit/           # Fast, isolated tests
├── integration/    # Component interaction tests
├── contract/       # API contract tests
├── performance/    # Load and benchmark tests
├── fixtures/       # Test data files
├── helpers/        # Shared test utilities
└── mocks/         # Generated mocks
```

### Test Naming
```go
// Good test names
TestServiceRegistry_Register_ValidService
TestServiceRegistry_Register_DuplicateError
TestServiceRegistry_Deregister_NotFound

// Bad test names
TestRegister
TestError
Test1
```

### Assertion Best Practices
```go
// Use testify for clear assertions
assert.NoError(t, err)
assert.Equal(t, expected, actual)
require.NotNil(t, result)

// Provide helpful messages
assert.NoError(t, err, "failed to register service %s", service.Name)

// Use require for critical assertions
require.NoError(t, err) // Test stops here if error
assert.Equal(t, expected, actual) // Test continues even if not equal
```

## Scripts and Tools

### Coverage Check Script
```bash
#!/bin/bash
# scripts/check-coverage.sh

TARGET=${1:-100}
COVERAGE=$(go test -cover ./... 2>&1 | grep -E "^ok" | awk '{print $5}' | sed 's/[^0-9.]//g' | awk '{sum+=$1; count++} END {print sum/count}')

echo "Current coverage: ${COVERAGE}%"
echo "Target coverage: ${TARGET}%"

if (( $(echo "$COVERAGE < $TARGET" | bc -l) )); then
    echo "❌ Coverage below target"
    exit 1
else
    echo "✅ Coverage meets target"
fi
```

### Find Untested Files
```bash
#!/bin/bash
# scripts/find-untested.sh

for file in $(find src -name "*.go" -not -name "*_test.go"); do
    testfile="${file%%.go}_test.go"
    if [ ! -f "$testfile" ]; then
        echo "Missing tests: $file"
    fi
done
```

## Next Steps

1. **Generate mocks**: `make mocks`
2. **Run coverage analysis**: `make test-coverage`
3. **Open coverage report**: `open coverage.html`
4. **Write tests for uncovered code**
5. **Set up CI/CD with coverage gates**
6. **Monitor coverage trends**

---
*Quick Start Guide v1.0.0*
*Updated: 2025-09-20*