package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
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
	"questionarie-service/storage"
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

	// Initialize MinIO storage (optional — graceful if not configured)
	var minioStorage *storage.MinIOStorage
	minioStorage, err = storage.NewMinIOStorage()
	if err != nil {
		log.Printf("WARNING: MinIO not configured, image upload disabled: %v", err)
	}

	// Initialize repositories
	companyRepo := repository.NewCompanyRepository(mongodb.Database)
	questionnaireRepo := repository.NewQuestionnaireRepository(mongodb.Database)
	companyQuestionnaireRepo := repository.NewCompanyQuestionnaireRepository(mongodb.Database)
	assignmentRepo := repository.NewAssignmentRepository(mongodb.Database)
	userMetadataRepo := repository.NewUserMetadataRepository(mongodb.Database)
	categoryRepo := repository.NewCategoryRepository(mongodb.Database)
	gamificationRepo := repository.NewGamificationRepository(mongodb.Database)

	// Seed default gamification data
	if err := gamificationRepo.SeedDefaultData(context.Background()); err != nil {
		log.Printf("WARNING: Failed to seed gamification defaults: %v", err)
	}

	// Initialize services
	questionnaireService := services.NewQuestionnaireService(questionnaireRepo)
	companyService := services.NewCompanyService(companyRepo, companyQuestionnaireRepo, questionnaireRepo, assignmentRepo, userMetadataRepo)
	userMetadataService := services.NewUserMetadataService(userMetadataRepo, companyRepo)
	gamificationService := services.NewGamificationService(gamificationRepo, userMetadataRepo)
	evaluationService := services.NewEvaluationService(questionnaireRepo, assignmentRepo)
	assignmentService := services.NewAssignmentService(assignmentRepo, companyQuestionnaireRepo, userMetadataRepo, questionnaireRepo, gamificationService, evaluationService)
	reportService := services.NewReportService(assignmentRepo, companyQuestionnaireRepo, userMetadataRepo, questionnaireRepo, companyRepo)
	categoryService := services.NewCategoryService(categoryRepo, questionnaireRepo)

	// Initialize handlers
	questionnaireHandler := handlers.NewQuestionnaireHandler(questionnaireService)
	companyHandler := handlers.NewCompanyHandler(companyService)
	userMetadataHandler := handlers.NewUserMetadataHandler(userMetadataService)
	assignmentHandler := handlers.NewAssignmentHandler(assignmentService)
	responseHandler := handlers.NewResponseHandler(assignmentService)
	reportHandler := handlers.NewReportHandler(reportService)
	categoryHandler := handlers.NewCategoryHandler(categoryService)
	gamificationHandler := handlers.NewGamificationHandler(gamificationService)

	var imageHandler *handlers.ImageHandler
	if minioStorage != nil {
		imageHandler = handlers.NewImageHandler(minioStorage)
	}

	// Create router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS middleware — restrict to allowed origins
	allowedOrigins := map[string]bool{
		"https://services.wemoova.com":    true,
		"https://qa.services.wemoova.com": true,
		"http://localhost:3000":           true,
	}
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if allowedOrigins[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "3600")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Request body size limit middleware (1MB for regular requests, skip for image uploads)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.URL.Path, "/images/upload") && !strings.Contains(r.URL.Path, "/questionnaires/import") {
				r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
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

		// Swagger UI (disabled in production)
		if os.Getenv("ENV") != "production" {
			r.Get("/swagger/doc.json", func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, "./docs/swagger.json")
			})
			r.Get("/swagger*", httpSwagger.Handler(
				httpSwagger.URL("/questionarie-service/swagger/doc.json"),
			))
		}

		// Public image serving (no auth required — images are accessed via <img> tags)
		if imageHandler != nil {
			r.Get("/api/v1/images/*", imageHandler.GetImage)
		}

		// Protected routes with JWT authentication
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.JWTAuth)

			// === Image Upload (Super Admin only) ===
			if imageHandler != nil {
				r.Group(func(r chi.Router) {
					r.Use(authMiddleware.RequireSuperAdmin())
					r.Post("/api/v1/images/upload", imageHandler.UploadImage)
					r.Delete("/api/v1/images", imageHandler.DeleteImage)
				})
			}

			// === Questionnaires READ (Company Admin+) ===
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireCompanyAdmin())

				r.Get("/api/v1/questionnaires", questionnaireHandler.GetQuestionnaires)
				r.Get("/api/v1/questionnaires/{id}", questionnaireHandler.GetQuestionnaireByID)
			})

			// === Questionnaires WRITE (Super Admin only) ===
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireSuperAdmin())

				r.Post("/api/v1/questionnaires/import", questionnaireHandler.ImportQuestionnaire)
				r.Post("/api/v1/questionnaires", questionnaireHandler.CreateQuestionnaire)
				r.Put("/api/v1/questionnaires/{id}", questionnaireHandler.UpdateQuestionnaire)
				r.Delete("/api/v1/questionnaires/{id}", questionnaireHandler.DeleteQuestionnaire)
				r.Patch("/api/v1/questionnaires/{id}/toggle-status", questionnaireHandler.ToggleQuestionnaireStatus)

				// Questions management
				r.Post("/api/v1/questionnaires/{id}/questions", questionnaireHandler.AddQuestion)
				r.Put("/api/v1/questionnaires/{id}/questions/{question_id}", questionnaireHandler.UpdateQuestion)
				r.Delete("/api/v1/questionnaires/{id}/questions/{question_id}", questionnaireHandler.RemoveQuestion)

				// Sections management
				r.Post("/api/v1/questionnaires/{id}/sections", questionnaireHandler.AddSection)
				r.Put("/api/v1/questionnaires/{id}/sections/{section_id}", questionnaireHandler.UpdateSection)
				r.Delete("/api/v1/questionnaires/{id}/sections/{section_id}", questionnaireHandler.DeleteSection)

				// Evaluation configuration
				r.Put("/api/v1/questionnaires/{id}/evaluation-config", questionnaireHandler.SetEvaluationConfig)
			})

			// === Companies (Super Admin only) ===
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireSuperAdmin())

				r.Post("/api/v1/companies", companyHandler.CreateCompany)
				r.Get("/api/v1/companies", companyHandler.GetCompanies)
				r.Get("/api/v1/companies/{id}", companyHandler.GetCompanyByID)
				r.Put("/api/v1/companies/{id}", companyHandler.UpdateCompany)
				r.Delete("/api/v1/companies/{id}", companyHandler.DeleteCompany)

				// Assign questionnaire to company
				r.Post("/api/v1/companies/{company_id}/questionnaires", companyHandler.AssignQuestionnaireToCompany)

				// Questionnaire stats & companies that use a questionnaire
				r.Get("/api/v1/questionnaires/{id}/stats", questionnaireHandler.GetQuestionnaireStats)
				r.Get("/api/v1/questionnaires/{id}/companies", companyHandler.GetQuestionnaireCompanies)

				// Categories (Super Admin only)
				r.Post("/api/v1/questionnaire-categories", categoryHandler.CreateCategory)
				r.Get("/api/v1/questionnaire-categories", categoryHandler.GetCategories)
				r.Get("/api/v1/questionnaire-categories/{id}", categoryHandler.GetCategoryByID)
				r.Put("/api/v1/questionnaire-categories/{id}", categoryHandler.UpdateCategory)
				r.Delete("/api/v1/questionnaire-categories/{id}", categoryHandler.DeleteCategory)
				r.Get("/api/v1/questionnaire-categories/{id}/questionnaires", categoryHandler.GetCategoryQuestionnaires)
			})

			// === Gamification Admin CRUD (Super Admin only) ===
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireSuperAdmin())

				r.Post("/api/v1/gamification/badges", gamificationHandler.CreateBadge)
				r.Get("/api/v1/gamification/badges", gamificationHandler.GetBadges)
				r.Put("/api/v1/gamification/badges/{id}", gamificationHandler.UpdateBadge)

				r.Post("/api/v1/gamification/achievements", gamificationHandler.CreateAchievement)
				r.Get("/api/v1/gamification/achievements", gamificationHandler.GetAchievements)
				r.Put("/api/v1/gamification/achievements/{id}", gamificationHandler.UpdateAchievement)

				r.Get("/api/v1/gamification/point-rules", gamificationHandler.GetPointRules)
				r.Put("/api/v1/gamification/point-rules/{id}", gamificationHandler.UpdatePointRule)
			})

			// === Gamification User (Employee+) ===
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireEmployee())

				r.Get("/api/v1/gamification/my-profile", gamificationHandler.GetMyProfile)
				r.Get("/api/v1/gamification/leaderboard", gamificationHandler.GetLeaderboard)
			})

			// === Gamification Admin Views (Company Admin+) ===
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireCompanyAdmin())

				r.Get("/api/v1/gamification/company/{company_id}/leaderboard", gamificationHandler.GetCompanyLeaderboard)
				r.Get("/api/v1/gamification/users/{user_id}/profile", gamificationHandler.GetUserProfile)
			})

			// === User Metadata - Get My Metadata (All authenticated users) ===
			r.Get("/api/v1/users/me/metadata", userMetadataHandler.GetMyMetadata)

			// === User Metadata - List Users (Super Admin, Company Admin, Supervisor) ===
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireSupervisor())

				r.Get("/api/v1/users/metadata", userMetadataHandler.ListUsers)
				r.Post("/api/v1/users/metadata/resolve-documents", userMetadataHandler.ResolveByDocuments)
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

			// === My Company (Company Admin+) ===
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireCompanyAdmin())

				r.Get("/api/v1/my-company", companyHandler.GetMyCompany)
			})

			// === Company Questionnaires (Company Admin+) ===
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireCompanyAdmin())

				r.Get("/api/v1/companies/{company_id}/questionnaires", companyHandler.GetCompanyQuestionnaires)
				r.Get("/api/v1/company-questionnaires/{id}", companyHandler.GetCompanyQuestionnaire)
				r.Put("/api/v1/company-questionnaires/{id}", companyHandler.UpdateCompanyQuestionnaire)

				// Lifecycle transitions
				r.Post("/api/v1/company-questionnaires/{id}/activate", companyHandler.ActivateCompanyQuestionnaire)
				r.Post("/api/v1/company-questionnaires/{id}/pause", companyHandler.PauseCompanyQuestionnaire)
				r.Post("/api/v1/company-questionnaires/{id}/close", companyHandler.CloseCompanyQuestionnaire)

				// Bulk assignment
				r.Post("/api/v1/company-questionnaires/{cq_id}/assign-all", companyHandler.AssignAllToCompany)
				r.Post("/api/v1/company-questionnaires/{cq_id}/assign-department", companyHandler.AssignToDepartment)

				// Company dashboard
				r.Get("/api/v1/companies/{company_id}/dashboard", companyHandler.GetCompanyDashboard)
			})

			// === Assignments (Supervisor+) ===
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireSupervisor())

				// Assign questionnaires to users
				r.Post("/api/v1/company-questionnaires/{cq_id}/assignments", assignmentHandler.AssignToUsers)
				r.Get("/api/v1/company-questionnaires/{cq_id}/assignments", assignmentHandler.GetAssignmentsByCompanyQuestionnaire)
				r.Delete("/api/v1/company-questionnaires/{cq_id}/assignments", assignmentHandler.CancelAllAssignments)

				// Progress & visibility
				r.Get("/api/v1/company-questionnaires/{cq_id}/progress", assignmentHandler.GetAssignmentProgress)
				r.Get("/api/v1/company-questionnaires/{cq_id}/pending-users", assignmentHandler.GetPendingUsers)
				r.Get("/api/v1/company-questionnaires/{cq_id}/in-progress-users", assignmentHandler.GetInProgressUsers)
				r.Get("/api/v1/company-questionnaires/{cq_id}/completed-users", assignmentHandler.GetCompletedUsers)

				// Reminders
				r.Post("/api/v1/company-questionnaires/{cq_id}/remind", companyHandler.SendReminder)

				// Cancel individual assignment
				r.Post("/api/v1/assignments/{id}/cancel", assignmentHandler.CancelAssignment)

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

			// === Reports (Supervisor+) ===
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireSupervisor())

				r.Get("/api/v1/reports/company-questionnaire/{cq_id}/completion", reportHandler.GetCompletionMetrics)
				r.Get("/api/v1/reports/company/{company_id}/overview", reportHandler.GetCompanyOverview)
				r.Get("/api/v1/reports/company/{company_id}/employees-progress", reportHandler.GetEmployeeProgress)
				r.Get("/api/v1/reports/assignments/{id}", reportHandler.GetIndividualReport)
			})

			// === Reports avanzados (Company Admin+) ===
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireCompanyAdmin())

				r.Get("/api/v1/reports/company-questionnaire/{cq_id}/answers", reportHandler.GetAnswerDistribution)
				r.Get("/api/v1/reports/company-questionnaire/{cq_id}/evaluation-summary", reportHandler.GetEvaluationSummary)
				r.Get("/api/v1/reports/company/{company_id}/trends", reportHandler.GetTrends)
				r.Get("/api/v1/reports/company-questionnaire/{cq_id}/export", reportHandler.ExportCSV)
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
