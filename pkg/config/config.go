package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Cache    CacheConfig
	Gateway  GatewayConfig
	Auth     AuthConfig
	Tracing  TracingConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	MaxHeaderBytes  int
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// CacheConfig holds cache-related configuration
type CacheConfig struct {
	Enabled         bool
	TTL             time.Duration
	CleanupInterval time.Duration
	MaxSize         int64
}

// GatewayConfig holds gateway-specific configuration
type GatewayConfig struct {
	MaxConcurrentRequests int
	RequestTimeout        time.Duration
	RateLimitEnabled      bool
	RateLimitPerSecond    int
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret     string
	TokenDuration time.Duration
	Enabled       bool
}

// TracingConfig holds tracing configuration
type TracingConfig struct {
	Enabled      bool
	OTELEndpoint string
	ServiceName  string
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            getEnv("SERVER_PORT", "8080"),
			ReadTimeout:     getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
			ShutdownTimeout: getDurationEnv("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
			MaxHeaderBytes:  getIntEnv("SERVER_MAX_HEADER_BYTES", 1<<20),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			DBName:          getEnv("DB_NAME", "isekai_gateway"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Cache: CacheConfig{
			Enabled:         getBoolEnv("CACHE_ENABLED", true),
			TTL:             getDurationEnv("CACHE_TTL", 5*time.Minute),
			CleanupInterval: getDurationEnv("CACHE_CLEANUP_INTERVAL", 10*time.Minute),
			MaxSize:         getInt64Env("CACHE_MAX_SIZE", 1000),
		},
		Gateway: GatewayConfig{
			MaxConcurrentRequests: getIntEnv("GATEWAY_MAX_CONCURRENT_REQUESTS", 1000),
			RequestTimeout:        getDurationEnv("GATEWAY_REQUEST_TIMEOUT", 30*time.Second),
			RateLimitEnabled:      getBoolEnv("GATEWAY_RATE_LIMIT_ENABLED", true),
			RateLimitPerSecond:    getIntEnv("GATEWAY_RATE_LIMIT_PER_SECOND", 100),
		},
		Auth: AuthConfig{
			JWTSecret:     getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			TokenDuration: getDurationEnv("JWT_TOKEN_DURATION", 24*time.Hour),
			Enabled:       getBoolEnv("AUTH_ENABLED", false),
		},
		Tracing: TracingConfig{
			Enabled:      getBoolEnv("TRACING_ENABLED", false),
			OTELEndpoint: getEnv("OTEL_ENDPOINT", "localhost:4318"),
			ServiceName:  getEnv("SERVICE_NAME", "isekai-gateway"),
		},
	}
}

// GetDSN returns the PostgreSQL connection string
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
