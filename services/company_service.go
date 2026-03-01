package services

import (
	"context"
	"fmt"
	"questionarie-service/models"
	"questionarie-service/repository"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CompanyDashboardItem represents a single questionnaire entry in the company dashboard
type CompanyDashboardItem struct {
	CompanyQuestionnaireID string                            `json:"company_questionnaire_id"`
	QuestionnaireTitle     string                            `json:"questionnaire_title"`
	Status                 models.CompanyQuestionnaireStatus `json:"status"`
	PeriodStart            string                            `json:"period_start"`
	PeriodEnd              string                            `json:"period_end"`
	TotalAssigned          int                               `json:"total_assigned"`
	Completed              int                               `json:"completed"`
	InProgress             int                               `json:"in_progress"`
	Pending                int                               `json:"pending"`
	CompletionRate         float64                           `json:"completion_rate"`
}

// ReminderResult describes which users would receive a reminder
type ReminderResult struct {
	CompanyQuestionnaireID string   `json:"company_questionnaire_id"`
	Target                 string   `json:"target"`
	UserIDs                []string `json:"user_ids"`
	Count                  int      `json:"count"`
	RemindersSent          int      `json:"reminders_sent"`
	LastReminderAt         *time.Time `json:"last_reminder_at,omitempty"`
}

// CompanyService handles business logic for companies
type CompanyService struct {
	companyRepo              *repository.CompanyRepository
	companyQuestionnaireRepo *repository.CompanyQuestionnaireRepository
	questionnaireRepo        *repository.QuestionnaireRepository
	assignmentRepo           *repository.AssignmentRepository
	userMetadataRepo         *repository.UserMetadataRepository
}

// NewCompanyService creates a new CompanyService
func NewCompanyService(
	companyRepo *repository.CompanyRepository,
	companyQuestionnaireRepo *repository.CompanyQuestionnaireRepository,
	questionnaireRepo *repository.QuestionnaireRepository,
	assignmentRepo *repository.AssignmentRepository,
	userMetadataRepo *repository.UserMetadataRepository,
) *CompanyService {
	return &CompanyService{
		companyRepo:              companyRepo,
		companyQuestionnaireRepo: companyQuestionnaireRepo,
		questionnaireRepo:        questionnaireRepo,
		assignmentRepo:           assignmentRepo,
		userMetadataRepo:         userMetadataRepo,
	}
}

// validateHexColor validates that a string is a valid 7-char hex color (e.g. #FF5500)
func validateHexColor(color string) error {
	if len(color) != 7 || color[0] != '#' {
		return fmt.Errorf("must be a 7-char hex color (e.g. #FF5500)")
	}
	for _, c := range color[1:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return fmt.Errorf("contains invalid hex character: %c", c)
		}
	}
	return nil
}

// validateBranding validates branding fields
func validateBranding(b *models.Branding) error {
	if b == nil {
		return nil
	}
	if b.PrimaryColor != "" {
		if err := validateHexColor(b.PrimaryColor); err != nil {
			return fmt.Errorf("invalid primary_color: %w", err)
		}
	}
	if b.SecondaryColor != "" {
		if err := validateHexColor(b.SecondaryColor); err != nil {
			return fmt.Errorf("invalid secondary_color: %w", err)
		}
	}
	return nil
}

// CreateCompany creates a new company (Super Admin only)
func (s *CompanyService) CreateCompany(ctx context.Context, name string, isActive bool, branding *models.Branding, customDomain *models.CustomDomain) (*models.Company, error) {
	if name == "" {
		return nil, fmt.Errorf("company name is required")
	}
	if len(name) < 3 {
		return nil, fmt.Errorf("company name must be at least 3 characters")
	}

	if err := validateBranding(branding); err != nil {
		return nil, err
	}

	if customDomain != nil && customDomain.Slug != "" {
		existing, err := s.companyRepo.GetBySlug(ctx, customDomain.Slug)
		if err != nil {
			return nil, fmt.Errorf("failed to check slug uniqueness: %w", err)
		}
		if existing != nil {
			return nil, fmt.Errorf("slug '%s' is already in use", customDomain.Slug)
		}
	}

	company := models.NewCompany(name)
	company.IsActive = isActive
	company.Branding = branding
	company.CustomDomain = customDomain

	if err := s.companyRepo.Create(ctx, company); err != nil {
		return nil, fmt.Errorf("failed to create company: %w", err)
	}

	return company, nil
}

