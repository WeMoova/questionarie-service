package services

import (
	"context"
	"fmt"
	"log/slog"
	"questionarie-service/models"
	"questionarie-service/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AssignmentService handles business logic for user assignments
type AssignmentService struct {
	assignmentRepo           *repository.AssignmentRepository
	companyQuestionnaireRepo *repository.CompanyQuestionnaireRepository
	userMetadataRepo         *repository.UserMetadataRepository
	questionnaireRepo        *repository.QuestionnaireRepository
	companyRepo              *repository.CompanyRepository
	categoryRepo             *repository.CategoryRepository
	gamificationService      *GamificationService
	evaluationService        *EvaluationService
}

// NewAssignmentService creates a new AssignmentService
func NewAssignmentService(
	assignmentRepo *repository.AssignmentRepository,
	companyQuestionnaireRepo *repository.CompanyQuestionnaireRepository,
	userMetadataRepo *repository.UserMetadataRepository,
	questionnaireRepo *repository.QuestionnaireRepository,
	companyRepo *repository.CompanyRepository,
	categoryRepo *repository.CategoryRepository,
	gamificationService *GamificationService,
	evaluationService *EvaluationService,
) *AssignmentService {
	return &AssignmentService{
		assignmentRepo:           assignmentRepo,
		companyQuestionnaireRepo: companyQuestionnaireRepo,
		userMetadataRepo:         userMetadataRepo,
		questionnaireRepo:        questionnaireRepo,
		companyRepo:              companyRepo,
		categoryRepo:             categoryRepo,
		gamificationService:      gamificationService,
		evaluationService:        evaluationService,
	}
}

// AssignmentProgress holds detailed progress info for a company questionnaire
type AssignmentProgress struct {
	CompanyQuestionnaireID string                            `json:"company_questionnaire_id"`
	Status                 models.CompanyQuestionnaireStatus `json:"status"`
	TotalAssigned          int                               `json:"total_assigned"`
	Completed              int                               `json:"completed"`
	InProgress             int                               `json:"in_progress"`
	Pending                int                               `json:"pending"`
	Cancelled              int                               `json:"cancelled"`
	CompletionRate         float64                           `json:"completion_rate"`
	Users                  []UserProgressItem                `json:"users"`
}

// UserProgressItem holds per-user progress
type UserProgressItem struct {
	UserID      string                   `json:"user_id"`
	Status      models.AssignmentStatus  `json:"status"`
	ProgressPct float64                  `json:"progress_pct"`
	AssignedAt  string                   `json:"assigned_at"`
	CompletedAt *string                  `json:"completed_at,omitempty"`
	StartedAt   *string                  `json:"started_at,omitempty"`
}

// AssignToUsers assigns a company questionnaire to multiple users
func (s *AssignmentService) AssignToUsers(
	ctx context.Context,
	assignedBy string,
	companyQuestionnaireID primitive.ObjectID,
	userIDs []string,
	isSuperAdmin bool,
) ([]*models.UserQuestionnaireAssignment, error) {
	if len(userIDs) == 0 {
		return nil, fmt.Errorf("user IDs list cannot be empty")
	}

	// Get company questionnaire
	cq, err := s.companyQuestionnaireRepo.GetByID(ctx, companyQuestionnaireID)
	if err != nil {
		return nil, fmt.Errorf("company questionnaire not found: %w", err)
	}

	if cq.Status != models.CQStatusActive {
		return nil, fmt.Errorf("company questionnaire is not active (current status: %s)", cq.Status)
	}

	// Verify questionnaire has questions
	questionnaire, err := s.questionnaireRepo.GetByID(ctx, cq.QuestionnaireID)
	if err != nil {
		return nil, fmt.Errorf("questionnaire not found: %w", err)
	}
	if len(questionnaire.Questions) == 0 {
		return nil, fmt.Errorf("questionnaire has no questions")
	}

	// If not super admin, verify authorization
	if !isSuperAdmin {
		// Get assigner metadata
		assignerMeta, err := s.userMetadataRepo.GetByID(ctx, assignedBy)
		if err != nil {
			return nil, fmt.Errorf("assigner metadata not found: %w", err)
		}

		// Verify company questionnaire belongs to assigner's company
		if cq.CompanyID != assignerMeta.CompanyID {
			return nil, fmt.Errorf("unauthorized: company questionnaire not in your company")
		}

		// Verify all target users belong to the same company
		for _, targetUserID := range userIDs {
			targetMeta, err := s.userMetadataRepo.GetByID(ctx, targetUserID)
			if err != nil {
				return nil, fmt.Errorf("user metadata not found for user %s: %w", targetUserID, err)
			}

			if targetMeta.CompanyID != assignerMeta.CompanyID {
				return nil, fmt.Errorf("unauthorized: cannot assign to user %s outside your company", targetUserID)
			}
		}
	}

	// Create assignments
	assignments := make([]*models.UserQuestionnaireAssignment, 0, len(userIDs))

	for _, userID := range userIDs {
		// Check for duplicate
		isDuplicate, err := s.assignmentRepo.CheckDuplicate(ctx, userID, companyQuestionnaireID)
		if err != nil {
			return nil, fmt.Errorf("failed to check duplicate for user %s: %w", userID, err)
		}
		if isDuplicate {
			// Skip duplicate, don't fail the entire operation
			continue
		}

		// Create assignment
		assignment := models.NewUserQuestionnaireAssignment(companyQuestionnaireID, userID, assignedBy)

		if err := s.assignmentRepo.Create(ctx, assignment); err != nil {
			return nil, fmt.Errorf("failed to create assignment for user %s: %w", userID, err)
		}

		assignments = append(assignments, assignment)
	}

	return assignments, nil
}

// GetAssignmentByID retrieves an assignment by ID (enriched with questionnaire data)
func (s *AssignmentService) GetAssignmentByID(ctx context.Context, id primitive.ObjectID) (*models.UserQuestionnaireAssignment, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.enrichAssignments(ctx, []*models.UserQuestionnaireAssignment{assignment})
	return assignment, nil
}

// GetUserAssignments retrieves all assignments for a user (enriched with questionnaire data)
func (s *AssignmentService) GetUserAssignments(ctx context.Context, userID string, status *models.AssignmentStatus) ([]*models.UserQuestionnaireAssignment, error) {
	assignments, err := s.assignmentRepo.GetByUserID(ctx, userID, status)
	if err != nil {
		return nil, err
	}
	s.enrichAssignments(ctx, assignments)
	return assignments, nil
}

// AssignmentQuestionsResult holds questions and sections for an assignment
type AssignmentQuestionsResult struct {
	Questions []models.Question `json:"questions"`
	Sections  []models.Section  `json:"sections,omitempty"`
}

// GetAssignmentQuestions returns the questions and sections for the questionnaire associated with an assignment
func (s *AssignmentService) GetAssignmentQuestions(ctx context.Context, assignmentID primitive.ObjectID) (*AssignmentQuestionsResult, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}

	cq, err := s.companyQuestionnaireRepo.GetByID(ctx, assignment.CompanyQuestionnaireID)
	if err != nil {
		return nil, fmt.Errorf("company questionnaire not found: %w", err)
	}

	questionnaire, err := s.questionnaireRepo.GetByID(ctx, cq.QuestionnaireID)
	if err != nil {
		return nil, fmt.Errorf("questionnaire not found: %w", err)
	}

	return &AssignmentQuestionsResult{
		Questions: questionnaire.Questions,
		Sections:  questionnaire.Sections,
	}, nil
}

