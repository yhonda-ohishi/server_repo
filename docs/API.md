# gRPC-First Multi-Protocol Gateway API Documentation

## Overview

The gRPC-First Multi-Protocol Gateway provides a unified interface to access ETC toll system services through multiple protocols:

- **REST API** - HTTP/JSON endpoints for web applications
- **gRPC** - High-performance binary protocol for microservices
- **JSON-RPC 2.0** - Simple RPC protocol for lightweight clients
- **gRPC-Web** - gRPC for browser applications (future implementation)

## Base URLs

### Single Mode (Default)
- REST API: `http://localhost:8080/api/v1`
- gRPC: `localhost:9090`
- JSON-RPC: `http://localhost:8080/jsonrpc`
- Health: `http://localhost:8080/health`
- Swagger UI: `http://localhost:8080/docs`

### Separate Mode
- REST API: `http://localhost:8080/api/v1`
- JSON-RPC: `http://localhost:8080/jsonrpc`
- gRPC: External server (configurable)

## Authentication

Currently, the API does not implement authentication. This is intended for development and testing purposes.

## User Service API

### REST Endpoints

#### List Users
```http
GET /api/v1/users?page_size=10&page_token=string
```

**Response:**
```json
{
  "users": [
    {
      "id": "user-123",
      "email": "user@example.com",
      "name": "John Doe",
      "phone_number": "090-1234-5678",
      "address": "Tokyo, Japan",
      "status": "active",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ],
  "next_page_token": "next_token_here"
}
```

#### Get User
```http
GET /api/v1/users/{id}
```

**Response:**
```json
{
  "id": "user-123",
  "email": "user@example.com",
  "name": "John Doe",
  "phone_number": "090-1234-5678",
  "address": "Tokyo, Japan",
  "status": "active",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

#### Create User
```http
POST /api/v1/users
Content-Type: application/json

{
  "email": "newuser@example.com",
  "name": "New User",
  "phone_number": "090-9999-8888",
  "address": "Osaka, Japan"
}
```

**Response:**
```json
{
  "id": "user-new",
  "email": "newuser@example.com",
  "name": "New User",
  "phone_number": "090-9999-8888",
  "address": "Osaka, Japan",
  "status": "active",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

#### Update User
```http
PUT /api/v1/users/{id}
Content-Type: application/json

{
  "email": "updated@example.com",
  "name": "Updated Name",
  "phone_number": "090-7777-6666",
  "address": "Kyoto, Japan"
}
```

#### Delete User
```http
DELETE /api/v1/users/{id}
```

### gRPC Service

```protobuf
service UserService {
  rpc CreateUser(CreateUserRequest) returns (User);
  rpc GetUser(GetUserRequest) returns (User);
  rpc UpdateUser(UpdateUserRequest) returns (User);
  rpc DeleteUser(DeleteUserRequest) returns (google.protobuf.Empty);
  rpc ListUsers(ListUsersRequest) returns (UserList);
}
```

### JSON-RPC Methods

#### user.get
```json
{
  "jsonrpc": "2.0",
  "method": "user.get",
  "params": {"id": "user-123"},
  "id": 1
}
```

#### user.create
```json
{
  "jsonrpc": "2.0",
  "method": "user.create",
  "params": {
    "email": "jsonrpc@example.com",
    "name": "JSON-RPC User",
    "phone_number": "090-5555-5555",
    "address": "JSON City"
  },
  "id": 2
}
```

#### user.list
```json
{
  "jsonrpc": "2.0",
  "method": "user.list",
  "params": {},
  "id": 3
}
```

## Transaction Service API

### REST Endpoints

#### Get Transaction
```http
GET /api/v1/transactions/{id}
```

**Response:**
```json
{
  "id": "txn-123",
  "card_id": "card-1",
  "entry_gate_id": "gate-001",
  "exit_gate_id": "gate-002",
  "entry_time": "2024-01-15T08:30:00Z",
  "exit_time": "2024-01-15T09:15:00Z",
  "distance": 45.5,
  "toll_amount": 1200,
  "discount_amount": 100,
  "final_amount": 1100,
  "payment_status": "completed",
  "transaction_date": "2024-01-15T09:15:00Z"
}
```

#### Get Transaction History
```http
GET /api/v1/transactions?card_id=card-1&start_date=2024-01-01T00:00:00Z&end_date=2024-01-31T23:59:59Z&page_size=10
```

**Response:**
```json
{
  "transactions": [
    {
      "id": "txn-1",
      "card_id": "card-1",
      "entry_gate_id": "gate-001",
      "exit_gate_id": "gate-002",
      "distance": 45.5,
      "toll_amount": 1200,
      "final_amount": 1100,
      "payment_status": "completed"
    }
  ],
  "next_page_token": "",
  "total_amount": 1100
}
```

### gRPC Service

```protobuf
service TransactionService {
  rpc GetTransaction(GetTransactionRequest) returns (Transaction);
  rpc GetTransactionHistory(GetTransactionHistoryRequest) returns (TransactionList);
}
```

### JSON-RPC Methods

#### transaction.get
```json
{
  "jsonrpc": "2.0",
  "method": "transaction.get",
  "params": {"id": "txn-123"},
  "id": 1
}
```

#### transaction.history
```json
{
  "jsonrpc": "2.0",
  "method": "transaction.history",
  "params": {"card_id": "card-1"},
  "id": 2
}
```

## ETC Card Service API

### REST Endpoints

#### List Cards
```http
GET /api/v1/cards?user_id=user-123
```

#### Get Card
```http
GET /api/v1/cards/{id}
```

#### Create Card
```http
POST /api/v1/cards
Content-Type: application/json

{
  "user_id": "user-123",
  "card_number": "1234-5678-9012-3456",
  "card_type": "personal"
}
```

## Payment Service API

### REST Endpoints

#### Process Payment
```http
POST /api/v1/payments
Content-Type: application/json

{
  "transaction_id": "txn-123",
  "amount": 1100,
  "payment_method": "credit_card"
}
```

#### Get Payment Status
```http
GET /api/v1/payments/{payment_id}
```

## Health and Monitoring

### Health Check
```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "checks": {
    "database": "ok",
    "redis": "ok",
    "grpc_server": "ok"
  }
}
```

### Readiness Check
```http
GET /ready
```

### Metrics
```http
GET /metrics
```

Returns Prometheus-format metrics.

## Error Handling

### HTTP Status Codes

- `200 OK` - Successful operation
- `201 Created` - Resource created successfully
- `204 No Content` - Successful deletion
- `400 Bad Request` - Invalid request data
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error

### gRPC Status Codes

- `OK` - Successful operation
- `INVALID_ARGUMENT` - Invalid request parameters
- `NOT_FOUND` - Resource not found
- `INTERNAL` - Internal server error

### JSON-RPC Error Codes

- `-32700` - Parse error
- `-32600` - Invalid Request
- `-32601` - Method not found
- `-32602` - Invalid params
- `-32603` - Internal error
- `-32000` - Application-specific error (e.g., resource not found)

### Error Response Format

#### REST API
```json
{
  "error": "User not found"
}
```

#### JSON-RPC
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32000,
    "message": "User not found",
    "data": "Additional error details"
  },
  "id": 1
}
```

## Rate Limiting

Currently not implemented. Consider implementing rate limiting for production use.

## CORS

CORS is enabled for all origins in development mode. Configure appropriately for production.

## Examples

### cURL Examples

#### Create a user via REST
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "name": "Test User",
    "phone_number": "090-1234-5678",
    "address": "Tokyo, Japan"
  }'
```

