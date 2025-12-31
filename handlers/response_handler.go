package handlers

import (
	"net/http"
	"questionarie-service/middleware"
	"questionarie-service/services"
	"questionarie-service/utils"

	"github.com/go-chi/chi/v5"
)

// ResponseHandler handles response-related HTTP requests
type ResponseHandler struct {
	service *services.AssignmentService
}

// NewResponseHandler creates a new ResponseHandler
func NewResponseHandler(service *services.AssignmentService) *ResponseHandler {
	return &ResponseHandler{
		service: service,
	}
}

// SaveResponse handles POST /api/v1/assignments/:id/responses
func (h *ResponseHandler) SaveResponse(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var req struct {
		QuestionID    string      `json:"question_id"`
		ResponseValue interface{} `json:"response_value"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	if req.QuestionID == "" {
		utils.BadRequest(w, "question_id is required")
		return
	}

	if req.ResponseValue == nil {
		utils.BadRequest(w, "response_value is required")
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())

	if err := h.service.SaveResponse(r.Context(), id, claims.Sub, req.QuestionID, req.ResponseValue); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Response saved successfully")
}

// UpdateResponses handles PUT /api/v1/assignments/:id/responses
func (h *ResponseHandler) UpdateResponses(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var req struct {
		Responses []struct {
			QuestionID    string      `json:"question_id"`
			ResponseValue interface{} `json:"response_value"`
		} `json:"responses"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	if len(req.Responses) == 0 {
		utils.BadRequest(w, "responses cannot be empty")
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())

	// Save each response
	for _, response := range req.Responses {
		if err := h.service.SaveResponse(r.Context(), id, claims.Sub, response.QuestionID, response.ResponseValue); err != nil {
			utils.HandleRepositoryError(w, err)
			return
		}
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Responses updated successfully")
}

// SubmitAssignment handles POST /api/v1/assignments/:id/submit
func (h *ResponseHandler) SubmitAssignment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())

	if err := h.service.SubmitAssignment(r.Context(), id, claims.Sub); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Assignment submitted successfully")
}
