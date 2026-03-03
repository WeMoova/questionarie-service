package handlers

import (
	"encoding/json"
	"io"
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

// GetDimensionSummary handles GET /api/v1/reports/company-questionnaire/:cq_id/dimension-summary
func (h *ReportHandler) GetDimensionSummary(w http.ResponseWriter, r *http.Request) {
	cqIDStr := chi.URLParam(r, "cq_id")
	cqID, err := utils.ValidateObjectID(cqIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	summary, err := h.service.GetDimensionSummary(r.Context(), cqID, claims.Sub, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, summary, "")
}

// GetDepartmentDimensions handles GET /api/v1/reports/company-questionnaire/:cq_id/department-dimensions
func (h *ReportHandler) GetDepartmentDimensions(w http.ResponseWriter, r *http.Request) {
	cqIDStr := chi.URLParam(r, "cq_id")
	cqID, err := utils.ValidateObjectID(cqIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	result, err := h.service.GetDepartmentDimensions(r.Context(), cqID, claims.Sub, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, result, "")
}

// GetRiskAnalysis handles GET /api/v1/reports/company-questionnaire/:cq_id/risk-analysis
func (h *ReportHandler) GetRiskAnalysis(w http.ResponseWriter, r *http.Request) {
	cqIDStr := chi.URLParam(r, "cq_id")
	cqID, err := utils.ValidateObjectID(cqIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	result, err := h.service.GetRiskAnalysis(r.Context(), cqID, claims.Sub, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, result, "")
}

// GetScoreDistribution handles GET /api/v1/reports/company-questionnaire/:cq_id/score-distribution
func (h *ReportHandler) GetScoreDistribution(w http.ResponseWriter, r *http.Request) {
	cqIDStr := chi.URLParam(r, "cq_id")
	cqID, err := utils.ValidateObjectID(cqIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	result, err := h.service.GetScoreDistribution(r.Context(), cqID, claims.Sub, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, result, "")
}

// GetFreeTextResponses handles GET /api/v1/reports/company-questionnaire/:cq_id/free-text-responses
func (h *ReportHandler) GetFreeTextResponses(w http.ResponseWriter, r *http.Request) {
	cqIDStr := chi.URLParam(r, "cq_id")
	cqID, err := utils.ValidateObjectID(cqIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	result, err := h.service.GetFreeTextResponses(r.Context(), cqID, claims.Sub, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, result, "")
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

// SendReportEmail handles POST /api/v1/reports/company-questionnaire/:cq_id/send-email
func (h *ReportHandler) SendReportEmail(w http.ResponseWriter, r *http.Request) {
	cqIDStr := chi.URLParam(r, "cq_id")
	cqID, err := utils.ValidateObjectID(cqIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	// Parse request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.BadRequest(w, "failed to read request body")
		return
	}
	defer r.Body.Close()

	var req struct {
		Recipients    []string          `json:"recipients"`
		Subject       string            `json:"subject"`
		CustomMessage string            `json:"custom_message"`
		Attachments   []services.EmailAttachment `json:"attachments,omitempty"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		utils.BadRequest(w, "invalid JSON format")
		return
	}

	if len(req.Recipients) == 0 {
		utils.BadRequest(w, "at least one recipient is required")
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	if err := h.service.SendReportEmail(r.Context(), cqID, claims.Sub, isSuperAdmin, req.Recipients, req.Subject, req.CustomMessage, req.Attachments); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Email enviado exitosamente")
}
