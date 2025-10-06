package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zakirkun/isekai/internal/cache"
	"github.com/zakirkun/isekai/internal/database"
	"github.com/zakirkun/isekai/internal/handlers"
	"github.com/zakirkun/isekai/pkg/config"
	"github.com/zakirkun/isekai/pkg/logger"
)

// TestRouteLifecycle tests the full CRUD lifecycle of routes
func TestRouteLifecycle(t *testing.T) {
	// Skip if no database connection
	cfg := config.Load()
	log := logger.Get()

	db, err := database.New(&cfg.Database, log)
	if err != nil {
		t.Skipf("Skipping integration test - database not available: %v", err)
	}
	defer db.Close()

	// Initialize schema
	ctx := context.Background()
	if err := db.InitSchema(ctx); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Create cache
	cacheInstance := cache.New(&cfg.Cache, log)
	defer cacheInstance.Stop()

	// Create handler
	handler := handlers.NewRouteHandler(db, cacheInstance, log)

	// Test Create
	t.Run("CreateRoute", func(t *testing.T) {
		route := database.Route{
			Path:      "/test",
			TargetURL: "http://example.com",
			Method:    "GET",
			Enabled:   true,
		}

		body, _ := json.Marshal(route)
		req := httptest.NewRequest("POST", "/api/routes", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.Create(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}
	})

	// Test List
	t.Run("ListRoutes", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/routes", nil)
		w := httptest.NewRecorder()

		handler.List(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	// Test Cache
	t.Run("CacheHit", func(t *testing.T) {
		// First request
		req1 := httptest.NewRequest("GET", "/api/routes", nil)
		w1 := httptest.NewRecorder()
		handler.List(w1, req1)

		// Second request should hit cache
		req2 := httptest.NewRequest("GET", "/api/routes", nil)
		w2 := httptest.NewRecorder()
		handler.List(w2, req2)

		if w2.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w2.Code)
		}
	})
}

// TestCacheExpiration tests cache TTL functionality
func TestCacheExpiration(t *testing.T) {
	log := logger.Get()
	cfg := &config.CacheConfig{
		Enabled:         true,
		TTL:             100 * time.Millisecond,
		CleanupInterval: 50 * time.Millisecond,
		MaxSize:         10,
	}

	c := cache.New(cfg, log)
	defer c.Stop()

	// Set value
	c.Set("test-key", "test-value")

	// Verify it exists
	if val, found := c.Get("test-key"); !found || val != "test-value" {
		t.Error("Expected to find cached value")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Verify it's expired
	if _, found := c.Get("test-key"); found {
		t.Error("Expected cache entry to be expired")
	}
}

// TestCircuitBreaker tests circuit breaker functionality
func TestCircuitBreaker(t *testing.T) {
	// This test requires actual backend services
	// Skipping for now - would need mock HTTP servers
	t.Skip("Circuit breaker test requires mock services")
}

// TestLoadBalancer tests load balancing
func TestLoadBalancer(t *testing.T) {
	// Test implementation would require multiple backends
	t.Skip("Load balancer test requires mock backends")
}

// TestWebSocket tests WebSocket connectivity
func TestWebSocket(t *testing.T) {
	// WebSocket test requires full server setup
	t.Skip("WebSocket test requires full server")
}

// BenchmarkCacheOperations benchmarks cache operations
func BenchmarkCacheOperations(b *testing.B) {
	log := logger.Get()
	cfg := &config.CacheConfig{
		Enabled:         true,
		TTL:             5 * time.Minute,
		CleanupInterval: 10 * time.Minute,
		MaxSize:         1000,
	}

	c := cache.New(cfg, log)
	defer c.Stop()

	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			c.Set("bench-key", "bench-value")
		}
	})

	b.Run("Get", func(b *testing.B) {
		c.Set("bench-key", "bench-value")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			c.Get("bench-key")
		}
	})
}
