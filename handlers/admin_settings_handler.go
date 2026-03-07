package handlers

import (
	"net/http"
	"questionarie-service/middleware"
	"questionarie-service/models"
	"questionarie-service/repository"
	"questionarie-service/utils"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// AdminSettingsHandler handles admin settings HTTP requests
type AdminSettingsHandler struct {
	repo *repository.AdminSettingsRepository
}

// NewAdminSettingsHandler creates a new AdminSettingsHandler
func NewAdminSettingsHandler(repo *repository.AdminSettingsRepository) *AdminSettingsHandler {
	return &AdminSettingsHandler{repo: repo}
}

// GetSettings handles GET /api/v1/public/admin-settings
// Public endpoint — returns the active theme. If a preset is scheduled for today, its theme is returned instead.
func (h *AdminSettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.repo.Get(r.Context())
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	activeTheme := settings.ActiveTheme

	// Check if any preset is scheduled for today
	today := time.Now().Format("2006-01-02")
	for _, preset := range settings.Presets {
		if preset.ScheduleStart != nil && preset.ScheduleEnd != nil {
			if today >= *preset.ScheduleStart && today <= *preset.ScheduleEnd {
				activeTheme = preset.Theme
				break
			}
		}
	}

	utils.RespondWithSuccess(w, http.StatusOK, activeTheme, "")
}

// GetSettingsFull handles GET /api/v1/admin/settings
// Protected endpoint — returns full settings including all presets.
func (h *AdminSettingsHandler) GetSettingsFull(w http.ResponseWriter, r *http.Request) {
	settings, err := h.repo.Get(r.Context())
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, settings, "")
}

// UpdateTheme handles PUT /api/v1/admin/settings/theme
// Updates the active theme.
func (h *AdminSettingsHandler) UpdateTheme(w http.ResponseWriter, r *http.Request) {
	var theme models.AdminTheme
	if err := utils.ParseRequestBody(r, &theme); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	claims, err := middleware.GetUserFromContext(r.Context())
	if err != nil {
		utils.Unauthorized(w, "unauthorized")
		return
	}

	if err := h.repo.UpdateTheme(r.Context(), theme, claims.Sub); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Theme updated successfully")
}

// CreatePreset handles POST /api/v1/admin/settings/presets
// Creates a new theme preset.
func (h *AdminSettingsHandler) CreatePreset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name          string           `json:"name"`
		Theme         models.AdminTheme `json:"theme"`
		ScheduleStart *string          `json:"schedule_start,omitempty"`
		ScheduleEnd   *string          `json:"schedule_end,omitempty"`
	}
	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	if req.Name == "" {
		utils.BadRequest(w, "name is required")
		return
	}

	preset := models.AdminThemePreset{
		ID:            uuid.New().String(),
		Name:          req.Name,
		Theme:         req.Theme,
		ScheduleStart: req.ScheduleStart,
		ScheduleEnd:   req.ScheduleEnd,
		CreatedAt:     time.Now(),
	}

	if err := h.repo.AddPreset(r.Context(), preset); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusCreated, preset, "Preset created successfully")
}

// UpdatePreset handles PUT /api/v1/admin/settings/presets/{presetId}
// Updates an existing theme preset.
func (h *AdminSettingsHandler) UpdatePreset(w http.ResponseWriter, r *http.Request) {
	presetID := chi.URLParam(r, "presetId")
	if presetID == "" {
		utils.BadRequest(w, "presetId is required")
		return
	}

	var req struct {
		Name          string           `json:"name"`
		Theme         models.AdminTheme `json:"theme"`
		ScheduleStart *string          `json:"schedule_start,omitempty"`
		ScheduleEnd   *string          `json:"schedule_end,omitempty"`
	}
	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	// Retrieve existing preset to preserve CreatedAt
	settings, err := h.repo.Get(r.Context())
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	var createdAt time.Time
	found := false
	for _, p := range settings.Presets {
		if p.ID == presetID {
			createdAt = p.CreatedAt
			found = true
			break
		}
	}
	if !found {
		utils.NotFound(w, "preset not found")
		return
	}

	preset := models.AdminThemePreset{
		ID:            presetID,
		Name:          req.Name,
		Theme:         req.Theme,
		ScheduleStart: req.ScheduleStart,
		ScheduleEnd:   req.ScheduleEnd,
		CreatedAt:     createdAt,
	}

	if err := h.repo.UpdatePreset(r.Context(), presetID, preset); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, preset, "Preset updated successfully")
}

// DeletePreset handles DELETE /api/v1/admin/settings/presets/{presetId}
// Deletes a theme preset.
func (h *AdminSettingsHandler) DeletePreset(w http.ResponseWriter, r *http.Request) {
	presetID := chi.URLParam(r, "presetId")
	if presetID == "" {
		utils.BadRequest(w, "presetId is required")
		return
	}

	if err := h.repo.DeletePreset(r.Context(), presetID); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Preset deleted successfully")
}

// ActivatePreset handles POST /api/v1/admin/settings/presets/{presetId}/activate
// Merges non-empty fields from the preset into the active theme.
func (h *AdminSettingsHandler) ActivatePreset(w http.ResponseWriter, r *http.Request) {
	presetID := chi.URLParam(r, "presetId")
	if presetID == "" {
		utils.BadRequest(w, "presetId is required")
		return
	}

	settings, err := h.repo.Get(r.Context())
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	// Find the preset
	var preset *models.AdminThemePreset
	for _, p := range settings.Presets {
		if p.ID == presetID {
			preset = &p
			break
		}
	}
	if preset == nil {
		utils.NotFound(w, "preset not found")
		return
	}

	// Merge non-empty fields from preset theme into active theme
	active := settings.ActiveTheme
	if preset.Theme.Logo != "" {
		active.Logo = preset.Theme.Logo
	}
	if preset.Theme.LogoIcon != "" {
		active.LogoIcon = preset.Theme.LogoIcon
	}
	if len(preset.Theme.LoginImages) > 0 {
		active.LoginImages = preset.Theme.LoginImages
	}
	if preset.Theme.PrimaryColor != "" {
		active.PrimaryColor = preset.Theme.PrimaryColor
	}
	if preset.Theme.SecondaryColor != "" {
		active.SecondaryColor = preset.Theme.SecondaryColor
	}
	if preset.Theme.AccentColor != "" {
		active.AccentColor = preset.Theme.AccentColor
	}
	if preset.Theme.WelcomeMessage != "" {
		active.WelcomeMessage = preset.Theme.WelcomeMessage
	}

	claims, err := middleware.GetUserFromContext(r.Context())
	if err != nil {
		utils.Unauthorized(w, "unauthorized")
		return
	}

	if err := h.repo.UpdateTheme(r.Context(), active, claims.Sub); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, active, "Preset activated successfully")
}
