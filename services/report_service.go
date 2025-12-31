package services

import (
	"context"
	"fmt"
	"questionarie-service/models"
	"questionarie-service/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ReportService handles business logic for reports and analytics
type ReportService struct {
	assignmentRepo           *repository.AssignmentRepository
	companyQuestionnaireRepo *repository.CompanyQuestionnaireRepository
	userMetadataRepo         *repository.UserMetadataRepository
	questionnaireRepo        *repository.QuestionnaireRepository
	companyRepo              *repository.CompanyRepository
}

// NewReportService creates a new ReportService
func NewReportService(
	assignmentRepo *repository.AssignmentRepository,
	companyQuestionnaireRepo *repository.CompanyQuestionnaireRepository,
	userMetadataRepo *repository.UserMetadataRepository,
	questionnaireRepo *repository.QuestionnaireRepository,
	companyRepo *repository.CompanyRepository,
) *ReportService {
	return &ReportService{
		assignmentRepo:           assignmentRepo,
		companyQuestionnaireRepo: companyQuestionnaireRepo,
		userMetadataRepo:         userMetadataRepo,
		questionnaireRepo:        questionnaireRepo,
		companyRepo:              companyRepo,
	}
}

// CompletionMetrics represents completion metrics for a questionnaire
type CompletionMetrics struct {
	CompanyQuestionnaireID primitive.ObjectID         `json:"company_questionnaire_id"`
	QuestionnaireTitle     string                     `json:"questionnaire_title"`
	CompanyName            string                     `json:"company_name"`
	PeriodStart            string                     `json:"period_start"`
	PeriodEnd              string                     `json:"period_end"`
	TotalEmployees         int64                      `json:"total_employees"`
	Assigned               int64                      `json:"assigned"`
	Pending                int64                      `json:"pending"`
	InProgress             int64                      `json:"in_progress"`
	Completed              int64                      `json:"completed"`
	NotStarted             int64                      `json:"not_started"`
	CompletionPercentage   float64                    `json:"completion_percentage"`
	AvgTimeToComplete      float64                    `json:"average_time_to_complete_minutes"`
	CompletionByDepartment []DepartmentCompletionStat `json:"completion_by_department,omitempty"`
}

// DepartmentCompletionStat represents completion statistics by department
type DepartmentCompletionStat struct {
	Department string  `json:"department"`
	Completed  int64   `json:"completed"`
	Total      int64   `json:"total"`
	Percentage float64 `json:"percentage"`
}

// GetCompletionMetrics retrieves completion metrics for a company questionnaire
func (s *ReportService) GetCompletionMetrics(ctx context.Context, companyQuestionnaireID primitive.ObjectID, userID string, isSuperAdmin bool) (*CompletionMetrics, error) {
	// Get company questionnaire
	cq, err := s.companyQuestionnaireRepo.GetByID(ctx, companyQuestionnaireID)
	if err != nil {
		return nil, fmt.Errorf("company questionnaire not found: %w", err)
	}

	// Verify authorization if not super admin
	if !isSuperAdmin {
		userMeta, err := s.userMetadataRepo.GetByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("user metadata not found: %w", err)
		}

		if cq.CompanyID != userMeta.CompanyID {
			return nil, fmt.Errorf("unauthorized: cannot access reports from other companies")
		}
	}

	// Get company info
	company, err := s.companyRepo.GetByID(ctx, cq.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("company not found: %w", err)
	}

	// Get questionnaire info
	questionnaire, err := s.questionnaireRepo.GetByID(ctx, cq.QuestionnaireID)
	if err != nil {
		return nil, fmt.Errorf("questionnaire not found: %w", err)
	}

	// Get total employees in company
	totalEmployees, err := s.userMetadataRepo.CountByCompany(ctx, cq.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("failed to count employees: %w", err)
	}

	// Get assignments
	assignments, err := s.assignmentRepo.GetByCompanyQuestionnaireID(ctx, companyQuestionnaireID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignments: %w", err)
	}

	// Get completion stats
	stats, err := s.assignmentRepo.GetCompletionStats(ctx, companyQuestionnaireID)
	if err != nil {
		return nil, fmt.Errorf("failed to get completion stats: %w", err)
	}

	// Get average completion time
	avgTime, err := s.assignmentRepo.GetAverageCompletionTime(ctx, companyQuestionnaireID)
	if err != nil {
		return nil, fmt.Errorf("failed to get average completion time: %w", err)
	}

	assigned := int64(len(assignments))
	pending := stats["pending"]
	inProgress := stats["in_progress"]
	completed := stats["completed"]
	notStarted := pending

	completionPercentage := 0.0
	if assigned > 0 {
		completionPercentage = (float64(completed) / float64(assigned)) * 100
	}

	metrics := &CompletionMetrics{
		CompanyQuestionnaireID: companyQuestionnaireID,
		QuestionnaireTitle:     questionnaire.Title,
		CompanyName:            company.Name,
		PeriodStart:            cq.PeriodStart.Format("2006-01-02"),
		PeriodEnd:              cq.PeriodEnd.Format("2006-01-02"),
		TotalEmployees:         totalEmployees,
		Assigned:               assigned,
		Pending:                pending,
		InProgress:             inProgress,
		Completed:              completed,
		NotStarted:             notStarted,
		CompletionPercentage:   completionPercentage,
		AvgTimeToComplete:      avgTime,
	}

	// Get completion by department
	deptStats, err := s.getCompletionByDepartment(ctx, cq.CompanyID, assignments)
	if err == nil {
		metrics.CompletionByDepartment = deptStats
	}

	return metrics, nil
}

