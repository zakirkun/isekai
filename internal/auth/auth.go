package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zakirkun/isekai/pkg/logger"
	"github.com/zakirkun/isekai/pkg/response"
)

var (
	ErrMissingToken  = errors.New("missing authorization token")
	ErrInvalidToken  = errors.New("invalid authorization token")
	ErrExpiredToken  = errors.New("token has expired")
	ErrInvalidClaims = errors.New("invalid token claims")
)

// Claims represents JWT claims
type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// AuthService handles authentication
type AuthService struct {
	secretKey []byte
	log       *logger.Logger
}

// NewAuthService creates a new auth service
func NewAuthService(secretKey string, log *logger.Logger) *AuthService {
	return &AuthService{
		secretKey: []byte(secretKey),
		log:       log,
	}
}

// GenerateToken generates a JWT token
func (a *AuthService) GenerateToken(userID, username string, roles []string, duration time.Duration) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.secretKey)
}

// ValidateToken validates a JWT token
func (a *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return a.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}

// Middleware provides JWT authentication middleware
func (a *AuthService) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Unauthorized(w, ErrMissingToken.Error())
				return
			}

			// Check Bearer prefix
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				response.Unauthorized(w, ErrInvalidToken.Error())
				return
			}

			// Validate token
			claims, err := a.ValidateToken(parts[1])
			if err != nil {
				response.Unauthorized(w, err.Error())
				return
			}

			// Add claims to context
			ctx := context.WithValue(r.Context(), "claims", claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole middleware checks if user has required role
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value("claims").(*Claims)
			if !ok {
				response.Forbidden(w, "Invalid authentication context")
				return
			}

			// Check if user has required role
			hasRole := false
			for _, r := range claims.Roles {
				if r == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				response.Forbidden(w, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetClaims retrieves claims from request context
func GetClaims(r *http.Request) (*Claims, error) {
	claims, ok := r.Context().Value("claims").(*Claims)
	if !ok {
		return nil, errors.New("no claims in context")
	}
	return claims, nil
}
