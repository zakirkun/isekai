package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zakirkun/isekai/internal/auth"
	"github.com/zakirkun/isekai/internal/cache"
	"github.com/zakirkun/isekai/internal/circuitbreaker"
	"github.com/zakirkun/isekai/internal/database"
	"github.com/zakirkun/isekai/internal/loadbalancer"
	"github.com/zakirkun/isekai/internal/metrics"
	"github.com/zakirkun/isekai/internal/proxy"
	"github.com/zakirkun/isekai/pkg/logger"
	"github.com/zakirkun/isekai/pkg/response"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("isekai-handlers")

// RouteHandler handles route CRUD operations
type RouteHandler struct {
	repo  *database.RouteRepository
	cache *cache.Cache
	log   *logger.Logger
}

// NewRouteHandler creates a new route handler
func NewRouteHandler(db *database.Database, cache *cache.Cache, log *logger.Logger) *RouteHandler {
	return &RouteHandler{
		repo:  database.NewRouteRepository(db),
		cache: cache,
		log:   log,
	}
}

// List handles listing all routes
// @Summary List all routes
// @Description Get a list of all configured routes
// @Tags routes
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/routes [get]
func (h *RouteHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Start tracing span
	ctx, span := tracer.Start(ctx, "handler.RouteHandler.List")
	defer span.End()

	// Try cache first
	cacheKey := "routes:all"
	if cached, found := h.cache.Get(cacheKey); found {
		span.SetAttributes(attribute.Bool("cache.hit", true))
		span.SetStatus(codes.Ok, "retrieved from cache")
		response.Success(w, "Routes retrieved from cache", cached)
		return
	}

	span.SetAttributes(attribute.Bool("cache.hit", false))

	routes, err := h.repo.FindAll(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to retrieve routes")
		h.log.Errorf("Failed to list routes: %v", err)
		response.InternalServerError(w, "Failed to retrieve routes")
		return
	}

	span.SetAttributes(attribute.Int("routes.count", len(routes)))

	// Cache the result
	h.cache.SetWithTTL(cacheKey, routes, 2*time.Minute)

	span.SetStatus(codes.Ok, "success")
	response.Success(w, "Routes retrieved", routes)
}

// Get handles getting a single route by ID
// @Summary Get route by ID
// @Description Get a specific route by its ID
// @Tags routes
// @Accept json
// @Produce json
// @Param id path int true "Route ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/routes/{id} [get]
func (h *RouteHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	// Start tracing span
	ctx, span := tracer.Start(ctx, "handler.RouteHandler.Get")
	defer span.End()

	id, err := strconv.Atoi(idStr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid route ID")
		response.BadRequest(w, "Invalid route ID")
		return
	}

	span.SetAttributes(attribute.Int("route.id", id))

	// Try cache first
	cacheKey := "route:" + idStr
	if cached, found := h.cache.Get(cacheKey); found {
		span.SetAttributes(attribute.Bool("cache.hit", true))
		span.SetStatus(codes.Ok, "route retrieved from cache")
		response.Success(w, "Route retrieved from cache", cached)
		return
	}

	span.SetAttributes(attribute.Bool("cache.hit", false))

	route, err := h.repo.FindByID(ctx, id)
	if err != nil {
		h.log.Errorf("Failed to get route %d: %v", id, err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "route not found")
		response.NotFound(w, "Route not found")
		return
	}

	// Cache the result
	h.cache.SetWithTTL(cacheKey, route, 2*time.Minute)

	span.SetStatus(codes.Ok, "route retrieved")
	response.Success(w, "Route retrieved", route)
}

