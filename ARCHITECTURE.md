# Isekai API Gateway - Project Structure

## Overview
A high-performance API Gateway built with Go, featuring PostgreSQL, in-memory caching, and concurrent request handling using goroutines.

## Technology Stack
- **Language**: Go 1.22
- **Router**: Chi v5 - Lightweight, idiomatic router
- **Database**: PostgreSQL with pgx/v5 - High-performance PostgreSQL driver
- **Caching**: Custom in-memory cache with TTL support
- **Concurrency**: Goroutines for background workers and request handling

## Project Structure

```
isekai/
├── cmd/
│   └── gateway/
│       └── main.go                 # Application entry point
│
├── internal/                       # Private application code
│   ├── core/
│   │   └── engine.go              # Core engine with goroutine workers
│   ├── database/
│   │   ├── database.go            # PostgreSQL connection & health checks
│   │   └── repository.go          # Route repository with CRUD operations
│   ├── cache/
│   │   └── cache.go               # In-memory cache with TTL & cleanup goroutine
│   ├── router/
│   │   └── router.go              # Chi router setup & handlers
│   ├── middleware/
│   │   └── middleware.go          # Middleware (Logger, CORS, Rate Limiter, etc.)
│   └── proxy/
│       └── proxy.go               # HTTP proxy for forwarding requests
│
├── pkg/                            # Public packages (can be imported)
│   ├── config/
│   │   └── config.go              # Configuration management
│   ├── logger/
│   │   └── logger.go              # Singleton logger
│   └── response/
│       └── response.go            # HTTP response helpers
│
├── bin/                            # Compiled binaries (gitignored)
├── .env.example                    # Environment variables template
├── .gitignore                      # Git ignore rules
├── docker-compose.yml              # Docker Compose setup
├── Dockerfile                      # Docker build configuration
├── go.mod                          # Go module dependencies
├── go.sum                          # Dependency checksums
├── Makefile                        # Build automation (Linux/Mac)
├── README.md                       # Project documentation
├── setup.ps1                       # Windows setup script
└── start.ps1                       # Windows start script
```

## Core Components

### 1. Engine (internal/core/engine.go)
- **Purpose**: Main application orchestrator
- **Goroutines**:
  - HTTP server (async)
  - Stats collector (background worker)
  - Health checker (background worker)
- **Features**:
  - Graceful shutdown with WaitGroup
  - Signal handling (SIGTERM, SIGINT)
  - Resource cleanup coordination

### 2. Database (internal/database/)
- **database.go**: 
  - PostgreSQL connection pool (pgxpool)
  - Connection pooling with configurable limits
  - Health checks
  - Schema initialization
- **repository.go**:
  - Route CRUD operations
  - Context-aware queries
  - Prepared statement support

### 3. Cache (internal/cache/cache.go)
- **Features**:
  - In-memory key-value store
  - TTL-based expiration
  - Automatic cleanup goroutine
  - LRU eviction when max size reached
  - Thread-safe with RWMutex

### 4. Proxy (internal/proxy/proxy.go)
- **Features**:
  - HTTP request forwarding
  - Header preservation
  - X-Forwarded headers
  - Configurable timeouts
  - Response streaming

### 5. Middleware (internal/middleware/middleware.go)
- **Logger**: Request/response logging with timing
- **CORS**: Cross-origin resource sharing
- **Recovery**: Panic recovery
- **Rate Limiter**: 
  - Per-client IP rate limiting
  - Sliding window algorithm
  - Background cleanup goroutine
- **Timeout**: Request timeout enforcement

### 6. Router (internal/router/router.go)
- **Built on Chi**: 
  - Lightweight routing
  - Middleware chaining
  - Route groups
  - Path parameters

## Database Schema

