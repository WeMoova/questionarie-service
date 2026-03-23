package handlers

import (
	"encoding/json"
	"log/slog"
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
// Auto-syncs UF value from CMF Chile API if available.
func (h *PricingConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	config, err := h.repo.Get(r.Context())
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	// Try to auto-update UF value from CMF
	if uf, err := utils.FetchCurrentUF(); err == nil {
		if uf.Value != config.UfValueCLP {
			config.UfValueCLP = uf.Value
			// Update in background — don't block the response
			go func() {
				if updateErr := h.repo.UpdateUFValue(r.Context(), uf.Value); updateErr != nil {
					slog.Warn("failed to auto-update UF value", "error", updateErr)
				}
			}()
		}
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

	// Auto-sync UF value from CMF before saving
	if uf, err := utils.FetchCurrentUF(); err == nil {
		config.UfValueCLP = uf.Value
	}

	if err := h.repo.Update(r.Context(), &config, claims.Sub); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	updated, _ := h.repo.Get(r.Context())
	utils.RespondWithSuccess(w, http.StatusOK, updated, "Pricing config updated successfully")
}

// GetUFValue returns the current UF value from CMF Chile API.
func (h *PricingConfigHandler) GetUFValue(w http.ResponseWriter, r *http.Request) {
	uf, err := utils.FetchCurrentUF()
	if err != nil {
		// Fallback to stored value in pricing config
		config, configErr := h.repo.Get(r.Context())
		if configErr != nil {
			utils.RespondWithError(w, http.StatusBadGateway, "Failed to fetch UF value: "+err.Error())
			return
		}
		utils.RespondWithSuccess(w, http.StatusOK, map[string]interface{}{
			"value":  config.UfValueCLP,
			"date":   config.UpdatedAt.Format("2006-01-02"),
			"source": "stored",
		}, "")
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, map[string]interface{}{
		"value":  uf.Value,
		"date":   uf.Date,
		"source": uf.Source,
	}, "")
}