// Create handles creating a new route
// @Summary Create a new route
// @Description Create a new route configuration
// @Tags routes
// @Accept json
// @Produce json
// @Param route body database.Route true "Route object"
// @Success 201 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Security BearerAuth
// @Router /api/routes [post]
func (h *RouteHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Start tracing span
	ctx, span := tracer.Start(ctx, "handler.RouteHandler.Create")
	defer span.End()

	var route database.Route
	if err := json.NewDecoder(r.Body).Decode(&route); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request body")
		response.BadRequest(w, "Invalid request body")
		return
	}

	// Validate required fields
	if route.Path == "" || route.TargetURL == "" {
		span.SetStatus(codes.Error, "missing required fields")
		response.BadRequest(w, "Path and target URL are required")
		return
	}

	span.SetAttributes(
		attribute.String("route.path", route.Path),
		attribute.String("route.method", route.Method),
		attribute.String("route.target_url", route.TargetURL),
	)

	if err := h.repo.Create(ctx, &route); err != nil {
		h.log.Errorf("Failed to create route: %v", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create route")
		response.InternalServerError(w, "Failed to create route")
		return
	}

	// Invalidate cache
	h.cache.Delete("routes:all")

	span.SetAttributes(attribute.Int("route.id", route.ID))
	span.SetStatus(codes.Ok, "route created")

	h.log.Infof("Route created: %s -> %s", route.Path, route.TargetURL)
	response.JSON(w, http.StatusCreated, response.Response{
		Success: true,
		Message: "Route created successfully",
		Data:    route,
	})
}

// Update handles updating an existing route
// @Summary Update a route
// @Description Update an existing route configuration
// @Tags routes
// @Accept json
// @Produce json
// @Param id path int true "Route ID"
// @Param route body database.Route true "Route object"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Security BearerAuth
// @Router /api/routes/{id} [put]
func (h *RouteHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	// Start tracing span
	ctx, span := tracer.Start(ctx, "handler.RouteHandler.Update")
	defer span.End()

	id, err := strconv.Atoi(idStr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid route ID")
		response.BadRequest(w, "Invalid route ID")
		return
	}

	span.SetAttributes(attribute.Int("route.id", id))

	var route database.Route
	if err := json.NewDecoder(r.Body).Decode(&route); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request body")
		response.BadRequest(w, "Invalid request body")
		return
	}

	route.ID = id

	// Validate required fields
	if route.Path == "" || route.TargetURL == "" {
		span.SetStatus(codes.Error, "missing required fields")
		response.BadRequest(w, "Path and target URL are required")
		return
	}

	span.SetAttributes(
		attribute.String("route.path", route.Path),
		attribute.String("route.method", route.Method),
		attribute.String("route.target_url", route.TargetURL),
	)

	if err := h.repo.Update(ctx, &route); err != nil {
		h.log.Errorf("Failed to update route %d: %v", id, err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update route")
		response.InternalServerError(w, "Failed to update route")
		return
	}

	// Invalidate cache
	h.cache.Delete("routes:all")
	h.cache.Delete("route:" + idStr)

	span.SetStatus(codes.Ok, "route updated")

	h.log.Infof("Route updated: %d", id)
	response.Success(w, "Route updated successfully", route)
}

