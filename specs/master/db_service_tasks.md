# Implementation Tasks: db_service gRPC Integration via bufconn

**Feature**: db_service gRPC Integration
**Branch**: master
**Type**: Backend Service Integration
**Priority**: High - Core functionality for single mode

## Overview
Integrate db_service gRPC services directly via bufconn for in-memory communication, eliminating repository layer abstractions.

## Tasks

### Setup & Configuration

**[X] T001: Configure db_service Module Integration**
- File: `go.mod`
- Add replace directive: `replace github.com/yhonda-ohishi/db_service => ../db_service`
- Import db_service proto packages
- Run `go mod tidy` to resolve dependencies

**[X] T002: Setup bufconn Configuration**
- File: `internal/config/bufconn_config.go`
- Define buffer size constant (1MB default)
- Create configuration struct for bufconn
- Add environment variable support for buffer size

### Test Definition (TDD) [P]

**T003: Create ETCMeisaiService Contract Tests [P]**
- File: `internal/services/etc_meisai_service_test.go`
- Test Create, Get, List, Update, Delete operations
- Verify proto field mappings (IcFr, IcTo, Price, etc.)
- Test optional fields handling

**T004: Create DTakoUriageKeihiService Contract Tests [P]**
- File: `internal/services/dtako_uriage_keihi_service_test.go`
- Test with SrchId as primary key
- Test float64 price field handling
- Verify RFC3339 datetime format

**T005: Create DTakoFerryRowsService Contract Tests [P]**
- File: `internal/services/dtako_ferry_rows_service_test.go`
- Test int32 ID field
- Test Japanese field names (UnkoNo, JomuinName1)
- Verify all required fields

**T006: Create ETCMeisaiMappingService Contract Tests [P]**
- File: `internal/services/etc_meisai_mapping_service_test.go`
- Test hash-to-ID mapping
- Test GetDTakoRowIDByHash functionality
- Verify relationship integrity

**T007: Create bufconn Integration Test [P]**
- File: `tests/integration/bufconn_integration_test.go`
- Test multiple services on same bufconn listener
- Test concurrent service calls
- Verify in-memory communication

### Core Implementation

**[X] T008: Implement bufconn Client Manager**
- File: `internal/client/bufconn_client.go`
- Create bufconn listener (1024 * 1024 bytes)
- Implement GetListener() method
- Add dialer function for clients

**[X] T009: Update ServiceRegistry for db_service**
- File: `internal/services/registry.go`
- Add db_service fields to ServiceRegistry struct
- Import `dbproto "github.com/yhonda-ohishi/db_service/src/proto"`
- Add IsSingleMode boolean field

**[X] T010: Implement NewServiceRegistryForSingleMode**
- File: `internal/services/registry.go`
- Create constructor for single mode
- Initialize mock db_service implementations
- Set IsSingleMode to true

**[X] T011: Create Mock ETCMeisaiService**
- File: `internal/services/db_mock_services.go`
- Implement dbproto.UnimplementedETCMeisaiServiceServer
- Add map[int64]*dbproto.ETCMeisai for storage
- Implement Create, Get, List, Update, Delete methods

**[X] T012: Create Mock DTakoUriageKeihiService**
- File: `internal/services/db_mock_services.go`
- Implement dbproto.UnimplementedDTakoUriageKeihiServiceServer
- Use map[string]*dbproto.DTakoUriageKeihi (SrchId as key)
- Generate SrchId if not provided (SRCH%06d format)

**[X] T013: Create Mock DTakoFerryRowsService**
- File: `internal/services/db_mock_services.go`
- Implement dbproto.UnimplementedDTakoFerryRowsServiceServer
- Use map[int32]*dbproto.DTakoFerryRows
- Handle int32 ID field correctly

**[X] T014: Create Mock ETCMeisaiMappingService**
- File: `internal/services/db_mock_services.go`
- Implement dbproto.UnimplementedETCMeisaiMappingServiceServer
- Create hash-to-DTakoRowId mappings
- Implement GetDTakoRowIDByHash method

**[X] T015: Update RegisterAll for db_service**
- File: `internal/services/registry.go`
- Add db_service registration in RegisterAll method
- Use dbproto.Register*ServiceServer functions
- Only register if IsSingleMode is true

**[X] T016: Update Gateway for Single Mode**
- File: `internal/gateway/simple_gateway.go`
- Change to use NewServiceRegistryForSingleMode()
- Ensure single bufconn listener for all services
- Start gRPC server with all services

### Integration & Protocol Conversion

