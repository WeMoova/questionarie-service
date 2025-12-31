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

	if !cq.IsActive {
		return nil, fmt.Errorf("company questionnaire is not active")
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
