package core

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/zakirkun/isekai/internal/auth"
	"github.com/zakirkun/isekai/internal/cache"
	"github.com/zakirkun/isekai/internal/circuitbreaker"
	"github.com/zakirkun/isekai/internal/database"
	"github.com/zakirkun/isekai/internal/loadbalancer"
	"github.com/zakirkun/isekai/internal/metrics"
	"github.com/zakirkun/isekai/internal/proxy"
	"github.com/zakirkun/isekai/internal/router"
	"github.com/zakirkun/isekai/internal/tracing"
	"github.com/zakirkun/isekai/internal/websocket"
	"github.com/zakirkun/isekai/pkg/config"
	"github.com/zakirkun/isekai/pkg/logger"
)

// EngineV2 represents the enhanced API gateway engine with all features
type EngineV2 struct {
	config      *config.Config
	log         *logger.Logger
	db          *database.Database
	cache       *cache.Cache
	proxy       *proxy.Proxy
	router      *router.RouterV2
	server      *http.Server
	authService *auth.AuthService
	metrics     *metrics.Metrics
	cb          *circuitbreaker.CircuitBreaker
	lb          *loadbalancer.LoadBalancer
	tracer      *tracing.TracerProvider
	wsHub       *websocket.Hub
	wsContext   context.Context
	wsCancel    context.CancelFunc
	wg          sync.WaitGroup
	shutdown    chan os.Signal
}

// NewV2 creates a new enhanced Engine instance with all features
func NewV2() (*EngineV2, error) {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	log := logger.Get()
	log.Info("Starting Isekai API Gateway v2.0...")
	log.Infof("Features enabled: Auth=%v, Tracing=%v, RateLimit=%v",
		cfg.Auth.Enabled, cfg.Tracing.Enabled, cfg.Gateway.RateLimitEnabled)

	// Initialize database
	db, err := database.New(&cfg.Database, log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize database schema
	ctx := context.Background()
	if err := db.InitSchema(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}

	// Initialize cache
	cacheInstance := cache.New(&cfg.Cache, log)

	// Initialize proxy
	proxyInstance := proxy.New(cfg.Gateway.RequestTimeout, log)

	// Initialize metrics
	metricsInstance := metrics.New()

	// Initialize auth service
	authService := auth.NewAuthService(cfg.Auth.JWTSecret, log)

	// Initialize circuit breaker
	cb := circuitbreaker.New(log, metricsInstance)

	// Initialize load balancer
	lb := loadbalancer.New(loadbalancer.RoundRobin)
	// TODO: Load backends from database/config

	// Initialize tracing (if enabled)
	var tracer *tracing.TracerProvider
	if cfg.Tracing.Enabled {
		tracer, err = tracing.New(cfg.Tracing.ServiceName, cfg.Tracing.OTELEndpoint)
		if err != nil {
			log.Warnf("Failed to initialize tracing: %v", err)
		} else {
			log.Infof("Distributed tracing enabled - sending to OTEL collector at %s", cfg.Tracing.OTELEndpoint)
		}
	}

	// Initialize WebSocket hub
	wsContext, wsCancel := context.WithCancel(context.Background())
	wsHub := websocket.NewHub(log)

	// Initialize router
	routerInstance := router.NewV2(
		db,
		cacheInstance,
		proxyInstance,
		cfg,
		log,
		authService,
		metricsInstance,
		cb,
		lb,
		wsHub,
	)

	// Create HTTP server
	server := &http.Server{
		Addr:           ":" + cfg.Server.Port,
		Handler:        routerInstance.Handler(),
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	// Setup shutdown signal channel
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	engine := &EngineV2{
		config:      cfg,
		log:         log,
		db:          db,
		cache:       cacheInstance,
		proxy:       proxyInstance,
		router:      routerInstance,
		server:      server,
		authService: authService,
		metrics:     metricsInstance,
		cb:          cb,
		lb:          lb,
		tracer:      tracer,
		wsHub:       wsHub,
		wsContext:   wsContext,
		wsCancel:    wsCancel,
		shutdown:    shutdown,
	}

	return engine, nil
}

// Start starts the engine
func (e *EngineV2) Start() error {
	// Start server in a goroutine
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.log.Infof("🚀 Server starting on port %s", e.config.Server.Port)
		e.log.Infof("📊 Metrics available at http://localhost:%s/metrics", e.config.Server.Port)
		e.log.Infof("📚 Swagger docs at http://localhost:%s/swagger/index.html", e.config.Server.Port)
		e.log.Infof("🔌 WebSocket endpoint at ws://localhost:%s/ws", e.config.Server.Port)

		if err := e.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			e.log.Errorf("Server error: %v", err)
		}
	}()

	// Start WebSocket hub
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.wsHub.Run(e.wsContext)
	}()

	// Start background workers
	e.startBackgroundWorkers()

	// Wait for shutdown signal
	<-e.shutdown
	e.log.Info("🛑 Shutdown signal received, gracefully shutting down...")

	return e.Stop()
}

// Stop stops the engine gracefully
func (e *EngineV2) Stop() error {
	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), e.config.Server.ShutdownTimeout)
	defer cancel()

	// Shutdown HTTP server first
	if err := e.server.Shutdown(ctx); err != nil {
		e.log.Errorf("Server shutdown error: %v", err)
		return err
	}

	// Stop WebSocket hub
	e.wsCancel()

	// Cleanup router (stops accepting new requests)
	e.router.Shutdown()

	// Stop cache background workers
	e.cache.Stop()

	// Wait for all background goroutines to finish BEFORE closing database
	e.log.Info("Waiting for background workers to finish...")
	e.wg.Wait()

	// Now safe to close database
	e.db.Close()

	// Shutdown tracer if enabled
	if e.tracer != nil {
		if err := e.tracer.Shutdown(ctx); err != nil {
			e.log.Errorf("Tracer shutdown error: %v", err)
		}
	}

	e.log.Info("✅ Server stopped gracefully")
	return nil
}

// startBackgroundWorkers starts background worker goroutines
func (e *EngineV2) startBackgroundWorkers() {
	// Stats collector worker
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.statsCollector()
	}()

	// Health check worker
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.healthChecker()
	}()

	// Circuit breaker monitor
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.circuitBreakerMonitor()
	}()

	e.log.Info("✅ Background workers started")
}

// statsCollector collects and logs statistics periodically
func (e *EngineV2) statsCollector() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats := map[string]interface{}{
				"cache_size":        e.cache.Size(),
				"websocket_clients": e.wsHub.GetClientCount(),
				"backends":          len(e.lb.GetAllBackends()),
			}
			e.log.Debugf("📊 Stats: %v", stats)
		case <-e.shutdown:
			return
		}
	}
}

// healthChecker performs periodic health checks
func (e *EngineV2) healthChecker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Skip health checks during shutdown
			select {
			case <-e.shutdown:
				return
			default:
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

			// Check database health
			if err := e.db.Health(ctx); err != nil {
				e.log.Warnf("⚠️  Database health check failed: %v", err)
			}

			// Check cache health
			if err := e.cache.Health(ctx); err != nil {
				e.log.Warnf("⚠️  Cache health check failed: %v", err)
			}

			cancel()
		case <-e.shutdown:
			return
		}
	}
}

// circuitBreakerMonitor monitors circuit breaker states
func (e *EngineV2) circuitBreakerMonitor() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			states := e.cb.GetAllStates()
			for name, state := range states {
				if state.String() == "open" {
					e.log.Warnf("🔴 Circuit breaker '%s' is OPEN", name)
				}
			}
		case <-e.shutdown:
			return
		}
	}
}
