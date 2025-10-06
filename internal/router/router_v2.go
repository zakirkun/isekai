package router

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/zakirkun/isekai/internal/auth"
	"github.com/zakirkun/isekai/internal/cache"
	"github.com/zakirkun/isekai/internal/circuitbreaker"
	"github.com/zakirkun/isekai/internal/database"
	"github.com/zakirkun/isekai/internal/handlers"
	"github.com/zakirkun/isekai/internal/loadbalancer"
	"github.com/zakirkun/isekai/internal/metrics"
	"github.com/zakirkun/isekai/internal/middleware"
	"github.com/zakirkun/isekai/internal/proxy"
	"github.com/zakirkun/isekai/internal/websocket"
	"github.com/zakirkun/isekai/pkg/config"
	"github.com/zakirkun/isekai/pkg/logger"
	"github.com/zakirkun/isekai/pkg/response"
)

// RouterV2 represents the enhanced HTTP router with all features
type RouterV2 struct {
	chi         *chi.Mux
	db          *database.Database
	cache       *cache.Cache
	proxy       *proxy.Proxy
	cfg         *config.Config
	log         *logger.Logger
	rl          *middleware.RateLimiter
	authService *auth.AuthService
	metrics     *metrics.Metrics
	cb          *circuitbreaker.CircuitBreaker
	lb          *loadbalancer.LoadBalancer
	wsHub       *websocket.Hub
}

// NewV2 creates a new enhanced router instance with all features
func NewV2(
	db *database.Database,
	cache *cache.Cache,
	proxy *proxy.Proxy,
	cfg *config.Config,
	log *logger.Logger,
	authService *auth.AuthService,
	metricsInstance *metrics.Metrics,
	cb *circuitbreaker.CircuitBreaker,
	lb *loadbalancer.LoadBalancer,
	wsHub *websocket.Hub,
) *RouterV2 {
	r := &RouterV2{
		chi:         chi.NewRouter(),
		db:          db,
		cache:       cache,
		proxy:       proxy,
		cfg:         cfg,
		log:         log,
		authService: authService,
		metrics:     metricsInstance,
		cb:          cb,
		lb:          lb,
		wsHub:       wsHub,
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
func (r *RouterV2) setupMiddleware() {
	// Recovery middleware (should be first)
	r.chi.Use(middleware.Recovery(r.log))

	// CORS middleware
	r.chi.Use(middleware.CORS())

	// Metrics middleware
	if r.metrics != nil {
		r.chi.Use(middleware.MetricsMiddleware(r.metrics))
	}

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
func (r *RouterV2) setupRoutes() {
	// Health check endpoint
	r.chi.Get("/health", r.healthHandler)

	// Metrics endpoint (Prometheus)
	r.chi.Handle("/metrics", promhttp.Handler())

	// Swagger documentation
	r.chi.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// WebSocket endpoint
	r.chi.Get("/ws", r.websocketHandler)

	// API routes
	r.chi.Route("/api", func(api chi.Router) {
		// Public endpoints
		api.Get("/status", r.statusHandler)

		// Auth endpoints
		authHandler := handlers.NewAuthHandler(r.authService, r.log)
		api.Post("/auth/login", authHandler.Login)

		// Protected route management endpoints
		api.Route("/routes", func(routes chi.Router) {
			routeHandler := handlers.NewRouteHandler(r.db, r.cache, r.log)

			// Public read endpoints
			routes.Get("/", routeHandler.List)
			routes.Get("/{id}", routeHandler.Get)

			// Protected write endpoints (require auth)
			if r.cfg.Auth.Enabled {
				routes.Group(func(protected chi.Router) {
					protected.Use(r.authService.Middleware())
					protected.Use(auth.RequireRole("admin"))

					protected.Post("/", routeHandler.Create)
					protected.Put("/{id}", routeHandler.Update)
					protected.Delete("/{id}", routeHandler.Delete)
				})
			} else {
				routes.Post("/", routeHandler.Create)
				routes.Put("/{id}", routeHandler.Update)
				routes.Delete("/{id}", routeHandler.Delete)
			}
		})

		// Circuit breaker status
		api.Get("/circuit-breaker/status", r.circuitBreakerStatus)

		// Load balancer status
		api.Get("/load-balancer/status", r.loadBalancerStatus)

		// WebSocket stats
		api.Get("/websocket/stats", r.websocketStats)
	})

	// Proxy all other requests
	proxyHandler := handlers.NewProxyHandler(r.db, r.proxy, r.cache, r.cb, r.lb, r.metrics, r.log)
	r.chi.HandleFunc("/*", proxyHandler.Handle)
}

// Handler returns the chi router
func (r *RouterV2) Handler() http.Handler {
	return r.chi
}

// Shutdown performs cleanup
func (r *RouterV2) Shutdown() {
	if r.rl != nil {
		r.rl.Stop()
	}
}

// healthHandler handles health check requests
func (r *RouterV2) healthHandler(w http.ResponseWriter, req *http.Request) {
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
func (r *RouterV2) statusHandler(w http.ResponseWriter, req *http.Request) {
	status := map[string]interface{}{
		"service": "Isekai API Gateway",
		"version": "2.0.0",
		"features": map[string]bool{
			"authentication":  r.cfg.Auth.Enabled,
			"tracing":         r.cfg.Tracing.Enabled,
			"rate_limiting":   r.cfg.Gateway.RateLimitEnabled,
			"circuit_breaker": true,
			"load_balancing":  true,
			"websocket":       true,
			"metrics":         true,
			"swagger":         true,
		},
		"cache": map[string]interface{}{
			"size": r.cache.Size(),
		},
		"websocket": map[string]interface{}{
			"connected_clients": r.wsHub.GetClientCount(),
		},
	}

	response.Success(w, "Status retrieved", status)
}

// circuitBreakerStatus returns circuit breaker status
func (r *RouterV2) circuitBreakerStatus(w http.ResponseWriter, req *http.Request) {
	states := r.cb.GetAllStates()

	stateStrings := make(map[string]string)
	for name, state := range states {
		stateStrings[name] = state.String()
	}

	response.Success(w, "Circuit breaker status", stateStrings)
}

// loadBalancerStatus returns load balancer status
func (r *RouterV2) loadBalancerStatus(w http.ResponseWriter, req *http.Request) {
	backends := r.lb.GetAllBackends()
	response.Success(w, "Load balancer status", backends)
}

// websocketStats returns WebSocket statistics
func (r *RouterV2) websocketStats(w http.ResponseWriter, req *http.Request) {
	stats := map[string]interface{}{
		"connected_clients": r.wsHub.GetClientCount(),
	}
	response.Success(w, "WebSocket stats", stats)
}

// websocketHandler handles WebSocket connections
func (r *RouterV2) websocketHandler(w http.ResponseWriter, req *http.Request) {
	// Generate client ID (you might want to use user ID from auth)
	clientID := fmt.Sprintf("client-%d", r.wsHub.GetClientCount()+1)

	websocket.ServeWS(r.wsHub, w, req, clientID)
}
