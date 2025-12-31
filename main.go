package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"

	"questionarie-service/db"
	"questionarie-service/handlers"
	authMiddleware "questionarie-service/middleware"
	"questionarie-service/repository"
	"questionarie-service/services"
)

func main() {
	// Load environment variables
	_ = godotenv.Load()

	// Initialize MongoDB connection
	mongodb, err := db.NewMongoDB()
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongodb.Close(context.Background())

	// Initialize repositories
	companyRepo := repository.NewCompanyRepository(mongodb.Database)
	questionnaireRepo := repository.NewQuestionnaireRepository(mongodb.Database)
	companyQuestionnaireRepo := repository.NewCompanyQuestionnaireRepository(mongodb.Database)
	assignmentRepo := repository.NewAssignmentRepository(mongodb.Database)
	userMetadataRepo := repository.NewUserMetadataRepository(mongodb.Database)

	// Initialize services
	questionnaireService := services.NewQuestionnaireService(questionnaireRepo)
	companyService := services.NewCompanyService(companyRepo, companyQuestionnaireRepo, questionnaireRepo)
	userMetadataService := services.NewUserMetadataService(userMetadataRepo, companyRepo)
	assignmentService := services.NewAssignmentService(assignmentRepo, companyQuestionnaireRepo, userMetadataRepo, questionnaireRepo)
	reportService := services.NewReportService(assignmentRepo, companyQuestionnaireRepo, userMetadataRepo, questionnaireRepo, companyRepo)

	// Initialize handlers
	questionnaireHandler := handlers.NewQuestionnaireHandler(questionnaireService)
	companyHandler := handlers.NewCompanyHandler(companyService)
	userMetadataHandler := handlers.NewUserMetadataHandler(userMetadataService)
	assignmentHandler := handlers.NewAssignmentHandler(assignmentService)
	responseHandler := handlers.NewResponseHandler(assignmentService)
	reportHandler := handlers.NewReportHandler(reportService)

	// Create router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Mount all routes under /questionarie-service prefix for ALB path-based routing
	r.Route("/questionarie-service", func(r chi.Router) {
		// Health endpoints (no auth required)
		r.Get("/health", handlers.HealthCheck)
		r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
			if err := mongodb.HealthCheck(r.Context()); err != nil {
				http.Error(w, "Database unhealthy", http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Ready"))
		})

		// Swagger UI (no auth required)
		r.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL("/questionarie-service/swagger/doc.json"),
		))

		// Serve swagger.json file
		r.Get("/swagger/doc.json", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./docs/swagger.json")
		})

		// Protected routes with JWT authentication
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.JWTAuth)

			// === Questionnaires (Super Admin only) ===
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireSuperAdmin())

				r.Post("/api/v1/questionnaires", questionnaireHandler.CreateQuestionnaire)
				r.Get("/api/v1/questionnaires", questionnaireHandler.GetQuestionnaires)
				r.Get("/api/v1/questionnaires/{id}", questionnaireHandler.GetQuestionnaireByID)
				r.Put("/api/v1/questionnaires/{id}", questionnaireHandler.UpdateQuestionnaire)
				r.Delete("/api/v1/questionnaires/{id}", questionnaireHandler.DeactivateQuestionnaire)

				// Questions management
				r.Post("/api/v1/questionnaires/{id}/questions", questionnaireHandler.AddQuestion)
				r.Put("/api/v1/questionnaires/{id}/questions/{question_id}", questionnaireHandler.UpdateQuestion)
				r.Delete("/api/v1/questionnaires/{id}/questions/{question_id}", questionnaireHandler.RemoveQuestion)
			})

			// === Companies (Super Admin only) ===
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireSuperAdmin())

				r.Post("/api/v1/companies", companyHandler.CreateCompany)
				r.Get("/api/v1/companies", companyHandler.GetCompanies)
				r.Get("/api/v1/companies/{id}", companyHandler.GetCompanyByID)
				r.Put("/api/v1/companies/{id}", companyHandler.UpdateCompany)

				// Assign questionnaire to company
				r.Post("/api/v1/companies/{company_id}/questionnaires", companyHandler.AssignQuestionnaireToCompany)
			})

			// === User Metadata (Super Admin only) ===
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireSuperAdmin())

				r.Post("/api/v1/users/metadata", userMetadataHandler.CreateUserMetadata)
				r.Get("/api/v1/users/metadata/{user_id}", userMetadataHandler.GetUserMetadata)
				r.Put("/api/v1/users/metadata/{user_id}", userMetadataHandler.UpdateUserMetadata)
				r.Delete("/api/v1/users/metadata/{user_id}", userMetadataHandler.DeleteUserMetadata)

				// Get users by company
				r.Get("/api/v1/companies/{company_id}/users", userMetadataHandler.GetUsersByCompany)
			})

			// === Company Questionnaires (Super Admin, Company Admin) ===
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireCompanyAdmin())

				r.Get("/api/v1/companies/{company_id}/questionnaires", companyHandler.GetCompanyQuestionnaires)
				r.Put("/api/v1/company-questionnaires/{id}", companyHandler.UpdateCompanyQuestionnaire)
			})

			// === Assignments (Company Admin, Supervisor) ===
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireSupervisor())

				// Assign questionnaires to users
				r.Post("/api/v1/company-questionnaires/{cq_id}/assignments", assignmentHandler.AssignToUsers)
				r.Get("/api/v1/company-questionnaires/{cq_id}/assignments", assignmentHandler.GetAssignmentsByCompanyQuestionnaire)

				// View company/team questionnaires
				r.Get("/api/v1/my-company/questionnaires", assignmentHandler.GetMyCompanyQuestionnaires)
				r.Get("/api/v1/my-team/assignments", assignmentHandler.GetMyTeamAssignments)
			})

			// === Responses (Employee - all authenticated users) ===
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireEmployee())

				// View my assignments
				r.Get("/api/v1/my-assignments", assignmentHandler.GetMyAssignments)
				r.Get("/api/v1/assignments/{id}", assignmentHandler.GetAssignmentByID)

				// Save responses
				r.Post("/api/v1/assignments/{id}/responses", responseHandler.SaveResponse)
				r.Put("/api/v1/assignments/{id}/responses", responseHandler.UpdateResponses)
				r.Post("/api/v1/assignments/{id}/submit", responseHandler.SubmitAssignment)
			})

			// === Reports (Company Admin, Supervisor) ===
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireSupervisor())

				r.Get("/api/v1/reports/company-questionnaire/{cq_id}/completion", reportHandler.GetCompletionMetrics)
				r.Get("/api/v1/reports/company/{company_id}/overview", reportHandler.GetCompanyOverview)
				r.Get("/api/v1/reports/company/{company_id}/employees-progress", reportHandler.GetEmployeeProgress)
			})
		})
	})

	// Server configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Server starting on port %s", port)
		log.Printf("Connected to MongoDB: %s", os.Getenv("MONGODB_DATABASE"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	<-done
	log.Println("Server shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server exited properly")
}
