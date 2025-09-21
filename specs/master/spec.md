# Feature Specification: gRPC-First Multi-Protocol Gateway Server

## Overview
Implement a gRPC-First architecture server that acts as a Multi-Protocol Gateway, providing automatic protocol conversion from gRPC to REST, gRPC-Web, and JSON-RPC2. The server integrates with the etc_meisai repository endpoints and provides OpenAPI/Swagger documentation.

## User Stories
1. As a backend developer, I want to define APIs once in Protocol Buffers and have them available in multiple protocols
2. As a frontend developer, I want to access the same APIs via REST, gRPC-Web, or JSON-RPC2
3. As an API consumer, I want interactive Swagger documentation for all endpoints
4. As a DevOps engineer, I want flexible deployment modes (single process or distributed)
5. As a system administrator, I want comprehensive health checks and monitoring

## Functional Requirements

### Core Features
1. **Multi-Protocol Support**
   - Native gRPC server
   - REST API via grpc-gateway automatic conversion
   - gRPC-Web support for browser clients
   - JSON-RPC2 custom implementation
   - OpenAPI/Swagger UI documentation

2. **Deployment Modes**
   - Single process mode (production): All components in one process using bufconn
   - Separate process mode (development): Each layer as independent service

3. **etc_meisai Integration**
   - Expose all endpoints from https://github.com/yhonda-ohishi/etc_meisai
   - Protocol buffer definitions for all services
   - Automatic REST endpoint generation

4. **Service Architecture**
   - database-repo: Database access layer
   - handlers-repo: Business logic layer
   - server-repo: Gateway and protocol conversion

## Non-Functional Requirements

### Performance
- In-memory gRPC communication via bufconn in single mode
- Minimal latency overhead for protocol conversion
- Connection pooling for database operations

### Reliability
- Graceful shutdown handling
- Connection retry logic
- Circuit breaker patterns

### Security
- TLS/mTLS support for gRPC
- CORS configuration for web clients
- Security headers for HTTP responses

### Observability
- Structured logging
- Health check endpoints
- Metrics preparation (future)

## Technical Constraints
- Go 1.21+ required
- Must use Fiber v2 for HTTP server
- grpc-gateway v2 for REST conversion
- Protocol Buffers v3

## Dependencies
- External: etc_meisai repository API specifications
- Internal: database-repo and handlers-repo modules
- Infrastructure: MySQL/PostgreSQL database

## Success Criteria
1. All etc_meisai endpoints accessible via all protocols
2. Swagger UI functioning with complete API documentation
3. Both deployment modes working correctly
4. Health checks passing
5. All tests passing with good coverage
6. Docker containerization complete

## Implementation Priorities
1. Protocol buffer definitions
2. Single process mode with bufconn
3. REST API via grpc-gateway
4. Swagger UI integration
5. gRPC-Web support
6. JSON-RPC2 implementation
7. Separate process mode
8. Docker and deployment