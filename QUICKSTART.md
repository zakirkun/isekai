# Quick Reference Guide - Isekai API Gateway

## 🚀 Quick Start

### First Time Setup
```powershell
# 1. Install dependencies
go mod download

# 2. Copy environment file
Copy-Item .env.example .env

# 3. Update .env with your database credentials

# 4. Start PostgreSQL (Docker)
docker-compose up -d postgres

# 5. Build and run
go run cmd/gateway/main.go
```

### Using Setup Script (Recommended)
```powershell
.\setup.ps1
```

## 📋 Common Commands

### Development
```powershell
# Run directly
go run cmd/gateway/main.go

# Build binary
go build -o bin/gateway.exe cmd/gateway/main.go

# Run binary
.\bin\gateway.exe

# Install dependencies
go mod download
go mod tidy

# Format code
go fmt ./...

# Run tests
go test ./...
```

### Docker
```powershell
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f gateway

# Stop all services
docker-compose down

# Rebuild and restart
docker-compose up -d --build
```

## 🔌 API Endpoints

### Health & Status
```bash
# Health check
curl http://localhost:8080/health

# Gateway status
curl http://localhost:8080/api/status
```

### Route Management
```bash
# List all routes
curl http://localhost:8080/api/routes

# Create a route
curl -X POST http://localhost:8080/api/routes \
  -H "Content-Type: application/json" \
  -d '{
    "path": "/api/users",
    "target_url": "http://backend:3000/users",
    "method": "GET",
    "enabled": true,
    "rate_limit": 100,
    "timeout": 30
  }'

# Get route by ID
curl http://localhost:8080/api/routes/1

# Update route
curl -X PUT http://localhost:8080/api/routes/1 \
  -H "Content-Type: application/json" \
  -d '{
    "path": "/api/users",
    "target_url": "http://backend:3000/users",
    "method": "GET",
    "enabled": true,
    "rate_limit": 200,
    "timeout": 30
  }'

# Delete route
curl -X DELETE http://localhost:8080/api/routes/1
```

## 🔧 Environment Variables

### Essential Configuration
```bash
# Server
SERVER_PORT=8080

# Database (Required)
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=isekai_gateway

# Cache
CACHE_ENABLED=true
CACHE_TTL=5m

# Gateway
GATEWAY_RATE_LIMIT_ENABLED=true
GATEWAY_RATE_LIMIT_PER_SECOND=100
```

See `.env.example` for complete list.

## 📊 Project Structure

```
isekai/
├── cmd/gateway/           # Main application
├── internal/              # Private code
│   ├── core/             # Core engine
│   ├── database/         # PostgreSQL
│   ├── cache/            # Caching
│   ├── router/           # HTTP routing
│   ├── middleware/       # Middleware
│   └── proxy/            # Request proxy
└── pkg/                   # Public packages
    ├── config/           # Configuration
    ├── logger/           # Logging
    └── response/         # HTTP responses
```

## 🐛 Troubleshooting

### Database Connection Failed
```powershell
# Check PostgreSQL is running
docker ps | Select-String postgres

# Start PostgreSQL
docker-compose up -d postgres

# Check connection
psql -h localhost -U postgres -d isekai_gateway
```

### Port Already in Use
```bash
# Change port in .env
SERVER_PORT=8081

# Or set environment variable
$env:SERVER_PORT="8081"
go run cmd/gateway/main.go
```

### Build Errors
```powershell
# Clean and rebuild
Remove-Item -Recurse bin/
go clean
go mod tidy
go build -o bin/gateway.exe cmd/gateway/main.go
```

## 📈 Performance Tips

1. **Connection Pooling**: Adjust `DB_MAX_OPEN_CONNS` based on load
2. **Cache TTL**: Lower TTL for frequently changing data
3. **Rate Limiting**: Set appropriate limits per endpoint
4. **Timeouts**: Configure based on backend response times

## 🔒 Security Checklist

- [ ] Change default database password
- [ ] Enable SSL for database (`DB_SSL_MODE=require`)
- [ ] Configure CORS allowed origins (currently allows all)
- [ ] Add authentication middleware
- [ ] Enable rate limiting
- [ ] Use HTTPS in production
- [ ] Sanitize user inputs
- [ ] Add request validation

## 📚 Key Features

✅ **Golang Core**: High-performance Go runtime  
✅ **PostgreSQL**: Reliable data persistence with pgx  
✅ **Caching**: In-memory cache with TTL  
✅ **Chi Router**: Lightweight, fast routing  
✅ **Goroutines**: Concurrent background workers  
✅ **Rate Limiting**: Per-client protection  
✅ **Health Checks**: Monitoring endpoints  
✅ **Graceful Shutdown**: No dropped connections  
✅ **Docker Support**: Easy deployment  

## 🎯 Next Development Steps

1. **Complete Route Handlers**: Implement full CRUD
2. **Add Authentication**: JWT or API keys
3. **Request Logging**: Log all proxied requests
4. **Metrics**: Add Prometheus metrics
5. **Load Balancing**: Round-robin to multiple backends
6. **Circuit Breaker**: Fault tolerance
7. **API Documentation**: OpenAPI/Swagger
8. **Integration Tests**: End-to-end testing
9. **WebSocket Support**: Real-time communication
10. **Distributed Tracing**: Request tracking

## 📞 Support

- Check `README.md` for detailed documentation
- Check `ARCHITECTURE.md` for system design
- Review code comments for implementation details

## 🎉 Success Indicators

Your gateway is working correctly if:
- `http://localhost:8080/health` returns `{"success":true}`
- No errors in console logs
- Database connection shown as "healthy"
- Cache size is reported in status endpoint

Happy coding! 🚀
