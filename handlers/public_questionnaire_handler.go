package handlers

import (
	"net/http"
	"questionarie-service/middleware"
	"questionarie-service/models"
	"questionarie-service/services"
	"questionarie-service/utils"

	"github.com/go-chi/chi/v5"
)

// PublicQuestionnaireHandler handles public (anonymous) questionnaire endpoints.
type PublicQuestionnaireHandler struct {
	service *services.PublicLinkService
}

// NewPublicQuestionnaireHandler creates a new PublicQuestionnaireHandler.
func NewPublicQuestionnaireHandler(service *services.PublicLinkService) *PublicQuestionnaireHandler {
	return &PublicQuestionnaireHandler{service: service}
}

// GetLinkInfo handles GET /api/v1/public/q/{slug}
// Returns public-facing metadata for a link (no auth required).
func (h *PublicQuestionnaireHandler) GetLinkInfo(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		utils.BadRequest(w, "slug is required")
		return
	}

	info, err := h.service.GetPublicLinkInfo(r.Context(), slug)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, info, "")
}

// GetQuestions handles GET /api/v1/public/q/{slug}/questions
// Returns the questions for the questionnaire behind the given slug (no auth required).
func (h *PublicQuestionnaireHandler) GetQuestions(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		utils.BadRequest(w, "slug is required")
		return
	}

	questions, err := h.service.GetPublicQuestions(r.Context(), slug)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, questions, "")
}

// StartSession handles POST /api/v1/public/q/{slug}/start
// Creates an anonymous assignment and returns a session token (no auth required).
func (h *PublicQuestionnaireHandler) StartSession(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		utils.BadRequest(w, "slug is required")
		return
	}

	var req struct {
		DemographicData map[string]string `json:"demographic_data"`
	}
	if err := utils.ParseRequestBody(r, &req); err != nil {
		// Allow empty body — demographic data is optional
		req.DemographicData = map[string]string{}
	}

	result, err := h.service.StartAnonymousSession(r.Context(), slug, req.DemographicData)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusCreated, result, "Session started")
}

// SaveResponses handles POST /api/v1/public/q/{slug}/responses
// Saves responses for an anonymous session (requires SessionTokenAuth middleware).
func (h *PublicQuestionnaireHandler) SaveResponses(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetSessionFromContext(r.Context())
	if claims == nil {
		utils.Unauthorized(w, "session token required")
		return
	}

	var req struct {
		Responses []models.Response `json:"responses"`
	}
	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	if len(req.Responses) == 0 {
		utils.BadRequest(w, "responses cannot be empty")
		return
	}

	if err := h.service.SaveAnonymousResponses(r.Context(), claims, req.Responses); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Responses saved successfully")
}

// SubmitSession handles POST /api/v1/public/q/{slug}/submit
// Marks the anonymous assignment as completed (requires SessionTokenAuth middleware).
func (h *PublicQuestionnaireHandler) SubmitSession(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetSessionFromContext(r.Context())
	if claims == nil {
		utils.Unauthorized(w, "session token required")
		return
	}

	result, err := h.service.SubmitAnonymous(r.Context(), claims)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, result, "Questionnaire submitted successfully")
}
