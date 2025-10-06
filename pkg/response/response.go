package response

import (
	"encoding/json"
	"net/http"
)

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// JSON sends a JSON response
func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// Success sends a success response
func Success(w http.ResponseWriter, message string, data interface{}) {
	JSON(w, http.StatusOK, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Error sends an error response
func Error(w http.ResponseWriter, statusCode int, message string) {
	JSON(w, statusCode, Response{
		Success: false,
		Error:   message,
	})
}

// BadRequest sends a 400 Bad Request response
func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, message)
}

// Unauthorized sends a 401 Unauthorized response
func Unauthorized(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnauthorized, message)
}

// Forbidden sends a 403 Forbidden response
func Forbidden(w http.ResponseWriter, message string) {
	Error(w, http.StatusForbidden, message)
}

// NotFound sends a 404 Not Found response
func NotFound(w http.ResponseWriter, message string) {
	Error(w, http.StatusNotFound, message)
}

// InternalServerError sends a 500 Internal Server Error response
func InternalServerError(w http.ResponseWriter, message string) {
	Error(w, http.StatusInternalServerError, message)
}

// ServiceUnavailable sends a 503 Service Unavailable response
func ServiceUnavailable(w http.ResponseWriter, message string) {
	Error(w, http.StatusServiceUnavailable, message)
}