// GetCompanyByID retrieves a company by ID
func (s *CompanyService) GetCompanyByID(ctx context.Context, id primitive.ObjectID) (*models.Company, error) {
	return s.companyRepo.GetByID(ctx, id)
}

// GetMyCompany retrieves the company of the authenticated company_admin via their user metadata.
func (s *CompanyService) GetMyCompany(ctx context.Context, userID string) (*models.Company, error) {
	userMeta, err := s.userMetadataRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("company not configured for this user: user metadata not found")
	}
	return s.companyRepo.GetByID(ctx, userMeta.CompanyID)
}

// GetAllCompanies retrieves all companies with pagination, enriched with active questionnaire count.
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

	companies, err := s.companyRepo.GetAll(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}

	for _, company := range companies {
		count, err := s.companyQuestionnaireRepo.CountByCompanyID(ctx, company.ID)
		if err == nil {
			company.QuestionnaireCount = int(count)
		}
	}

	return companies, nil
}

// UpdateCompany updates a company
func (s *CompanyService) UpdateCompany(ctx context.Context, id primitive.ObjectID, name string, isActive *bool, branding *models.Branding, customDomain *models.CustomDomain) error {
	company, err := s.companyRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if name != "" {
		company.Name = name
	}
	if isActive != nil {
		company.IsActive = *isActive
	}
	if branding != nil {
		if err := validateBranding(branding); err != nil {
			return err
		}
		company.Branding = branding
	}
	if customDomain != nil {
		if customDomain.Slug != "" {
			existing, err := s.companyRepo.GetBySlug(ctx, customDomain.Slug)
			if err != nil {
				return fmt.Errorf("failed to check slug uniqueness: %w", err)
			}
			if existing != nil && existing.ID != id {
				return fmt.Errorf("slug '%s' is already in use", customDomain.Slug)
			}
		}
		company.CustomDomain = customDomain
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

// GetCompanyQuestionnaires retrieves all questionnaires assigned to a company,
// enriched with the questionnaire title and description.
func (s *CompanyService) GetCompanyQuestionnaires(ctx context.Context, companyID primitive.ObjectID, activeOnly bool) ([]*models.CompanyQuestionnaire, error) {
	cqs, err := s.companyQuestionnaireRepo.GetByCompanyID(ctx, companyID, activeOnly)
	if err != nil {
		return nil, err
	}

	for _, cq := range cqs {
		q, err := s.questionnaireRepo.GetByID(ctx, cq.QuestionnaireID)
		if err == nil {
			cq.QuestionnaireTitle = q.Title
			cq.QuestionnaireDescription = q.Description
		}
	}

	return cqs, nil
}

// GetActiveCompanyQuestionnaires retrieves active questionnaires for a company in current period
func (s *CompanyService) GetActiveCompanyQuestionnaires(ctx context.Context, companyID primitive.ObjectID) ([]*models.CompanyQuestionnaire, error) {
	return s.companyQuestionnaireRepo.GetActiveByCompanyAndPeriod(ctx, companyID)
}

// UpdateCompanyQuestionnaire updates a company questionnaire assignment period
func (s *CompanyService) UpdateCompanyQuestionnaire(ctx context.Context, id primitive.ObjectID, periodStart, periodEnd time.Time, isActive bool, userID string, isSuperAdmin bool) error {
	cq, err := s.companyQuestionnaireRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Validate company ownership for non-super-admins
	if !isSuperAdmin {
		userMeta, err := s.userMetadataRepo.GetByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("unauthorized: user metadata not found")
		}
		if cq.CompanyID != userMeta.CompanyID {
			return fmt.Errorf("unauthorized: company questionnaire not in your company")
		}
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
	// Sync status with isActive for backward compatibility
	if isActive && (cq.Status == models.CQStatusPaused || cq.Status == models.CQStatusDraft) {
		cq.Status = models.CQStatusActive
	} else if !isActive && cq.Status == models.CQStatusActive {
		cq.Status = models.CQStatusClosed
	}

	return s.companyQuestionnaireRepo.Update(ctx, id, cq)
}

// DeactivateCompanyQuestionnaire deactivates a company questionnaire
func (s *CompanyService) DeactivateCompanyQuestionnaire(ctx context.Context, id primitive.ObjectID) error {
	return s.companyQuestionnaireRepo.Deactivate(ctx, id)
}

// DeleteCompanyQuestionnaire hard-deletes a company questionnaire if no responses exist
func (s *CompanyService) DeleteCompanyQuestionnaire(ctx context.Context, cqID primitive.ObjectID, userID string, isSuperAdmin bool) error {
	cq, err := s.companyQuestionnaireRepo.GetByID(ctx, cqID)
	if err != nil {
		return err
	}

	if !isSuperAdmin {
		userMeta, err := s.userMetadataRepo.GetByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("unauthorized: user metadata not found")
		}
		if cq.CompanyID != userMeta.CompanyID {
			return fmt.Errorf("unauthorized: company questionnaire not in your company")
		}
	}

	// Check if any assignment has responses
	assignments, err := s.assignmentRepo.GetByCompanyQuestionnaireID(ctx, cqID)
	if err != nil {
		return fmt.Errorf("failed to check assignments: %w", err)
	}
	for _, a := range assignments {
		if len(a.Responses) > 0 {
			return fmt.Errorf("no se puede eliminar: existen respuestas registradas")
		}
	}

	// Cancel all pending/in-progress assignments
	if len(assignments) > 0 {
		if _, err := s.assignmentRepo.CancelAllPendingByCompanyQuestionnaire(ctx, cqID, userID, "company questionnaire deleted"); err != nil {
			return fmt.Errorf("failed to cancel assignments: %w", err)
		}
	}

	// Delete the company questionnaire
	return s.companyQuestionnaireRepo.Delete(ctx, cqID)
}

// DeactivateCompany soft-deletes a company
func (s *CompanyService) DeactivateCompany(ctx context.Context, id primitive.ObjectID) error {
	company, err := s.companyRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	company.IsActive = false
	company.UpdatedAt = time.Now()
	return s.companyRepo.Update(ctx, id, company)
}

// GetCompanyQuestionnaireByID retrieves a single CompanyQuestionnaire by ID with auth check
func (s *CompanyService) GetCompanyQuestionnaireByID(ctx context.Context, id primitive.ObjectID, userID string, isSuperAdmin bool) (*models.CompanyQuestionnaire, error) {
	cq, err := s.companyQuestionnaireRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !isSuperAdmin {
		userMeta, err := s.userMetadataRepo.GetByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("user metadata not found: %w", err)
		}
		if cq.CompanyID != userMeta.CompanyID {
			return nil, fmt.Errorf("unauthorized: company questionnaire not in your company")
		}
	}

	return cq, nil
}

