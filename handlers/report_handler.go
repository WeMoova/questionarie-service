package handlers

import (
	"net/http"
	"questionarie-service/middleware"
	"questionarie-service/services"
	"questionarie-service/utils"

	"github.com/go-chi/chi/v5"
)

// ReportHandler handles report-related HTTP requests
type ReportHandler struct {
	service *services.ReportService
}

// NewReportHandler creates a new ReportHandler
func NewReportHandler(service *services.ReportService) *ReportHandler {
	return &ReportHandler{
		service: service,
	}
}

// GetCompletionMetrics handles GET /api/v1/reports/company-questionnaire/:cq_id/completion
func (h *ReportHandler) GetCompletionMetrics(w http.ResponseWriter, r *http.Request) {
	cqIDStr := chi.URLParam(r, "cq_id")
	cqID, err := utils.ValidateObjectID(cqIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	metrics, err := h.service.GetCompletionMetrics(r.Context(), cqID, claims.Sub, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, metrics, "")
}

// GetCompanyOverview handles GET /api/v1/reports/company/:company_id/overview
func (h *ReportHandler) GetCompanyOverview(w http.ResponseWriter, r *http.Request) {
	companyIDStr := chi.URLParam(r, "company_id")
	companyID, err := utils.ValidateObjectID(companyIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	overview, err := h.service.GetCompanyOverview(r.Context(), companyID, claims.Sub, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, overview, "")
}

// GetEmployeeProgress handles GET /api/v1/reports/company/:company_id/employees-progress
func (h *ReportHandler) GetEmployeeProgress(w http.ResponseWriter, r *http.Request) {
	companyIDStr := chi.URLParam(r, "company_id")
	companyID, err := utils.ValidateObjectID(companyIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	progress, err := h.service.GetEmployeeProgress(r.Context(), companyID, claims.Sub, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, progress, "")
}