### routes table
```sql
id              SERIAL PRIMARY KEY
path            VARCHAR(255) NOT NULL UNIQUE
target_url      VARCHAR(500) NOT NULL
method          VARCHAR(10) NOT NULL DEFAULT 'GET'
enabled         BOOLEAN NOT NULL DEFAULT true
rate_limit      INTEGER DEFAULT 0
timeout         INTEGER DEFAULT 30
created_at      TIMESTAMP NOT NULL DEFAULT NOW()
updated_at      TIMESTAMP NOT NULL DEFAULT NOW()
```

### request_logs table
```sql
id              SERIAL PRIMARY KEY
route_id        INTEGER REFERENCES routes(id)
method          VARCHAR(10) NOT NULL
path            VARCHAR(255) NOT NULL
status_code     INTEGER NOT NULL
response_time   INTEGER NOT NULL
client_ip       VARCHAR(45)
user_agent      TEXT
created_at      TIMESTAMP NOT NULL DEFAULT NOW()
```

## API Endpoints

### System Endpoints
- `GET /health` - Health check (database, cache)
- `GET /api/status` - Gateway status & metrics

### Route Management
- `GET    /api/routes` - List all routes
- `POST   /api/routes` - Create a route
- `GET    /api/routes/{id}` - Get route by ID
- `PUT    /api/routes/{id}` - Update route
- `DELETE /api/routes/{id}` - Delete route

### Proxy
- `* /*` - Proxy all other requests to configured backends

## Concurrency Model

### Goroutines Usage

1. **HTTP Server**
   - Handles incoming HTTP requests
   - Chi router manages request routing
   - Each request handled in separate goroutine (by net/http)

2. **Cache Cleanup Worker**
   - Runs every N minutes (configurable)
   - Removes expired cache entries
   - Prevents memory bloat

3. **Stats Collector Worker**
   - Collects metrics periodically
   - Logs cache size, request counts, etc.
   - Non-blocking background task

4. **Health Checker Worker**
   - Periodic health checks for database & cache
   - Logs warnings on failures
   - Helps with monitoring

5. **Rate Limiter Cleanup**
   - Removes old rate limit entries
   - Prevents memory leaks
   - Runs in background

### Synchronization
- **WaitGroup**: Coordinates graceful shutdown
- **Channels**: Signal handling for shutdown
- **Mutex/RWMutex**: Thread-safe cache & rate limiter
- **Context**: Request cancellation & timeouts

## Configuration

All configuration via environment variables (see .env.example):

### Categories
1. **Server**: Port, timeouts, header limits
2. **Database**: Connection string, pool settings
3. **Cache**: TTL, cleanup interval, max size
4. **Gateway**: Concurrent requests, timeouts, rate limits

## Getting Started

### Windows (PowerShell)
```powershell
# Setup (first time)
.\setup.ps1

# Start gateway
.\start.ps1

# Or run directly
go run cmd/gateway/main.go
```

### Linux/Mac
```bash
# Setup
make install

# Build
make build

# Run
make run

# Or
go run cmd/gateway/main.go
```

### Docker
```bash
# Start all services (PostgreSQL + Gateway)
docker-compose up -d

# View logs
docker-compose logs -f

# Stop
docker-compose down
```

## Development Workflow

1. **Make changes** to code
2. **Test** with `go test ./...`
3. **Build** with `go build` or `make build`
4. **Run** with `go run cmd/gateway/main.go`
5. **Check health**: `curl http://localhost:8080/health`

## Performance Features

✅ **Connection Pooling**: pgx connection pool for PostgreSQL  
✅ **In-Memory Caching**: Fast cache with TTL  
✅ **Goroutine Workers**: Async background tasks  
✅ **Rate Limiting**: Protect against abuse  
✅ **Graceful Shutdown**: No dropped connections  
✅ **Request Timeouts**: Prevent hanging requests  
✅ **Middleware Chain**: Efficient request processing  

## Next Steps

1. Implement full route CRUD handlers
2. Add request logging to database
3. Add authentication/authorization middleware
4. Add metrics export (Prometheus)
5. Add OpenAPI/Swagger documentation
6. Add integration tests
7. Add load balancing support
8. Add circuit breaker pattern
9. Add distributed tracing
10. Add WebSocket support

## License
MIT License