// ChangeCompanyQuestionnaireStatus transitions the status of a CompanyQuestionnaire
func (s *CompanyService) ChangeCompanyQuestionnaireStatus(ctx context.Context, id primitive.ObjectID, newStatus models.CompanyQuestionnaireStatus, changedBy, reason string, isSuperAdmin bool) error {
	cq, err := s.companyQuestionnaireRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if !isSuperAdmin {
		userMeta, err := s.userMetadataRepo.GetByID(ctx, changedBy)
		if err != nil {
			return fmt.Errorf("user metadata not found: %w", err)
		}
		if cq.CompanyID != userMeta.CompanyID {
			return fmt.Errorf("unauthorized: company questionnaire not in your company")
		}
	}

	if err := cq.TransitionStatus(newStatus, changedBy, reason); err != nil {
		return err
	}

	historyEntry := cq.StatusHistory[len(cq.StatusHistory)-1]
	return s.companyQuestionnaireRepo.UpdateStatus(ctx, id, newStatus, historyEntry)
}

// GetQuestionnaireCompanies returns all companies that have a specific questionnaire assigned
func (s *CompanyService) GetQuestionnaireCompanies(ctx context.Context, questionnaireID primitive.ObjectID) ([]map[string]interface{}, error) {
	cqs, err := s.companyQuestionnaireRepo.GetByQuestionnaireID(ctx, questionnaireID)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(cqs))
	for _, cq := range cqs {
		company, err := s.companyRepo.GetByID(ctx, cq.CompanyID)
		if err != nil {
			continue
		}
		result = append(result, map[string]interface{}{
			"company_questionnaire_id": cq.ID.Hex(),
			"company_id":               cq.CompanyID.Hex(),
			"company_name":             company.Name,
			"status":                   cq.Status,
			"period_start":             cq.PeriodStart.Format("2006-01-02"),
			"period_end":               cq.PeriodEnd.Format("2006-01-02"),
			"is_active":                cq.IsActive,
		})
	}

	return result, nil
}

