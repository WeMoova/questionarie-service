package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const CompanyContextKey contextKey = "api_company"

// APICompanyInfo holds the company information extracted from a validated API token.
type APICompanyInfo struct {
	CompanyID   primitive.ObjectID
	CompanyName string
	TokenID     primitive.ObjectID
	TokenName   string
}

// APITokenValidator is the interface that the middleware needs to validate tokens.
// Implemented by services.APITokenService.
type APITokenValidator interface {
	ValidateTokenRaw(ctx context.Context, rawToken string) (*APICompanyInfo, error)
}

// APIKeyAuth creates a middleware that validates company API tokens (X-API-Key header).
func APIKeyAuth(validator APITokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				// Also accept Authorization: Bearer wm_live_...
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer wm_live_") {
					apiKey = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}

			if apiKey == "" {
				respondError(w, "Missing X-API-Key header", http.StatusUnauthorized)
				return
			}

			info, err := validator.ValidateTokenRaw(r.Context(), apiKey)
			if err != nil {
				slog.Error("api key validation failed", "error", err, "path", r.URL.Path)
				respondError(w, "API key validation failed", http.StatusInternalServerError)
				return
			}
			if info == nil {
				slog.Warn("invalid api key", "path", r.URL.Path)
				respondError(w, "Invalid or inactive API key", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), CompanyContextKey, info)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetCompanyFromContext retrieves the API company info from request context.
func GetCompanyFromContext(ctx context.Context) (*APICompanyInfo, error) {
	info, ok := ctx.Value(CompanyContextKey).(*APICompanyInfo)
	if !ok || info == nil {
		return nil, fmt.Errorf("company info not found in context")
	}
	return info, nil
}
