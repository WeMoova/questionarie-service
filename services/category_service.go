package services

import (
	"context"
	"fmt"
	"questionarie-service/models"
	"questionarie-service/repository"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CategoryService handles business logic for questionnaire categories
type CategoryService struct {
	categoryRepo      *repository.CategoryRepository
	questionnaireRepo *repository.QuestionnaireRepository
}

// NewCategoryService creates a new CategoryService
func NewCategoryService(categoryRepo *repository.CategoryRepository, questionnaireRepo *repository.QuestionnaireRepository) *CategoryService {
	return &CategoryService{
		categoryRepo:      categoryRepo,
		questionnaireRepo: questionnaireRepo,
	}
}

// CreateCategory creates a new questionnaire category
func (s *CategoryService) CreateCategory(ctx context.Context, name, description string) (*models.QuestionnaireCategory, error) {
	if len(name) < 3 {
		return nil, fmt.Errorf("category name must be at least 3 characters")
	}
	category := models.NewQuestionnaireCategory(name, description)
	if err := s.categoryRepo.Create(ctx, category); err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}
	return category, nil
}

// GetOrCreateByName finds a category by name (case-insensitive) or creates it if it doesn't exist
func (s *CategoryService) GetOrCreateByName(ctx context.Context, name string) (*models.QuestionnaireCategory, error) {
	if len(name) < 3 {
		return nil, fmt.Errorf("category name must be at least 3 characters")
	}
	existing, err := s.categoryRepo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	// Create new category
	category := models.NewQuestionnaireCategory(name, "")
	if err := s.categoryRepo.Create(ctx, category); err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}
	return category, nil
}

// GetAllCategories retrieves all categories
func (s *CategoryService) GetAllCategories(ctx context.Context, activeOnly bool) ([]*models.QuestionnaireCategory, error) {
	return s.categoryRepo.GetAll(ctx, activeOnly)
}

// GetCategoryByID retrieves a category by ID
func (s *CategoryService) GetCategoryByID(ctx context.Context, id primitive.ObjectID) (*models.QuestionnaireCategory, error) {
	return s.categoryRepo.GetByID(ctx, id)
}

// UpdateCategory updates a category
func (s *CategoryService) UpdateCategory(ctx context.Context, id primitive.ObjectID, name, description string, isActive *bool) error {
	category, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if name != "" {
		if len(name) < 3 {
			return fmt.Errorf("category name must be at least 3 characters")
		}
		category.Name = name
	}
	if description != "" {
		category.Description = description
	}
	if isActive != nil {
		category.IsActive = *isActive
	}
	category.UpdatedAt = time.Now()
	return s.categoryRepo.Update(ctx, id, category)
}

// DeleteCategory permanently removes a category
func (s *CategoryService) DeleteCategory(ctx context.Context, id primitive.ObjectID) error {
	return s.categoryRepo.Delete(ctx, id)
}

// GetCategoryIDsForQuestionnaires returns the set of category IDs used by the given questionnaire IDs
func (s *CategoryService) GetCategoryIDsForQuestionnaires(ctx context.Context, questionnaireIDs []primitive.ObjectID) (map[primitive.ObjectID]bool, error) {
	filter := repository.QuestionnaireFilter{IDs: questionnaireIDs}
	questionnaires, err := s.questionnaireRepo.GetAll(ctx, 1, int64(len(questionnaireIDs)+1), filter)
	if err != nil {
		return nil, err
	}
	categoryIDs := make(map[primitive.ObjectID]bool)
	for _, q := range questionnaires {
		if q.CategoryID != nil {
			categoryIDs[*q.CategoryID] = true
		}
	}
	return categoryIDs, nil
}

// GetQuestionnairesByCategory retrieves all questionnaires for a category
func (s *CategoryService) GetQuestionnairesByCategory(ctx context.Context, categoryID primitive.ObjectID) ([]*models.Questionnaire, error) {
	if _, err := s.categoryRepo.GetByID(ctx, categoryID); err != nil {
		return nil, fmt.Errorf("category not found: %w", err)
	}
	return s.questionnaireRepo.GetByCategoryID(ctx, categoryID)
}
