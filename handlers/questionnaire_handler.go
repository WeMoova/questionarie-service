package handlers

import (
	"net/http"
	"questionarie-service/middleware"
	"questionarie-service/models"
	"questionarie-service/services"
	"questionarie-service/utils"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// QuestionnaireHandler handles questionnaire-related HTTP requests
type QuestionnaireHandler struct {
	service *services.QuestionnaireService
}

// NewQuestionnaireHandler creates a new QuestionnaireHandler
func NewQuestionnaireHandler(service *services.QuestionnaireService) *QuestionnaireHandler {
	return &QuestionnaireHandler{
		service: service,
	}
}

// CreateQuestionnaire handles POST /api/v1/questionnaires
func (h *QuestionnaireHandler) CreateQuestionnaire(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	questionnaire, err := h.service.CreateQuestionnaire(r.Context(), req.Title, req.Description, claims.Sub)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusCreated, questionnaire, "Questionnaire created successfully")
}

// GetQuestionnaires handles GET /api/v1/questionnaires
func (h *QuestionnaireHandler) GetQuestionnaires(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)
	pageSize, _ := strconv.ParseInt(r.URL.Query().Get("page_size"), 10, 64)
	activeOnly := r.URL.Query().Get("active") == "true"

	questionnaires, err := h.service.GetAllQuestionnaires(r.Context(), page, pageSize, activeOnly)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, questionnaires, "")
}

// GetQuestionnaireByID handles GET /api/v1/questionnaires/:id
func (h *QuestionnaireHandler) GetQuestionnaireByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	questionnaire, err := h.service.GetQuestionnaireByID(r.Context(), id)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, questionnaire, "")
}

// UpdateQuestionnaire handles PUT /api/v1/questionnaires/:id
func (h *QuestionnaireHandler) UpdateQuestionnaire(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		IsActive    *bool  `json:"is_active"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	if err := h.service.UpdateQuestionnaire(r.Context(), id, req.Title, req.Description, isActive); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Questionnaire updated successfully")
}

// DeactivateQuestionnaire handles DELETE /api/v1/questionnaires/:id
func (h *QuestionnaireHandler) DeactivateQuestionnaire(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	if err := h.service.DeactivateQuestionnaire(r.Context(), id); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Questionnaire deactivated successfully")
}

// AddQuestion handles POST /api/v1/questionnaires/:id/questions
func (h *QuestionnaireHandler) AddQuestion(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var req struct {
		QuestionText string                 `json:"question_text"`
		QuestionType string                 `json:"question_type"`
		Options      map[string]interface{} `json:"options"`
		OrderIndex   int                    `json:"order_index"`
		IsRequired   bool                   `json:"is_required"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	if err := utils.ValidateQuestionType(req.QuestionType); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	question := models.NewQuestion(
		req.QuestionText,
		models.QuestionType(req.QuestionType),
		req.OrderIndex,
		req.IsRequired,
	)

	if req.Options != nil {
		question.Options = req.Options
	}

	if err := h.service.AddQuestion(r.Context(), id, *question); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusCreated, question, "Question added successfully")
}

// UpdateQuestion handles PUT /api/v1/questionnaires/:id/questions/:question_id
func (h *QuestionnaireHandler) UpdateQuestion(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	questionID := chi.URLParam(r, "question_id")
	if questionID == "" {
		utils.BadRequest(w, "question_id is required")
		return
	}

	var req struct {
		QuestionText string                 `json:"question_text"`
		QuestionType string                 `json:"question_type"`
		Options      map[string]interface{} `json:"options"`
		OrderIndex   int                    `json:"order_index"`
		IsRequired   bool                   `json:"is_required"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	question := models.Question{
		QuestionID:   questionID,
		QuestionText: req.QuestionText,
		QuestionType: models.QuestionType(req.QuestionType),
		Options:      req.Options,
		OrderIndex:   req.OrderIndex,
		IsRequired:   req.IsRequired,
	}

	if err := h.service.UpdateQuestion(r.Context(), id, questionID, question); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Question updated successfully")
}

// RemoveQuestion handles DELETE /api/v1/questionnaires/:id/questions/:question_id
func (h *QuestionnaireHandler) RemoveQuestion(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	questionID := chi.URLParam(r, "question_id")
	if questionID == "" {
		utils.BadRequest(w, "question_id is required")
		return
	}

	if err := h.service.RemoveQuestion(r.Context(), id, questionID); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Question removed successfully")
}
