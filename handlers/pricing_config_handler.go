package handlers

import (
	"encoding/json"
	"net/http"
	"questionarie-service/middleware"
	"questionarie-service/models"
	"questionarie-service/repository"
	"questionarie-service/utils"
)

type PricingConfigHandler struct {
	repo *repository.PricingConfigRepository
}

func NewPricingConfigHandler(repo *repository.PricingConfigRepository) *PricingConfigHandler {
	return &PricingConfigHandler{repo: repo}
}

// GetConfig returns the pricing configuration (public or admin).
func (h *PricingConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	config, err := h.repo.Get(r.Context())
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}
	utils.RespondWithSuccess(w, http.StatusOK, config, "")
}

// UpdateConfig replaces the entire pricing configuration (super_admin only).
func (h *PricingConfigHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var config models.PricingConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		utils.BadRequest(w, "Invalid request body: "+err.Error())
		return
	}

	claims, err := middleware.GetUserFromContext(r.Context())
	if err != nil {
		utils.Unauthorized(w, "unauthorized")
		return
	}

	if err := h.repo.Update(r.Context(), &config, claims.Sub); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	updated, _ := h.repo.Get(r.Context())
	utils.RespondWithSuccess(w, http.StatusOK, updated, "Pricing config updated successfully")
}
