package handlers

import (
	"net/http"
	"questionarie-service/middleware"
	"questionarie-service/services"
	"questionarie-service/utils"

	"github.com/go-chi/chi/v5"
)

// APITokenHandler handles API token HTTP requests
type APITokenHandler struct {
	service *services.APITokenService
}

// NewAPITokenHandler creates a new APITokenHandler
func NewAPITokenHandler(service *services.APITokenService) *APITokenHandler {
	return &APITokenHandler{service: service}
}

// GenerateToken handles POST /api/v1/companies/{company_id}/api-tokens
func (h *APITokenHandler) GenerateToken(w http.ResponseWriter, r *http.Request) {
	companyIDStr := chi.URLParam(r, "company_id")
	companyID, err := utils.ValidateObjectID(companyIDStr)
	if err != nil {
		utils.BadRequest(w, "Invalid company ID")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
	if req.Name == "" {
		utils.BadRequest(w, "Token name is required")
		return
	}

	claims, err := middleware.GetUserFromContext(r.Context())
	if err != nil {
		utils.Unauthorized(w, "unauthorized")
		return
	}

	token, err := h.service.GenerateToken(r.Context(), companyID, req.Name, claims.Sub)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusCreated, token, "API token generated successfully")
}

// ListTokens handles GET /api/v1/companies/{company_id}/api-tokens
func (h *APITokenHandler) ListTokens(w http.ResponseWriter, r *http.Request) {
	companyIDStr := chi.URLParam(r, "company_id")
	companyID, err := utils.ValidateObjectID(companyIDStr)
	if err != nil {
		utils.BadRequest(w, "Invalid company ID")
		return
	}

	tokens, err := h.service.ListByCompany(r.Context(), companyID)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, tokens, "")
}

// RevokeToken handles DELETE /api/v1/api-tokens/{token_id}
func (h *APITokenHandler) RevokeToken(w http.ResponseWriter, r *http.Request) {
	tokenIDStr := chi.URLParam(r, "token_id")
	tokenID, err := utils.ValidateObjectID(tokenIDStr)
	if err != nil {
		utils.BadRequest(w, "Invalid token ID")
		return
	}

	if err := h.service.RevokeToken(r.Context(), tokenID); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "API token revoked successfully")
}

// ToggleToken handles PATCH /api/v1/api-tokens/{token_id}/toggle
func (h *APITokenHandler) ToggleToken(w http.ResponseWriter, r *http.Request) {
	tokenIDStr := chi.URLParam(r, "token_id")
	tokenID, err := utils.ValidateObjectID(tokenIDStr)
	if err != nil {
		utils.BadRequest(w, "Invalid token ID")
		return
	}

	if err := h.service.ToggleToken(r.Context(), tokenID); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "API token toggled successfully")
}
