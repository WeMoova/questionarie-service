package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"questionarie-service/middleware"
	"questionarie-service/models"
	"questionarie-service/services"
	"questionarie-service/utils"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CompanyHandler handles company-related HTTP requests
type CompanyHandler struct {
	service          *services.CompanyService
	fusionAuthService *services.FusionAuthService
}

// NewCompanyHandler creates a new CompanyHandler
func NewCompanyHandler(service *services.CompanyService, fusionAuthService *services.FusionAuthService) *CompanyHandler {
	return &CompanyHandler{
		service:          service,
		fusionAuthService: fusionAuthService,
	}
}

// CreateCompany handles POST /api/v1/companies
func (h *CompanyHandler) CreateCompany(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string                 `json:"name"`
		IsActive     *bool                  `json:"is_active"`
		Branding     *models.Branding       `json:"branding"`
		CustomDomain *models.CustomDomain   `json:"custom_domain"`
		Settings     *models.CompanySettings `json:"settings"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	// Default is_active to true if not provided
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	company, err := h.service.CreateCompany(r.Context(), req.Name, isActive, req.Branding, req.CustomDomain, req.Settings)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	// Create FusionAuth tenant for the company (synchronous)
	if h.fusionAuthService != nil && company.CustomDomain != nil && company.CustomDomain.Slug != "" {
		result, err := h.fusionAuthService.CreateTenantForCompany(r.Context(), company)
		if err != nil {
			slog.Error("failed to create FusionAuth tenant", "company_id", company.ID.Hex(), "error", err)
			// Don't fail company creation — tenant can be created later
		} else {
			if err := h.service.UpdateCompanyFusionAuth(r.Context(), company.ID, result.TenantID, result.ApplicationID, result.ClientID, result.ClientSecret); err != nil {
				slog.Error("failed to save FusionAuth IDs", "company_id", company.ID.Hex(), "error", err)
			} else {
				company.FusionAuthTenantID = result.TenantID
				company.FusionAuthApplicationID = result.ApplicationID
				company.FusionAuthClientID = result.ClientID
				company.FusionAuthClientSecret = result.ClientSecret
			}
		}
	}

	utils.RespondWithSuccess(w, http.StatusCreated, company, "Company created successfully")
}

// GetCompanies handles GET /api/v1/companies
func (h *CompanyHandler) GetCompanies(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)
	pageSize, _ := strconv.ParseInt(r.URL.Query().Get("page_size"), 10, 64)

	companies, err := h.service.GetAllCompanies(r.Context(), page, pageSize)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, companies, "")
}

// GetMyCompany handles GET /api/v1/my-company
// Returns the company of the authenticated company_admin based on their user metadata.
func (h *CompanyHandler) GetMyCompany(w http.ResponseWriter, r *http.Request) {
	claims, err := middleware.GetUserFromContext(r.Context())
	if err != nil {
		utils.Unauthorized(w, "unauthorized")
		return
	}

	company, err := h.service.GetMyCompany(r.Context(), claims.Sub)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, company, "")
}

// GetCompanyByID handles GET /api/v1/companies/:id
func (h *CompanyHandler) GetCompanyByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	company, err := h.service.GetCompanyByID(r.Context(), id)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, company, "")
}

// UpdateCompany handles PUT /api/v1/companies/:id
func (h *CompanyHandler) UpdateCompany(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var req struct {
		Name         string                 `json:"name"`
		IsActive     *bool                  `json:"is_active"`
		Branding     *models.Branding       `json:"branding"`
		CustomDomain *models.CustomDomain   `json:"custom_domain"`
		Settings     *models.CompanySettings `json:"settings"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	if err := h.service.UpdateCompany(r.Context(), id, req.Name, req.IsActive, req.Branding, req.CustomDomain, req.Settings); err != nil {
		slog.Error("UpdateCompany failed", "company_id", id.Hex(), "error", err.Error())
		utils.HandleRepositoryError(w, err)
		return
	}

	// Sync branding to FusionAuth tenant if branding changed
	if req.Branding != nil && h.fusionAuthService != nil {
		company, err := h.service.GetCompanyByID(r.Context(), id)
		if err == nil && company.FusionAuthTenantID != "" {
			go func() {
				if err := h.fusionAuthService.SyncBrandingToTenant(context.Background(), company); err != nil {
					slog.Error("failed to sync branding to FusionAuth", "company_id", id.Hex(), "error", err)
				}
			}()
		}
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Company updated successfully")
}

// AssignQuestionnaireToCompany handles POST /api/v1/companies/:company_id/questionnaires
func (h *CompanyHandler) AssignQuestionnaireToCompany(w http.ResponseWriter, r *http.Request) {
	companyIDStr := chi.URLParam(r, "company_id")
	companyID, err := utils.ValidateObjectID(companyIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var req struct {
		QuestionnaireID string `json:"questionnaire_id"`
		PeriodStart     string `json:"period_start"`
		PeriodEnd       string `json:"period_end"`
		DisplayMode     string `json:"display_mode"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	questionnaireID, err := utils.ValidateObjectID(req.QuestionnaireID)
	if err != nil {
		utils.BadRequest(w, "invalid questionnaire_id: "+err.Error())
		return
	}

	periodStart, err := time.Parse("2006-01-02", req.PeriodStart)
	if err != nil {
		utils.BadRequest(w, "invalid period_start format (use YYYY-MM-DD)")
		return
	}

	periodEnd, err := time.Parse("2006-01-02", req.PeriodEnd)
	if err != nil {
		utils.BadRequest(w, "invalid period_end format (use YYYY-MM-DD)")
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	cq, err := h.service.AssignQuestionnaireToCompany(r.Context(), companyID, questionnaireID, claims.Sub, periodStart, periodEnd, req.DisplayMode)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusCreated, cq, "Questionnaire assigned to company successfully")
}

// GetCompanyQuestionnaires handles GET /api/v1/companies/:company_id/questionnaires
func (h *CompanyHandler) GetCompanyQuestionnaires(w http.ResponseWriter, r *http.Request) {
	companyIDStr := chi.URLParam(r, "company_id")
	companyID, err := utils.ValidateObjectID(companyIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	activeOnly := r.URL.Query().Get("active") == "true"

	questionnaires, err := h.service.GetCompanyQuestionnaires(r.Context(), companyID, activeOnly)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, questionnaires, "")
}

// DeleteCompany handles DELETE /api/v1/companies/:id
func (h *CompanyHandler) DeleteCompany(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	if err := h.service.DeleteCompany(r.Context(), id); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Company deleted successfully")
}

// GetCompanyQuestionnaire handles GET /api/v1/company-questionnaires/:id
func (h *CompanyHandler) GetCompanyQuestionnaire(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	cq, err := h.service.GetCompanyQuestionnaireByID(r.Context(), id, claims.Sub, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, cq, "")
}

// ActivateCompanyQuestionnaire handles POST /api/v1/company-questionnaires/:id/activate
func (h *CompanyHandler) ActivateCompanyQuestionnaire(w http.ResponseWriter, r *http.Request) {
	h.changeCompanyQuestionnaireStatus(w, r, "active")
}

// PauseCompanyQuestionnaire handles POST /api/v1/company-questionnaires/:id/pause
func (h *CompanyHandler) PauseCompanyQuestionnaire(w http.ResponseWriter, r *http.Request) {
	h.changeCompanyQuestionnaireStatus(w, r, "paused")
}

// CloseCompanyQuestionnaire handles POST /api/v1/company-questionnaires/:id/close
func (h *CompanyHandler) CloseCompanyQuestionnaire(w http.ResponseWriter, r *http.Request) {
	h.changeCompanyQuestionnaireStatus(w, r, "closed")
}

func (h *CompanyHandler) changeCompanyQuestionnaireStatus(w http.ResponseWriter, r *http.Request, newStatus string) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	utils.ParseRequestBody(r, &req)

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	if err := h.service.ChangeCompanyQuestionnaireStatus(
		r.Context(), id,
		models.CompanyQuestionnaireStatus(newStatus),
		claims.Sub, req.Reason, isSuperAdmin,
	); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Status updated to "+newStatus)
}

// AssignAllToCompany handles POST /api/v1/company-questionnaires/:cq_id/assign-all
func (h *CompanyHandler) AssignAllToCompany(w http.ResponseWriter, r *http.Request) {
	cqIDStr := chi.URLParam(r, "cq_id")
	cqID, err := utils.ValidateObjectID(cqIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	userIDs, err := h.service.AssignAllToCompany(r.Context(), cqID, claims.Sub, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, map[string]interface{}{
		"user_ids":    userIDs,
		"total_users": len(userIDs),
	}, "Use POST /assignments with these user_ids to create assignments")
}

// AssignToDepartment handles POST /api/v1/company-questionnaires/:cq_id/assign-department
func (h *CompanyHandler) AssignToDepartment(w http.ResponseWriter, r *http.Request) {
	cqIDStr := chi.URLParam(r, "cq_id")
	cqID, err := utils.ValidateObjectID(cqIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var req struct {
		Department string `json:"department"`
	}
	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
	if req.Department == "" {
		utils.BadRequest(w, "department is required")
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	userIDs, err := h.service.AssignToDepartment(r.Context(), cqID, req.Department, claims.Sub, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, map[string]interface{}{
		"department":  req.Department,
		"user_ids":    userIDs,
		"total_users": len(userIDs),
	}, "Department users ready for assignment")
}

// SendReminder handles POST /api/v1/company-questionnaires/:cq_id/remind
func (h *CompanyHandler) SendReminder(w http.ResponseWriter, r *http.Request) {
	cqIDStr := chi.URLParam(r, "cq_id")
	cqID, err := utils.ValidateObjectID(cqIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var req struct {
		Target  string `json:"target"`
		Message string `json:"message"`
	}
	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
	if req.Target == "" {
		req.Target = "pending"
	}
	if req.Target != "pending" && req.Target != "in_progress" && req.Target != "all" {
		utils.BadRequest(w, "target must be one of: pending, in_progress, all")
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	result, err := h.service.RegisterReminder(r.Context(), cqID, req.Target, claims.Sub, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, result, "Reminder registered successfully")
}

// DeleteCompanyQuestionnaire handles DELETE /api/v1/company-questionnaires/:id
func (h *CompanyHandler) DeleteCompanyQuestionnaire(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	if err := h.service.DeleteCompanyQuestionnaire(r.Context(), id, claims.Sub, isSuperAdmin); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Company questionnaire deleted")
}

// GetCompanyDashboard handles GET /api/v1/companies/:company_id/dashboard
func (h *CompanyHandler) GetCompanyDashboard(w http.ResponseWriter, r *http.Request) {
	companyIDStr := chi.URLParam(r, "company_id")
	companyID, err := utils.ValidateObjectID(companyIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	dashboard, err := h.service.GetCompanyDashboard(r.Context(), companyID, claims.Sub, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, dashboard, "")
}

// GetQuestionnaireCompanies handles GET /api/v1/questionnaires/:id/companies
func (h *CompanyHandler) GetQuestionnaireCompanies(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	companies, err := h.service.GetQuestionnaireCompanies(r.Context(), id)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, companies, "")
}

// UpdateCompanyQuestionnaire handles PUT /api/v1/company-questionnaires/:id
func (h *CompanyHandler) UpdateCompanyQuestionnaire(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var req struct {
		PeriodStart string `json:"period_start"`
		PeriodEnd   string `json:"period_end"`
		IsActive    *bool  `json:"is_active"`
		DisplayMode string `json:"display_mode"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var periodStart, periodEnd time.Time

	if req.PeriodStart != "" {
		periodStart, err = time.Parse("2006-01-02", req.PeriodStart)
		if err != nil {
			utils.BadRequest(w, "invalid period_start format")
			return
		}
	}

	if req.PeriodEnd != "" {
		periodEnd, err = time.Parse("2006-01-02", req.PeriodEnd)
		if err != nil {
			utils.BadRequest(w, "invalid period_end format")
			return
		}
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	if err := h.service.UpdateCompanyQuestionnaire(r.Context(), id, periodStart, periodEnd, isActive, req.DisplayMode, claims.Sub, isSuperAdmin); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Company questionnaire updated successfully")
}

// GetCompanyBrandingBySlug handles GET /api/v1/public/company-branding/{slug}
// Public endpoint — no authentication required.
func (h *CompanyHandler) GetCompanyBrandingBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		utils.BadRequest(w, "slug is required")
		return
	}

	result, err := h.service.GetCompanyBrandingBySlug(r.Context(), slug)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}
	if result == nil {
		utils.RespondWithError(w, http.StatusNotFound, "company not found")
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, result, "")
}

// GetColorConfig handles GET /api/v1/company-questionnaires/{id}/color-config
func (h *CompanyHandler) GetColorConfig(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.BadRequest(w, "invalid company questionnaire ID")
		return
	}

	config, err := h.service.GetColorConfig(r.Context(), id)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, config, "")
}

