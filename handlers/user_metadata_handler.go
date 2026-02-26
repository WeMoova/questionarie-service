package handlers

import (
	"net/http"
	"questionarie-service/middleware"
	"questionarie-service/services"
	"questionarie-service/utils"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
		UserID         string `json:"user_id"`
		CompanyID      string `json:"company_id"`
		SupervisorID   string `json:"supervisor_id"`
		Department     string `json:"department"`
		DocumentType   string `json:"document_type"`
		DocumentNumber string `json:"document_number"`
		Phone          string `json:"phone"`
		Position       string `json:"position"`
		EmployeeCode   string `json:"employee_code"`
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

	metadata, err := h.service.CreateUserMetadata(r.Context(), req.UserID, companyID, req.SupervisorID, req.Department, req.DocumentType, req.DocumentNumber, req.Phone, req.Position, req.EmployeeCode)
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
		CompanyID      string `json:"company_id"`
		SupervisorID   string `json:"supervisor_id"`
		Department     string `json:"department"`
		DocumentType   string `json:"document_type"`
		DocumentNumber string `json:"document_number"`
		Phone          string `json:"phone"`
		Position       string `json:"position"`
		EmployeeCode   string `json:"employee_code"`
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

	if err := h.service.UpdateUserMetadata(r.Context(), userID, companyID, req.SupervisorID, req.Department, req.DocumentType, req.DocumentNumber, req.Phone, req.Position, req.EmployeeCode); err != nil {
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

// GetMyMetadata handles GET /api/v1/users/me/metadata
func (h *UserMetadataHandler) GetMyMetadata(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetUserFromContext(r.Context())

	metadata, err := h.service.GetUserMetadata(r.Context(), claims.Sub)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	// Build enhanced response with role from JWT
	response := map[string]interface{}{
		"user_id":    metadata.ID,
		"company_id": metadata.CompanyID.Hex(),
		"created_at": metadata.CreatedAt,
	}

	// Add role from JWT (first role)
	if len(claims.Roles) > 0 {
		response["role"] = claims.Roles[0]
	}

	// Add optional fields
	if metadata.SupervisorID != "" {
		response["supervisor_id"] = metadata.SupervisorID
	}
	if metadata.Department != "" {
		response["department"] = metadata.Department
	}

	utils.RespondWithSuccess(w, http.StatusOK, response, "")
}

// ResolveByDocuments resolves document numbers to FusionAuth user IDs
func (h *UserMetadataHandler) ResolveByDocuments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, _ := middleware.GetUserFromContext(ctx)
	isSuperAdmin := middleware.IsSuperAdmin(ctx)

	var req struct {
		DocumentNumbers []string `json:"document_numbers"`
		CompanyID       string   `json:"company_id"`
	}
	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, "invalid request body")
		return
	}
	if len(req.DocumentNumbers) == 0 {
		utils.BadRequest(w, "document_numbers cannot be empty")
		return
	}

	var companyID primitive.ObjectID
	if req.CompanyID != "" {
		var err error
		companyID, err = primitive.ObjectIDFromHex(req.CompanyID)
		if err != nil {
			utils.BadRequest(w, "invalid company_id")
			return
		}
	}

	resolved, notFound, err := h.service.ResolveByDocuments(ctx, claims.Sub, isSuperAdmin, companyID, req.DocumentNumbers)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, map[string]interface{}{
		"resolved":  resolved,
		"not_found": notFound,
	}, "")
}

// ListUsers handles GET /api/v1/users/metadata
func (h *UserMetadataHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	companyIDStr := r.URL.Query().Get("company_id")
	supervisorID := r.URL.Query().Get("supervisor_id")
	department := r.URL.Query().Get("department")
	page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)
	pageSize, _ := strconv.ParseInt(r.URL.Query().Get("page_size"), 10, 64)

	// Parse company_id if provided
	var companyID *primitive.ObjectID
	if companyIDStr != "" {
		cid, err := utils.ValidateObjectID(companyIDStr)
		if err != nil {
			utils.BadRequest(w, "invalid company_id: "+err.Error())
			return
		}
		companyID = &cid
	}

	// Parse supervisor_id if provided
	var supervisorIDPtr *string
	if supervisorID != "" {
		supervisorIDPtr = &supervisorID
	}

	// Parse department if provided
	var departmentPtr *string
	if department != "" {
		departmentPtr = &department
	}

	// Call service with role-based filtering
	users, total, err := h.service.ListUsersWithFilters(r.Context(), companyID, supervisorIDPtr, departmentPtr, page, pageSize)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	// Build response
	response := map[string]interface{}{
		"users": users,
		"pagination": map[string]interface{}{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_pages": (total + pageSize - 1) / pageSize,
		},
	}

	utils.RespondWithSuccess(w, http.StatusOK, response, "")
}
