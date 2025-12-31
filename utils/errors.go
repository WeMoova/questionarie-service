package utils

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// SuccessResponse represents a standard success response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// RespondWithError sends an error response
func RespondWithError(w http.ResponseWriter, code int, message string) {
	RespondWithJSON(w, code, ErrorResponse{
		Error:   http.StatusText(code),
		Message: message,
		Code:    code,
	})
}

// RespondWithJSON sends a JSON response
func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// RespondWithSuccess sends a success response with data
func RespondWithSuccess(w http.ResponseWriter, code int, data interface{}, message string) {
	RespondWithJSON(w, code, SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// Common error responses
var (
	// BadRequest returns a 400 error
	BadRequest = func(w http.ResponseWriter, message string) {
		RespondWithError(w, http.StatusBadRequest, message)
	}

	// Unauthorized returns a 401 error
	Unauthorized = func(w http.ResponseWriter, message string) {
		RespondWithError(w, http.StatusUnauthorized, message)
	}

	// Forbidden returns a 403 error
	Forbidden = func(w http.ResponseWriter, message string) {
		RespondWithError(w, http.StatusForbidden, message)
	}

	// NotFound returns a 404 error
	NotFound = func(w http.ResponseWriter, message string) {
		RespondWithError(w, http.StatusNotFound, message)
	}

	// Conflict returns a 409 error
	Conflict = func(w http.ResponseWriter, message string) {
		RespondWithError(w, http.StatusConflict, message)
	}

	// InternalServerError returns a 500 error
	InternalServerError = func(w http.ResponseWriter, message string) {
		RespondWithError(w, http.StatusInternalServerError, message)
	}

	// ValidationError returns a 422 error
	ValidationError = func(w http.ResponseWriter, message string) {
		RespondWithError(w, http.StatusUnprocessableEntity, message)
	}
)

// HandleRepositoryError converts repository errors to HTTP responses
func HandleRepositoryError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	errMsg := err.Error()

	// Check for specific error patterns
	switch {
	case contains(errMsg, "not found"):
		NotFound(w, errMsg)
	case contains(errMsg, "duplicate") || contains(errMsg, "already exists"):
		Conflict(w, errMsg)
	case contains(errMsg, "unauthorized"):
		Forbidden(w, errMsg)
	case contains(errMsg, "invalid") || contains(errMsg, "validation"):
		ValidationError(w, errMsg)
	default:
		InternalServerError(w, "An error occurred while processing your request")
	}
}

// contains checks if a string contains a substring (case-insensitive helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
