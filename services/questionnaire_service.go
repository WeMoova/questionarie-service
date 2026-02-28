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
func (s *QuestionnaireService) CreateQuestionnaire(ctx context.Context, title, description, createdBy string, tags []string, categoryID *primitive.ObjectID, coverImage string) (*models.Questionnaire, error) {
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if len(title) < 5 {
		return nil, fmt.Errorf("title must be at least 5 characters")
	}

	questionnaire := models.NewQuestionnaire(title, description, createdBy)
	questionnaire.CoverImage = coverImage
	if tags != nil {
		questionnaire.Tags = tags
	}
	if categoryID != nil {
		questionnaire.CategoryID = categoryID
	}

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
func (s *QuestionnaireService) UpdateQuestionnaire(ctx context.Context, id primitive.ObjectID, title, description string, isActive bool, tags []string, categoryID *primitive.ObjectID, coverImage *string) error {
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
	if coverImage != nil {
		questionnaire.CoverImage = *coverImage
	}
	questionnaire.IsActive = isActive
	if tags != nil {
		questionnaire.Tags = tags
	}
	if categoryID != nil {
		questionnaire.CategoryID = categoryID
	}

	return s.repo.Update(ctx, id, questionnaire)
}

// GetQuestionnaireUsageStats returns per-questionnaire usage stats (companies assigned, total assignments)
func (s *QuestionnaireService) GetQuestionnaireUsageStats(ctx context.Context, id primitive.ObjectID) (map[string]interface{}, error) {
	q, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"questionnaire_id":    q.ID.Hex(),
		"title":               q.Title,
		"is_active":           q.IsActive,
		"total_questions":     len(q.Questions),
		"required_questions":  countRequired(q.Questions),
		"tags":                q.Tags,
		"category_id":         q.CategoryID,
	}, nil
}

func countRequired(questions []models.Question) int {
	count := 0
	for _, q := range questions {
		if q.IsRequired {
			count++
		}
	}
	return count
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

// AddSection adds a section to a questionnaire
func (s *QuestionnaireService) AddSection(ctx context.Context, questionnaireID primitive.ObjectID, section models.Section) error {
	return s.repo.AddSection(ctx, questionnaireID, section)
}

// UpdateSection updates a section within a questionnaire
func (s *QuestionnaireService) UpdateSection(ctx context.Context, questionnaireID primitive.ObjectID, sectionID string, section models.Section) error {
	return s.repo.UpdateSection(ctx, questionnaireID, sectionID, section)
}

// DeleteSection removes a section from a questionnaire
func (s *QuestionnaireService) DeleteSection(ctx context.Context, questionnaireID primitive.ObjectID, sectionID string) error {
	return s.repo.DeleteSection(ctx, questionnaireID, sectionID)
}

// SetEvaluationConfig sets the evaluation configuration for a questionnaire
func (s *QuestionnaireService) SetEvaluationConfig(ctx context.Context, questionnaireID primitive.ObjectID, config *models.EvaluationConfig) error {
	return s.repo.SetEvaluationConfig(ctx, questionnaireID, config)
}

// ImportQuestionnaire creates a complete questionnaire with sections, questions, and evaluation config in one call
func (s *QuestionnaireService) ImportQuestionnaire(ctx context.Context, title, description, createdBy string, tags []string, categoryID *primitive.ObjectID, coverImage string, sections []models.Section, questions []models.Question, evaluationConfig *models.EvaluationConfig) (*models.Questionnaire, error) {
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if len(title) < 5 {
		return nil, fmt.Errorf("title must be at least 5 characters")
	}

	questionnaire := models.NewQuestionnaire(title, description, createdBy)
	questionnaire.CoverImage = coverImage
	if tags != nil {
		questionnaire.Tags = tags
	}
	if categoryID != nil {
		questionnaire.CategoryID = categoryID
	}
	if sections != nil {
		questionnaire.Sections = sections
	}
	if evaluationConfig != nil {
		questionnaire.EvaluationConfig = evaluationConfig
	}

	// Assign UUIDs to questions that don't have IDs
	for i := range questions {
		if questions[i].QuestionID == "" {
			q := models.NewQuestion(questions[i].QuestionText, questions[i].QuestionType, questions[i].OrderIndex, questions[i].IsRequired)
			questions[i].QuestionID = q.QuestionID
		}
		if questions[i].Options == nil {
			questions[i].Options = make(map[string]interface{})
		}
	}
	questionnaire.Questions = questions

	if err := s.repo.Create(ctx, questionnaire); err != nil {
		return nil, fmt.Errorf("failed to import questionnaire: %w", err)
	}

	return questionnaire, nil
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
