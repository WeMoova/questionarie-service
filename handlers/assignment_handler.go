package handlers

import (
	"net/http"
	"questionarie-service/middleware"
	"questionarie-service/models"
	"questionarie-service/services"
	"questionarie-service/utils"

	"github.com/go-chi/chi/v5"
)

// AssignmentHandler handles assignment-related HTTP requests
type AssignmentHandler struct {
	service *services.AssignmentService
}

// NewAssignmentHandler creates a new AssignmentHandler
func NewAssignmentHandler(service *services.AssignmentService) *AssignmentHandler {
	return &AssignmentHandler{
		service: service,
	}
}

// AssignToUsers handles POST /api/v1/company-questionnaires/:cq_id/assignments
func (h *AssignmentHandler) AssignToUsers(w http.ResponseWriter, r *http.Request) {
	cqIDStr := chi.URLParam(r, "cq_id")
	cqID, err := utils.ValidateObjectID(cqIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var req struct {
		UserIDs []string `json:"user_ids"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	if len(req.UserIDs) == 0 {
		utils.BadRequest(w, "user_ids cannot be empty")
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	assignments, err := h.service.AssignToUsers(r.Context(), claims.Sub, cqID, req.UserIDs, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusCreated, map[string]interface{}{
		"assignments":       assignments,
		"total_created":     len(assignments),
		"total_requested":   len(req.UserIDs),
	}, "Users assigned successfully")
}

// GetAssignmentsByCompanyQuestionnaire handles GET /api/v1/company-questionnaires/:cq_id/assignments
func (h *AssignmentHandler) GetAssignmentsByCompanyQuestionnaire(w http.ResponseWriter, r *http.Request) {
	cqIDStr := chi.URLParam(r, "cq_id")
	cqID, err := utils.ValidateObjectID(cqIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	assignments, err := h.service.GetCompanyQuestionnaireAssignments(r.Context(), cqID)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, assignments, "")
}

// GetMyAssignments handles GET /api/v1/my-assignments
func (h *AssignmentHandler) GetMyAssignments(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetUserFromContext(r.Context())

	statusStr := r.URL.Query().Get("status")
	var status *models.AssignmentStatus
	if statusStr != "" {
		if err := utils.ValidateAssignmentStatus(statusStr); err != nil {
			utils.BadRequest(w, err.Error())
			return
		}
		s := models.AssignmentStatus(statusStr)
		status = &s
	}

	assignments, err := h.service.GetUserAssignments(r.Context(), claims.Sub, status)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, assignments, "")
}

// GetAssignmentByID handles GET /api/v1/assignments/:id
func (h *AssignmentHandler) GetAssignmentByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	assignment, err := h.service.GetAssignmentByID(r.Context(), id)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	// Verify ownership unless super admin
	claims, _ := middleware.GetUserFromContext(r.Context())
	if !middleware.IsSuperAdmin(r.Context()) && assignment.UserID != claims.Sub {
		utils.Forbidden(w, "unauthorized to access this assignment")
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, assignment, "")
}

// GetMyCompanyQuestionnaires handles GET /api/v1/my-company/questionnaires
func (h *AssignmentHandler) GetMyCompanyQuestionnaires(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetUserFromContext(r.Context())

	questionnaires, err := h.service.GetMyCompanyQuestionnaires(r.Context(), claims.Sub)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, questionnaires, "")
}

// GetMyTeamAssignments handles GET /api/v1/my-team/assignments
func (h *AssignmentHandler) GetMyTeamAssignments(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetUserFromContext(r.Context())

	assignments, err := h.service.GetMyTeamAssignments(r.Context(), claims.Sub)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, assignments, "")
}
