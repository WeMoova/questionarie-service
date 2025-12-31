package handlers

import (
	"net/http"
	"questionarie-service/middleware"
	"questionarie-service/services"
	"questionarie-service/utils"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// CompanyHandler handles company-related HTTP requests
type CompanyHandler struct {
	service *services.CompanyService
}

// NewCompanyHandler creates a new CompanyHandler
func NewCompanyHandler(service *services.CompanyService) *CompanyHandler {
	return &CompanyHandler{
		service: service,
	}
}

// CreateCompany handles POST /api/v1/companies
func (h *CompanyHandler) CreateCompany(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	company, err := h.service.CreateCompany(r.Context(), req.Name)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
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
		Name string `json:"name"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	if err := h.service.UpdateCompany(r.Context(), id, req.Name); err != nil {
		utils.HandleRepositoryError(w, err)
		return
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
	cq, err := h.service.AssignQuestionnaireToCompany(r.Context(), companyID, questionnaireID, claims.Sub, periodStart, periodEnd)
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

	if err := h.service.UpdateCompanyQuestionnaire(r.Context(), id, periodStart, periodEnd, isActive); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Company questionnaire updated successfully")
}
