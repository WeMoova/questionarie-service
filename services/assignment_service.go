package services

import (
	"context"
	"fmt"
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
}

// NewAssignmentService creates a new AssignmentService
func NewAssignmentService(
	assignmentRepo *repository.AssignmentRepository,
	companyQuestionnaireRepo *repository.CompanyQuestionnaireRepository,
	userMetadataRepo *repository.UserMetadataRepository,
	questionnaireRepo *repository.QuestionnaireRepository,
) *AssignmentService {
	return &AssignmentService{
		assignmentRepo:           assignmentRepo,
		companyQuestionnaireRepo: companyQuestionnaireRepo,
		userMetadataRepo:         userMetadataRepo,
		questionnaireRepo:        questionnaireRepo,
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

	if len(assignments) == 0 {
		return nil, fmt.Errorf("no new assignments created (all users already assigned)")
	}

	return assignments, nil
}

// GetAssignmentByID retrieves an assignment by ID
func (s *AssignmentService) GetAssignmentByID(ctx context.Context, id primitive.ObjectID) (*models.UserQuestionnaireAssignment, error) {
	return s.assignmentRepo.GetByID(ctx, id)
}

// GetUserAssignments retrieves all assignments for a user
func (s *AssignmentService) GetUserAssignments(ctx context.Context, userID string, status *models.AssignmentStatus) ([]*models.UserQuestionnaireAssignment, error) {
	return s.assignmentRepo.GetByUserID(ctx, userID, status)
}

// GetCompanyQuestionnaireAssignments retrieves all assignments for a company questionnaire
func (s *AssignmentService) GetCompanyQuestionnaireAssignments(ctx context.Context, cqID primitive.ObjectID) ([]*models.UserQuestionnaireAssignment, error) {
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
	return s.assignmentRepo.UpdateStatus(ctx, assignmentID, models.AssignmentStatusCompleted)
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
