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

	"github.com/zakirkun/isekai/internal/cache"
	"github.com/zakirkun/isekai/internal/database"
	"github.com/zakirkun/isekai/internal/proxy"
	"github.com/zakirkun/isekai/internal/router"
	"github.com/zakirkun/isekai/pkg/config"
	"github.com/zakirkun/isekai/pkg/logger"
)

// Engine represents the core API gateway engine
type Engine struct {
	config   *config.Config
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	proxy    *proxy.Proxy
	router   *router.Router
	server   *http.Server
	wg       sync.WaitGroup
	shutdown chan os.Signal
}

// New creates a new Engine instance
func New() (*Engine, error) {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	log := logger.Get()
	log.Info("Starting Isekai API Gateway...")

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

	// Initialize router
	routerInstance := router.New(db, cacheInstance, proxyInstance, cfg, log)

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

	engine := &Engine{
		config:   cfg,
		log:      log,
		db:       db,
		cache:    cacheInstance,
		proxy:    proxyInstance,
		router:   routerInstance,
		server:   server,
		shutdown: shutdown,
	}

	return engine, nil
}

// Start starts the engine
func (e *Engine) Start() error {
	// Start server in a goroutine
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.log.Infof("Server starting on port %s", e.config.Server.Port)
		if err := e.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			e.log.Errorf("Server error: %v", err)
		}
	}()

	// Start background workers
	e.startBackgroundWorkers()

	// Wait for shutdown signal
	<-e.shutdown
	e.log.Info("Shutdown signal received, gracefully shutting down...")

	return e.Stop()
}

// Stop stops the engine gracefully
func (e *Engine) Stop() error {
	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), e.config.Server.ShutdownTimeout)
	defer cancel()

	// Shutdown HTTP server
	if err := e.server.Shutdown(ctx); err != nil {
		e.log.Errorf("Server shutdown error: %v", err)
		return err
	}

	// Cleanup resources
	e.router.Shutdown()
	e.cache.Stop()
	e.db.Close()

	// Wait for all goroutines to finish
	e.wg.Wait()

	e.log.Info("Server stopped gracefully")
	return nil
}

// startBackgroundWorkers starts background worker goroutines
func (e *Engine) startBackgroundWorkers() {
	// Example: Stats collector worker
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.statsCollector()
	}()

	// Example: Health check worker
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.healthChecker()
	}()

	e.log.Info("Background workers started")
}

// statsCollector collects and logs statistics periodically
func (e *Engine) statsCollector() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats := map[string]interface{}{
				"cache_size": e.cache.Size(),
			}
			e.log.Debugf("Stats: %v", stats)
		case <-e.shutdown:
			return
		}
	}
}

// healthChecker performs periodic health checks
func (e *Engine) healthChecker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx := context.Background()

			// Check database health
			if err := e.db.Health(ctx); err != nil {
				e.log.Warnf("Database health check failed: %v", err)
			}

			// Check cache health
			if err := e.cache.Health(ctx); err != nil {
				e.log.Warnf("Cache health check failed: %v", err)
			}
		case <-e.shutdown:
			return
		}
	}
}

// GetConfig returns the engine configuration
func (e *Engine) GetConfig() *config.Config {
	return e.config
}

// GetLogger returns the engine logger
func (e *Engine) GetLogger() *logger.Logger {
	return e.log
}