// AssignAllToCompany assigns a company questionnaire to every user in the company
func (s *CompanyService) AssignAllToCompany(ctx context.Context, cqID primitive.ObjectID, assignedBy string, isSuperAdmin bool) ([]string, error) {
	cq, err := s.companyQuestionnaireRepo.GetByID(ctx, cqID)
	if err != nil {
		return nil, fmt.Errorf("company questionnaire not found: %w", err)
	}

	if !isSuperAdmin {
		userMeta, err := s.userMetadataRepo.GetByID(ctx, assignedBy)
		if err != nil {
			return nil, fmt.Errorf("user metadata not found: %w", err)
		}
		if cq.CompanyID != userMeta.CompanyID {
			return nil, fmt.Errorf("unauthorized: company questionnaire not in your company")
		}
	}

	users, err := s.userMetadataRepo.GetByCompanyID(ctx, cq.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get company users: %w", err)
	}

	userIDs := make([]string, 0, len(users))
	for _, u := range users {
		userIDs = append(userIDs, u.ID)
	}

	return userIDs, nil
}

// AssignToDepartment returns user IDs for a department within the company questionnaire's company
func (s *CompanyService) AssignToDepartment(ctx context.Context, cqID primitive.ObjectID, department, assignedBy string, isSuperAdmin bool) ([]string, error) {
	cq, err := s.companyQuestionnaireRepo.GetByID(ctx, cqID)
	if err != nil {
		return nil, fmt.Errorf("company questionnaire not found: %w", err)
	}

	if !isSuperAdmin {
		userMeta, err := s.userMetadataRepo.GetByID(ctx, assignedBy)
		if err != nil {
			return nil, fmt.Errorf("user metadata not found: %w", err)
		}
		if cq.CompanyID != userMeta.CompanyID {
			return nil, fmt.Errorf("unauthorized: company questionnaire not in your company")
		}
	}

	users, err := s.userMetadataRepo.GetByCompanyAndDepartment(ctx, cq.CompanyID, department)
	if err != nil {
		return nil, fmt.Errorf("failed to get department users: %w", err)
	}

	userIDs := make([]string, 0, len(users))
	for _, u := range users {
		userIDs = append(userIDs, u.ID)
	}

	return userIDs, nil
}

