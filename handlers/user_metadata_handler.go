package handlers

import (
	"net/http"
	"questionarie-service/services"
	"questionarie-service/utils"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// UserMetadataHandler handles user metadata-related HTTP requests
type UserMetadataHandler struct {
	service *services.UserMetadataService
}

// NewUserMetadataHandler creates a new UserMetadataHandler
func NewUserMetadataHandler(service *services.UserMetadataService) *UserMetadataHandler {
	return &UserMetadataHandler{
		service: service,
	}
}

// CreateUserMetadata handles POST /api/v1/users/metadata
func (h *UserMetadataHandler) CreateUserMetadata(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID       string `json:"user_id"`
		CompanyID    string `json:"company_id"`
		SupervisorID string `json:"supervisor_id"`
		Department   string `json:"department"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	if req.UserID == "" {
		utils.BadRequest(w, "user_id is required")
		return
	}

	if req.CompanyID == "" {
		utils.BadRequest(w, "company_id is required")
		return
	}

	companyID, err := utils.ValidateObjectID(req.CompanyID)
	if err != nil {
		utils.BadRequest(w, "invalid company_id: "+err.Error())
		return
	}

	metadata, err := h.service.CreateUserMetadata(r.Context(), req.UserID, companyID, req.SupervisorID, req.Department)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusCreated, metadata, "User metadata created successfully")
}

// GetUserMetadata handles GET /api/v1/users/metadata/:user_id
func (h *UserMetadataHandler) GetUserMetadata(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		utils.BadRequest(w, "user_id is required")
		return
	}

	metadata, err := h.service.GetUserMetadata(r.Context(), userID)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, metadata, "")
}

// UpdateUserMetadata handles PUT /api/v1/users/metadata/:user_id
func (h *UserMetadataHandler) UpdateUserMetadata(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		utils.BadRequest(w, "user_id is required")
		return
	}

	var req struct {
		CompanyID    string `json:"company_id"`
		SupervisorID string `json:"supervisor_id"`
		Department   string `json:"department"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	companyID, err := utils.ValidateObjectID(req.CompanyID)
	if err != nil {
		utils.BadRequest(w, "invalid company_id: "+err.Error())
		return
	}

	if err := h.service.UpdateUserMetadata(r.Context(), userID, companyID, req.SupervisorID, req.Department); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "User metadata updated successfully")
}

// DeleteUserMetadata handles DELETE /api/v1/users/metadata/:user_id
func (h *UserMetadataHandler) DeleteUserMetadata(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		utils.BadRequest(w, "user_id is required")
		return
	}

	if err := h.service.DeleteUserMetadata(r.Context(), userID); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "User metadata deleted successfully")
}

// GetUsersByCompany handles GET /api/v1/companies/:company_id/users
func (h *UserMetadataHandler) GetUsersByCompany(w http.ResponseWriter, r *http.Request) {
	companyIDStr := chi.URLParam(r, "company_id")
	companyID, err := utils.ValidateObjectID(companyIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)
	pageSize, _ := strconv.ParseInt(r.URL.Query().Get("page_size"), 10, 64)

	users, err := h.service.GetUsersByCompany(r.Context(), companyID, page, pageSize)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, users, "")
}
