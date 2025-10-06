package database

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("isekai-database")

// Route represents a gateway route
type Route struct {
	ID        int       `json:"id"`
	Path      string    `json:"path"`
	TargetURL string    `json:"target_url"`
	Method    string    `json:"method"`
	Enabled   bool      `json:"enabled"`
	RateLimit int       `json:"rate_limit"`
	Timeout   int       `json:"timeout"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RouteRepository handles route database operations
type RouteRepository struct {
	db *Database
}

// NewRouteRepository creates a new route repository
func NewRouteRepository(db *Database) *RouteRepository {
	return &RouteRepository{db: db}
}

// FindAll retrieves all routes
func (r *RouteRepository) FindAll(ctx context.Context) ([]Route, error) {
	// Start tracing span
	ctx, span := tracer.Start(ctx, "repository.RouteRepository.FindAll")
	defer span.End()

	query := `
		SELECT id, path, target_url, method, enabled, rate_limit, timeout, created_at, updated_at
		FROM routes
		ORDER BY id
	`

	span.SetAttributes(attribute.String("db.query", "SELECT routes"))

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "query failed")
		return nil, err
	}
	defer rows.Close()

	var routes []Route
	for rows.Next() {
		var route Route
		err := rows.Scan(
			&route.ID,
			&route.Path,
			&route.TargetURL,
			&route.Method,
			&route.Enabled,
			&route.RateLimit,
			&route.Timeout,
			&route.CreatedAt,
			&route.UpdatedAt,
		)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "scan failed")
			return nil, err
		}
		routes = append(routes, route)
	}

	span.SetAttributes(attribute.Int("routes.count", len(routes)))
	span.SetStatus(codes.Ok, "success")

	return routes, nil
}

// FindByID retrieves a route by ID
func (r *RouteRepository) FindByID(ctx context.Context, id int) (*Route, error) {
	// Start tracing span
	ctx, span := tracer.Start(ctx, "repository.RouteRepository.FindByID",
		trace.WithAttributes(
			attribute.Int("route.id", id),
		),
	)
	defer span.End()

	query := `
		SELECT id, path, target_url, method, enabled, rate_limit, timeout, created_at, updated_at
		FROM routes
		WHERE id = $1
	`

	span.SetAttributes(attribute.String("db.query", "SELECT route by ID"))

	var route Route
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&route.ID,
		&route.Path,
		&route.TargetURL,
		&route.Method,
		&route.Enabled,
		&route.RateLimit,
		&route.Timeout,
		&route.CreatedAt,
		&route.UpdatedAt,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "route not found")
		return nil, err
	}

	span.SetAttributes(
		attribute.String("route.path", route.Path),
		attribute.String("route.method", route.Method),
		attribute.String("route.target_url", route.TargetURL),
	)
	span.SetStatus(codes.Ok, "route found")
	return &route, nil
}

// FindByPath retrieves a route by path and method
func (r *RouteRepository) FindByPath(ctx context.Context, path, method string) (*Route, error) {
	// Start tracing span
	ctx, span := tracer.Start(ctx, "repository.RouteRepository.FindByPath",
		trace.WithAttributes(
			attribute.String("route.path", path),
			attribute.String("route.method", method),
		),
	)
	defer span.End()

	query := `
		SELECT id, path, target_url, method, enabled, rate_limit, timeout, created_at, updated_at
		FROM routes
		WHERE path = $1 AND method = $2 AND enabled = true
	`

	span.SetAttributes(attribute.String("db.query", "SELECT route by path"))

	var route Route
	err := r.db.Pool.QueryRow(ctx, query, path, method).Scan(
		&route.ID,
		&route.Path,
		&route.TargetURL,
		&route.Method,
		&route.Enabled,
		&route.RateLimit,
		&route.Timeout,
		&route.CreatedAt,
		&route.UpdatedAt,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "route not found")
		return nil, err
	}

	span.SetAttributes(
		attribute.Int("route.id", route.ID),
		attribute.String("route.target_url", route.TargetURL),
	)
	span.SetStatus(codes.Ok, "success")

	return &route, nil
}

// Create creates a new route
func (r *RouteRepository) Create(ctx context.Context, route *Route) error {
	// Start tracing span
	ctx, span := tracer.Start(ctx, "repository.RouteRepository.Create",
		trace.WithAttributes(
			attribute.String("route.path", route.Path),
			attribute.String("route.method", route.Method),
			attribute.String("route.target_url", route.TargetURL),
		),
	)
	defer span.End()

	query := `
		INSERT INTO routes (path, target_url, method, enabled, rate_limit, timeout)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	err := r.db.Pool.QueryRow(
		ctx,
		query,
		route.Path,
		route.TargetURL,
		route.Method,
		route.Enabled,
		route.RateLimit,
		route.Timeout,
	).Scan(&route.ID, &route.CreatedAt, &route.UpdatedAt)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create route")
		return err
	}

	span.SetAttributes(attribute.Int("route.id", route.ID))
	span.SetStatus(codes.Ok, "route created")
	return nil
}