// enrichAssignments populates transient fields on assignments (questionnaire title, company name, etc.)
func (s *AssignmentService) enrichAssignments(ctx context.Context, assignments []*models.UserQuestionnaireAssignment) {
	// Cache to avoid repeated lookups
	cqCache := make(map[primitive.ObjectID]*models.CompanyQuestionnaire)
	qCache := make(map[primitive.ObjectID]*models.Questionnaire)
	companyCache := make(map[primitive.ObjectID]*models.Company)
	categoryCache := make(map[primitive.ObjectID]string)

	for _, a := range assignments {
		// Get company questionnaire
		cq, ok := cqCache[a.CompanyQuestionnaireID]
		if !ok {
			var err error
			cq, err = s.companyQuestionnaireRepo.GetByID(ctx, a.CompanyQuestionnaireID)
			if err != nil {
				continue
			}
			cqCache[a.CompanyQuestionnaireID] = cq
		}

		// Get questionnaire
		q, ok := qCache[cq.QuestionnaireID]
		if !ok {
			var err error
			q, err = s.questionnaireRepo.GetByID(ctx, cq.QuestionnaireID)
			if err != nil {
				continue
			}
			qCache[cq.QuestionnaireID] = q
		}

		a.QuestionnaireTitle = q.Title
		a.QuestionnaireDescription = q.Description
		a.TotalQuestions = len(q.Questions)
		a.QuestionnaireID = q.ID.Hex()

		// Get category name
		if q.CategoryID != nil {
			catName, ok := categoryCache[*q.CategoryID]
			if !ok {
				cat, err := s.categoryRepo.GetByID(ctx, *q.CategoryID)
				if err == nil {
					catName = cat.Name
				}
				categoryCache[*q.CategoryID] = catName
			}
			a.QuestionnaireCategory = catName
		}

		// Get company name
		company, ok := companyCache[cq.CompanyID]
		if !ok {
			var err error
			company, err = s.companyRepo.GetByID(ctx, cq.CompanyID)
			if err != nil {
				continue
			}
			companyCache[cq.CompanyID] = company
		}
		a.CompanyName = company.Name

		// Set display_mode from company questionnaire
		if string(cq.DisplayMode) != "" {
			a.DisplayMode = string(cq.DisplayMode)
		} else {
			a.DisplayMode = "step_by_step"
		}
	}
}