**[X] T017: Add db_service Health Checks**
- File: `internal/health/db_service_health.go`
- Add health check for each db_service
- Implement service availability detection
- Return status in health endpoint

**[X] T018: Configure grpc-gateway for db_service**
- File: `internal/gateway/db_service_routes.go`
- Register db_service endpoints
- Configure JSON marshaling for Japanese fields
- Setup proper error mapping

**[X] T019: Create REST Route Mappings**
- File: `internal/gateway/simple_gateway.go`
- Map /api/v1/db/etc-meisai endpoints
- Map /api/v1/db/dtako-uriage-keihi endpoints
- Map /api/v1/db/dtako-ferry-rows endpoints

**T020: Update Swagger Generation**
- File: `swagger/db_service.swagger.json`
- Generate from db_service proto files
- Merge with existing swagger specs
- Update Swagger UI configuration

### Testing & Validation

**T021: Create Single Mode E2E Test**
- File: `tests/e2e/single_mode_db_service_test.go`
- Start server in single mode
- Test all db_service endpoints
- Verify bufconn communication

**T022: Create REST API Tests for db_service [P]**
- File: `tests/e2e/db_service_rest_test.go`
- Test ETCMeisai CRUD via REST
- Test DTako services via REST
- Verify JSON response format

**T023: Create Performance Benchmarks [P]**
- File: `tests/benchmark/db_service_bench_test.go`
- Benchmark bufconn vs network calls
- Measure memory usage
- Test concurrent request handling

### Documentation & Configuration

**T024: Update Configuration Documentation**
- File: `docs/configuration.md`
- Document DEPLOYMENT_MODE settings
- Explain bufconn configuration
- Add db_service integration notes

**T025: Create db_service Integration Guide**
- File: `docs/db_service_integration.md`
- Document architecture decisions
- Explain ServiceRegistry pattern
- Add troubleshooting section

**T026: Update README**
- File: `README.md`
- Add db_service prerequisites
- Update installation instructions
- Add single mode examples

## Parallel Execution Strategy

### Phase 1: Tests (All Parallel)
```bash
Task agent T003 &
Task agent T004 &
Task agent T005 &
Task agent T006 &
Task agent T007 &
wait
```

### Phase 2: Core Implementation (Sequential for same file)
```bash
# ServiceRegistry updates (same file)
Task agent T009
Task agent T010
Task agent T015

# Mock services (same file)
Task agent T011
Task agent T012
Task agent T013
Task agent T014
```

### Phase 3: Integration (Mixed)
```bash
# Can run in parallel
Task agent T017 &
Task agent T018 &
Task agent T019 &
wait

# Gateway update
Task agent T016
```

### Phase 4: Validation (All Parallel)
```bash
Task agent T021 &
Task agent T022 &
Task agent T023 &
wait
```

## Dependencies

**Critical Path:**
1. T001-T002 (Setup) →
2. T003-T007 (Tests) →
3. T008-T014 (Core Implementation) →
4. T015-T016 (Registration) →
5. T017-T020 (Integration) →
6. T021-T023 (Validation)

**Parallel Opportunities:**
- All test creation tasks (T003-T007)
- Documentation tasks (T024-T026)
- Benchmark and REST tests (T022-T023)

## File Impact Matrix

| File | Tasks | Sequential? |
|------|-------|-------------|
| `go.mod` | T001 | No |
| `internal/services/registry.go` | T009, T010, T015 | Yes |
| `internal/services/db_mock_services.go` | T011-T014 | Yes |
| `internal/gateway/simple_gateway.go` | T016 | No |
| Test files | T003-T007, T021-T023 | No |
| Documentation | T024-T026 | No |

## Success Criteria

1. ✅ db_service services registered and accessible via bufconn
2. ✅ All mock services implement correct proto interfaces
3. ✅ REST endpoints auto-generated from gRPC
4. ✅ Single mode uses in-memory communication only
5. ✅ All tests passing with proper field mappings
6. ✅ No repository layer dependencies
7. ✅ Health checks report db_services as healthy
8. ✅ Swagger documentation includes db_service endpoints

## Technical Notes

- bufconn buffer size: 1MB (1024 * 1024 bytes)
- db_service proto package: `github.com/yhonda-ohishi/db_service/src/proto`
- Primary keys: ETCMeisai(int64), DTakoUriageKeihi(string), DTakoFerryRows(int32)
- All services use dbproto.Unimplemented*ServiceServer embedding
- Mock data stored in memory maps with mutex protection
- ServiceRegistry pattern enables clean mode switching