// getCompletionByDepartment calculates completion statistics by department
func (s *ReportService) getCompletionByDepartment(ctx context.Context, companyID primitive.ObjectID, assignments []*models.UserQuestionnaireAssignment) ([]DepartmentCompletionStat, error) {
	// Get all users in company
	users, err := s.userMetadataRepo.GetByCompanyID(ctx, companyID)
	if err != nil {
		return nil, err
	}

	// Create map of user ID to department
	userDepts := make(map[string]string)
	for _, user := range users {
		if user.Department != "" {
			userDepts[user.ID] = user.Department
		}
	}

	// Count assignments by department
	deptStats := make(map[string]*DepartmentCompletionStat)

	for _, assignment := range assignments {
		dept := userDepts[assignment.UserID]
		if dept == "" {
			dept = "Unassigned"
		}

		if _, exists := deptStats[dept]; !exists {
			deptStats[dept] = &DepartmentCompletionStat{
				Department: dept,
			}
		}

		deptStats[dept].Total++
		if assignment.Status == models.AssignmentStatusCompleted {
			deptStats[dept].Completed++
		}
	}

	// Calculate percentages
	result := make([]DepartmentCompletionStat, 0, len(deptStats))
	for _, stat := range deptStats {
		if stat.Total > 0 {
			stat.Percentage = (float64(stat.Completed) / float64(stat.Total)) * 100
		}
		result = append(result, *stat)
	}

	return result, nil
}

// CompanyOverview represents overview statistics for a company
type CompanyOverview struct {
	CompanyID              string                       `json:"company_id"`
	CompanyName            string                       `json:"company_name"`
	TotalEmployees         int64                        `json:"total_employees"`
	TotalQuestionnaires    int                          `json:"total_questionnaires"`
	ActiveQuestionnaires   int                          `json:"active_questionnaires"`
	TotalAssignments       int                          `json:"total_assignments"`
	CompletedAssignments   int                          `json:"completed_assignments"`
	OverallCompletion      float64                      `json:"overall_completion_percentage"`
	QuestionnaireBreakdown []QuestionnaireBreakdownStat `json:"questionnaire_breakdown"`
}

// QuestionnaireBreakdownStat represents statistics for a specific questionnaire
type QuestionnaireBreakdownStat struct {
	QuestionnaireID      string  `json:"questionnaire_id"`
	QuestionnaireTitle   string  `json:"questionnaire_title"`
	Assigned             int     `json:"assigned"`
	Completed            int     `json:"completed"`
	CompletionPercentage float64 `json:"completion_percentage"`
}

