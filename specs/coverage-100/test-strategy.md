# Test Strategy: 100% Coverage Implementation

## Testing Philosophy

### Core Principles
1. **Behavior over Implementation**: Test what the code does, not how
2. **Fast Feedback**: Unit tests run in milliseconds
3. **Independent Tests**: No shared state between tests
4. **Deterministic Results**: Same input always produces same output
5. **Maintainable Tests**: Clear naming, minimal setup, obvious assertions

## Test Categories

### 1. Unit Tests (60% of tests)
**Purpose**: Test individual functions/methods in isolation
**Location**: Same package as code under test
**Execution Time**: < 5 seconds total
**Dependencies**: All mocked

#### Coverage Requirements
- Every exported function
- Every error condition
- Every branch/condition
- Edge cases and boundaries

#### Test Structure
```go
func TestServiceName_MethodName(t *testing.T) {
    // Arrange
    mock := NewMockDependency()
    svc := NewService(mock)

    // Act
    result, err := svc.Method(input)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### 2. Integration Tests (25% of tests)
**Purpose**: Test component interactions
**Location**: tests/integration/
**Execution Time**: < 30 seconds total
**Dependencies**: Real where possible, mocked externals

#### Scenarios
- Service-to-service communication
- Database operations with test DB
- Cache operations with test Redis
- End-to-end request flows
- Circuit breaker behavior

### 3. Contract Tests (10% of tests)
**Purpose**: Validate API contracts
**Location**: tests/contract/
**Execution Time**: < 10 seconds
**Dependencies**: HTTP test server

#### Coverage
- Request/response schemas
- Status codes
- Error formats
- Authentication flows
- Content negotiation

### 4. Performance Tests (5% of tests)
**Purpose**: Validate performance requirements
**Location**: tests/performance/
**Execution Time**: Variable (not in CI)
**Dependencies**: Full stack

#### Types
- Load tests: Normal traffic patterns
- Stress tests: Breaking points
- Spike tests: Traffic bursts
- Soak tests: Memory leaks

## Mock Strategy

### Interface Mocking with gomock
```bash
# Generate mocks
mockgen -source=src/services/cache/cache.go -destination=tests/mocks/cache_mock.go
```

### Mock Organization
```
tests/mocks/
├── services/
│   ├── cache_mock.go
│   ├── auth_mock.go
│   └── router_mock.go
├── clients/
│   └── client_mock.go
└── repositories/
    └── repo_mock.go
```

### Mock Builders
```go
func NewMockCacheWithData(data map[string]string) *MockCache {
    mock := NewMockCache(ctrl)
    for k, v := range data {
        mock.EXPECT().Get(k).Return(v, nil).AnyTimes()
    }
    return mock
}
```

## Test Data Management

### Fixture Files
```
tests/fixtures/
├── models/
│   ├── service_valid.json
│   ├── service_invalid.json
│   └── service_edge_cases.json
├── requests/
│   ├── batch_valid.json
│   └── batch_invalid.json
└── responses/
    └── error_responses.json
```

### Test Data Factories
```go
package testdata

func ValidService(opts ...Option) *models.ServiceRegistry {
    svc := &models.ServiceRegistry{
        ID:       uuid.New().String(),
        Name:     "test-service",
        Type:     models.ServiceTypeDB,
        // defaults
    }
    for _, opt := range opts {
        opt(svc)
    }
    return svc
}
```

### Golden Files
For complex outputs, use golden files:
```go
func TestComplexOutput(t *testing.T) {
    output := GenerateOutput()
    golden := filepath.Join("testdata", t.Name()+".golden")

    if *update {
        ioutil.WriteFile(golden, output, 0644)
    }

    expected, _ := ioutil.ReadFile(golden)
    assert.Equal(t, expected, output)
}
```

## Parallel Execution

### Unit Test Parallelization
```go
func TestParallel(t *testing.T) {
    t.Parallel() // Mark test as parallel

    testCases := []struct{
        name string
        // ...
    }{
        // cases
    }

    for _, tc := range testCases {
        tc := tc // Capture range variable
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel() // Subtests also parallel
            // test logic
        })
    }
}
```

### Integration Test Isolation
```go
func TestIntegration(t *testing.T) {
    // Each test gets its own database
    db := setupTestDB(t)
    defer cleanupDB(db)

    // Test logic
}
```

## Test Helpers

### Assertion Helpers
```go
package helpers

