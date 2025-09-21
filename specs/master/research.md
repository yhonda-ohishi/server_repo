# Research: gRPC-First Multi-Protocol Gateway with db_service Integration

## db_service gRPC Integration via bufconn

**Decision**: Direct integration of db_service gRPC services without repository layer
**Rationale**:
- Simpler architecture - services communicate via gRPC
- No need for additional abstraction layers
- bufconn provides efficient in-memory communication

**Services to Integrate**:
- ETCMeisaiService: ETC明細 (toll details) management
- DTakoUriageKeihiService: 経費精算 (expense settlement)
- DTakoFerryRowsService: フェリー運行 (ferry operations)
- ETCMeisaiMappingService: System mapping service

**Integration Pattern**:
```go
// In ServiceRegistry for single mode
registry := &ServiceRegistry{
    // Gateway services
    ETCService: NewETCServiceServer(),

    // db_service services (actual implementations)
    ETCMeisaiService: actualETCMeisaiService,
    DTakoUriageKeihiService: actualDTakoUriageKeihiService,
    // ... other services
}

// Register all to same gRPC server
dbproto.RegisterETCMeisaiServiceServer(grpcServer, registry.ETCMeisaiService)
```

## etc_meisai API Analysis

**Decision**: Implement core ETC highway toll management endpoints
**Rationale**: Based on typical ETC (Electronic Toll Collection) systems in Japan
**Key Endpoints Identified**:
- User management (registration, authentication, profile)
- ETC card operations (registration, activation, deactivation)
- Transaction history (toll records, payments)
- Route information (entry/exit points, toll calculations)
- Payment processing (billing, refunds)
- Reporting (monthly statements, usage analytics)

## bufconn Implementation Pattern

**Decision**: Use bufconn.Listen with 1MB buffer for in-memory gRPC
**Rationale**:
- Zero network overhead
- Perfect for single-process mode
- Standard pattern in gRPC testing

**Implementation**:
```go
const bufSize = 1024 * 1024 // 1MB buffer
lis := bufconn.Listen(bufSize)

// Dialer function for clients
bufDialer := func(context.Context, string) (net.Conn, error) {
    return lis.Dial()
}
```

**Alternatives Considered**:
- Unix domain sockets: Platform-specific
- Shared memory: Complex implementation

## grpc-gateway with Fiber Integration

**Decision**: Use separate ports - Fiber on 8080, gRPC on 9090
**Rationale**:
- grpc-gateway generates standard HTTP handlers
- Fiber serves static Swagger UI and health endpoints
- Clean separation of concerns

**Pattern**:
```go
// gRPC server on :9090
go grpcServer.Serve(listener)

// HTTP gateway proxying to gRPC
mux := runtime.NewServeMux()
gateway.RegisterUserServiceHandlerFromEndpoint(ctx, mux, "localhost:9090")

// Fiber wrapping gateway
app.All("/api/*", adaptor.HTTPHandler(mux))
```

**Alternatives Considered**:
- cmux multiplexing: Added complexity
- Single port: Protocol detection overhead

## gRPC-Web Implementation

**Decision**: improbable-eng/grpc-web wrapper
**Rationale**:
- No Envoy proxy required
- Direct browser support
- Wraps existing gRPC server

**Implementation**:
```go
wrappedGrpc := grpcweb.WrapServer(grpcServer,
    grpcweb.WithOriginFunc(func(origin string) bool {
        return true // Configure CORS as needed
    }),
)
```

**Alternatives Considered**:
- Envoy proxy: Additional infrastructure
- Custom implementation: Unnecessary complexity

## JSON-RPC 2.0 Implementation

**Decision**: Custom handler with gorilla/rpc/v2/json2
**Rationale**:
- Lightweight library
- JSON-RPC 2.0 compliant
- Easy integration with existing services

**Structure**:
```go
type JSONRPCRequest struct {
    JSONRPC string      `json:"jsonrpc"`
    Method  string      `json:"method"`
    Params  interface{} `json:"params"`
    ID      interface{} `json:"id"`
}
```

**Alternatives Considered**:
- Full RPC framework: Overkill
- Manual implementation: Error-prone