#### Get transaction via JSON-RPC
```bash
curl -X POST http://localhost:8080/jsonrpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "transaction.get",
    "params": {"id": "txn-1"},
    "id": 1
  }'
```

### gRPC Client Examples

#### Go Client
```go
conn, err := grpc.Dial("localhost:9090", grpc.WithInsecure())
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewUserServiceClient(conn)
user, err := client.GetUser(context.Background(), &pb.GetUserRequest{
    Id: "user-123",
})
```

#### Python Client
```python
import grpc
import user_pb2
import user_pb2_grpc

channel = grpc.insecure_channel('localhost:9090')
stub = user_pb2_grpc.UserServiceStub(channel)

response = stub.GetUser(user_pb2.GetUserRequest(id='user-123'))
```

## Protocol Buffers Schema

Full protocol buffer definitions are available in the `/swagger` endpoint when the server is running, or in the `proto/` directory of the source code.

## Performance Considerations

- **Protocol Overhead**: gRPC < REST < JSON-RPC for performance
- **Payload Size**: gRPC uses binary encoding, REST/JSON-RPC use JSON
- **Connection Reuse**: gRPC maintains persistent connections
- **Caching**: Consider implementing response caching for read-heavy operations

## Development and Testing

### Running the Server
```bash
# Single mode (default)
./bin/server

# With environment variables
DEPLOYMENT_MODE=single SERVER_HTTP_PORT=8080 ./bin/server
```

### Testing Endpoints
```bash
# Health check
curl http://localhost:8080/health

# API endpoints
curl http://localhost:8080/api/v1/users

# JSON-RPC
curl -X POST http://localhost:8080/jsonrpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "method": "user.list", "id": 1}'
```

### Interactive Documentation

Visit `http://localhost:8080/docs` for interactive Swagger UI documentation.

## Protocol Compatibility

The gateway ensures that data returned by all protocols (REST, gRPC, JSON-RPC) is consistent and follows the same schema, enabling seamless switching between protocols for different use cases.