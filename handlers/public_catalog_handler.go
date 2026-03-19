package handlers

import (
	"net/http"
	"questionarie-service/repository"
	"questionarie-service/utils"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PublicCatalogHandler struct {
	questionnaireRepo *repository.QuestionnaireRepository
	categoryRepo      *repository.CategoryRepository
}

func NewPublicCatalogHandler(qRepo *repository.QuestionnaireRepository, cRepo *repository.CategoryRepository) *PublicCatalogHandler {
	return &PublicCatalogHandler{
		questionnaireRepo: qRepo,
		categoryRepo:      cRepo,
	}
}

type PublicQuestionnaire struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	CoverImage     string   `json:"cover_image,omitempty"`
	Tags           []string `json:"tags"`
	CategoryID     string   `json:"category_id,omitempty"`
	CategoryName   string   `json:"category_name,omitempty"`
	QuestionCount  int      `json:"question_count"`
	SectionCount   int      `json:"section_count"`
	DimensionNames []string `json:"dimension_names,omitempty"`
	SectionNames   []string `json:"section_names,omitempty"`
	CreatedAt      string   `json:"created_at"`
}

type PublicQuestionnaireDetail struct {
	PublicQuestionnaire
	Sections []PublicSection `json:"sections,omitempty"`
}

type PublicSection struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type PublicCategory struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func (h *PublicCatalogHandler) GetPublicQuestionnaires(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	search := r.URL.Query().Get("search")
	categoryIDStr := r.URL.Query().Get("category_id")

	page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.ParseInt(r.URL.Query().Get("page_size"), 10, 64)
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	isActive := true
	filter := repository.QuestionnaireFilter{
		Search:   search,
		IsActive: &isActive,
	}

	if categoryIDStr != "" {
		catID, err := primitive.ObjectIDFromHex(categoryIDStr)
		if err == nil {
			filter.CategoryID = &catID
		}
	}

	questionnaires, err := h.questionnaireRepo.GetAll(ctx, page, pageSize, filter)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	// Build category name map
	categories, _ := h.categoryRepo.GetAll(ctx, true)
	catMap := make(map[string]string)
	for _, c := range categories {
		catMap[c.ID.Hex()] = c.Name
	}

	var items []PublicQuestionnaire
	for _, q := range questionnaires {
		pq := PublicQuestionnaire{
			ID:            q.ID.Hex(),
			Title:         q.Title,
			Description:   q.Description,
			CoverImage:    q.CoverImage,
			Tags:          q.Tags,
			QuestionCount: len(q.Questions),
			SectionCount:  len(q.Sections),
			CreatedAt:     q.CreatedAt.Format("2006-01-02"),
		}
		if pq.Tags == nil {
			pq.Tags = []string{}
		}
		if q.CategoryID != nil {
			pq.CategoryID = q.CategoryID.Hex()
			pq.CategoryName = catMap[q.CategoryID.Hex()]
		}
		if q.EvaluationConfig != nil && q.EvaluationConfig.Enabled {
			for _, dim := range q.EvaluationConfig.Dimensions {
				pq.DimensionNames = append(pq.DimensionNames, dim.Name)
			}
		}
		for _, sec := range q.Sections {
			pq.SectionNames = append(pq.SectionNames, sec.Title)
		}
		items = append(items, pq)
	}
	if items == nil {
		items = []PublicQuestionnaire{}
	}

	utils.RespondWithSuccess(w, http.StatusOK, map[string]any{
		"items":     items,
		"total":     len(items),
		"page":      page,
		"page_size": pageSize,
	}, "")
}

func (h *PublicCatalogHandler) GetPublicQuestionnaireByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	objectID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid questionnaire ID")
		return
	}

	q, err := h.questionnaireRepo.GetByID(ctx, objectID)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}
	if !q.IsActive {
		utils.BadRequest(w, "Questionnaire not found")
		return
	}

	categories, _ := h.categoryRepo.GetAll(ctx, true)
	catMap := make(map[string]string)
	for _, c := range categories {
		catMap[c.ID.Hex()] = c.Name
	}

	detail := PublicQuestionnaireDetail{
		PublicQuestionnaire: PublicQuestionnaire{
			ID:            q.ID.Hex(),
			Title:         q.Title,
			Description:   q.Description,
			CoverImage:    q.CoverImage,
			Tags:          q.Tags,
			QuestionCount: len(q.Questions),
			SectionCount:  len(q.Sections),
			CreatedAt:     q.CreatedAt.Format("2006-01-02"),
		},
	}
	if detail.Tags == nil {
		detail.Tags = []string{}
	}
	if q.CategoryID != nil {
		detail.CategoryID = q.CategoryID.Hex()
		detail.CategoryName = catMap[q.CategoryID.Hex()]
	}
	if q.EvaluationConfig != nil && q.EvaluationConfig.Enabled {
		for _, dim := range q.EvaluationConfig.Dimensions {
			detail.DimensionNames = append(detail.DimensionNames, dim.Name)
		}
	}
	for _, sec := range q.Sections {
		detail.Sections = append(detail.Sections, PublicSection{
			Title:       sec.Title,
			Description: sec.Description,
		})
		detail.SectionNames = append(detail.SectionNames, sec.Title)
	}

	utils.RespondWithSuccess(w, http.StatusOK, detail, "")
}

func (h *PublicCatalogHandler) GetPublicCategories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	categories, err := h.categoryRepo.GetAll(ctx, true)
	if err != nil {
		utils.HandleRepositoryError(w, err)
		return
	}

	isActive := true
	allQ, _ := h.questionnaireRepo.GetAll(ctx, 1, 1000, repository.QuestionnaireFilter{IsActive: &isActive})
	countMap := make(map[string]int)
	for _, q := range allQ {
		if q.CategoryID != nil {
			countMap[q.CategoryID.Hex()]++
		}
	}

	var result []PublicCategory
	for _, c := range categories {
		count := countMap[c.ID.Hex()]
		if count > 0 {
			result = append(result, PublicCategory{
				ID:    c.ID.Hex(),
				Name:  c.Name,
				Count: count,
			})
		}
	}
	if result == nil {
		result = []PublicCategory{}
	}

	utils.RespondWithSuccess(w, http.StatusOK, result, "")
}