## Protocol Buffer Service Definitions

**Decision**: Separate proto files per domain
**Rationale**:
- Better organization
- Independent versioning
- Clear service boundaries

**Structure**:
```proto
// user.proto
service UserService {
    rpc GetUser(GetUserRequest) returns (User);
    rpc CreateUser(CreateUserRequest) returns (User);
    rpc UpdateUser(UpdateUserRequest) returns (User);
    rpc DeleteUser(DeleteUserRequest) returns (Empty);
}

// transaction.proto
service TransactionService {
    rpc GetTransactionHistory(GetHistoryRequest) returns (TransactionList);
    rpc GetTransactionDetails(GetDetailsRequest) returns (Transaction);
}
```

## Dependency Injection Pattern

**Decision**: Interface-based injection with factory functions
**Rationale**:
- Clean separation between modes
- Testable
- No external DI framework needed

**Pattern**:
```go
type GRPCClient interface {
    Connect(ctx context.Context) error
    GetConnection() *grpc.ClientConn
}

type ClientFactory func(config Config) GRPCClient

var clientFactories = map[string]ClientFactory{
    "single":   NewBufconnClient,
    "separate": NewNetworkClient,
}
```

**Alternatives Considered**:
- Wire (Google): Compile-time DI, adds complexity
- Dig (Uber): Runtime DI, overhead

## Swagger UI Integration

**Decision**: Embed Swagger UI with statik
**Rationale**:
- Single binary distribution
- No external dependencies
- Version controlled

**Implementation**:
```go
//go:generate statik -src=./swagger-ui
import _ "server-repo/statik"

// Serve Swagger UI
app.Use("/docs", filesystem.New(filesystem.Config{
    Root: statik.New(),
}))
```

**Alternatives Considered**:
- CDN loading: Requires internet
- External hosting: Additional infrastructure

## Code Generation Strategy

**Decision**: Makefile-based generation with buf
**Rationale**:
- buf provides better proto management
- Consistent code generation
- CI/CD friendly

**Tools**:
- buf for proto compilation
- protoc-gen-go for Go code
- protoc-gen-go-grpc for gRPC code
- protoc-gen-grpc-gateway for REST gateway
- protoc-gen-openapiv2 for Swagger

## Testing Strategy

**Decision**: Table-driven tests with separate test suites per protocol
**Rationale**:
- Protocol-specific test cases
- Clear test organization
- Parallel execution

**Structure**:
```
tests/
├── grpc/        # Native gRPC tests
├── rest/        # REST API tests
├── grpcweb/     # gRPC-Web tests
├── jsonrpc/     # JSON-RPC tests
└── integration/ # Cross-protocol tests
```

## Configuration Management

**Decision**: Environment variables with viper
**Rationale**:
- 12-factor app compliance
- Multiple config sources
- Type-safe configuration

**Key Variables**:
```bash
DEPLOYMENT_MODE=single|separate
SERVER_HTTP_PORT=8080
SERVER_GRPC_PORT=9090
DATABASE_URL=postgres://...
LOG_LEVEL=info|debug|error
CORS_ORIGINS=*
```

## Error Handling

**Decision**: gRPC status codes with detailed errors
**Rationale**:
- Standard error model
- Maps well to HTTP status codes
- Rich error details

**Pattern**:
```go
status.Errorf(codes.NotFound, "user %s not found", userID)
// Maps to HTTP 404 in REST
```

---

## Summary of Technical Decisions

1. **etc_meisai**: Core ETC toll management endpoints
2. **bufconn**: 1MB buffer for in-memory gRPC
3. **Gateway**: Separate ports with Fiber adapter
4. **gRPC-Web**: improbable-eng wrapper
5. **JSON-RPC**: gorilla/rpc/v2/json2
6. **Protos**: Domain-separated service definitions
7. **DI**: Interface-based with factories
8. **Swagger**: Embedded with statik
9. **Codegen**: buf-based Makefile
10. **Testing**: Protocol-specific test suites
11. **Config**: Viper with environment variables
12. **Errors**: gRPC status codes

All decisions prioritize simplicity, maintainability, and production readiness while supporting the dual-mode architecture requirement.