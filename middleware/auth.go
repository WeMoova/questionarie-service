package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

type contextKey string

const UserContextKey contextKey = "user"

type JWTClaims struct {
	Sub   string   `json:"sub"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

var (
	jwksCache     jwk.Set
	jwksCacheMux  sync.RWMutex
	jwksCacheTime time.Time
	cacheDuration = 24 * time.Hour
)

// JWTAuth middleware validates FusionAuth JWT tokens
func JWTAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondError(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			respondError(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		claims, err := validateToken(tokenString)
		if err != nil {
			respondError(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func validateToken(tokenString string) (*JWTClaims, error) {
	fusionAuthURL := os.Getenv("FUSIONAUTH_URL")
	if fusionAuthURL == "" {
		return nil, fmt.Errorf("FUSIONAUTH_URL not configured")
	}

	jwksURL := fmt.Sprintf("%s/.well-known/jwks.json", fusionAuthURL)

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get kid from token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid not found in token header")
		}

		// Fetch public key from JWKS
		publicKey, err := fetchPublicKey(jwksURL, kid)
		if err != nil {
			return nil, err
		}

		return publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

func fetchPublicKey(jwksURL, kid string) (interface{}, error) {
	set, err := getJWKS(jwksURL)
	if err != nil {
		return nil, err
	}

	key, found := set.LookupKeyID(kid)
	if !found {
		return nil, fmt.Errorf("key with kid %s not found", kid)
	}

	var rawKey interface{}
	if err := key.Raw(&rawKey); err != nil {
		return nil, fmt.Errorf("failed to get raw key: %w", err)
	}

	return rawKey, nil
}

func getJWKS(jwksURL string) (jwk.Set, error) {
	// Check cache
	jwksCacheMux.RLock()
	if jwksCache != nil && time.Since(jwksCacheTime) < cacheDuration {
		defer jwksCacheMux.RUnlock()
		return jwksCache, nil
	}
	jwksCacheMux.RUnlock()

	// Fetch JWKS
	set, err := jwk.Fetch(context.Background(), jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	// Update cache
	jwksCacheMux.Lock()
	jwksCache = set
	jwksCacheTime = time.Now()
	jwksCacheMux.Unlock()

	return set, nil
}

func respondError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// GetUserFromContext retrieves user claims from request context
func GetUserFromContext(ctx context.Context) (*JWTClaims, error) {
	claims, ok := ctx.Value(UserContextKey).(*JWTClaims)
	if !ok {
		return nil, fmt.Errorf("user not found in context")
	}
	return claims, nil
}