// Update updates an existing route
func (r *RouteRepository) Update(ctx context.Context, route *Route) error {
	// Start tracing span
	ctx, span := tracer.Start(ctx, "repository.RouteRepository.Update",
		trace.WithAttributes(
			attribute.Int("route.id", route.ID),
			attribute.String("route.path", route.Path),
			attribute.String("route.method", route.Method),
		),
	)
	defer span.End()

	query := `
		UPDATE routes
		SET path = $1, target_url = $2, method = $3, enabled = $4, rate_limit = $5, timeout = $6, updated_at = NOW()
		WHERE id = $7
		RETURNING updated_at
	`

	err := r.db.Pool.QueryRow(
		ctx,
		query,
		route.Path,
		route.TargetURL,
		route.Method,
		route.Enabled,
		route.RateLimit,
		route.Timeout,
		route.ID,
	).Scan(&route.UpdatedAt)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update route")
		return err
	}

	span.SetStatus(codes.Ok, "route updated")
	return nil
}

// Delete deletes a route
func (r *RouteRepository) Delete(ctx context.Context, id int) error {
	// Start tracing span
	ctx, span := tracer.Start(ctx, "repository.RouteRepository.Delete",
		trace.WithAttributes(
			attribute.Int("route.id", id),
		),
	)
	defer span.End()

	query := `DELETE FROM routes WHERE id = $1`
	cmdTag, err := r.db.Pool.Exec(ctx, query, id)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete route")
		return err
	}

	span.SetAttributes(attribute.Int64("rows_affected", cmdTag.RowsAffected()))
	span.SetStatus(codes.Ok, "route deleted")
	return nil
}

// RequestLog represents a logged request
type RequestLog struct {
	ID           int       `json:"id"`
	RouteID      *int      `json:"route_id,omitempty"` // Nullable - may not have a matching route
	Method       string    `json:"method"`
	Path         string    `json:"path"`
	StatusCode   int       `json:"status_code"`
	ResponseTime int       `json:"response_time"`
	ClientIP     string    `json:"client_ip"`
	UserAgent    string    `json:"user_agent"`
	CreatedAt    time.Time `json:"created_at"`
}

// RequestLogRepository handles request log database operations
type RequestLogRepository struct {
	db *Database
}

// NewRequestLogRepository creates a new request log repository
func NewRequestLogRepository(db *Database) *RequestLogRepository {
	return &RequestLogRepository{db: db}
}

// Create creates a new request log entry
func (r *RequestLogRepository) Create(ctx context.Context, log *RequestLog) error {
	// Start tracing span
	ctx, span := tracer.Start(ctx, "repository.RequestLogRepository.Create",
		trace.WithAttributes(
			attribute.String("log.method", log.Method),
			attribute.String("log.path", log.Path),
			attribute.Int("log.status_code", log.StatusCode),
		),
	)
	defer span.End()

	query := `
		INSERT INTO request_logs (route_id, method, path, status_code, response_time, client_ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	err := r.db.Pool.QueryRow(
		ctx,
		query,
		log.RouteID,
		log.Method,
		log.Path,
		log.StatusCode,
		log.ResponseTime,
		log.ClientIP,
		log.UserAgent,
	).Scan(&log.ID, &log.CreatedAt)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create request log")
		return err
	}

	span.SetAttributes(attribute.Int("log.id", log.ID))
	span.SetStatus(codes.Ok, "request log created")
	return nil
}

// FindByRouteID retrieves logs for a specific route
func (r *RequestLogRepository) FindByRouteID(ctx context.Context, routeID int, limit int) ([]RequestLog, error) {
	// Start tracing span
	ctx, span := tracer.Start(ctx, "repository.RequestLogRepository.FindByRouteID",
		trace.WithAttributes(
			attribute.Int("route.id", routeID),
			attribute.Int("query.limit", limit),
		),
	)
	defer span.End()

	query := `
		SELECT id, route_id, method, path, status_code, response_time, client_ip, user_agent, created_at
		FROM request_logs
		WHERE route_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Pool.Query(ctx, query, routeID, limit)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to query request logs")
		return nil, err
	}
	defer rows.Close()

	var logs []RequestLog
	for rows.Next() {
		var log RequestLog
		err := rows.Scan(
			&log.ID,
			&log.RouteID,
			&log.Method,
			&log.Path,
			&log.StatusCode,
			&log.ResponseTime,
			&log.ClientIP,
			&log.UserAgent,
			&log.CreatedAt,
		)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to scan request log")
			return nil, err
		}
		logs = append(logs, log)
	}

	span.SetAttributes(attribute.Int("logs.count", len(logs)))
	span.SetStatus(codes.Ok, "request logs retrieved")
	return logs, nil
}