// RegisterReminder records that a reminder was sent for a company questionnaire
func (s *CompanyService) RegisterReminder(ctx context.Context, cqID primitive.ObjectID, target, userID string, isSuperAdmin bool) (*ReminderResult, error) {
	cq, err := s.companyQuestionnaireRepo.GetByID(ctx, cqID)
	if err != nil {
		return nil, fmt.Errorf("company questionnaire not found: %w", err)
	}

	if !isSuperAdmin {
		userMeta, err := s.userMetadataRepo.GetByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("user metadata not found: %w", err)
		}
		if cq.CompanyID != userMeta.CompanyID {
			return nil, fmt.Errorf("unauthorized: company questionnaire not in your company")
		}
	}

	// Determine target users based on target param
	var statusFilter *models.AssignmentStatus
	switch target {
	case "pending":
		s := models.AssignmentStatusPending
		statusFilter = &s
	case "in_progress":
		s := models.AssignmentStatusInProgress
		statusFilter = &s
	}

	assignments, err := s.assignmentRepo.GetByCompanyQuestionnaireID(ctx, cqID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignments: %w", err)
	}

	targetUserIDs := make([]string, 0)
	for _, a := range assignments {
		if statusFilter == nil || a.Status == *statusFilter {
			targetUserIDs = append(targetUserIDs, a.UserID)
		}
	}

	now := time.Now()
	if err := s.companyQuestionnaireRepo.UpdateReminder(ctx, cqID, now); err != nil {
		return nil, fmt.Errorf("failed to register reminder: %w", err)
	}

	return &ReminderResult{
		CompanyQuestionnaireID: cqID.Hex(),
		Target:                 target,
		UserIDs:                targetUserIDs,
		Count:                  len(targetUserIDs),
		RemindersSent:          cq.RemindersSent + 1,
		LastReminderAt:         &now,
	}, nil
}

// GetCompanyDashboard returns all company questionnaires with completion stats
func (s *CompanyService) GetCompanyDashboard(ctx context.Context, companyID primitive.ObjectID, userID string, isSuperAdmin bool) ([]CompanyDashboardItem, error) {
	if !isSuperAdmin {
		userMeta, err := s.userMetadataRepo.GetByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("user metadata not found: %w", err)
		}
		if companyID != userMeta.CompanyID {
			return nil, fmt.Errorf("unauthorized: cannot access other company dashboard")
		}
	}

	cqs, err := s.companyQuestionnaireRepo.GetByCompanyID(ctx, companyID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get company questionnaires: %w", err)
	}

	items := make([]CompanyDashboardItem, 0, len(cqs))
	for _, cq := range cqs {
		questionnaire, err := s.questionnaireRepo.GetByID(ctx, cq.QuestionnaireID)
		if err != nil {
			continue
		}

		assignments, err := s.assignmentRepo.GetByCompanyQuestionnaireID(ctx, cq.ID)
		if err != nil {
			continue
		}

		var completed, inProgress, pending int
		for _, a := range assignments {
			switch a.Status {
			case models.AssignmentStatusCompleted:
				completed++
			case models.AssignmentStatusInProgress:
				inProgress++
			case models.AssignmentStatusPending:
				pending++
			}
		}

		total := len(assignments)
		rate := 0.0
		if total > 0 {
			rate = float64(completed) / float64(total) * 100
		}

		items = append(items, CompanyDashboardItem{
			CompanyQuestionnaireID: cq.ID.Hex(),
			QuestionnaireTitle:     questionnaire.Title,
			Status:                 cq.Status,
			PeriodStart:            cq.PeriodStart.Format("2006-01-02"),
			PeriodEnd:              cq.PeriodEnd.Format("2006-01-02"),
			TotalAssigned:          total,
			Completed:              completed,
			InProgress:             inProgress,
			Pending:                pending,
			CompletionRate:         rate,
		})
	}

	return items, nil
}

// GetCompanyStats returns statistics about a company
// PublicCompanyBranding is the subset of company data returned by the public branding endpoint
type PublicCompanyBranding struct {
	CompanyName string          `json:"company_name"`
	Branding    *models.Branding `json:"branding,omitempty"`
}

// GetCompanyBrandingBySlug looks up an active company by slug and returns only branding info.
// Returns nil, nil if the slug is not found or the company is inactive.
func (s *CompanyService) GetCompanyBrandingBySlug(ctx context.Context, slug string) (*PublicCompanyBranding, error) {
	if slug == "" {
		return nil, fmt.Errorf("slug is required")
	}
	company, err := s.companyRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if company == nil || !company.IsActive {
		return nil, nil
	}
	return &PublicCompanyBranding{
		CompanyName: company.Name,
		Branding:    company.Branding,
	}, nil
}

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
