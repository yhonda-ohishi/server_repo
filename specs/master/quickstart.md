# Quickstart: gRPC-First Multi-Protocol Gateway with db_service Integration

## Prerequisites

- Go 1.21+ installed
- Protocol buffer compiler (protoc)
- Make utility
- Docker (optional)

## Installation

```bash
# Clone the repository
git clone https://github.com/yhonda-ohishi/db-handler-server.git
cd db-handler-server

# Clone db_service repository (adjacent to server_repo)
cd ..
git clone https://github.com/yhonda-ohishi/db_service.git
cd server_repo

# Install dependencies
go mod download

# Add db_service as local module
go mod edit -replace github.com/yhonda-ohishi/db_service=../db_service

# Install code generation tools
make install-tools

# Generate code from proto files
make generate
```

## Running the Server

### Single Process Mode (Recommended for Production)

```bash
# Set environment variables
export DEPLOYMENT_MODE=single
export LOG_LEVEL=info

# Run the server with integrated db_service via bufconn
go run cmd/server/main.go

# Server will start with:
# - HTTP on port 8080
# - gRPC services via bufconn (in-memory)
# - db_service services registered directly
# - No external database required (uses mock data)
```

Server will start with:
- HTTP/REST API on port 8080
- gRPC server on port 9090 (internal)
- All services in single process using bufconn

### Separate Process Mode (Development)

```bash
# Terminal 1: Start database-repo
cd database-repo
go run cmd/server/main.go

# Terminal 2: Start handlers-repo
cd handlers-repo
go run cmd/server/main.go

# Terminal 3: Start server-repo gateway
export DEPLOYMENT_MODE=separate
export DATABASE_GRPC_URL=localhost:50051
export HANDLERS_GRPC_URL=localhost:50052
go run cmd/server/main.go
```

## Testing db_service Integration

### Test db_service via REST (auto-converted from gRPC)

```bash
# Get ETC明細 list
curl http://localhost:8080/api/v1/db/etc-meisai

# Create ETC明細 record
curl -X POST http://localhost:8080/api/v1/db/etc-meisai \
  -H "Content-Type: application/json" \
  -d '{
    "date_to": "2024-01-15",
    "ic_fr": "東京IC",
    "ic_to": "横浜IC",
    "price": 1500,
    "etc_num": "1234-5678-9012-3456"
  }'

# Get DTako Uriage Keihi list
curl http://localhost:8080/api/v1/db/dtako-uriage-keihi

# Get DTako Ferry Rows
curl http://localhost:8080/api/v1/db/dtako-ferry-rows
```

### Test db_service via gRPC (internal bufconn)

```bash
# Since db_service runs via bufconn internally,
# you can test it through the gateway's gRPC reflection:

# List db_service services
grpcurl -plaintext localhost:9090 list | grep db_service

# Get ETC明細
grpcurl -plaintext -d '{}' \
  localhost:9090 db_service.ETCMeisaiService/List
```

## Testing Each Protocol

### 1. REST API

```bash
# Create a user
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "name": "Test User",
    "phone_number": "090-1234-5678",
    "address": "Tokyo, Japan"
  }'

# Get a user
curl http://localhost:8080/api/v1/users/{user-id}

# List users
curl http://localhost:8080/api/v1/users?page_size=10

# Get transaction history
curl "http://localhost:8080/api/v1/transactions?card_id={card-id}"
```

### 2. gRPC (using grpcurl)

```bash
# Install grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# List services
grpcurl -plaintext localhost:9090 list

# Describe service
grpcurl -plaintext localhost:9090 describe etc_meisai.v1.UserService

# Call GetUser
grpcurl -plaintext -d '{"id": "user-123"}' \
  localhost:9090 etc_meisai.v1.UserService/GetUser

# Create User
grpcurl -plaintext -d '{
  "email": "test@example.com",
  "name": "Test User",
  "phone_number": "090-1234-5678",
  "address": "Tokyo, Japan"
}' localhost:9090 etc_meisai.v1.UserService/CreateUser
```

### 3. gRPC-Web (Browser)

