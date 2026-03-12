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
	repo     *repository.QuestionnaireRepository
	cqRepo   *repository.CompanyQuestionnaireRepository
	aRepo    *repository.AssignmentRepository
	linkRepo *repository.PublicLinkRepository
}

// NewQuestionnaireService creates a new QuestionnaireService
func NewQuestionnaireService(repo *repository.QuestionnaireRepository, cqRepo *repository.CompanyQuestionnaireRepository, aRepo *repository.AssignmentRepository, linkRepo *repository.PublicLinkRepository) *QuestionnaireService {
	return &QuestionnaireService{
		repo:     repo,
		cqRepo:   cqRepo,
		aRepo:    aRepo,
		linkRepo: linkRepo,
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

// GetAllQuestionnaires retrieves all questionnaires with pagination and filters
func (s *QuestionnaireService) GetAllQuestionnaires(ctx context.Context, page, pageSize int64, filter repository.QuestionnaireFilter) ([]*models.Questionnaire, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	questionnaires, err := s.repo.GetAll(ctx, page, pageSize, filter)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.Count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return questionnaires, total, nil
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

// DeleteQuestionnaire permanently removes a questionnaire and all related
// company_questionnaire assignments and user assignments (cascade delete).
func (s *QuestionnaireService) DeleteQuestionnaire(ctx context.Context, id primitive.ObjectID) error {
	// 1. Find all company_questionnaire IDs that reference this questionnaire
	cqIDs, err := s.cqRepo.GetIDsByQuestionnaireID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find related company questionnaires: %w", err)
	}

	// 2. Delete public links for those company questionnaires
	if len(cqIDs) > 0 {
		if _, err := s.linkRepo.DeleteByCompanyQuestionnaireIDs(ctx, cqIDs); err != nil {
			return fmt.Errorf("failed to delete related public links: %w", err)
		}
	}

	// 3. Delete user assignments linked to those company questionnaires
	if len(cqIDs) > 0 {
		if _, err := s.aRepo.DeleteByCompanyQuestionnaireIDs(ctx, cqIDs); err != nil {
			return fmt.Errorf("failed to delete related user assignments: %w", err)
		}
	}

	// 4. Delete the company questionnaire assignments themselves
	if _, err := s.cqRepo.DeleteByQuestionnaireID(ctx, id); err != nil {
		return fmt.Errorf("failed to delete related company questionnaires: %w", err)
	}

	// 5. Delete the questionnaire
	return s.repo.Delete(ctx, id)
}

// ToggleQuestionnaireActive sets the active status of a questionnaire
func (s *QuestionnaireService) ToggleQuestionnaireActive(ctx context.Context, id primitive.ObjectID, isActive bool) error {
	return s.repo.ToggleActive(ctx, id, isActive)
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
	total, err := s.repo.Count(ctx, repository.QuestionnaireFilter{})
	if err != nil {
		return nil, err
	}

	activeVal := true
	active, err := s.repo.Count(ctx, repository.QuestionnaireFilter{IsActive: &activeVal})
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

// DuplicateQuestionnaire creates a copy of an existing questionnaire with a new title and optional category
func (s *QuestionnaireService) DuplicateQuestionnaire(ctx context.Context, sourceID primitive.ObjectID, title, createdBy string, categoryID *primitive.ObjectID) (*models.Questionnaire, error) {
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if len(title) < 5 {
		return nil, fmt.Errorf("title must be at least 5 characters")
	}

	source, err := s.repo.GetByID(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("source questionnaire not found: %w", err)
	}

	dup := models.NewQuestionnaire(title, source.Description, createdBy)
	dup.CoverImage = source.CoverImage
	dup.Tags = append([]string{}, source.Tags...)
	dup.Sections = append([]models.Section{}, source.Sections...)
	if source.EvaluationConfig != nil {
		configCopy := *source.EvaluationConfig
		configCopy.Dimensions = append([]models.DimensionConfig{}, source.EvaluationConfig.Dimensions...)
		dup.EvaluationConfig = &configCopy
	}

	// Use provided categoryID, or fall back to original
	if categoryID != nil {
		dup.CategoryID = categoryID
	} else {
		dup.CategoryID = source.CategoryID
	}

	// Deep-copy questions with new UUIDs
	questions := make([]models.Question, len(source.Questions))
	for i, q := range source.Questions {
		questions[i] = q
		newQ := models.NewQuestion(q.QuestionText, q.QuestionType, q.OrderIndex, q.IsRequired)
		questions[i].QuestionID = newQ.QuestionID
	}
	dup.Questions = questions

	if err := s.repo.Create(ctx, dup); err != nil {
		return nil, fmt.Errorf("failed to duplicate questionnaire: %w", err)
	}

	return dup, nil
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
