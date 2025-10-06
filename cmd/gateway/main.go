package main

import (
	"os"

	"github.com/joho/godotenv"
	_ "github.com/zakirkun/isekai/docs" // swagger docs (bukan internal/docs)

	"github.com/zakirkun/isekai/internal/core"
	"github.com/zakirkun/isekai/pkg/logger"
)

// @title Isekai API Gateway
// @version 2.0
// @description A high-performance API Gateway built with Go featuring PostgreSQL, caching, circuit breaker, load balancing, distributed tracing, WebSocket, and more
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@isekai-gateway.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	log := logger.Get()

	// Create and start the enhanced engine
	engine, err := core.NewV2()
	if err != nil {
		log.Fatalf("Failed to create engine: %v", err)
	}

	// Start the engine
	if err := engine.Start(); err != nil {
		log.Errorf("Engine error: %v", err)
		os.Exit(1)
	}
}
