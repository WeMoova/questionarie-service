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

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	assignments, err := h.service.GetCompanyQuestionnaireAssignments(r.Context(), cqID, claims.Sub, isSuperAdmin)
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

// GetAssignmentQuestions handles GET /api/v1/assignments/:id/questions
func (h *AssignmentHandler) GetAssignmentQuestions(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	// Verify ownership: get assignment first
	assignment, err := h.service.GetAssignmentByID(r.Context(), id)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	if !middleware.IsSuperAdmin(r.Context()) && assignment.UserID != claims.Sub {
		utils.Forbidden(w, "unauthorized to access this assignment")
		return
	}

	questions, err := h.service.GetAssignmentQuestions(r.Context(), id)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, questions, "")
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

// CancelAssignment handles POST /api/v1/assignments/:id/cancel
func (h *AssignmentHandler) CancelAssignment(w http.ResponseWriter, r *http.Request) {
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

	if err := h.service.CancelAssignment(r.Context(), id, claims.Sub, req.Reason, isSuperAdmin); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Assignment cancelled successfully")
}

// CancelAllAssignments handles DELETE /api/v1/company-questionnaires/:cq_id/assignments
func (h *AssignmentHandler) CancelAllAssignments(w http.ResponseWriter, r *http.Request) {
	cqIDStr := chi.URLParam(r, "cq_id")
	cqID, err := utils.ValidateObjectID(cqIDStr)
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

	count, err := h.service.CancelAllPendingByCompanyQuestionnaire(r.Context(), cqID, claims.Sub, req.Reason, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, map[string]interface{}{
		"cancelled_count": count,
	}, "Assignments cancelled successfully")
}

// GetAssignmentProgress handles GET /api/v1/company-questionnaires/:cq_id/progress
func (h *AssignmentHandler) GetAssignmentProgress(w http.ResponseWriter, r *http.Request) {
	cqIDStr := chi.URLParam(r, "cq_id")
	cqID, err := utils.ValidateObjectID(cqIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	progress, err := h.service.GetAssignmentProgress(r.Context(), cqID, claims.Sub, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, progress, "")
}

// GetPendingUsers handles GET /api/v1/company-questionnaires/:cq_id/pending-users
func (h *AssignmentHandler) GetPendingUsers(w http.ResponseWriter, r *http.Request) {
	h.getAssignmentsByStatus(w, r, models.AssignmentStatusPending)
}

// GetInProgressUsers handles GET /api/v1/company-questionnaires/:cq_id/in-progress-users
func (h *AssignmentHandler) GetInProgressUsers(w http.ResponseWriter, r *http.Request) {
	h.getAssignmentsByStatus(w, r, models.AssignmentStatusInProgress)
}

// GetCompletedUsers handles GET /api/v1/company-questionnaires/:cq_id/completed-users
func (h *AssignmentHandler) GetCompletedUsers(w http.ResponseWriter, r *http.Request) {
	h.getAssignmentsByStatus(w, r, models.AssignmentStatusCompleted)
}

func (h *AssignmentHandler) getAssignmentsByStatus(w http.ResponseWriter, r *http.Request, status models.AssignmentStatus) {
	cqIDStr := chi.URLParam(r, "cq_id")
	cqID, err := utils.ValidateObjectID(cqIDStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	isSuperAdmin := middleware.IsSuperAdmin(r.Context())

	assignments, err := h.service.GetAssignmentsByStatus(r.Context(), cqID, status, claims.Sub, isSuperAdmin)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, assignments, "")
}
