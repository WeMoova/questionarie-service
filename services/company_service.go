package services

import (
	"context"
	"fmt"
	"questionarie-service/models"
	"questionarie-service/repository"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CompanyService handles business logic for companies
type CompanyService struct {
	companyRepo              *repository.CompanyRepository
	companyQuestionnaireRepo *repository.CompanyQuestionnaireRepository
	questionnaireRepo        *repository.QuestionnaireRepository
}

// NewCompanyService creates a new CompanyService
func NewCompanyService(
	companyRepo *repository.CompanyRepository,
	companyQuestionnaireRepo *repository.CompanyQuestionnaireRepository,
	questionnaireRepo *repository.QuestionnaireRepository,
) *CompanyService {
	return &CompanyService{
		companyRepo:              companyRepo,
		companyQuestionnaireRepo: companyQuestionnaireRepo,
		questionnaireRepo:        questionnaireRepo,
	}
}

// CreateCompany creates a new company (Super Admin only)
func (s *CompanyService) CreateCompany(ctx context.Context, name string) (*models.Company, error) {
	if name == "" {
		return nil, fmt.Errorf("company name is required")
	}
	if len(name) < 3 {
		return nil, fmt.Errorf("company name must be at least 3 characters")
	}

	company := models.NewCompany(name)

	if err := s.companyRepo.Create(ctx, company); err != nil {
		return nil, fmt.Errorf("failed to create company: %w", err)
	}

	return company, nil
}

// GetCompanyByID retrieves a company by ID
func (s *CompanyService) GetCompanyByID(ctx context.Context, id primitive.ObjectID) (*models.Company, error) {
	return s.companyRepo.GetByID(ctx, id)
}

// GetAllCompanies retrieves all companies with pagination
func (s *CompanyService) GetAllCompanies(ctx context.Context, page, pageSize int64) ([]*models.Company, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return s.companyRepo.GetAll(ctx, page, pageSize)
}

// UpdateCompany updates a company
func (s *CompanyService) UpdateCompany(ctx context.Context, id primitive.ObjectID, name string) error {
	company, err := s.companyRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if name != "" {
		company.Name = name
	}
	company.UpdatedAt = time.Now()

	return s.companyRepo.Update(ctx, id, company)
}

// DeleteCompany deletes a company
func (s *CompanyService) DeleteCompany(ctx context.Context, id primitive.ObjectID) error {
	// TODO: Check if company has users or active questionnaires before deleting
	return s.companyRepo.Delete(ctx, id)
}

// SearchCompaniesByName searches companies by name
func (s *CompanyService) SearchCompaniesByName(ctx context.Context, name string) ([]*models.Company, error) {
	if name == "" {
		return nil, fmt.Errorf("search name cannot be empty")
	}
	return s.companyRepo.SearchByName(ctx, name)
}

// AssignQuestionnaireToCompany assigns a questionnaire to a company
func (s *CompanyService) AssignQuestionnaireToCompany(
	ctx context.Context,
	companyID, questionnaireID primitive.ObjectID,
	assignedBy string,
	periodStart, periodEnd time.Time,
) (*models.CompanyQuestionnaire, error) {
	// Validate company exists
	if _, err := s.companyRepo.GetByID(ctx, companyID); err != nil {
		return nil, fmt.Errorf("company not found: %w", err)
	}

	// Validate questionnaire exists and is active
	questionnaire, err := s.questionnaireRepo.GetByID(ctx, questionnaireID)
	if err != nil {
		return nil, fmt.Errorf("questionnaire not found: %w", err)
	}
	if !questionnaire.IsActive {
		return nil, fmt.Errorf("questionnaire is not active")
	}
	if len(questionnaire.Questions) == 0 {
		return nil, fmt.Errorf("questionnaire has no questions")
	}

	// Validate period
	if periodStart.After(periodEnd) || periodStart.Equal(periodEnd) {
		return nil, fmt.Errorf("period start must be before period end")
	}

	// Check for duplicate assignment in overlapping period
	isDuplicate, err := s.companyQuestionnaireRepo.CheckDuplicate(ctx, companyID, questionnaireID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to check duplicate: %w", err)
	}
	if isDuplicate {
		return nil, fmt.Errorf("questionnaire already assigned to this company for overlapping period")
	}

	// Create assignment
	cq := models.NewCompanyQuestionnaire(companyID, questionnaireID, assignedBy, periodStart, periodEnd)

	if err := s.companyQuestionnaireRepo.Create(ctx, cq); err != nil {
		return nil, fmt.Errorf("failed to assign questionnaire: %w", err)
	}

	return cq, nil
}

// GetCompanyQuestionnaires retrieves all questionnaires assigned to a company
func (s *CompanyService) GetCompanyQuestionnaires(ctx context.Context, companyID primitive.ObjectID, activeOnly bool) ([]*models.CompanyQuestionnaire, error) {
	return s.companyQuestionnaireRepo.GetByCompanyID(ctx, companyID, activeOnly)
}

// GetActiveCompanyQuestionnaires retrieves active questionnaires for a company in current period
func (s *CompanyService) GetActiveCompanyQuestionnaires(ctx context.Context, companyID primitive.ObjectID) ([]*models.CompanyQuestionnaire, error) {
	return s.companyQuestionnaireRepo.GetActiveByCompanyAndPeriod(ctx, companyID)
}

// UpdateCompanyQuestionnaire updates a company questionnaire assignment
func (s *CompanyService) UpdateCompanyQuestionnaire(ctx context.Context, id primitive.ObjectID, periodStart, periodEnd time.Time, isActive bool) error {
	cq, err := s.companyQuestionnaireRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Validate period if provided
	if !periodStart.IsZero() && !periodEnd.IsZero() {
		if periodStart.After(periodEnd) || periodStart.Equal(periodEnd) {
			return fmt.Errorf("period start must be before period end")
		}
		cq.PeriodStart = periodStart
		cq.PeriodEnd = periodEnd
	}

	cq.IsActive = isActive

	return s.companyQuestionnaireRepo.Update(ctx, id, cq)
}

// DeactivateCompanyQuestionnaire deactivates a company questionnaire
func (s *CompanyService) DeactivateCompanyQuestionnaire(ctx context.Context, id primitive.ObjectID) error {
	return s.companyQuestionnaireRepo.Deactivate(ctx, id)
}

// GetCompanyStats returns statistics about a company
func (s *CompanyService) GetCompanyStats(ctx context.Context, companyID primitive.ObjectID) (map[string]interface{}, error) {
	company, err := s.companyRepo.GetByID(ctx, companyID)
	if err != nil {
		return nil, err
	}

	questionnaires, err := s.companyQuestionnaireRepo.GetByCompanyID(ctx, companyID, false)
	if err != nil {
		return nil, err
	}

	activeQuestionnaires, err := s.companyQuestionnaireRepo.GetByCompanyID(ctx, companyID, true)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"company_id":             company.ID.Hex(),
		"company_name":           company.Name,
		"total_questionnaires":   len(questionnaires),
		"active_questionnaires":  len(activeQuestionnaires),
		"created_at":             company.CreatedAt,
	}, nil
}
