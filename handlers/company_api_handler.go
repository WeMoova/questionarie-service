package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"questionarie-service/middleware"
	"questionarie-service/services"
	"questionarie-service/utils"
	"time"

	"github.com/go-chi/chi/v5"
)

// CompanyAPIHandler handles company-facing API endpoints authenticated via API key.
type CompanyAPIHandler struct {
	reportService *services.ReportService
	httpClient    *http.Client
}

// NewCompanyAPIHandler creates a new CompanyAPIHandler.
func NewCompanyAPIHandler(reportService *services.ReportService) *CompanyAPIHandler {
	return &CompanyAPIHandler{
		reportService: reportService,
		httpClient:    &http.Client{Timeout: 120 * time.Second},
	}
}

func (h *CompanyAPIHandler) getReportServiceURL() string {
	u := os.Getenv("REPORT_SERVICE_URL")
	if u == "" {
		return "http://report-service.report-service.svc.cluster.local:8080"
	}
	return u
}

// GeneratePDF handles GET /api/v1/company-api/reports/{cq_id}/pdf
func (h *CompanyAPIHandler) GeneratePDF(w http.ResponseWriter, r *http.Request) {
	cqIDStr := chi.URLParam(r, "cq_id")
	cqID, err := utils.ValidateObjectID(cqIDStr)
	if err != nil {
		utils.BadRequest(w, "Invalid company questionnaire ID")
		return
	}

	companyInfo, err := middleware.GetCompanyFromContext(r.Context())
	if err != nil {
		utils.Unauthorized(w, "missing company context")
		return
	}

	// Verify the cq belongs to this company
	if err := h.reportService.VerifyCompanyOwnership(r.Context(), cqID, companyInfo.CompanyID); err != nil {
		utils.Forbidden(w, "This questionnaire does not belong to your company")
		return
	}

	// Gather all report data (pass isSuperAdmin=true since ownership is already verified)
	metrics, _ := h.reportService.GetCompletionMetrics(r.Context(), cqID, "", true)
	dimensionSummary, _ := h.reportService.GetDimensionSummary(r.Context(), cqID, "", true)
	departments, _ := h.reportService.GetDepartmentDimensions(r.Context(), cqID, "", true)
	riskProfiles, _ := h.reportService.GetRiskAnalysis(r.Context(), cqID, "", true)

	// Get company info for branding
	companyName := companyInfo.CompanyName
	questionnaireName := "Reporte"
	var companyLogo, companyColor string
	if metrics != nil {
		questionnaireName = metrics.QuestionnaireTitle
		companyName = metrics.CompanyName
	}

	// Build request payload for report-service
	payload := map[string]any{
		"questionnaire_name": questionnaireName,
		"company_name":       companyName,
		"company_logo":       companyLogo,
		"company_color":      companyColor,
		"metrics":            metrics,
		"dimension_summary":  dimensionSummary,
		"departments":        departments,
		"risk_profiles":      riskProfiles,
	}

	h.proxyToReportService(w, r, "/report-service/api/v1/pdf/report", payload)
}

// GenerateExcel handles GET /api/v1/company-api/reports/{cq_id}/excel
func (h *CompanyAPIHandler) GenerateExcel(w http.ResponseWriter, r *http.Request) {
	cqIDStr := chi.URLParam(r, "cq_id")
	cqID, err := utils.ValidateObjectID(cqIDStr)
	if err != nil {
		utils.BadRequest(w, "Invalid company questionnaire ID")
		return
	}

	companyInfo, err := middleware.GetCompanyFromContext(r.Context())
	if err != nil {
		utils.Unauthorized(w, "missing company context")
		return
	}

	if err := h.reportService.VerifyCompanyOwnership(r.Context(), cqID, companyInfo.CompanyID); err != nil {
		utils.Forbidden(w, "This questionnaire does not belong to your company")
		return
	}

	// Get free text data
	freeText, err := h.reportService.GetFreeTextResponses(r.Context(), cqID, "", true)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	metrics, _ := h.reportService.GetCompletionMetrics(r.Context(), cqID, "", true)
	questionnaireName := "Reporte"
	companyName := companyInfo.CompanyName
	if metrics != nil {
		questionnaireName = metrics.QuestionnaireTitle
		companyName = metrics.CompanyName
	}

	payload := map[string]any{
		"questionnaire_name": questionnaireName,
		"company_name":       companyName,
		"free_text_data":     freeText,
	}

	h.proxyToReportService(w, r, "/report-service/api/v1/excel/freetext", payload)
}

// proxyToReportService forwards a JSON payload to the report-service and streams the response back.
func (h *CompanyAPIHandler) proxyToReportService(w http.ResponseWriter, r *http.Request, path string, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal report payload", "error", err)
		utils.BadRequest(w, "failed to prepare report data")
		return
	}

	reportURL := fmt.Sprintf("%s%s", h.getReportServiceURL(), path)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, reportURL, bytes.NewReader(body))
	if err != nil {
		slog.Error("failed to create report-service request", "error", err)
		utils.BadRequest(w, "failed to create report request")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	// Forward the API key so report-service can authenticate
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		slog.Error("report-service call failed", "error", err, "url", reportURL)
		utils.RespondWithError(w, http.StatusBadGateway, "Report generation service unavailable")
		return
	}
	defer resp.Body.Close()

	// Stream the response back to the client
	for _, key := range []string{"Content-Type", "Content-Disposition", "Content-Length"} {
		if v := resp.Header.Get(key); v != "" {
			w.Header().Set(key, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
