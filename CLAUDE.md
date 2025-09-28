# Claude Code Context - gRPC-First Gateway Server

## Project Overview
This is a gRPC-First Multi-Protocol Gateway server that automatically converts gRPC services to REST, gRPC-Web, and JSON-RPC2. It implements the etc_meisai_scraper (Japanese ETC toll system scraper) API endpoints.

## Architecture
- **gRPC-First**: All services defined in Protocol Buffers
- **Multi-Protocol**: Automatic conversion to REST, gRPC-Web, JSON-RPC2
- **Dual Mode**: Single process (production) or distributed (development)
- **Gateway Pattern**: server-repo acts as gateway to database-repo and handlers-repo

## Tech Stack
- Go 1.21+
- Fiber v2 (HTTP framework)
- gRPC v1.59.0 + grpc-gateway v2
- Protocol Buffers v3
- bufconn (in-memory gRPC)
- PostgreSQL/MySQL

## Key Directories
```
server-repo/
├── cmd/server/          # Entry point
├── internal/gateway/    # Protocol conversion logic
├── internal/client/     # gRPC clients (bufconn/network)
├── proto/              # Protocol buffer definitions
├── swagger/            # Auto-generated OpenAPI specs
└── tests/              # Protocol-specific tests
```

## Development Commands
```bash
# Generate code from protos
make generate

# Run in single mode (recommended)
DEPLOYMENT_MODE=single go run cmd/server/main.go

# Run tests
make test

# Access Swagger UI
open http://localhost:8080/docs
```

## Current Implementation Status
- [x] Project structure defined
- [x] Protocol buffer contracts created
- [x] Research completed for all technical decisions
- [ ] Code generation setup pending
- [ ] Gateway implementation pending
- [ ] Testing framework pending

## Important Patterns
1. **Deployment Mode Switching**: Use DEPLOYMENT_MODE env var
2. **Protocol Conversion**: grpc-gateway handles REST automatically
3. **In-Memory Communication**: bufconn for zero-latency in single mode
4. **Error Handling**: gRPC status codes map to HTTP status

## Common Tasks
- Add new service: Define in proto, run `make generate`
- Test endpoint: Use Swagger UI at /docs
- Debug gRPC: Use grpcurl tool
- Monitor health: Check /health/status endpoint

## Recent Changes
- Initial project setup based on gRPC-First architecture
- Protocol buffer definitions for User and Transaction services
- Research completed for all technical components
- Data model defined for ETC toll system

## Known Issues
- None yet (new project)

## Testing Approach
- Unit tests per protocol handler
- Integration tests for protocol conversion
- E2E tests for complete flows
- Load tests with vegeta

## Performance Targets
- Minimal protocol conversion overhead
- In-memory communication in single mode
- Connection pooling for databases
- Sub-100ms response times for cached data