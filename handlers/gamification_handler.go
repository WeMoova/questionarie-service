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

// GamificationHandler handles gamification-related HTTP requests
type GamificationHandler struct {
	service *services.GamificationService
}

// NewGamificationHandler creates a new GamificationHandler
func NewGamificationHandler(service *services.GamificationService) *GamificationHandler {
	return &GamificationHandler{service: service}
}

// ── Point Rules ─────────────────────────────────────────────────────────────

func (h *GamificationHandler) GetPointRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.service.GetPointRules(r.Context())
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}
	utils.RespondWithSuccess(w, http.StatusOK, rules, "")
}

func (h *GamificationHandler) UpdatePointRule(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ValidateObjectID(chi.URLParam(r, "id"))
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Points      int    `json:"points"`
		IsActive    *bool  `json:"is_active"`
	}
	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	rule := &models.PointRule{
		Name:        req.Name,
		Description: req.Description,
		Points:      req.Points,
		IsActive:    true,
	}
	if req.IsActive != nil {
		rule.IsActive = *req.IsActive
	}

	if err := h.service.UpdatePointRule(r.Context(), id, rule); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}
	utils.RespondWithSuccess(w, http.StatusOK, nil, "Point rule updated successfully")
}

// ── Badges ──────────────────────────────────────────────────────────────────

func (h *GamificationHandler) GetBadges(w http.ResponseWriter, r *http.Request) {
	badges, err := h.service.GetBadges(r.Context())
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}
	utils.RespondWithSuccess(w, http.StatusOK, badges, "")
}

func (h *GamificationHandler) CreateBadge(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string               `json:"name"`
		Description string               `json:"description"`
		IconURL     string               `json:"icon_url"`
		Criteria    models.BadgeCriteria  `json:"criteria"`
	}
	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	if req.Name == "" {
		utils.BadRequest(w, "name is required")
		return
	}

	badge := &models.Badge{
		ID:          primitive.NewObjectID(),
		Name:        req.Name,
		Description: req.Description,
		IconURL:     req.IconURL,
		Criteria:    req.Criteria,
		IsActive:    true,
	}

	if err := h.service.CreateBadge(r.Context(), badge); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}
	utils.RespondWithSuccess(w, http.StatusCreated, badge, "Badge created successfully")
}

func (h *GamificationHandler) UpdateBadge(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ValidateObjectID(chi.URLParam(r, "id"))
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var req struct {
		Name        string               `json:"name"`
		Description string               `json:"description"`
		IconURL     string               `json:"icon_url"`
		Criteria    models.BadgeCriteria  `json:"criteria"`
		IsActive    *bool                 `json:"is_active"`
	}
	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	badge := &models.Badge{
		Name:        req.Name,
		Description: req.Description,
		IconURL:     req.IconURL,
		Criteria:    req.Criteria,
		IsActive:    true,
	}
	if req.IsActive != nil {
		badge.IsActive = *req.IsActive
	}

	if err := h.service.UpdateBadge(r.Context(), id, badge); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}
	utils.RespondWithSuccess(w, http.StatusOK, nil, "Badge updated successfully")
}

// ── Achievements ────────────────────────────────────────────────────────────

func (h *GamificationHandler) GetAchievements(w http.ResponseWriter, r *http.Request) {
	achievements, err := h.service.GetAchievements(r.Context())
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}
	utils.RespondWithSuccess(w, http.StatusOK, achievements, "")
}

func (h *GamificationHandler) CreateAchievement(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string                     `json:"name"`
		Description string                     `json:"description"`
		IconURL     string                     `json:"icon_url"`
		Category    models.AchievementCategory `json:"category"`
		Threshold   int                        `json:"threshold"`
		PointReward int                        `json:"point_reward"`
	}
	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	if req.Name == "" {
		utils.BadRequest(w, "name is required")
		return
	}

	achievement := &models.Achievement{
		ID:          primitive.NewObjectID(),
		Name:        req.Name,
		Description: req.Description,
		IconURL:     req.IconURL,
		Category:    req.Category,
		Threshold:   req.Threshold,
		PointReward: req.PointReward,
		IsActive:    true,
	}

	if err := h.service.CreateAchievement(r.Context(), achievement); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}
	utils.RespondWithSuccess(w, http.StatusCreated, achievement, "Achievement created successfully")
}

func (h *GamificationHandler) UpdateAchievement(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ValidateObjectID(chi.URLParam(r, "id"))
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var req struct {
		Name        string                     `json:"name"`
		Description string                     `json:"description"`
		IconURL     string                     `json:"icon_url"`
		Category    models.AchievementCategory `json:"category"`
		Threshold   int                        `json:"threshold"`
		PointReward int                        `json:"point_reward"`
		IsActive    *bool                      `json:"is_active"`
	}
	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	achievement := &models.Achievement{
		Name:        req.Name,
		Description: req.Description,
		IconURL:     req.IconURL,
		Category:    req.Category,
		Threshold:   req.Threshold,
		PointReward: req.PointReward,
		IsActive:    true,
	}
	if req.IsActive != nil {
		achievement.IsActive = *req.IsActive
	}

	if err := h.service.UpdateAchievement(r.Context(), id, achievement); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}
	utils.RespondWithSuccess(w, http.StatusOK, nil, "Achievement updated successfully")
}

// ── User-facing ─────────────────────────────────────────────────────────────

func (h *GamificationHandler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetUserFromContext(r.Context())
	profile, err := h.service.GetUserProfile(r.Context(), claims.Sub)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}
	utils.RespondWithSuccess(w, http.StatusOK, profile, "")
}

func (h *GamificationHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	companyIDStr := r.URL.Query().Get("company_id")
	var companyID *primitive.ObjectID
	if companyIDStr != "" {
		id, err := utils.ValidateObjectID(companyIDStr)
		if err != nil {
			utils.BadRequest(w, "invalid company_id")
			return
		}
		companyID = &id
	}

	entries, err := h.service.GetLeaderboard(r.Context(), companyID, limit)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}
	utils.RespondWithSuccess(w, http.StatusOK, entries, "")
}

// ── Admin views ─────────────────────────────────────────────────────────────

func (h *GamificationHandler) GetCompanyLeaderboard(w http.ResponseWriter, r *http.Request) {
	companyID, err := utils.ValidateObjectID(chi.URLParam(r, "company_id"))
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	entries, err := h.service.GetLeaderboard(r.Context(), &companyID, limit)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}
	utils.RespondWithSuccess(w, http.StatusOK, entries, "")
}

func (h *GamificationHandler) GetUserProfile(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		utils.BadRequest(w, "user_id is required")
		return
	}

	profile, err := h.service.GetUserProfile(r.Context(), userID)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}
	utils.RespondWithSuccess(w, http.StatusOK, profile, "")
}