// GetCompanyQuestionnaireAssignments retrieves all assignments for a company questionnaire
func (s *AssignmentService) GetCompanyQuestionnaireAssignments(ctx context.Context, cqID primitive.ObjectID, userID string, isSuperAdmin bool) ([]*models.UserQuestionnaireAssignment, error) {
	cq, err := s.companyQuestionnaireRepo.GetByID(ctx, cqID)
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

	return s.assignmentRepo.GetByCompanyQuestionnaireID(ctx, cqID)
}

// SaveResponse saves or updates a response for a question
func (s *AssignmentService) SaveResponse(
	ctx context.Context,
	assignmentID primitive.ObjectID,
	userID string,
	questionID string,
	responseValue interface{},
) error {
	// Get assignment
	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return err
	}

	// Verify ownership
	if assignment.UserID != userID {
		return fmt.Errorf("unauthorized: assignment does not belong to user")
	}

	// Verify assignment is not completed
	if assignment.Status == models.AssignmentStatusCompleted {
		return fmt.Errorf("cannot modify completed assignment")
	}

	// Get company questionnaire to check period
	cq, err := s.companyQuestionnaireRepo.GetByID(ctx, assignment.CompanyQuestionnaireID)
	if err != nil {
		return err
	}

	// Verify period is active
	if !cq.IsWithinPeriod() {
		return fmt.Errorf("questionnaire period has expired")
	}

	// Create response
	response := models.NewResponse(questionID, responseValue)

	// Add/update response
	return s.assignmentRepo.AddOrUpdateResponse(ctx, assignmentID, *response)
}