func AssertErrorCode(t *testing.T, err error, code string) {
    t.Helper()
    var apiErr *models.ErrorDetails
    require.True(t, errors.As(err, &apiErr))
    assert.Equal(t, code, apiErr.Code)
}
```

### Test Servers
```go
func NewTestServer(t *testing.T) *httptest.Server {
    t.Helper()
    router := routes.SetupRoutes(testDeps())
    server := httptest.NewServer(router)
    t.Cleanup(server.Close)
    return server
}
```

### Wait Helpers
```go
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration) {
    t.Helper()
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if condition() {
            return
        }
        time.Sleep(10 * time.Millisecond)
    }
    t.Fatal("condition not met within timeout")
}
```

## Coverage Requirements

### Minimum Coverage by Component
| Component | Line Coverage | Branch Coverage | Notes |
|-----------|--------------|-----------------|-------|
| Models | 100% | 100% | All validation paths |
| Services | 100% | 100% | Core business logic |
| Handlers | 100% | 100% | All HTTP paths |
| Middleware | 100% | 100% | All conditions |
| Config | 95% | 95% | Exclude impossible errors |
| CLI | 90% | 90% | Exclude interactive parts |
| Main | 80% | 80% | Exclude signal handling |

### Coverage Exemptions
```go
// coverage:ignore - Signal handling cannot be tested
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
```

## CI/CD Integration

### Test Execution Pipeline
```yaml
test:
  stage: test
  script:
    - go test -race -coverprofile=coverage.out ./...
    - go tool cover -func=coverage.out
    - go tool cover -html=coverage.out -o coverage.html
  coverage: '/total:\s+\(statements\)\s+(\d+.\d+)%/'
  artifacts:
    reports:
      coverage_report:
        coverage_format: cobertura
        path: coverage.xml
```

### Coverage Gates
```yaml
coverage:
  stage: quality
  script:
    - coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    - if (( $(echo "$coverage < 100" | bc -l) )); then exit 1; fi
```

## Test Maintenance

### Test Review Checklist
- [ ] Test name clearly describes what is being tested
- [ ] Arrange-Act-Assert structure is clear
- [ ] No unnecessary setup
- [ ] Assertions are specific
- [ ] Error messages are helpful
- [ ] Test is independent
- [ ] Test is deterministic

### Refactoring Tests
When refactoring:
1. Run tests to ensure they pass
2. Refactor production code
3. Run tests again
4. If tests fail, fix production code (not tests)
5. Only modify tests if behavior changes

## Performance Test Strategy

### Load Test Scenarios
```go
func TestLoadNormalTraffic(t *testing.T) {
    rate := vegeta.Rate{Freq: 100, Per: time.Second}
    duration := 30 * time.Second

    attacker := vegeta.NewAttacker()
    metrics := &vegeta.Metrics{}

    for res := range attacker.Attack(targeter, rate, duration, "Normal Load") {
        metrics.Add(res)
    }

    assert.Less(t, metrics.Latencies.P95, 100*time.Millisecond)
    assert.Greater(t, metrics.Success, 0.99)
}
```

### Benchmark Tests
```go
func BenchmarkServiceRegistry_Register(b *testing.B) {
    registry := setupRegistry()
    service := testService()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        registry.Register(service)
    }
}
```

---
*Strategy Version: 1.0.0*
*Last Updated: 2025-09-20*