```javascript
// Install grpc-web client
// npm install grpc-web

const {UserServiceClient} = require('./generated/user_grpc_web_pb');
const {GetUserRequest} = require('./generated/user_pb');

const client = new UserServiceClient('http://localhost:8080');

const request = new GetUserRequest();
request.setId('user-123');

client.getUser(request, {}, (err, response) => {
  if (err) {
    console.error(err);
  } else {
    console.log(response.toObject());
  }
});
```

### 4. JSON-RPC 2.0

```bash
# Call GetUser via JSON-RPC
curl -X POST http://localhost:8080/jsonrpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "UserService.GetUser",
    "params": {
      "id": "user-123"
    },
    "id": 1
  }'

# Batch request
curl -X POST http://localhost:8080/jsonrpc \
  -H "Content-Type: application/json" \
  -d '[
    {
      "jsonrpc": "2.0",
      "method": "UserService.GetUser",
      "params": {"id": "user-123"},
      "id": 1
    },
    {
      "jsonrpc": "2.0",
      "method": "UserService.GetUser",
      "params": {"id": "user-456"},
      "id": 2
    }
  ]'
```

## Swagger UI

Access the interactive API documentation:

```
http://localhost:8080/docs
```

The Swagger UI provides:
- Complete API documentation
- Interactive API testing
- Request/response schemas
- Authentication setup

## Health Checks

```bash
# Liveness probe
curl http://localhost:8080/health/live

# Readiness probe
curl http://localhost:8080/health/ready

# Detailed health status
curl http://localhost:8080/health/status
```

## Configuration

### Environment Variables

```bash
# Deployment mode
DEPLOYMENT_MODE=single|separate

# Server ports
SERVER_HTTP_PORT=8080
SERVER_GRPC_PORT=9090

# Database
DATABASE_URL=postgres://user:pass@localhost:5432/etcmeisai

# Logging
LOG_LEVEL=debug|info|warn|error

# CORS
CORS_ORIGINS=http://localhost:3000,https://app.example.com

# For separate mode
DATABASE_GRPC_URL=localhost:50051
HANDLERS_GRPC_URL=localhost:50052
```

### Configuration File (config.yaml)

```yaml
deployment:
  mode: single

server:
  http_port: 8080
  grpc_port: 9090

database:
  url: postgres://user:pass@localhost:5432/etcmeisai
  max_connections: 25
  idle_connections: 5

logging:
  level: info
  format: json

cors:
  allowed_origins:
    - http://localhost:3000
    - https://app.example.com
```

## Docker Deployment

```bash
# Build image
docker build -t etc-meisai-gateway .

# Run single mode
docker run -p 8080:8080 -p 9090:9090 \
  -e DEPLOYMENT_MODE=single \
  -e DATABASE_URL=$DATABASE_URL \
  etc-meisai-gateway

# Docker Compose
docker-compose up
```

## Testing

```bash
# Run unit tests
make test

# Run integration tests
make test-integration

# Run all tests with coverage
make test-coverage

# Test specific protocol
go test ./tests/rest/...
go test ./tests/grpc/...
go test ./tests/grpcweb/...
go test ./tests/jsonrpc/...
```

## Troubleshooting

### Common Issues

1. **Port already in use**
   ```bash
   # Check what's using the port
   lsof -i :8080
   # Kill the process or use different port
   export SERVER_HTTP_PORT=8081
   ```

2. **Database connection failed**
   - Verify DATABASE_URL is correct
   - Check database is running
   - Verify network connectivity

3. **gRPC connection refused**
   - In separate mode, verify backend services are running
   - Check GRPC_URL environment variables

4. **CORS errors in browser**
   - Add origin to CORS_ORIGINS environment variable
   - Restart the server

### Debug Mode

```bash
# Enable debug logging
export LOG_LEVEL=debug

# Enable gRPC verbose logging
export GRPC_GO_LOG_VERBOSITY_LEVEL=99
export GRPC_GO_LOG_SEVERITY_LEVEL=info

# Run with race detector
go run -race cmd/server/main.go
```

## Next Steps

1. Review the [API Documentation](./contracts/)
2. Explore the [Data Model](./data-model.md)
3. Check [Implementation Plan](./plan.md)
4. Run the test suite
5. Deploy to production

## Support

For issues or questions:
- GitHub Issues: https://github.com/yhonda-ohishi/db-handler-server/issues
- Documentation: /docs
- API Reference: /swagger