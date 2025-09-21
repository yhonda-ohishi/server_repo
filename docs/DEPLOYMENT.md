# Deployment Guide: gRPC-First Multi-Protocol Gateway

## Overview

This guide covers deployment strategies for the gRPC-First Multi-Protocol Gateway in various environments, from development to production.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Deployment Modes](#deployment-modes)
3. [Local Development](#local-development)
4. [Docker Deployment](#docker-deployment)
5. [Production Deployment](#production-deployment)
6. [Configuration Management](#configuration-management)
7. [Monitoring and Observability](#monitoring-and-observability)
8. [Security Considerations](#security-considerations)
9. [Troubleshooting](#troubleshooting)

## Prerequisites

### System Requirements

- **CPU**: 2+ cores recommended
- **Memory**: 512MB minimum, 2GB recommended
- **Storage**: 1GB for application and logs
- **Network**: Ports 8080 (HTTP) and 9090 (gRPC)

### Software Dependencies

- **Go 1.21+** (for building from source)
- **Docker 20.10+** (for containerized deployment)
- **docker-compose 2.0+** (for multi-service deployment)
- **Protocol Buffers compiler** (protoc)

## Deployment Modes

The gateway supports two deployment modes:

### Single Mode (Default)
- All services run in a single process
- Uses bufconn for zero-latency gRPC communication
- Ideal for development and small-scale deployments
- Simpler configuration and debugging

### Separate Mode
- Gateway connects to external gRPC services
- Supports distributed architecture
- Better for microservices environments
- Requires network configuration

## Local Development

### Quick Start

1. **Clone and build:**
```bash
git clone <repository-url>
cd db-handler-server
make build
```

2. **Run in single mode:**
```bash
./bin/server
```

3. **Verify deployment:**
```bash
# Health check
curl http://localhost:8080/health

# API endpoint
curl http://localhost:8080/api/v1/users

# JSON-RPC
curl -X POST http://localhost:8080/jsonrpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "method": "user.list", "id": 1}'
```

### Development Configuration

Create `.env` file:
```env
DEPLOYMENT_MODE=single
SERVER_HTTP_PORT=8080
SERVER_GRPC_PORT=9090
LOG_LEVEL=debug
CORS_ORIGINS=http://localhost:3000,http://localhost:8080
```

### Hot Reload Development

```bash
# Install air for hot reload
go install github.com/cosmtrek/air@latest

# Run with hot reload
air
```

## Docker Deployment

### Single Container

1. **Build image:**
```bash
docker build -t grpc-gateway:latest .
```

2. **Run container:**
```bash
docker run -d \
  --name grpc-gateway \
  -p 8080:8080 \
  -p 9090:9090 \
  -e DEPLOYMENT_MODE=single \
  -e LOG_LEVEL=info \
  grpc-gateway:latest
```

3. **Check container health:**
```bash
docker ps
docker logs grpc-gateway
```

### Multi-Service Deployment

1. **Start all services:**
```bash
docker-compose up -d
```

2. **Scale gateway instances:**
```bash
docker-compose up -d --scale gateway=3
```

3. **Monitor services:**
```bash
docker-compose ps
docker-compose logs -f gateway
```

### Docker Environment Variables

```env
# Deployment configuration
DEPLOYMENT_MODE=single|separate
SERVER_HTTP_PORT=8080
SERVER_GRPC_PORT=9090

# External service connections (separate mode)
EXTERNAL_GRPC_ADDRESS=grpc-server:9090

# Logging and monitoring
LOG_LEVEL=info|debug|warn|error
METRICS_ENABLED=true

# Security
CORS_ORIGINS=https://yourdomain.com
TLS_ENABLED=false
```

## Production Deployment

### Kubernetes Deployment

1. **Create namespace:**
```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: grpc-gateway
```

2. **Deploy application:**
```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grpc-gateway
  namespace: grpc-gateway
spec:
  replicas: 3
  selector:
    matchLabels:
      app: grpc-gateway
  template:
    metadata:
      labels:
        app: grpc-gateway
    spec:
      containers:
      - name: gateway
        image: grpc-gateway:v1.0.0
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: grpc
        env:
        - name: DEPLOYMENT_MODE
          value: "single"
        - name: LOG_LEVEL
          value: "info"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

3. **Create service:**
```yaml
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: grpc-gateway-service
  namespace: grpc-gateway
spec:
  selector:
    app: grpc-gateway
  ports:
  - name: http
    port: 80
    targetPort: 8080
  - name: grpc
    port: 9090
    targetPort: 9090
  type: LoadBalancer
```

4. **Deploy:**
```bash
kubectl apply -f namespace.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

### Load Balancer Configuration (Nginx)

```nginx
# /etc/nginx/sites-available/grpc-gateway
upstream grpc_gateway {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081;
    server 127.0.0.1:8082;
}

upstream grpc_backend {
    server 127.0.0.1:9090;
    server 127.0.0.1:9091;
    server 127.0.0.1:9092;
}

server {
    listen 80;
    server_name your-domain.com;

    # HTTP/REST/JSON-RPC endpoints
    location / {
        proxy_pass http://grpc_gateway;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Health check
    location /health {
        proxy_pass http://grpc_gateway;
        access_log off;
    }
}

# gRPC load balancing (requires nginx with grpc module)
server {
    listen 9090 http2;

    location / {
        grpc_pass grpc://grpc_backend;
        grpc_set_header Host $host;
    }
}
```

### Systemd Service (Linux)

```ini
# /etc/systemd/system/grpc-gateway.service
[Unit]
Description=gRPC-First Multi-Protocol Gateway
After=network.target

[Service]
Type=simple
User=gateway
Group=gateway
WorkingDirectory=/opt/grpc-gateway
ExecStart=/opt/grpc-gateway/bin/server
ExecReload=/bin/kill -HUP $MAINPID
Restart=always
RestartSec=5
Environment=DEPLOYMENT_MODE=single
Environment=SERVER_HTTP_PORT=8080
Environment=SERVER_GRPC_PORT=9090
Environment=LOG_LEVEL=info

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/grpc-gateway/logs

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable grpc-gateway
sudo systemctl start grpc-gateway
sudo systemctl status grpc-gateway
```

## Configuration Management

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DEPLOYMENT_MODE` | `single` | Deployment mode: `single` or `separate` |
| `SERVER_HTTP_PORT` | `8080` | HTTP server port |
| `SERVER_GRPC_PORT` | `9090` | gRPC server port |
| `LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `CORS_ORIGINS` | `*` | Allowed CORS origins |
| `METRICS_ENABLED` | `true` | Enable Prometheus metrics |
| `EXTERNAL_GRPC_ADDRESS` | - | External gRPC server address (separate mode) |

### Configuration Files

Create `config.yaml`:
```yaml
deployment:
  mode: single

server:
  http_port: 8080
  grpc_port: 9090

logging:
  level: info
  format: json

cors:
  origins:
    - "https://yourdomain.com"
    - "https://api.yourdomain.com"

metrics:
  enabled: true
  path: "/metrics"
```

### Secrets Management

Use environment variables or secret management systems:

```bash
# Kubernetes secrets
kubectl create secret generic grpc-gateway-secrets \
  --from-literal=database-url="postgres://user:pass@db:5432/dbname" \
  --from-literal=api-key="your-api-key"

# Docker secrets
echo "postgres://user:pass@db:5432/dbname" | docker secret create db_url -
```

## Monitoring and Observability

### Health Checks

The gateway provides several health endpoints:

- `/health` - Liveness probe
- `/ready` - Readiness probe
- `/metrics` - Prometheus metrics

### Prometheus Metrics

Key metrics exported:

- `grpc_gateway_requests_total` - Total number of requests
- `grpc_gateway_request_duration_seconds` - Request duration histogram
- `grpc_gateway_active_connections` - Active connections
- `grpc_gateway_errors_total` - Total number of errors

### Logging

Structured JSON logging with correlation IDs:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "msg": "Request processed",
  "request_id": "req-123",
  "method": "GET",
  "path": "/api/v1/users",
  "status": 200,
  "duration": "15ms"
}
```

### Grafana Dashboard

Import the provided dashboard for monitoring:

```bash
# Import dashboard
curl -X POST http://grafana:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @monitoring/grafana/dashboards/grpc-gateway.json
```

## Security Considerations

### Network Security

1. **Firewall rules:**
```bash
# Allow only necessary ports
ufw allow 8080/tcp
ufw allow 9090/tcp
ufw deny 22/tcp  # Disable SSH from public
```

2. **TLS/SSL:**
```bash
# Generate certificates
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365

# Configure TLS
export TLS_ENABLED=true
export TLS_CERT_FILE=/etc/ssl/certs/cert.pem
export TLS_KEY_FILE=/etc/ssl/private/key.pem
```

### Container Security

1. **Run as non-root user:**
```dockerfile
RUN adduser -D -s /bin/sh gateway
USER gateway
```

2. **Security scanning:**
```bash
# Scan image for vulnerabilities
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  -v $HOME/Library/Caches:/root/.cache/ \
  aquasec/trivy:latest image grpc-gateway:latest
```

### API Security

1. **Rate limiting:**
```nginx
# Nginx rate limiting
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
limit_req zone=api burst=20 nodelay;
```

2. **CORS configuration:**
```env
CORS_ORIGINS=https://yourdomain.com,https://app.yourdomain.com
```

## Troubleshooting

### Common Issues

#### Port Already in Use
```bash
# Check what's using the port
lsof -i :8080
netstat -tulpn | grep 8080

# Kill process
kill -9 <PID>
```

#### Connection Refused
```bash
# Check if service is running
systemctl status grpc-gateway
docker ps | grep gateway

# Check logs
journalctl -u grpc-gateway -f
docker logs grpc-gateway
```

#### High Memory Usage
```bash
# Check memory usage
docker stats grpc-gateway
kubectl top pods -n grpc-gateway

# Adjust resource limits
docker run --memory=512m grpc-gateway:latest
```

### Debug Mode

Enable debug logging:
```bash
LOG_LEVEL=debug ./bin/server
```

### Performance Profiling

```bash
# Enable pprof endpoint
export PPROF_ENABLED=true

# Get CPU profile
go tool pprof http://localhost:8080/debug/pprof/profile

# Get memory profile
go tool pprof http://localhost:8080/debug/pprof/heap
```

### Health Check Scripts

```bash
#!/bin/bash
# health-check.sh

HEALTH_URL="http://localhost:8080/health"
TIMEOUT=5

if curl -f -s --max-time $TIMEOUT $HEALTH_URL > /dev/null; then
    echo "✓ Gateway is healthy"
    exit 0
else
    echo "✗ Gateway is unhealthy"
    exit 1
fi
```

## Backup and Recovery

### Configuration Backup
```bash
# Backup configuration
tar -czf grpc-gateway-config-$(date +%Y%m%d).tar.gz \
  /opt/grpc-gateway/config/ \
  /etc/systemd/system/grpc-gateway.service
```

### Database Backup (if using external DB)
```bash
# PostgreSQL backup
pg_dump -h postgres-host -U username dbname > backup.sql

# Restore
psql -h postgres-host -U username dbname < backup.sql
```

## Scaling Considerations

### Horizontal Scaling

1. **Stateless design** - Gateway is stateless and can be scaled horizontally
2. **Load balancing** - Use load balancers to distribute traffic
3. **Service discovery** - Use tools like Consul or etcd for service discovery

### Vertical Scaling

1. **CPU** - Scale based on request processing needs
2. **Memory** - Monitor for memory leaks and adjust limits
3. **Network** - Ensure adequate network bandwidth

## Maintenance

### Updates and Rollbacks

```bash
# Rolling update (Kubernetes)
kubectl set image deployment/grpc-gateway gateway=grpc-gateway:v1.1.0

# Rollback
kubectl rollout undo deployment/grpc-gateway

# Docker update
docker pull grpc-gateway:latest
docker-compose up -d
```

### Log Rotation

```bash
# Configure logrotate
cat > /etc/logrotate.d/grpc-gateway << EOF
/opt/grpc-gateway/logs/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    postrotate
        systemctl reload grpc-gateway
    endscript
}
EOF
```

## Support

For deployment issues:

1. Check logs first: `docker logs grpc-gateway` or `journalctl -u grpc-gateway`
2. Verify configuration: Ensure all required environment variables are set
3. Test connectivity: Use curl or grpcurl to test endpoints
4. Monitor resources: Check CPU, memory, and disk usage
5. Review documentation: API.md for endpoint details

## Performance Benchmarks

Expected performance characteristics:

- **Throughput**: 10,000+ requests/second (single instance)
- **Latency**: < 10ms protocol conversion overhead
- **Memory**: < 100MB base usage
- **CPU**: Scales linearly with request volume

For production load testing:
```bash
# HTTP load test
ab -n 10000 -c 100 http://localhost:8080/api/v1/users

# gRPC load test
ghz --insecure --total=10000 --concurrency=100 \
  --call=pb.UserService.ListUsers \
  localhost:9090
```