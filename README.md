
<div align="center">
  <img src="./isekai-logo.png" alt="Isekai API Gateway Logo" width="400"/>
  
  # Isekai API Gateway

  **A high-performance API Gateway built with Go**
  
  [![Go Version](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go)](https://golang.org)
  [![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
  [![PostgreSQL](https://img.shields.io/badge/PostgreSQL-12+-336791?logo=postgresql)](https://www.postgresql.org)
  [![Prometheus](https://img.shields.io/badge/Prometheus-Metrics-E6522C?logo=prometheus)](https://prometheus.io)
  [![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-Tracing-000000?logo=opentelemetry)](https://opentelemetry.io)
  
  *A feature-rich gateway with PostgreSQL, caching, authentication, tracing, and real-time capabilities*
</div>

---

## Features

### Core Features
- **Fast Routing**: Built on top of Chi router
- **Database**: PostgreSQL with pgx driver
- **Caching**: In-memory caching with TTL support
- **Concurrency**: Goroutine-based concurrent request handling
- **Rate Limiting**: Built-in rate limiting per client
- **Middleware**: Logger, CORS, Recovery, Timeout, and Rate Limiting
- **Proxy**: Request forwarding to backend services
- **Health Checks**: Database and cache health monitoring
- **Graceful Shutdown**: Clean shutdown with connection draining

### Advanced Features ✨
- **Full Route CRUD API**: Complete REST API for route management with cache integration
- **Request Logging**: Automatic database logging of all proxied requests with performance tracking
- **Authentication & Authorization**: JWT-based auth with Role-Based Access Control (RBAC)
- **Prometheus Metrics**: Comprehensive metrics export for monitoring (requests, latency, cache, circuit breaker states)
- **OpenAPI/Swagger**: Auto-generated interactive API documentation
- **Load Balancing**: Multiple algorithms (Round Robin, Least Connections) with health checking
- **Circuit Breaker**: Fault tolerance with automatic failure detection and recovery
- **Distributed Tracing**: OpenTelemetry integration with Jaeger for request flow visibility
- **WebSocket Support**: Full-duplex real-time communication with hub-based connection management
- **Integration Tests**: Comprehensive test suite with benchmarks and coverage reports

## Architecture

```
├── cmd/
│   └── gateway/          # Main application entry point
├── internal/
│   ├── core/             # Core engine implementation
│   ├── database/         # PostgreSQL database layer
│   ├── cache/            # In-memory caching layer
│   ├── router/           # HTTP routing and handlers
│   ├── middleware/       # HTTP middleware
│   └── proxy/            # Request proxying logic
└── pkg/
    ├── config/           # Configuration management
    ├── logger/           # Logging utilities
    └── response/         # HTTP response helpers
```

## Prerequisites

- Go 1.22 or higher
- PostgreSQL 12 or higher

## Installation

1. Clone the repository:
```bash
git clone https://github.com/zakirkun/isekai.git
cd isekai
```

2. Install dependencies:
```bash
go mod tidy
```

3. Set up environment variables (see Configuration section)

4. Run the application:
```bash
go run cmd/gateway/main.go
```

## Configuration

Configure the gateway using environment variables:

### Server Configuration
- `SERVER_PORT` - Server port (default: 8080)
- `SERVER_READ_TIMEOUT` - Read timeout (default: 15s)
- `SERVER_WRITE_TIMEOUT` - Write timeout (default: 15s)
- `SERVER_SHUTDOWN_TIMEOUT` - Graceful shutdown timeout (default: 30s)

### Database Configuration
- `DB_HOST` - PostgreSQL host (default: localhost)
- `DB_PORT` - PostgreSQL port (default: 5432)
- `DB_USER` - Database user (default: postgres)
- `DB_PASSWORD` - Database password (default: postgres)
- `DB_NAME` - Database name (default: isekai_gateway)
- `DB_SSL_MODE` - SSL mode (default: disable)
- `DB_MAX_OPEN_CONNS` - Max open connections (default: 25)
- `DB_MAX_IDLE_CONNS` - Max idle connections (default: 5)

### Cache Configuration
- `CACHE_ENABLED` - Enable caching (default: true)
- `CACHE_TTL` - Cache TTL (default: 5m)
- `CACHE_CLEANUP_INTERVAL` - Cleanup interval (default: 10m)
- `CACHE_MAX_SIZE` - Max cache entries (default: 1000)

### Gateway Configuration
- `GATEWAY_MAX_CONCURRENT_REQUESTS` - Max concurrent requests (default: 1000)
- `GATEWAY_REQUEST_TIMEOUT` - Request timeout (default: 30s)
- `GATEWAY_RATE_LIMIT_ENABLED` - Enable rate limiting (default: true)
- `GATEWAY_RATE_LIMIT_PER_SECOND` - Requests per second per client (default: 100)

### Authentication Configuration
- `AUTH_ENABLED` - Enable JWT authentication (default: false)
- `JWT_SECRET` - Secret key for JWT signing (required if auth enabled)
- `JWT_TOKEN_DURATION` - Token expiration duration (default: 24h)

### Tracing Configuration
- `TRACING_ENABLED` - Enable OpenTelemetry tracing (default: false)
- `OTLP_ENDPOINT` - OpenTelemetry collector endpoint (default: localhost:4318)
- `SERVICE_NAME` - Service name for tracing (default: isekai-gateway)

## API Endpoints

### Health & Status
```
GET /health                          # Health check endpoint
GET /api/status                      # Gateway status
```

### Authentication
```
POST /api/auth/login                 # Login and get JWT token
```

### Route Management
```
GET    /api/routes                   # List all routes
POST   /api/routes                   # Create a route (requires auth if enabled)
GET    /api/routes/{id}              # Get a route by ID
PUT    /api/routes/{id}              # Update a route (requires auth if enabled)
DELETE /api/routes/{id}              # Delete a route (requires auth if enabled)
```

### Monitoring & Observability
```
GET /metrics                         # Prometheus metrics endpoint
GET /swagger/index.html              # Swagger UI documentation
GET /swagger/doc.json                # OpenAPI JSON specification
```

### Load Balancer & Circuit Breaker
```
GET /api/load-balancer/status        # Load balancer status
GET /api/circuit-breaker/status      # Circuit breaker status
```

### WebSocket
```
WS /ws                               # WebSocket connection endpoint
GET /api/websocket/stats             # WebSocket statistics
```

## Development

### Run tests
```bash
# Run all tests
go test ./...

# Run integration tests
go test ./internal/integration/...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run benchmarks
go test -bench=. ./internal/integration/...
```

### Build
```bash
go build -o bin/gateway cmd/gateway/main.go
```

### Run with custom port
```bash
SERVER_PORT=3000 go run cmd/gateway/main.go
```

### Generate Swagger documentation
```bash
# Install swag
go install github.com/swaggo/swag/cmd/swag@latest

# Generate documentation
swag init -g cmd/gateway/main.go -o docs
```

## Tech Stack

- **Language**: Go 1.23
- **Router**: Chi v5
- **Database**: PostgreSQL with pgx/v5
- **Concurrency**: Goroutines and sync primitives
- **Caching**: In-memory with expiration
- **Authentication**: JWT with RBAC
- **Metrics**: Prometheus
- **Tracing**: OpenTelemetry + Jaeger
- **WebSocket**: Gorilla WebSocket
- **Circuit Breaker**: Sony gobreaker
- **Load Balancing**: Custom implementation (Round Robin, Least Connections)
- **API Documentation**: Swagger/OpenAPI with swaggo

## 🎯 Feature Highlights

### 🔐 Authentication & Security
- JWT-based authentication with configurable token duration
- Role-Based Access Control (RBAC) for fine-grained permissions
- Rate limiting to prevent abuse and DDoS attacks
- Request validation and sanitization

### 📊 Observability & Monitoring
- **Prometheus Metrics**: Track requests, latency, cache performance, and more
- **Distributed Tracing**: OpenTelemetry integration with Jaeger for request flow visualization
- **Request Logging**: All proxied requests logged to database with performance metrics
- **Health Checks**: Monitor database and cache connectivity

### ⚡ Performance & Reliability
- **Load Balancing**: Multiple strategies (Round Robin, Least Connections) with health checking
- **Circuit Breaker**: Automatic failure detection and recovery to prevent cascade failures
- **Caching**: In-memory caching with TTL for improved response times
- **Connection Pooling**: Efficient database connection management
- **Concurrent Processing**: Goroutine-based request handling

### 🌐 Real-Time Communication
- **WebSocket Support**: Full-duplex communication with hub-based connection management
- **Broadcast Messaging**: Send messages to all connected clients
- **Connection Tracking**: Monitor active WebSocket connections and statistics

### 📝 Developer Experience
- **Interactive API Documentation**: Swagger UI for exploring and testing APIs
- **Comprehensive Tests**: Integration tests with benchmarks and coverage reports
- **Hot Reload**: Easy development workflow
- **Docker Support**: Containerized deployment with Docker Compose

## 📚 Documentation

- [Architecture Overview](ARCHITECTURE.md) - System design and components
- [Quick Start Guide](QUICKSTART.md) - Common tasks and examples
- [Features Guide](FEATURES.md) - Complete feature documentation with examples
- [Observability Stack](OBSERVABILITY.md) - Monitoring, metrics, and tracing
- [API Documentation](http://localhost:8080/swagger) - Swagger UI (when running)

## 🚀 Quick Examples

### Creating a Route
```bash
curl -X POST http://localhost:8080/api/routes \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "path": "/api/users",
    "target_url": "http://backend:3000/users",
    "method": "GET",
    "enabled": true,
    "rate_limit": 100,
    "timeout": 30
  }'
```

### Authenticating
```bash
# Login to get JWT token
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "password"}'

# Use the token in subsequent requests
curl http://localhost:8080/api/routes \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### WebSocket Connection (JavaScript)
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = () => {
    console.log('Connected to gateway!');
    ws.send(JSON.stringify({
        type: 'message',
        payload: 'Hello from client!'
    }));
};

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    console.log('Received:', data);
};
```

### Monitoring with Prometheus
```bash
# View all available metrics
curl http://localhost:8080/metrics

# Query in Prometheus
# Example: Rate of HTTP requests
rate(isekai_http_requests_total[5m])
```

## 🔍 Observability

Run the full observability stack with OpenTelemetry, Prometheus, Jaeger, and Grafana:

```bash
docker-compose -f docker-compose.observability.yml up -d
```

Access:
- **Grafana**: http://localhost:3000 (admin/admin)
  - Pre-configured dashboards for request metrics, latency, and system health
- **Jaeger**: http://localhost:16686
  - Distributed tracing UI for request flow visualization
- **Prometheus**: http://localhost:9090
  - Metrics scraping and querying interface
- **Swagger UI**: http://localhost:8080/swagger/index.html
  - Interactive API documentation

### Available Metrics

The gateway exposes comprehensive Prometheus metrics at `/metrics`:

- `isekai_http_requests_total` - Total HTTP requests by method, path, and status
- `isekai_http_request_duration_seconds` - Request duration histogram
- `isekai_active_connections` - Current active connections
- `isekai_cache_hits_total` - Cache hit counter
- `isekai_cache_misses_total` - Cache miss counter
- `isekai_proxy_errors_total` - Proxy error counter by backend
- `isekai_database_query_duration_seconds` - Database query duration
- `isekai_circuit_breaker_state` - Circuit breaker states by backend

See [OBSERVABILITY.md](OBSERVABILITY.md) for detailed monitoring guide.

## 🏆 Performance Benchmarks

The gateway is designed for high performance:

- ⚡ **Async Request Logging**: Non-blocking database writes
- 💾 **Connection Pooling**: Configurable pool sizes for optimal resource usage
- 🚀 **In-Memory Caching**: Sub-millisecond cache lookups
- 🔄 **Load Balancing**: Efficient traffic distribution
- ⚠️ **Circuit Breaker**: Fast-fail for unhealthy backends
- 📊 **Optimized Metrics**: Low-overhead instrumentation

Run benchmarks:
```bash
go test -bench=. ./internal/integration/...
```

## 🛠️ Production Checklist

Before deploying to production, ensure:

```bash
# Security
✓ AUTH_ENABLED=true
✓ JWT_SECRET=<strong-random-secret>
✓ DB_SSL_MODE=require
✓ GATEWAY_RATE_LIMIT_ENABLED=true

# Performance
✓ DB_MAX_OPEN_CONNS=25 (adjust based on load)
✓ CACHE_ENABLED=true
✓ GATEWAY_REQUEST_TIMEOUT=30s

# Observability
✓ TRACING_ENABLED=true (if using)
✓ Prometheus scraping configured
✓ Grafana dashboards imported
✓ Jaeger collector running (if using tracing)
```

## License

MIT License
