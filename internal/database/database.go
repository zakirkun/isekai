package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zakirkun/isekai/pkg/config"
	"github.com/zakirkun/isekai/pkg/logger"
)

// Database represents the database connection
type Database struct {
	Pool *pgxpool.Pool
	log  *logger.Logger
}

// New creates a new database connection
func New(cfg *config.DatabaseConfig, log *logger.Logger) (*Database, error) {
	connString := cfg.GetDSN()

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection string: %w", err)
	}

	// Set connection pool settings
	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	log.Info("Database connection established successfully")

	return &Database{
		Pool: pool,
		log:  log,
	}, nil
}

// Close closes the database connection
func (db *Database) Close() {
	if db.Pool != nil {
		db.Pool.Close()
		db.log.Info("Database connection closed")
	}
}

// Health checks the database health
func (db *Database) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return db.Pool.Ping(ctx)
}

// InitSchema initializes the database schema
func (db *Database) InitSchema(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS routes (
			id SERIAL PRIMARY KEY,
			path VARCHAR(255) NOT NULL UNIQUE,
			target_url VARCHAR(500) NOT NULL,
			method VARCHAR(10) NOT NULL DEFAULT 'GET',
			enabled BOOLEAN NOT NULL DEFAULT true,
			rate_limit INTEGER DEFAULT 0,
			timeout INTEGER DEFAULT 30,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS request_logs (
			id SERIAL PRIMARY KEY,
			route_id INTEGER REFERENCES routes(id) ON DELETE SET NULL,
			method VARCHAR(10) NOT NULL,
			path VARCHAR(255) NOT NULL,
			status_code INTEGER NOT NULL,
			response_time INTEGER NOT NULL,
			client_ip VARCHAR(45),
			user_agent TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_routes_path ON routes(path);
		CREATE INDEX IF NOT EXISTS idx_routes_enabled ON routes(enabled);
		CREATE INDEX IF NOT EXISTS idx_request_logs_created_at ON request_logs(created_at);
		CREATE INDEX IF NOT EXISTS idx_request_logs_route_id ON request_logs(route_id);
	`

	_, err := db.Pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	db.log.Info("Database schema initialized successfully")
	return nil
}