// SubmitAssignment marks an assignment as completed
func (s *AssignmentService) SubmitAssignment(ctx context.Context, assignmentID primitive.ObjectID, userID string) error {
	// Get assignment
	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return err
	}

	// Verify ownership
	if assignment.UserID != userID {
		return fmt.Errorf("unauthorized: assignment does not belong to user")
	}

	// Verify assignment is not already completed
	if assignment.Status == models.AssignmentStatusCompleted {
		return fmt.Errorf("assignment already completed")
	}

	// Get questionnaire to validate all required questions are answered
	cq, err := s.companyQuestionnaireRepo.GetByID(ctx, assignment.CompanyQuestionnaireID)
	if err != nil {
		return err
	}

	questionnaire, err := s.questionnaireRepo.GetByID(ctx, cq.QuestionnaireID)
	if err != nil {
		return err
	}

	// Count required questions
	requiredCount := 0
	for _, q := range questionnaire.Questions {
		if q.IsRequired {
			requiredCount++
		}
	}

	// Count answered required questions
	answeredRequired := 0
	for _, response := range assignment.Responses {
		for _, q := range questionnaire.Questions {
			if q.QuestionID == response.QuestionID && q.IsRequired {
				answeredRequired++
				break
			}
		}
	}

	if answeredRequired < requiredCount {
		return fmt.Errorf("not all required questions answered (%d/%d)", answeredRequired, requiredCount)
	}

	// Mark as completed
	if err := s.assignmentRepo.UpdateStatus(ctx, assignmentID, models.AssignmentStatusCompleted); err != nil {
		return err
	}

	// Evaluate assignment if questionnaire has evaluation config
	if s.evaluationService != nil {
		evalResult, err := s.evaluationService.EvaluateAssignment(ctx, assignmentID, questionnaire.ID)
		if err != nil {
			slog.Error("evaluation failed", "assignment_id", assignmentID.Hex(), "error", err)
		} else if evalResult != nil {
			if err := s.assignmentRepo.SetEvaluationResult(ctx, assignmentID, evalResult); err != nil {
				slog.Error("failed to store evaluation result", "assignment_id", assignmentID.Hex(), "error", err)
			}
		}
	}

	// Award gamification points (fire-and-forget, don't block submission)
	if s.gamificationService != nil {
		go func() {
			bgCtx := context.Background()
			s.gamificationService.AwardPointsForCompletion(bgCtx, userID, assignmentID.Hex())
			s.gamificationService.UpdateStreak(bgCtx, userID)
			s.gamificationService.CheckAndAwardBadges(bgCtx, userID)
			s.gamificationService.CheckAndAwardAchievements(bgCtx, userID)
			slog.Info("gamification completed", "user_id", userID, "assignment_id", assignmentID.Hex())
		}()
	}

	return nil
}

// DeleteAssignment deletes an assignment
func (s *AssignmentService) DeleteAssignment(ctx context.Context, assignmentID primitive.ObjectID) error {
	return s.assignmentRepo.Delete(ctx, assignmentID)
}

// GetMyTeamAssignments retrieves assignments for users supervised by the given supervisor
func (s *AssignmentService) GetMyTeamAssignments(ctx context.Context, supervisorID string) ([]*models.UserQuestionnaireAssignment, error) {
	// Get all users supervised by this supervisor
	users, err := s.userMetadataRepo.GetBySupervisorID(ctx, supervisorID)
	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return []*models.UserQuestionnaireAssignment{}, nil
	}

	// Get assignments for all supervised users
	var allAssignments []*models.UserQuestionnaireAssignment
	for _, user := range users {
		assignments, err := s.assignmentRepo.GetByUserID(ctx, user.ID, nil)
		if err != nil {
			continue
		}
		allAssignments = append(allAssignments, assignments...)
	}

	return allAssignments, nil
}

// GetMyCompanyQuestionnaires retrieves active questionnaires for a company admin
func (s *AssignmentService) GetMyCompanyQuestionnaires(ctx context.Context, userID string) ([]*models.CompanyQuestionnaire, error) {
	// Get user metadata to find company
	userMeta, err := s.userMetadataRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user metadata not found: %w", err)
	}

	// Get company questionnaires
	return s.companyQuestionnaireRepo.GetByCompanyID(ctx, userMeta.CompanyID, true)
}

