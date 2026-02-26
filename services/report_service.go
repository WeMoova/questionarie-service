package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"questionarie-service/models"
	"questionarie-service/repository"
	"strconv"
	"time"

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

// QuestionAnswerDistribution holds answer stats for a single question
type QuestionAnswerDistribution struct {
	QuestionID   string                 `json:"question_id"`
	QuestionText string                 `json:"question_text"`
	QuestionType string                 `json:"question_type"`
	TotalAnswers int                    `json:"total_answers"`
	Distribution map[string]int         `json:"distribution"`
	Average      *float64               `json:"average,omitempty"`
}

// GetAnswerDistribution returns answer distribution per question for a company questionnaire
func (s *ReportService) GetAnswerDistribution(ctx context.Context, cqID primitive.ObjectID, userID string, isSuperAdmin bool) ([]QuestionAnswerDistribution, error) {
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
			return nil, fmt.Errorf("unauthorized: cannot access reports from other companies")
		}
	}

	questionnaire, err := s.questionnaireRepo.GetByID(ctx, cq.QuestionnaireID)
	if err != nil {
		return nil, fmt.Errorf("questionnaire not found: %w", err)
	}

	assignments, err := s.assignmentRepo.GetByCompanyQuestionnaireID(ctx, cqID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignments: %w", err)
	}

	// Build distribution per question
	result := make([]QuestionAnswerDistribution, 0, len(questionnaire.Questions))
	for _, q := range questionnaire.Questions {
		dist := QuestionAnswerDistribution{
			QuestionID:   q.QuestionID,
			QuestionText: q.QuestionText,
			QuestionType: string(q.QuestionType),
			Distribution: make(map[string]int),
		}

		var numericSum float64
		numericCount := 0

		for _, a := range assignments {
			if a.Status != models.AssignmentStatusCompleted {
				continue
			}
			for _, r := range a.Responses {
				if r.QuestionID != q.QuestionID {
					continue
				}
				dist.TotalAnswers++
				// Extract the "value" key from the response map for distribution
				val := r.ResponseValue["value"]
				key := fmt.Sprintf("%v", val)
				dist.Distribution[key]++

				// Try numeric average for likert/numeric types
				if q.QuestionType == models.QuestionTypeLikertScale {
					if num, ok := val.(float64); ok {
						numericSum += num
						numericCount++
					}
				}
			}
		}

		if numericCount > 0 {
			avg := numericSum / float64(numericCount)
			dist.Average = &avg
		}

		result = append(result, dist)
	}

	return result, nil
}

// TrendPeriod holds completion data for a single time period
type TrendPeriod struct {
	CompanyQuestionnaireID string    `json:"company_questionnaire_id"`
	PeriodStart            string    `json:"period_start"`
	PeriodEnd              string    `json:"period_end"`
	TotalAssigned          int       `json:"total_assigned"`
	Completed              int       `json:"completed"`
	CompletionRate         float64   `json:"completion_rate"`
	AvgTimeMinutes         float64   `json:"avg_time_minutes"`
}

// GetTrends compares completion across multiple periods for the same questionnaire in a company
func (s *ReportService) GetTrends(ctx context.Context, companyID primitive.ObjectID, userID string, isSuperAdmin bool) ([]TrendPeriod, error) {
	if !isSuperAdmin {
		userMeta, err := s.userMetadataRepo.GetByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("user metadata not found: %w", err)
		}
		if companyID != userMeta.CompanyID {
			return nil, fmt.Errorf("unauthorized: cannot access reports from other companies")
		}
	}

	cqs, err := s.companyQuestionnaireRepo.GetByCompanyID(ctx, companyID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get company questionnaires: %w", err)
	}

	trends := make([]TrendPeriod, 0, len(cqs))
	for _, cq := range cqs {
		assignments, err := s.assignmentRepo.GetByCompanyQuestionnaireID(ctx, cq.ID)
		if err != nil {
			continue
		}

		completed := 0
		var totalMs float64
		completedCount := 0
		for _, a := range assignments {
			if a.Status == models.AssignmentStatusCompleted {
				completed++
				if a.StartedAt != nil && a.CompletedAt != nil {
					totalMs += a.CompletedAt.Sub(*a.StartedAt).Minutes()
					completedCount++
				}
			}
		}

		total := len(assignments)
		rate := 0.0
		if total > 0 {
			rate = float64(completed) / float64(total) * 100
		}
		avgTime := 0.0
		if completedCount > 0 {
			avgTime = totalMs / float64(completedCount)
		}

		trends = append(trends, TrendPeriod{
			CompanyQuestionnaireID: cq.ID.Hex(),
			PeriodStart:            cq.PeriodStart.Format("2006-01-02"),
			PeriodEnd:              cq.PeriodEnd.Format("2006-01-02"),
			TotalAssigned:          total,
			Completed:              completed,
			CompletionRate:         rate,
			AvgTimeMinutes:         avgTime,
		})
	}

	return trends, nil
}

// IndividualReport holds full report for a single employee assignment
type IndividualReport struct {
	AssignmentID   string                 `json:"assignment_id"`
	UserID         string                 `json:"user_id"`
	Status         models.AssignmentStatus `json:"status"`
	AssignedAt     string                 `json:"assigned_at"`
	StartedAt      *string                `json:"started_at,omitempty"`
	CompletedAt    *string                `json:"completed_at,omitempty"`
	TimeToComplete *float64               `json:"time_to_complete_minutes,omitempty"`
	TotalQuestions int                    `json:"total_questions"`
	AnsweredCount  int                    `json:"answered_count"`
	ProgressPct    float64                `json:"progress_pct"`
	Responses      []ResponseDetail       `json:"responses"`
}