// GetCompanyOverview retrieves overview statistics for a company
func (s *ReportService) GetCompanyOverview(ctx context.Context, companyID primitive.ObjectID, userID string, isSuperAdmin bool) (*CompanyOverview, error) {
	// Verify authorization if not super admin
	if !isSuperAdmin {
		userMeta, err := s.userMetadataRepo.GetByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("user metadata not found: %w", err)
		}

		if companyID != userMeta.CompanyID {
			return nil, fmt.Errorf("unauthorized: cannot access reports from other companies")
		}
	}

	// Get company info
	company, err := s.companyRepo.GetByID(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("company not found: %w", err)
	}

	// Get total employees
	totalEmployees, err := s.userMetadataRepo.CountByCompany(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to count employees: %w", err)
	}

	// Get all company questionnaires
	companyQuestionnaires, err := s.companyQuestionnaireRepo.GetByCompanyID(ctx, companyID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get company questionnaires: %w", err)
	}

	activeCount := 0
	for _, cq := range companyQuestionnaires {
		if cq.IsActive {
			activeCount++
		}
	}

	// Get breakdown for each questionnaire
	breakdown := make([]QuestionnaireBreakdownStat, 0, len(companyQuestionnaires))
	totalAssignments := 0
	totalCompleted := 0

	for _, cq := range companyQuestionnaires {
		questionnaire, err := s.questionnaireRepo.GetByID(ctx, cq.QuestionnaireID)
		if err != nil {
			continue
		}

		assignments, err := s.assignmentRepo.GetByCompanyQuestionnaireID(ctx, cq.ID)
		if err != nil {
			continue
		}

		completed := 0
		for _, a := range assignments {
			if a.Status == models.AssignmentStatusCompleted {
				completed++
			}
		}

		assigned := len(assignments)
		totalAssignments += assigned
		totalCompleted += completed

		completionPct := 0.0
		if assigned > 0 {
			completionPct = (float64(completed) / float64(assigned)) * 100
		}

		breakdown = append(breakdown, QuestionnaireBreakdownStat{
			QuestionnaireID:      cq.QuestionnaireID.Hex(),
			QuestionnaireTitle:   questionnaire.Title,
			Assigned:             assigned,
			Completed:            completed,
			CompletionPercentage: completionPct,
		})
	}

	overallCompletion := 0.0
	if totalAssignments > 0 {
		overallCompletion = (float64(totalCompleted) / float64(totalAssignments)) * 100
	}

	return &CompanyOverview{
		CompanyID:              companyID.Hex(),
		CompanyName:            company.Name,
		TotalEmployees:         totalEmployees,
		TotalQuestionnaires:    len(companyQuestionnaires),
		ActiveQuestionnaires:   activeCount,
		TotalAssignments:       totalAssignments,
		CompletedAssignments:   totalCompleted,
		OverallCompletion:      overallCompletion,
		QuestionnaireBreakdown: breakdown,
	}, nil
}

// GetEmployeeProgress retrieves progress information for all employees in a company
func (s *ReportService) GetEmployeeProgress(ctx context.Context, companyID primitive.ObjectID, userID string, isSuperAdmin bool) ([]map[string]interface{}, error) {
	// Verify authorization
	if !isSuperAdmin {
		userMeta, err := s.userMetadataRepo.GetByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("user metadata not found: %w", err)
		}

		if companyID != userMeta.CompanyID {
			return nil, fmt.Errorf("unauthorized: cannot access reports from other companies")
		}
	}

	// Get all employees in company
	employees, err := s.userMetadataRepo.GetByCompanyID(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get employees: %w", err)
	}

	progress := make([]map[string]interface{}, 0, len(employees))

	for _, employee := range employees {
		// Get assignments for employee
		assignments, err := s.assignmentRepo.GetByUserID(ctx, employee.ID, nil)
		if err != nil {
			continue
		}

		totalAssignments := len(assignments)
		completed := 0
		inProgress := 0
		pending := 0

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

		completionRate := 0.0
		if totalAssignments > 0 {
			completionRate = (float64(completed) / float64(totalAssignments)) * 100
		}

		progress = append(progress, map[string]interface{}{
			"user_id":          employee.ID,
			"department":       employee.Department,
			"total_assigned":   totalAssignments,
			"completed":        completed,
			"in_progress":      inProgress,
			"pending":          pending,
			"completion_rate":  completionRate,
		})
	}

	return progress, nil
}