// Delete handles deleting a route
// @Summary Delete a route
// @Description Delete a route by its ID
// @Tags routes
// @Accept json
// @Produce json
// @Param id path int true "Route ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Security BearerAuth
// @Router /api/routes/{id} [delete]
func (h *RouteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	// Start tracing span
	ctx, span := tracer.Start(ctx, "handler.RouteHandler.Delete")
	defer span.End()

	id, err := strconv.Atoi(idStr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid route ID")
		response.BadRequest(w, "Invalid route ID")
		return
	}

	span.SetAttributes(attribute.Int("route.id", id))

	if err := h.repo.Delete(ctx, id); err != nil {
		h.log.Errorf("Failed to delete route %d: %v", id, err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete route")
		response.InternalServerError(w, "Failed to delete route")
		return
	}

	// Invalidate cache
	h.cache.Delete("routes:all")
	h.cache.Delete("route:" + idStr)

	span.SetStatus(codes.Ok, "route deleted")

	h.log.Infof("Route deleted: %d", id)
	response.Success(w, "Route deleted successfully", nil)
}

// ProxyHandler handles proxying requests
type ProxyHandler struct {
	repo           *database.RouteRepository
	proxy          *proxy.Proxy
	cache          *cache.Cache
	cb             *circuitbreaker.CircuitBreaker
	lb             *loadbalancer.LoadBalancer
	metrics        *metrics.Metrics
	log            *logger.Logger
	requestLogRepo *database.RequestLogRepository
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(
	db *database.Database,
	proxy *proxy.Proxy,
	cache *cache.Cache,
	cb *circuitbreaker.CircuitBreaker,
	lb *loadbalancer.LoadBalancer,
	metrics *metrics.Metrics,
	log *logger.Logger,
) *ProxyHandler {
	return &ProxyHandler{
		repo:           database.NewRouteRepository(db),
		proxy:          proxy,
		cache:          cache,
		cb:             cb,
		lb:             lb,
		metrics:        metrics,
		log:            log,
		requestLogRepo: database.NewRequestLogRepository(db),
	}
}

// Handle handles proxy requests with circuit breaker and load balancing
func (h *ProxyHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startTime := time.Now()

	// Start tracing span
	ctx, span := tracer.Start(ctx, "handler.ProxyHandler.Handle",
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.path", r.URL.Path),
			attribute.String("http.client_ip", r.RemoteAddr),
		),
	)
	defer span.End()

	// Find matching route
	route, err := h.repo.FindByPath(ctx, r.URL.Path, r.Method)
	if err != nil {
		span.SetAttributes(attribute.Bool("route.found", false))
		span.SetStatus(codes.Error, "route not found")
		h.log.Debugf("No route found for %s %s", r.Method, r.URL.Path)
		response.NotFound(w, "Route not found")

		// Log failed request with no route
		h.logRequest(ctx, nil, r.Method, r.URL.Path, http.StatusNotFound, time.Since(startTime), r)
		return
	}

	span.SetAttributes(
		attribute.Bool("route.found", true),
		attribute.Int("route.id", route.ID),
		attribute.String("route.target_url", route.TargetURL),
		attribute.Bool("route.enabled", route.Enabled),
	)

	if !route.Enabled {
		response.ServiceUnavailable(w, "Route is disabled")
		routeIDPtr := &route.ID
		h.logRequest(ctx, routeIDPtr, r.Method, r.URL.Path, http.StatusServiceUnavailable, time.Since(startTime), r)
		return
	}

	// Use circuit breaker for proxying
	result, err := h.cb.Execute(route.TargetURL, func() (interface{}, error) {
		return nil, h.proxy.ForwardAndCopy(ctx, w, r, route.TargetURL)
	})

	duration := time.Since(startTime)
	statusCode := http.StatusOK

	if err != nil {
		h.log.Errorf("Proxy error for %s: %v", route.TargetURL, err)
		h.metrics.ProxyErrors.WithLabelValues(route.TargetURL, "circuit_breaker").Inc()
		response.ServiceUnavailable(w, "Service temporarily unavailable")
		statusCode = http.StatusServiceUnavailable
	}

	// Log request with route ID
	routeIDPtr := &route.ID
	h.logRequest(ctx, routeIDPtr, r.Method, r.URL.Path, statusCode, duration, r)

	_ = result
}

// logRequest logs request to database
func (h *ProxyHandler) logRequest(ctx context.Context, routeID *int, method, path string, statusCode int, duration time.Duration, r *http.Request) {
	go func() {
		logEntry := &database.RequestLog{
			RouteID:      routeID,
			Method:       method,
			Path:         path,
			StatusCode:   statusCode,
			ResponseTime: int(duration.Milliseconds()),
			ClientIP:     r.RemoteAddr,
			UserAgent:    r.UserAgent(),
		}

		if err := h.requestLogRepo.Create(context.Background(), logEntry); err != nil {
			h.log.Errorf("Failed to log request: %v", err)
		}
	}()
}

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService *auth.AuthService
	log         *logger.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *auth.AuthService, log *logger.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		log:         log,
	}
}

// Login handles user login
// @Summary User login
// @Description Authenticate user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body object true "Login credentials"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	// TODO: Validate credentials against database
	// For now, simple hardcoded check
	if credentials.Username != "admin" || credentials.Password != "password" {
		response.Unauthorized(w, "Invalid credentials")
		return
	}

	// Generate token
	token, err := h.authService.GenerateToken(
		"1",
		credentials.Username,
		[]string{"admin"},
		24*time.Hour,
	)

	if err != nil {
		h.log.Errorf("Failed to generate token: %v", err)
		response.InternalServerError(w, "Failed to generate token")
		return
	}

	response.Success(w, "Login successful", map[string]string{
		"token": token,
	})
}
