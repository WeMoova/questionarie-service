package services

import (
	"context"
	"fmt"
	"questionarie-service/models"
	"questionarie-service/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// QuestionnaireService handles business logic for questionnaires
type QuestionnaireService struct {
	repo *repository.QuestionnaireRepository
}

// NewQuestionnaireService creates a new QuestionnaireService
func NewQuestionnaireService(repo *repository.QuestionnaireRepository) *QuestionnaireService {
	return &QuestionnaireService{
		repo: repo,
	}
}

// CreateQuestionnaire creates a new questionnaire (Super Admin only)
func (s *QuestionnaireService) CreateQuestionnaire(ctx context.Context, title, description, createdBy string) (*models.Questionnaire, error) {
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if len(title) < 5 {
		return nil, fmt.Errorf("title must be at least 5 characters")
	}

	questionnaire := models.NewQuestionnaire(title, description, createdBy)

	if err := s.repo.Create(ctx, questionnaire); err != nil {
		return nil, fmt.Errorf("failed to create questionnaire: %w", err)
	}

	return questionnaire, nil
}

// GetQuestionnaireByID retrieves a questionnaire by ID
func (s *QuestionnaireService) GetQuestionnaireByID(ctx context.Context, id primitive.ObjectID) (*models.Questionnaire, error) {
	return s.repo.GetByID(ctx, id)
}

// GetAllQuestionnaires retrieves all questionnaires with pagination
func (s *QuestionnaireService) GetAllQuestionnaires(ctx context.Context, page, pageSize int64, activeOnly bool) ([]*models.Questionnaire, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return s.repo.GetAll(ctx, page, pageSize, activeOnly)
}

// GetQuestionnairesByCreator retrieves questionnaires created by a user
func (s *QuestionnaireService) GetQuestionnairesByCreator(ctx context.Context, creatorID string) ([]*models.Questionnaire, error) {
	return s.repo.GetByCreator(ctx, creatorID)
}

// UpdateQuestionnaire updates a questionnaire
func (s *QuestionnaireService) UpdateQuestionnaire(ctx context.Context, id primitive.ObjectID, title, description string, isActive bool) error {
	questionnaire, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if title != "" {
		questionnaire.Title = title
	}
	if description != "" {
		questionnaire.Description = description
	}
	questionnaire.IsActive = isActive

	return s.repo.Update(ctx, id, questionnaire)
}

// DeactivateQuestionnaire deactivates a questionnaire
func (s *QuestionnaireService) DeactivateQuestionnaire(ctx context.Context, id primitive.ObjectID) error {
	return s.repo.Deactivate(ctx, id)
}

// AddQuestion adds a question to a questionnaire
func (s *QuestionnaireService) AddQuestion(ctx context.Context, questionnaireID primitive.ObjectID, question models.Question) error {
	// Validate question
	if question.QuestionText == "" {
		return fmt.Errorf("question text is required")
	}
	if len(question.QuestionText) < 5 {
		return fmt.Errorf("question text must be at least 5 characters")
	}

	// Validate question type
	validTypes := map[models.QuestionType]bool{
		models.QuestionTypeMultipleChoice: true,
		models.QuestionTypeLikertScale:    true,
		models.QuestionTypeFreeText:       true,
		models.QuestionTypeYesNo:          true,
	}
	if !validTypes[question.QuestionType] {
		return fmt.Errorf("invalid question type")
	}

	return s.repo.AddQuestion(ctx, questionnaireID, question)
}

// UpdateQuestion updates a specific question
func (s *QuestionnaireService) UpdateQuestion(ctx context.Context, questionnaireID primitive.ObjectID, questionID string, question models.Question) error {
	if question.QuestionText == "" {
		return fmt.Errorf("question text is required")
	}

	return s.repo.UpdateQuestion(ctx, questionnaireID, questionID, question)
}

// RemoveQuestion removes a question from a questionnaire
func (s *QuestionnaireService) RemoveQuestion(ctx context.Context, questionnaireID primitive.ObjectID, questionID string) error {
	return s.repo.RemoveQuestion(ctx, questionnaireID, questionID)
}

// GetQuestionnaireStats returns statistics about questionnaires
func (s *QuestionnaireService) GetQuestionnaireStats(ctx context.Context) (map[string]interface{}, error) {
	total, err := s.repo.Count(ctx, false)
	if err != nil {
		return nil, err
	}

	active, err := s.repo.Count(ctx, true)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total":    total,
		"active":   active,
		"inactive": total - active,
	}, nil
}

// ValidateQuestionnaire validates that a questionnaire is complete and ready to be assigned
func (s *QuestionnaireService) ValidateQuestionnaire(ctx context.Context, id primitive.ObjectID) error {
	questionnaire, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if !questionnaire.IsActive {
		return fmt.Errorf("questionnaire is not active")
	}

	if len(questionnaire.Questions) == 0 {
		return fmt.Errorf("questionnaire must have at least one question")
	}

	return nil
}
