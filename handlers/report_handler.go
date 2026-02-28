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

// GetAnswerDistribution handles GET /api/v1/reports/company-questionnaire/:cq_id/answers
func (h *ReportHandler) GetAnswerDistribution(w http.ResponseWriter, r *http.Request) {
	cqIDStr := chi.URLParam(r, "cq_id")
	cqID, err := utils.ValidateObjectID(cqIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	distribution, err := h.service.GetAnswerDistribution(r.Context(), cqID, claims.Sub, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, distribution, "")
}

// GetTrends handles GET /api/v1/reports/company/:company_id/trends
func (h *ReportHandler) GetTrends(w http.ResponseWriter, r *http.Request) {
	companyIDStr := chi.URLParam(r, "company_id")
	companyID, err := utils.ValidateObjectID(companyIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	trends, err := h.service.GetTrends(r.Context(), companyID, claims.Sub, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, trends, "")
}

// GetIndividualReport handles GET /api/v1/reports/assignments/:id
func (h *ReportHandler) GetIndividualReport(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	report, err := h.service.GetIndividualReport(r.Context(), id, claims.Sub, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, report, "")
}

// GetEvaluationSummary handles GET /api/v1/reports/company-questionnaire/:cq_id/evaluation-summary
func (h *ReportHandler) GetEvaluationSummary(w http.ResponseWriter, r *http.Request) {
	cqIDStr := chi.URLParam(r, "cq_id")
	cqID, err := utils.ValidateObjectID(cqIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	summary, err := h.service.GetEvaluationSummary(r.Context(), cqID, claims.Sub, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, summary, "")
}

// ExportCSV handles GET /api/v1/reports/company-questionnaire/:cq_id/export
func (h *ReportHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	cqIDStr := chi.URLParam(r, "cq_id")
	cqID, err := utils.ValidateObjectID(cqIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	data, err := h.service.ExportCSV(r.Context(), cqID, claims.Sub, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\"questionnaire_responses.csv\"")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