// CancelAssignment cancels a specific assignment (supervisor or admin only)
func (s *AssignmentService) CancelAssignment(ctx context.Context, assignmentID primitive.ObjectID, cancelledBy, reason string, isSuperAdmin bool) error {
	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return err
	}

	if assignment.Status == models.AssignmentStatusCompleted {
		return fmt.Errorf("cannot cancel a completed assignment")
	}
	if assignment.Status == models.AssignmentStatusCancelled {
		return fmt.Errorf("assignment is already cancelled")
	}

	if !isSuperAdmin {
		// Verify the canceller belongs to the same company as the assignment
		cq, err := s.companyQuestionnaireRepo.GetByID(ctx, assignment.CompanyQuestionnaireID)
		if err != nil {
			return err
		}
		cancellerMeta, err := s.userMetadataRepo.GetByID(ctx, cancelledBy)
		if err != nil {
			return fmt.Errorf("canceller metadata not found: %w", err)
		}
		if cq.CompanyID != cancellerMeta.CompanyID {
			return fmt.Errorf("unauthorized: assignment does not belong to your company")
		}
	}

	return s.assignmentRepo.CancelAssignment(ctx, assignmentID, cancelledBy, reason)
}

// CancelAllPendingByCompanyQuestionnaire cancels all non-completed assignments for a CQ
func (s *AssignmentService) CancelAllPendingByCompanyQuestionnaire(ctx context.Context, cqID primitive.ObjectID, cancelledBy string, reason string, isSuperAdmin bool) (int64, error) {
	cq, err := s.companyQuestionnaireRepo.GetByID(ctx, cqID)
	if err != nil {
		return 0, err
	}

	if !isSuperAdmin {
		cancellerMeta, err := s.userMetadataRepo.GetByID(ctx, cancelledBy)
		if err != nil {
			return 0, fmt.Errorf("canceller metadata not found: %w", err)
		}
		if cq.CompanyID != cancellerMeta.CompanyID {
			return 0, fmt.Errorf("unauthorized: company questionnaire not in your company")
		}
	}

	return s.assignmentRepo.CancelAllPendingByCompanyQuestionnaire(ctx, cqID, cancelledBy, reason)
}

// GetAssignmentProgress returns detailed progress for a company questionnaire
func (s *AssignmentService) GetAssignmentProgress(ctx context.Context, cqID primitive.ObjectID, userID string, isSuperAdmin bool) (*AssignmentProgress, error) {
	cq, err := s.companyQuestionnaireRepo.GetByID(ctx, cqID)
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

	questionnaire, err := s.questionnaireRepo.GetByID(ctx, cq.QuestionnaireID)
	if err != nil {
		return nil, err
	}
	totalQuestions := len(questionnaire.Questions)

	assignments, err := s.assignmentRepo.GetByCompanyQuestionnaireID(ctx, cqID)
	if err != nil {
		return nil, err
	}

	var completed, inProgress, pending, cancelled int
	userItems := make([]UserProgressItem, 0, len(assignments))

	for _, a := range assignments {
		switch a.Status {
		case models.AssignmentStatusCompleted:
			completed++
		case models.AssignmentStatusInProgress:
			inProgress++
		case models.AssignmentStatusPending:
			pending++
		case models.AssignmentStatusCancelled:
			cancelled++
		}

		progressPct := 0.0
		if totalQuestions > 0 {
			progressPct = float64(len(a.Responses)) / float64(totalQuestions) * 100
		}

		item := UserProgressItem{
			UserID:      a.UserID,
			Status:      a.Status,
			ProgressPct: progressPct,
			AssignedAt:  a.AssignedAt.Format("2006-01-02T15:04:05Z"),
		}
		if a.CompletedAt != nil {
			s := a.CompletedAt.Format("2006-01-02T15:04:05Z")
			item.CompletedAt = &s
		}
		if a.StartedAt != nil {
			s := a.StartedAt.Format("2006-01-02T15:04:05Z")
			item.StartedAt = &s
		}
		userItems = append(userItems, item)
	}

	total := len(assignments)
	rate := 0.0
	if total > 0 {
		rate = float64(completed) / float64(total) * 100
	}

	return &AssignmentProgress{
		CompanyQuestionnaireID: cqID.Hex(),
		Status:                 cq.Status,
		TotalAssigned:          total,
		Completed:              completed,
		InProgress:             inProgress,
		Pending:                pending,
		Cancelled:              cancelled,
		CompletionRate:         rate,
		Users:                  userItems,
	}, nil
}

