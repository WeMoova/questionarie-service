package handlers

import (
	"io"
	"net/http"
	"questionarie-service/storage"
	"questionarie-service/utils"

	"github.com/go-chi/chi/v5"
)

// ImageHandler handles image upload/download operations
type ImageHandler struct {
	storage *storage.MinIOStorage
}

// NewImageHandler creates a new ImageHandler
func NewImageHandler(s *storage.MinIOStorage) *ImageHandler {
	return &ImageHandler{storage: s}
}

// UploadImage handles POST /api/v1/images/upload
// Accepts multipart/form-data with field "image" and optional "folder" field
func (h *ImageHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	// Override body limit for image uploads (5MB)
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)

	if err := r.ParseMultipartForm(5 << 20); err != nil {
		utils.BadRequest(w, "file too large or invalid form data")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		utils.BadRequest(w, "image file is required")
		return
	}
	defer file.Close()

	folder := r.FormValue("folder")
	if folder == "" {
		folder = "general"
	}

	objectPath, err := h.storage.UploadImage(r.Context(), file, header.Size, header.Filename, folder)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusCreated, map[string]string{"url": objectPath}, "Image uploaded successfully")
}

// GetImage handles GET /api/v1/images/*
// Streams image from MinIO to client
func (h *ImageHandler) GetImage(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "*")
	if path == "" {
		utils.BadRequest(w, "image path is required")
		return
	}

	object, err := h.storage.GetObject(r.Context(), path)
	if err != nil {
		utils.NotFound(w, "image not found")
		return
	}
	defer object.Close()

	info, err := object.Stat()
	if err != nil {
		utils.NotFound(w, "image not found")
		return
	}

	w.Header().Set("Content-Type", info.ContentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	io.Copy(w, object)
}

// DeleteImage handles DELETE /api/v1/images
func (h *ImageHandler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := utils.ParseRequestBody(r, &req); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	if req.URL == "" {
		utils.BadRequest(w, "url is required")
		return
	}

	if err := h.storage.DeleteImage(r.Context(), req.URL); err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, nil, "Image deleted successfully")
}
