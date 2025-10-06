package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/zakirkun/isekai/internal/cache"
	"github.com/zakirkun/isekai/internal/database"
	"github.com/zakirkun/isekai/internal/middleware"
	"github.com/zakirkun/isekai/internal/proxy"
	"github.com/zakirkun/isekai/pkg/config"
	"github.com/zakirkun/isekai/pkg/logger"
	"github.com/zakirkun/isekai/pkg/response"
)

// Router represents the HTTP router
type Router struct {
	chi   *chi.Mux
	db    *database.Database
	cache *cache.Cache
	proxy *proxy.Proxy
	cfg   *config.Config
	log   *logger.Logger
	rl    *middleware.RateLimiter
}

// New creates a new router instance
func New(db *database.Database, cache *cache.Cache, proxy *proxy.Proxy, cfg *config.Config, log *logger.Logger) *Router {
	r := &Router{
		chi:   chi.NewRouter(),
		db:    db,
		cache: cache,
		proxy: proxy,
		cfg:   cfg,
		log:   log,
	}

	// Initialize rate limiter if enabled
	if cfg.Gateway.RateLimitEnabled {
		r.rl = middleware.NewRateLimiter(cfg.Gateway.RateLimitPerSecond, log)
	}

	r.setupMiddleware()
	r.setupRoutes()

	return r
}

// setupMiddleware sets up global middleware
func (r *Router) setupMiddleware() {
	// Recovery middleware (should be first)
	r.chi.Use(middleware.Recovery(r.log))

	// CORS middleware
	r.chi.Use(middleware.CORS())

	// Logger middleware
	r.chi.Use(middleware.Logger(r.log))

	// Rate limiting middleware
	if r.cfg.Gateway.RateLimitEnabled && r.rl != nil {
		r.chi.Use(middleware.RateLimit(r.rl))
	}

	// Timeout middleware
	r.chi.Use(middleware.Timeout(r.cfg.Gateway.RequestTimeout))
}

// setupRoutes sets up all routes
func (r *Router) setupRoutes() {
	// Health check endpoint
	r.chi.Get("/health", r.healthHandler)

	// API routes
	r.chi.Route("/api", func(api chi.Router) {
		api.Get("/status", r.statusHandler)

		// Route management endpoints
		api.Route("/routes", func(routes chi.Router) {
			routes.Get("/", r.listRoutesHandler)
			routes.Post("/", r.createRouteHandler)
			routes.Get("/{id}", r.getRouteHandler)
			routes.Put("/{id}", r.updateRouteHandler)
			routes.Delete("/{id}", r.deleteRouteHandler)
		})
	})

	// Proxy all other requests
	r.chi.HandleFunc("/*", r.proxyHandler)
}

// Handler returns the chi router
func (r *Router) Handler() http.Handler {
	return r.chi
}

// Shutdown performs cleanup
func (r *Router) Shutdown() {
	if r.rl != nil {
		r.rl.Stop()
	}
}

// healthHandler handles health check requests
func (r *Router) healthHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	health := map[string]interface{}{
		"status": "ok",
		"checks": map[string]string{},
	}

	// Check database
	if err := r.db.Health(ctx); err != nil {
		health["checks"].(map[string]string)["database"] = "unhealthy"
		health["status"] = "degraded"
	} else {
		health["checks"].(map[string]string)["database"] = "healthy"
	}

	// Check cache
	if err := r.cache.Health(ctx); err != nil {
		health["checks"].(map[string]string)["cache"] = "unhealthy"
		health["status"] = "degraded"
	} else {
		health["checks"].(map[string]string)["cache"] = "healthy"
	}

	response.Success(w, "Health check completed", health)
}

// statusHandler returns the gateway status
func (r *Router) statusHandler(w http.ResponseWriter, req *http.Request) {
	status := map[string]interface{}{
		"service": "Isekai API Gateway",
		"version": "1.0.0",
		"cache": map[string]interface{}{
			"size": r.cache.Size(),
		},
	}

	response.Success(w, "Status retrieved", status)
}

// Placeholder handlers (to be implemented with full CRUD operations)
func (r *Router) listRoutesHandler(w http.ResponseWriter, req *http.Request) {
	response.Success(w, "Routes listed", []interface{}{})
}

func (r *Router) createRouteHandler(w http.ResponseWriter, req *http.Request) {
	response.Success(w, "Route created", nil)
}

func (r *Router) getRouteHandler(w http.ResponseWriter, req *http.Request) {
	response.Success(w, "Route retrieved", nil)
}

func (r *Router) updateRouteHandler(w http.ResponseWriter, req *http.Request) {
	response.Success(w, "Route updated", nil)
}

func (r *Router) deleteRouteHandler(w http.ResponseWriter, req *http.Request) {
	response.Success(w, "Route deleted", nil)
}

func (r *Router) proxyHandler(w http.ResponseWriter, req *http.Request) {
	// This is a placeholder - will be implemented with actual routing logic
	response.NotFound(w, "Route not found")
}