// GetAssignmentsByStatus returns assignments for a CQ filtered by status
func (s *AssignmentService) GetAssignmentsByStatus(ctx context.Context, cqID primitive.ObjectID, status models.AssignmentStatus, userID string, isSuperAdmin bool) ([]*models.UserQuestionnaireAssignment, error) {
	cq, err := s.companyQuestionnaireRepo.GetByID(ctx, cqID)
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

	return s.assignmentRepo.GetByCompanyQuestionnaireIDAndStatus(ctx, cqID, status)
}

// GetMyQuestionnaires returns active company questionnaires for the user's company
// with the user's assignment status for each one.
func (s *AssignmentService) GetMyQuestionnaires(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	// Get user's company
	userMeta, err := s.userMetadataRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user metadata not found: %w", err)
	}

	// Get active CQs for the company
	cqs, err := s.companyQuestionnaireRepo.GetByCompanyID(ctx, userMeta.CompanyID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get company questionnaires: %w", err)
	}

	result := make([]map[string]interface{}, 0, len(cqs))
	for _, cq := range cqs {
		// Only show active CQs to employees
		if cq.Status != models.CQStatusActive {
			continue
		}

		displayMode := string(cq.DisplayMode)
		if displayMode == "" {
			displayMode = "step_by_step"
		}

		item := map[string]interface{}{
			"company_questionnaire_id": cq.ID.Hex(),
			"period_start":             cq.PeriodStart,
			"period_end":               cq.PeriodEnd,
			"assignment_status":        "not_started",
			"display_mode":             displayMode,
		}

		// Enrich with questionnaire data
		q, err := s.questionnaireRepo.GetByID(ctx, cq.QuestionnaireID)
		if err == nil {
			item["questionnaire_title"] = q.Title
			item["questionnaire_description"] = q.Description
			item["total_questions"] = len(q.Questions)
			if q.CategoryID != nil {
				cat, catErr := s.categoryRepo.GetByID(ctx, *q.CategoryID)
				if catErr == nil {
					item["questionnaire_category"] = cat.Name
				}
			}
		}

		// Check if user has an existing assignment
		assignment, err := s.assignmentRepo.GetByUserAndCQ(ctx, userID, cq.ID)
		if err == nil && assignment != nil {
			item["assignment_id"] = assignment.ID.Hex()
			item["assignment_status"] = string(assignment.Status)
		}

		result = append(result, item)
	}

	return result, nil
}

// StartQuestionnaire creates an assignment for a user if one doesn't exist,
// or returns the existing one. Used when an employee clicks "Start" on a company questionnaire.
func (s *AssignmentService) StartQuestionnaire(ctx context.Context, userID string, cqID primitive.ObjectID) (*models.UserQuestionnaireAssignment, error) {
	// Get and validate CQ
	cq, err := s.companyQuestionnaireRepo.GetByID(ctx, cqID)
	if err != nil {
		return nil, fmt.Errorf("company questionnaire not found: %w", err)
	}
	if cq.Status != models.CQStatusActive {
		return nil, fmt.Errorf("company questionnaire is not active")
	}

	// Validate user belongs to the same company
	userMeta, err := s.userMetadataRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user metadata not found: %w", err)
	}
	if cq.CompanyID != userMeta.CompanyID {
		return nil, fmt.Errorf("unauthorized: questionnaire not in your company")
	}

	// Check for existing non-cancelled assignment
	existing, err := s.assignmentRepo.GetByUserAndCQ(ctx, userID, cqID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing assignment: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	// Create new assignment
	assignment := models.NewUserQuestionnaireAssignment(cqID, userID, "self")
	if err := s.assignmentRepo.Create(ctx, assignment); err != nil {
		return nil, fmt.Errorf("failed to create assignment: %w", err)
	}

	return assignment, nil
}
