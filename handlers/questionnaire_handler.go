package handlers

import (
	"net/http"
	"questionarie-service/middleware"
	"questionarie-service/models"
	"questionarie-service/services"
	"questionarie-service/utils"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
		Title       string   `json:"title"`
		Description string   `json:"description"`
		CoverImage  string   `json:"cover_image"`
		Tags        []string `json:"tags"`
		CategoryID  string   `json:"category_id"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var categoryID *primitive.ObjectID
	if req.CategoryID != "" {
		id, err := utils.ValidateObjectID(req.CategoryID)
		if err != nil {
			utils.BadRequest(w, "invalid category_id: "+err.Error())
			return
		}
		categoryID = &id
	}

	claims, _ := middleware.GetUserFromContext(r.Context())
	questionnaire, err := h.service.CreateQuestionnaire(r.Context(), req.Title, req.Description, claims.Sub, req.Tags, categoryID, req.CoverImage)
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
		Title       string   `json:"title"`
		Description string   `json:"description"`
		CoverImage  *string  `json:"cover_image"`
		IsActive    *bool    `json:"is_active"`
		Tags        []string `json:"tags"`
		CategoryID  string   `json:"category_id"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	var categoryID *primitive.ObjectID
	if req.CategoryID != "" {
		cid, err := utils.ValidateObjectID(req.CategoryID)
		if err != nil {
			utils.BadRequest(w, "invalid category_id: "+err.Error())
			return
		}
		categoryID = &cid
	}

	if err := h.service.UpdateQuestionnaire(r.Context(), id, req.Title, req.Description, isActive, req.Tags, categoryID, req.CoverImage); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Questionnaire updated successfully")
}

// GetQuestionnaireStats handles GET /api/v1/questionnaires/:id/stats
func (h *QuestionnaireHandler) GetQuestionnaireStats(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	stats, err := h.service.GetQuestionnaireUsageStats(r.Context(), id)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, stats, "")
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
		QuestionText  string                 `json:"question_text"`
		QuestionType  string                 `json:"question_type"`
		ImageURL      string                 `json:"image_url"`
		Options       map[string]interface{} `json:"options"`
		OrderIndex    int                    `json:"order_index"`
		IsRequired    bool                   `json:"is_required"`
		SectionID     string                 `json:"section_id"`
		DimensionCode string                 `json:"dimension_code"`
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
	question.ImageURL = req.ImageURL
	question.SectionID = req.SectionID
	question.DimensionCode = req.DimensionCode

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
		QuestionText  string                 `json:"question_text"`
		QuestionType  string                 `json:"question_type"`
		ImageURL      string                 `json:"image_url"`
		Options       map[string]interface{} `json:"options"`
		OrderIndex    int                    `json:"order_index"`
		IsRequired    bool                   `json:"is_required"`
		SectionID     string                 `json:"section_id"`
		DimensionCode string                 `json:"dimension_code"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	question := models.Question{
		QuestionID:    questionID,
		QuestionText:  req.QuestionText,
		QuestionType:  models.QuestionType(req.QuestionType),
		ImageURL:      req.ImageURL,
		Options:       req.Options,
		OrderIndex:    req.OrderIndex,
		IsRequired:    req.IsRequired,
		SectionID:     req.SectionID,
		DimensionCode: req.DimensionCode,
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

// AddSection handles POST /api/v1/questionnaires/:id/sections
func (h *QuestionnaireHandler) AddSection(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		OrderIndex  int    `json:"order_index"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	if req.Title == "" {
		utils.BadRequest(w, "title is required")
		return
	}

	section := models.Section{
		ID:          primitive.NewObjectID().Hex(),
		Title:       req.Title,
		Description: req.Description,
		OrderIndex:  req.OrderIndex,
	}

	if err := h.service.AddSection(r.Context(), id, section); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusCreated, section, "Section added successfully")
}

// UpdateSection handles PUT /api/v1/questionnaires/:id/sections/:section_id
func (h *QuestionnaireHandler) UpdateSection(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	sectionID := chi.URLParam(r, "section_id")
	if sectionID == "" {
		utils.BadRequest(w, "section_id is required")
		return
	}

	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		OrderIndex  int    `json:"order_index"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	section := models.Section{
		ID:          sectionID,
		Title:       req.Title,
		Description: req.Description,
		OrderIndex:  req.OrderIndex,
	}

	if err := h.service.UpdateSection(r.Context(), id, sectionID, section); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Section updated successfully")
}

// DeleteSection handles DELETE /api/v1/questionnaires/:id/sections/:section_id
func (h *QuestionnaireHandler) DeleteSection(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	sectionID := chi.URLParam(r, "section_id")
	if sectionID == "" {
		utils.BadRequest(w, "section_id is required")
		return
	}

	if err := h.service.DeleteSection(r.Context(), id, sectionID); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Section deleted successfully")
}

// ImportQuestionnaire handles POST /api/v1/questionnaires/import
// Creates a complete questionnaire with sections, questions, and evaluation config in one call
func (h *QuestionnaireHandler) ImportQuestionnaire(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title            string                  `json:"title"`
		Description      string                  `json:"description"`
		CoverImage       string                  `json:"cover_image"`
		Tags             []string                `json:"tags"`
		CategoryID       string                  `json:"category_id"`
		Sections         []models.Section        `json:"sections"`
		Questions        []models.Question       `json:"questions"`
		EvaluationConfig *models.EvaluationConfig `json:"evaluation_config"`
	}

	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var categoryID *primitive.ObjectID
	if req.CategoryID != "" {
		id, err := utils.ValidateObjectID(req.CategoryID)
		if err != nil {
			utils.BadRequest(w, "invalid category_id: "+err.Error())
			return
		}
		categoryID = &id
	}

	claims, _ := middleware.GetUserFromContext(r.Context())

	questionnaire, err := h.service.ImportQuestionnaire(
		r.Context(),
		req.Title,
		req.Description,
		claims.Sub,
		req.Tags,
		categoryID,
		req.CoverImage,
		req.Sections,
		req.Questions,
		req.EvaluationConfig,
	)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusCreated, questionnaire, "Questionnaire imported successfully")
}

// SetEvaluationConfig handles PUT /api/v1/questionnaires/:id/evaluation-config
func (h *QuestionnaireHandler) SetEvaluationConfig(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var config models.EvaluationConfig
	if err := utils.ParseRequestBody(r, &config); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	if err := h.service.SetEvaluationConfig(r.Context(), id, &config); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Evaluation config updated successfully")
}
