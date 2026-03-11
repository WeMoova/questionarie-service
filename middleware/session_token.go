package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"questionarie-service/utils"
)

type SessionClaims struct {
	AssignmentID string `json:"assignment_id"`
	LinkID       string `json:"link_id"`
	jwt.RegisteredClaims
}

type sessionContextKey struct{}

var SessionContextKey = sessionContextKey{}

func getSessionSecret() []byte {
	secret := os.Getenv("SESSION_TOKEN_SECRET")
	if secret == "" {
		secret = "wemoova-session-default-secret-change-in-prod"
	}
	return []byte(secret)
}

func GenerateSessionToken(assignmentID, linkID primitive.ObjectID) (string, error) {
	claims := SessionClaims{
		AssignmentID: assignmentID.Hex(),
		LinkID:       linkID.Hex(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(4 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "wemoova-public",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getSessionSecret())
}

func SessionTokenAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			utils.RespondWithError(w, http.StatusUnauthorized, "Session token required")
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.ParseWithClaims(tokenStr, &SessionClaims{}, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return getSessionSecret(), nil
		})
		if err != nil || !token.Valid {
			utils.RespondWithError(w, http.StatusUnauthorized, "Invalid or expired session token")
			return
		}

		claims, ok := token.Claims.(*SessionClaims)
		if !ok {
			utils.RespondWithError(w, http.StatusUnauthorized, "Invalid session claims")
			return
		}

		ctx := context.WithValue(r.Context(), SessionContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetSessionFromContext(ctx context.Context) *SessionClaims {
	claims, _ := ctx.Value(SessionContextKey).(*SessionClaims)
	return claims
}
