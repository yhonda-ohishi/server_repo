# Implementation Plan: gRPC-First Multi-Protocol Gateway

**Branch**: `master` | **Date**: 2025-09-21 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/master/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → If not found: ERROR "No feature spec at {path}"
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → Detect Project Type from context (web=frontend+backend, mobile=app+api)
   → Set Structure Decision based on project type
3. Fill the Constitution Check section based on the content of the constitution document.
4. Evaluate Constitution Check section below
   → If violations exist: Document in Complexity Tracking
   → If no justification possible: ERROR "Simplify approach first"
   → Update Progress Tracking: Initial Constitution Check
5. Execute Phase 0 → research.md
   → If NEEDS CLARIFICATION remain: ERROR "Resolve unknowns"
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file
7. Re-evaluate Constitution Check section
   → If new violations: Refactor design, return to Phase 1
   → Update Progress Tracking: Post-Design Constitution Check
8. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
9. STOP - Ready for /tasks command
```

## Summary
Implement a gRPC-First Multi-Protocol Gateway server that provides automatic protocol conversion from gRPC to REST, gRPC-Web, and JSON-RPC2. The server uses a layered architecture with database-repo, handlers-repo, and server-repo components, supporting both single-process (production) and distributed (development) deployment modes.

## Technical Context
**Language/Version**: Go 1.21+
**Primary Dependencies**:
- Fiber v2 (HTTP server framework)
- gRPC v1.59.0 (RPC framework)
- grpc-gateway v2.18.1 (REST conversion)
- protobuf v1.31.0 (Protocol Buffers)
- bufconn (in-memory gRPC connections)
**Storage**: MySQL/PostgreSQL via database-repo layer
**Testing**:
- Go standard testing package
- testify for assertions
- grpc testing utilities
**Target Platform**: Linux/Windows server, Docker containers
**Project Type**: single (backend multi-protocol gateway)
**Performance Goals**:
- Minimal protocol conversion overhead
- In-memory communication in single mode
- Connection pooling for databases
**Constraints**:
- Must integrate with etc_meisai endpoints
- Support both single and distributed modes
- OpenAPI documentation required
**Scale/Scope**:
- Initial: etc_meisai endpoints
- Extensible to additional services

**User-Provided Context**: db_serviceのgRPCサービスをbufconn経由で直接登録 リポジトリは不要
- db_service gRPC services registered directly via bufconn
- No repository layer needed - direct gRPC service integration
- In single mode: db_service services run in-process via bufconn
- Services: ETCMeisaiService, DTakoUriageKeihiService, DTakoFerryRowsService, ETCMeisaiMappingService

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Based on constitutional principles:
- [X] Library-First: Each repo (database, handlers, server) as standalone module
- [X] CLI Interface: Server management via command-line flags
- [X] Test-First: Protocol buffer contracts define tests
- [X] Integration Testing: Multi-protocol testing required
- [X] Observability: Health checks and structured logging
- [X] Simplicity: Start with core protocols, add features incrementally

## Project Structure

### Documentation (this feature)
```
specs/master/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (proto files)
└── tasks.md             # Phase 2 output (/tasks command)
```

### Source Code (repository root)
```
server-repo/
├── cmd/
│   └── server/
│       └── main.go                 # Entry point with mode switching
├── internal/
│   ├── gateway/
│   │   ├── gateway.go             # Multi-protocol gateway
│   │   ├── grpc_web.go           # gRPC-Web support
│   │   └── jsonrpc.go            # JSON-RPC2 implementation
│   ├── client/
│   │   ├── bufconn.go            # In-memory gRPC client
│   │   └── network.go            # Network gRPC client
│   ├── config/
│   │   └── config.go             # Configuration management
│   └── health/
│       └── health.go             # Health checks
├── proto/                         # Protocol Buffers definitions
│   └── user.proto                # From etc_meisai specs
├── swagger/                       # Auto-generated OpenAPI
│   └── user.swagger.json
├── tests/
│   ├── integration/              # Protocol conversion tests
│   └── e2e/                     # End-to-end tests
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── go.mod
```

**Structure Decision**: Single project with modular internal packages

## Phase 0: Outline & Research

1. **Extract unknowns from Technical Context**:
   - etc_meisai API endpoint specifications
   - bufconn implementation patterns
   - grpc-gateway configuration for Fiber
   - gRPC-Web implementation without Envoy
   - JSON-RPC2 protocol specification
   - Protocol buffer service definitions from etc_meisai
   - Dependency injection patterns for mode switching
   - Swagger UI integration with grpc-gateway

2. **Generate and dispatch research agents**:
   ```
   Task 1: "Analyze etc_meisai repository for API endpoints and data models"
   Task 2: "Research bufconn patterns for in-memory gRPC communication"
   Task 3: "Find grpc-gateway integration patterns with Fiber v2"
   Task 4: "Research gRPC-Web implementation without Envoy proxy"
   Task 5: "Study JSON-RPC2 specification and Go implementations"
   Task 6: "Extract Protocol Buffer definitions from etc_meisai requirements"
   Task 7: "Research dependency injection for dual-mode operation"
   Task 8: "Find Swagger UI integration patterns with grpc-gateway"
   ```

3. **Consolidate findings** in `research.md`

**Output**: research.md with all technical decisions

## Phase 1: Design & Contracts
*Prerequisites: research.md complete*

1. **Extract entities from etc_meisai** → `data-model.md`:
   - User entity
   - Transaction entity
   - ETCCard entity
   - Route entity
   - Payment entity
   - Report entity

2. **Generate Protocol Buffer contracts** → `/contracts/`:
   - user.proto (User service definitions)
   - transaction.proto (Transaction operations)
   - etc_card.proto (Card management)
   - route.proto (Route information)
   - payment.proto (Payment processing)
   - common.proto (shared types)

3. **Generate OpenAPI specifications**:
   - Auto-generated from proto files via grpc-gateway
   - Swagger annotations in proto files
   - Output to swagger/ directory

4. **Create integration test scenarios**:
   - Single mode with bufconn tests
   - Distributed mode with network tests
   - Protocol conversion tests (gRPC → REST)
   - gRPC-Web browser simulation tests
   - JSON-RPC2 request/response tests

5. **Create quickstart guide** → `quickstart.md`:
   - Single mode startup
   - Distributed mode startup
   - Testing each protocol
   - Swagger UI access

**Output**: data-model.md, /contracts/*.proto, quickstart.md

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do*

**Task Generation Strategy**:
- Protocol buffer definition tasks [P]
- Code generation tasks (protoc, grpc-gateway)
- Gateway implementation tasks
- Client implementation (bufconn + network) [P]
- Protocol handler tasks [P]
- Configuration management
- Health check implementation
- Docker configuration
- Testing tasks

**Ordering Strategy**:
1. Proto definitions and generation
2. Core gateway framework
3. Client implementations
4. Protocol handlers
5. Configuration and health
6. Testing and validation
7. Containerization

**Estimated Output**: 35-40 numbered tasks in tasks.md

## Phase 3+: Future Implementation
*Beyond /plan command scope*

**Phase 3**: Task execution via /tasks command
**Phase 4**: Implementation following tasks
**Phase 5**: Validation and deployment

## Complexity Tracking
*No violations - architecture follows modular design with clear separation*

## Progress Tracking

**Phase Status**:
- [X] Phase 0: Research complete (db_service integration added)
- [X] Phase 1: Design complete (db_service contracts added)
- [X] Phase 2: Task planning complete (bufconn integration defined)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [X] Initial Constitution Check: PASS
- [X] Post-Design Constitution Check: PASS
- [X] All NEEDS CLARIFICATION resolved
- [X] Complexity deviations documented (none)
- [X] db_service integration via bufconn specified

---
*Based on Constitution v2.1.1 and root plan.md specifications*