package handlers

import (
	"net/http"
	"questionarie-service/middleware"
	"questionarie-service/models"
	"questionarie-service/services"
	"questionarie-service/utils"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CategoryHandler handles questionnaire category HTTP requests
type CategoryHandler struct {
	service        *services.CategoryService
	companyService *services.CompanyService
}

// NewCategoryHandler creates a new CategoryHandler
func NewCategoryHandler(service *services.CategoryService, companyService *services.CompanyService) *CategoryHandler {
	return &CategoryHandler{
		service:        service,
		companyService: companyService,
	}
}

// CreateCategory handles POST /api/v1/questionnaire-categories
func (h *CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	category, err := h.service.CreateCategory(r.Context(), req.Name, req.Description)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusCreated, category, "Category created successfully")
}

// GetCategories handles GET /api/v1/questionnaire-categories
func (h *CategoryHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active") == "true"

	categories, err := h.service.GetAllCategories(r.Context(), activeOnly)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	// For company_admin with visibility "assigned", filter to only categories
	// that belong to their assigned questionnaires
	if !middleware.IsSuperAdmin(r.Context()) && h.companyService != nil {
		claims, _ := middleware.GetUserFromContext(r.Context())
		if claims != nil {
			company, err := h.companyService.GetMyCompany(r.Context(), claims.Sub)
			if err == nil && company != nil &&
				company.Settings != nil &&
				company.Settings.QuestionnaireVisibility == "assigned" {
				// Get assigned questionnaires to extract their category IDs
				cqs, err := h.companyService.GetCompanyQuestionnaires(r.Context(), company.ID, false)
				if err != nil {
					utils.HandleRepositoryError(w, err)
					return
				}
				qIDs := make([]primitive.ObjectID, 0, len(cqs))
				for _, cq := range cqs {
					qIDs = append(qIDs, cq.QuestionnaireID)
				}
				allowedCategoryIDs, err := h.service.GetCategoryIDsForQuestionnaires(r.Context(), qIDs)
				if err != nil {
					utils.HandleRepositoryError(w, err)
					return
				}
				// Filter categories
				filtered := make([]*models.QuestionnaireCategory, 0)
				for _, cat := range categories {
					if allowedCategoryIDs[cat.ID] {
						filtered = append(filtered, cat)
					}
				}
				utils.RespondWithSuccess(w, http.StatusOK, filtered, "")
				return
			}
		}
	}

	utils.RespondWithSuccess(w, http.StatusOK, categories, "")
}

// GetCategoryByID handles GET /api/v1/questionnaire-categories/:id
func (h *CategoryHandler) GetCategoryByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	category, err := h.service.GetCategoryByID(r.Context(), id)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, category, "")
}

// UpdateCategory handles PUT /api/v1/questionnaire-categories/:id
func (h *CategoryHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IsActive    *bool  `json:"is_active"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	if err := h.service.UpdateCategory(r.Context(), id, req.Name, req.Description, req.IsActive); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Category updated successfully")
}

// DeleteCategory handles DELETE /api/v1/questionnaire-categories/:id
func (h *CategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	if err := h.service.DeleteCategory(r.Context(), id); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Category deleted successfully")
}

// GetCategoryQuestionnaires handles GET /api/v1/questionnaire-categories/:id/questionnaires
func (h *CategoryHandler) GetCategoryQuestionnaires(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	questionnaires, err := h.service.GetQuestionnairesByCategory(r.Context(), id)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, questionnaires, "")
}