// ResponseDetail enriches a response with the question text
type ResponseDetail struct {
	QuestionID    string      `json:"question_id"`
	QuestionText  string      `json:"question_text"`
	QuestionType  string      `json:"question_type"`
	ResponseValue interface{} `json:"response_value"`
	AnsweredAt    string      `json:"answered_at"`
}

// GetIndividualReport returns a full report for a single assignment
func (s *ReportService) GetIndividualReport(ctx context.Context, assignmentID primitive.ObjectID, userID string, isSuperAdmin bool) (*IndividualReport, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}

	cq, err := s.companyQuestionnaireRepo.GetByID(ctx, assignment.CompanyQuestionnaireID)
	if err != nil {
		return nil, err
	}

	if !isSuperAdmin {
		userMeta, err := s.userMetadataRepo.GetByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("user metadata not found: %w", err)
		}
		if cq.CompanyID != userMeta.CompanyID {
			return nil, fmt.Errorf("unauthorized: cannot access this assignment report")
		}
	}

	questionnaire, err := s.questionnaireRepo.GetByID(ctx, cq.QuestionnaireID)
	if err != nil {
		return nil, err
	}

	questionMap := make(map[string]models.Question)
	for _, q := range questionnaire.Questions {
		questionMap[q.QuestionID] = q
	}

	details := make([]ResponseDetail, 0, len(assignment.Responses))
	for _, r := range assignment.Responses {
		q, ok := questionMap[r.QuestionID]
		if !ok {
			continue
		}
		details = append(details, ResponseDetail{
			QuestionID:    r.QuestionID,
			QuestionText:  q.QuestionText,
			QuestionType:  string(q.QuestionType),
			ResponseValue: r.GetValue(),
			AnsweredAt:    r.AnsweredAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	totalQ := len(questionnaire.Questions)
	progressPct := 0.0
	if totalQ > 0 {
		progressPct = float64(len(assignment.Responses)) / float64(totalQ) * 100
	}

	report := &IndividualReport{
		AssignmentID:   assignment.ID.Hex(),
		UserID:         assignment.UserID,
		Status:         assignment.Status,
		AssignedAt:     assignment.AssignedAt.Format("2006-01-02T15:04:05Z"),
		TotalQuestions: totalQ,
		AnsweredCount:  len(assignment.Responses),
		ProgressPct:    progressPct,
		Responses:      details,
	}

	if assignment.StartedAt != nil {
		s := assignment.StartedAt.Format("2006-01-02T15:04:05Z")
		report.StartedAt = &s
	}
	if assignment.CompletedAt != nil {
		s := assignment.CompletedAt.Format("2006-01-02T15:04:05Z")
		report.CompletedAt = &s
		if assignment.StartedAt != nil {
			mins := assignment.CompletedAt.Sub(*assignment.StartedAt).Minutes()
			report.TimeToComplete = &mins
		}
	}

	return report, nil
}

// ExportCSV generates an anonymized CSV of all completed responses for a company questionnaire
func (s *ReportService) ExportCSV(ctx context.Context, cqID primitive.ObjectID, userID string, isSuperAdmin bool) ([]byte, error) {
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
			return nil, fmt.Errorf("unauthorized: cannot export from other companies")
		}
	}

	questionnaire, err := s.questionnaireRepo.GetByID(ctx, cq.QuestionnaireID)
	if err != nil {
		return nil, err
	}

	assignments, err := s.assignmentRepo.GetByCompanyQuestionnaireID(ctx, cqID)
	if err != nil {
		return nil, err
	}

	// Fetch user metadata for department info (anonymized)
	users, err := s.userMetadataRepo.GetByCompanyID(ctx, cq.CompanyID)
	if err != nil {
		return nil, err
	}
	userDept := make(map[string]string)
	for _, u := range users {
		userDept[u.ID] = u.Department
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	// Header row
	header := []string{"row_id", "department", "status", "assigned_at", "completed_at", "time_to_complete_minutes"}
	for _, q := range questionnaire.Questions {
		header = append(header, q.QuestionText)
	}
	w.Write(header)

	rowNum := 1
	for _, a := range assignments {
		if a.Status != models.AssignmentStatusCompleted {
			continue
		}

		row := []string{
			strconv.Itoa(rowNum),
			userDept[a.UserID],
			string(a.Status),
			a.AssignedAt.Format(time.RFC3339),
		}

		if a.CompletedAt != nil {
			row = append(row, a.CompletedAt.Format(time.RFC3339))
			if a.StartedAt != nil {
				mins := a.CompletedAt.Sub(*a.StartedAt).Minutes()
				row = append(row, fmt.Sprintf("%.1f", mins))
			} else {
				row = append(row, "")
			}
		} else {
			row = append(row, "", "")
		}

		// Response per question (in order)
		responseMap := make(map[string]string)
		for _, r := range a.Responses {
			responseMap[r.QuestionID] = fmt.Sprintf("%v", r.GetValue())
		}
		for _, q := range questionnaire.Questions {
			row = append(row, responseMap[q.QuestionID])
		}

		w.Write(row)
		rowNum++
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("failed to write CSV: %w", err)
	}

	return buf.Bytes(), nil
}
