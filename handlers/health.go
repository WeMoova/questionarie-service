package handlers

import (
	"encoding/json"
	"net/http"

	"questionarie-service/db"
)

// HealthCheck handles the /health endpoint
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "questionarie-service",
	})
}

// ReadinessCheck handles the /ready endpoint
func ReadinessCheck(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Check database connection
		if err := database.HealthCheck(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "not ready",
				"checks": map[string]string{
					"database": "unhealthy: " + err.Error(),
				},
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ready",
			"checks": map[string]string{
				"database": "healthy",
			},
		})
	}
}

// ExampleHandler is a sample protected endpoint
func ExampleHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Hello from questionarie-service",
		"path":    r.URL.Path,
	})
}