// UpdateColorConfig handles PUT /api/v1/company-questionnaires/{id}/color-config
func (h *CompanyHandler) UpdateColorConfig(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.BadRequest(w, "invalid company questionnaire ID")
		return
	}

	var input models.ColorConfig
	if err := utils.ParseRequestBody(r, &input); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	result, err := h.service.UpdateColorConfig(r.Context(), id, &input)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, result, "Color config updated successfully")
}

// GetCompanyAuthConfig handles GET /api/v1/public/company-auth-config/{slug}
// Public endpoint — returns FusionAuth auth config for the employee app login.
func (h *CompanyHandler) GetCompanyAuthConfig(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		utils.BadRequest(w, "slug is required")
		return
	}

	company, err := h.service.GetCompanyBySlug(r.Context(), slug)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}
	if company == nil || !company.IsActive {
		utils.RespondWithError(w, http.StatusNotFound, "company not found")
		return
	}

	if company.FusionAuthTenantID == "" {
		utils.RespondWithError(w, http.StatusNotFound, "company auth not configured")
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, map[string]string{
		"tenant_id":      company.FusionAuthTenantID,
		"application_id": company.FusionAuthApplicationID,
		"client_id":      company.FusionAuthClientID,
		"client_secret":  company.FusionAuthClientSecret,
	}, "")
}
