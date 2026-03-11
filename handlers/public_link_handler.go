package handlers

import (
	"encoding/json"
	"net/http"
	"questionarie-service/models"
	"questionarie-service/services"
	"questionarie-service/utils"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
)

// PublicLinkHandler handles CRUD operations for public links (admin-facing).
type PublicLinkHandler struct {
	service *services.PublicLinkService
}

// NewPublicLinkHandler creates a new PublicLinkHandler.
func NewPublicLinkHandler(service *services.PublicLinkService) *PublicLinkHandler {
	return &PublicLinkHandler{service: service}
}

// CreateLink handles POST /api/v1/company-questionnaires/{id}/public-links
func (h *PublicLinkHandler) CreateLink(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	cqID, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var link models.PublicLink
	if err := utils.ParseRequestBody(r, &link); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	if link.Slug == "" {
		utils.BadRequest(w, "slug is required")
		return
	}

	if err := h.service.CreateLink(r.Context(), cqID, &link); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusCreated, link, "Public link created successfully")
}

// ListLinks handles GET /api/v1/company-questionnaires/{id}/public-links
func (h *PublicLinkHandler) ListLinks(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	cqID, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	links, err := h.service.GetLinksByCQ(r.Context(), cqID)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, links, "")
}

// UpdateLink handles PUT /api/v1/public-links/{id}
func (h *PublicLinkHandler) UpdateLink(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	linkID, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	var raw map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		utils.BadRequest(w, "invalid JSON body")
		return
	}

	update := bson.M{}
	for key, val := range raw {
		switch key {
		case "slug", "label", "is_active", "demographic_fields", "result_config":
			update[key] = val
		}
	}

	if len(update) == 0 {
		utils.BadRequest(w, "no valid fields to update")
		return
	}

	if err := h.service.UpdateLink(r.Context(), linkID, update); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Public link updated successfully")
}

// DeleteLink handles DELETE /api/v1/public-links/{id}
func (h *PublicLinkHandler) DeleteLink(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	linkID, err := utils.ValidateObjectID(idStr)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	if err := h.service.DeleteLink(r.Context(), linkID); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Public link deleted successfully")
}
