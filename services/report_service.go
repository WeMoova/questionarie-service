package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"questionarie-service/models"
	"questionarie-service/repository"
	"sort"
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
	AssignmentID     string                    `json:"assignment_id"`
	UserID           string                    `json:"user_id"`
	Status           models.AssignmentStatus   `json:"status"`
	AssignedAt       string                    `json:"assigned_at"`
	StartedAt        *string                   `json:"started_at,omitempty"`
	CompletedAt      *string                   `json:"completed_at,omitempty"`
	TimeToComplete   *float64                  `json:"time_to_complete_minutes,omitempty"`
	TotalQuestions   int                       `json:"total_questions"`
	AnsweredCount    int                       `json:"answered_count"`
	ProgressPct      float64                   `json:"progress_pct"`
	Responses        []ResponseDetail          `json:"responses"`
	EvaluationResult *models.EvaluationResult  `json:"evaluation_result,omitempty"`
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
		AssignmentID:     assignment.ID.Hex(),
		UserID:           assignment.UserID,
		Status:           assignment.Status,
		AssignedAt:       assignment.AssignedAt.Format("2006-01-02T15:04:05Z"),
		TotalQuestions:   totalQ,
		AnsweredCount:    len(assignment.Responses),
		ProgressPct:      progressPct,
		Responses:        details,
		EvaluationResult: assignment.EvaluationResult,
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

// EvaluationSummary holds aggregated evaluation scores for a company questionnaire
type EvaluationSummary struct {
	CompanyQuestionnaireID string                      `json:"company_questionnaire_id"`
	QuestionnaireTitle     string                      `json:"questionnaire_title"`
	TotalEvaluated         int                         `json:"total_evaluated"`
	DimensionAverages      []DimensionAverageStat      `json:"dimension_averages"`
	GeneralInterpretation  string                      `json:"general_interpretation,omitempty"`
}

// DimensionAverageStat holds average scores for a single dimension
type DimensionAverageStat struct {
	DimensionCode string             `json:"dimension_code"`
	DimensionName string             `json:"dimension_name"`
	AvgScore      float64            `json:"avg_score"`
	MaxScore      int                `json:"max_score"`
	Direction     string             `json:"direction"`
	LevelDistribution map[string]int `json:"level_distribution"`
}

// GetEvaluationSummary returns aggregated evaluation scores for a company questionnaire
func (s *ReportService) GetEvaluationSummary(ctx context.Context, cqID primitive.ObjectID, userID string, isSuperAdmin bool) (*EvaluationSummary, error) {
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

	assignments, err := s.assignmentRepo.GetByCompanyQuestionnaireIDAndStatus(ctx, cqID, models.AssignmentStatusCompleted)
	if err != nil {
		return nil, fmt.Errorf("failed to get completed assignments: %w", err)
	}

	// Collect all evaluation results
	var evaluatedAssignments []*models.UserQuestionnaireAssignment
	for _, a := range assignments {
		if a.EvaluationResult != nil && len(a.EvaluationResult.DimensionScores) > 0 {
			evaluatedAssignments = append(evaluatedAssignments, a)
		}
	}

	if len(evaluatedAssignments) == 0 {
		return &EvaluationSummary{
			CompanyQuestionnaireID: cqID.Hex(),
			QuestionnaireTitle:     questionnaire.Title,
			TotalEvaluated:         0,
			DimensionAverages:      []DimensionAverageStat{},
		}, nil
	}

	// Aggregate scores per dimension
	type dimAccum struct {
		name      string
		maxScore  int
		direction string
		sumScore  float64
		count     int
		levels    map[string]int
	}
	dimMap := make(map[string]*dimAccum)

	for _, a := range evaluatedAssignments {
		for _, ds := range a.EvaluationResult.DimensionScores {
			acc, ok := dimMap[ds.DimensionCode]
			if !ok {
				acc = &dimAccum{
					name:      ds.DimensionName,
					maxScore:  ds.MaxScore,
					direction: ds.Direction,
					levels:    make(map[string]int),
				}
				dimMap[ds.DimensionCode] = acc
			}
			acc.sumScore += ds.RawScore
			acc.count++
			acc.levels[ds.Level]++
		}
	}

	dimAverages := make([]DimensionAverageStat, 0, len(dimMap))
	for code, acc := range dimMap {
		avg := 0.0
		if acc.count > 0 {
			avg = acc.sumScore / float64(acc.count)
		}
		dimAverages = append(dimAverages, DimensionAverageStat{
			DimensionCode:     code,
			DimensionName:     acc.name,
			AvgScore:          avg,
			MaxScore:          acc.maxScore,
			Direction:         acc.direction,
			LevelDistribution: acc.levels,
		})
	}

	interpretation := ""
	if questionnaire.EvaluationConfig != nil {
		interpretation = questionnaire.EvaluationConfig.GeneralInterpretation
	}

	return &EvaluationSummary{
		CompanyQuestionnaireID: cqID.Hex(),
		QuestionnaireTitle:     questionnaire.Title,
		TotalEvaluated:         len(evaluatedAssignments),
		DimensionAverages:      dimAverages,
		GeneralInterpretation:  interpretation,
	}, nil
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
	hasEval := questionnaire.EvaluationConfig != nil && questionnaire.EvaluationConfig.Enabled
	header := []string{"row_id", "department", "status", "assigned_at", "completed_at", "time_to_complete_minutes"}
	for _, q := range questionnaire.Questions {
		header = append(header, q.QuestionText)
	}
	if hasEval {
		for _, dim := range questionnaire.EvaluationConfig.Dimensions {
			header = append(header, dim.Code+"_score")
			header = append(header, dim.Code+"_level")
		}
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

		// Evaluation scores per dimension
		if hasEval && a.EvaluationResult != nil {
			scoreMap := make(map[string]models.DimensionScore)
			for _, ds := range a.EvaluationResult.DimensionScores {
				scoreMap[ds.DimensionCode] = ds
			}
			for _, dim := range questionnaire.EvaluationConfig.Dimensions {
				if ds, ok := scoreMap[dim.Code]; ok {
					row = append(row, fmt.Sprintf("%.1f", ds.RawScore))
					row = append(row, ds.LevelLabel)
				} else {
					row = append(row, "", "")
				}
			}
		} else if hasEval {
			for range questionnaire.EvaluationConfig.Dimensions {
				row = append(row, "", "")
			}
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

// --- Advanced Report Endpoints (generic, config-driven) ---

// DimensionSummary holds full dimension-level report data for a company questionnaire
type DimensionSummary struct {
	QuestionnaireTitle    string                   `json:"questionnaire_title"`
	ScoringMethod         string                   `json:"scoring_method"`
	TotalEvaluated        int                      `json:"total_evaluated"`
	TotalAssigned         int                      `json:"total_assigned"`
	Dimensions            []DimensionSummaryStat   `json:"dimensions"`
	GeneralInterpretation string                   `json:"general_interpretation,omitempty"`
}

// DimensionSummaryStat holds aggregated statistics for a single dimension
type DimensionSummaryStat struct {
	Code        string                  `json:"code"`
	Name        string                  `json:"name"`
	Direction   string                  `json:"direction"`
	MaxScore    int                     `json:"max_score"`
	AvgScore    float64                 `json:"avg_score"`
	MedianScore float64                 `json:"median_score"`
	MinObserved float64                 `json:"min_observed"`
	MaxObserved float64                 `json:"max_observed"`
	Thresholds  []ThresholdDistribution `json:"thresholds"`
}

// ThresholdDistribution holds count/percentage for a single threshold level
type ThresholdDistribution struct {
	Level      string  `json:"level"`
	Label      string  `json:"label"`
	Min        int     `json:"min"`
	Max        int     `json:"max"`
	Color      string  `json:"color"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// GetDimensionSummary returns a full dimension-level report for a company questionnaire
func (s *ReportService) GetDimensionSummary(ctx context.Context, cqID primitive.ObjectID, userID string, isSuperAdmin bool) (*DimensionSummary, error) {
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

	if questionnaire.EvaluationConfig == nil || !questionnaire.EvaluationConfig.Enabled {
		return &DimensionSummary{
			QuestionnaireTitle: questionnaire.Title,
			TotalEvaluated:     0,
			Dimensions:         []DimensionSummaryStat{},
		}, nil
	}

	evalConfig := questionnaire.EvaluationConfig

	// Get all assignments for total count
	allAssignments, err := s.assignmentRepo.GetByCompanyQuestionnaireID(ctx, cqID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignments: %w", err)
	}

	// Filter to completed with evaluation results
	var evaluated []*models.UserQuestionnaireAssignment
	for _, a := range allAssignments {
		if a.EvaluationResult != nil && len(a.EvaluationResult.DimensionScores) > 0 {
			evaluated = append(evaluated, a)
		}
	}

	// Build per-dimension accumulators
	type dimAccum struct {
		scores []float64
		levels map[string]int
	}
	dimData := make(map[string]*dimAccum)
	for _, dim := range evalConfig.Dimensions {
		dimData[dim.Code] = &dimAccum{levels: make(map[string]int)}
	}

	for _, a := range evaluated {
		for _, ds := range a.EvaluationResult.DimensionScores {
			acc, ok := dimData[ds.DimensionCode]
			if !ok {
				continue
			}
			acc.scores = append(acc.scores, ds.RawScore)
			acc.levels[ds.Level]++
		}
	}

	totalEval := len(evaluated)
	dimensions := make([]DimensionSummaryStat, 0, len(evalConfig.Dimensions))

	for _, dim := range evalConfig.Dimensions {
		acc := dimData[dim.Code]

		stat := DimensionSummaryStat{
			Code:      dim.Code,
			Name:      dim.Name,
			Direction: dim.ScoringDirection,
			MaxScore:  dim.MaxScore,
		}

		if len(acc.scores) > 0 {
			sort.Float64s(acc.scores)
			sum := 0.0
			for _, v := range acc.scores {
				sum += v
			}
			stat.AvgScore = math.Round(sum/float64(len(acc.scores))*10) / 10
			stat.MinObserved = acc.scores[0]
			stat.MaxObserved = acc.scores[len(acc.scores)-1]

			// Median
			n := len(acc.scores)
			if n%2 == 0 {
				stat.MedianScore = (acc.scores[n/2-1] + acc.scores[n/2]) / 2
			} else {
				stat.MedianScore = acc.scores[n/2]
			}
		}

		// Threshold distribution
		thresholds := make([]ThresholdDistribution, 0, len(dim.Thresholds))
		for _, t := range dim.Thresholds {
			minVal, maxVal := 0, 0
			if t.MinScore != nil {
				minVal = *t.MinScore
			}
			if t.MaxScore != nil {
				maxVal = *t.MaxScore
			}
			count := acc.levels[t.Level]
			pct := 0.0
			if totalEval > 0 {
				pct = math.Round(float64(count)/float64(totalEval)*1000) / 10
			}
			thresholds = append(thresholds, ThresholdDistribution{
				Level:      t.Level,
				Label:      t.Label,
				Min:        minVal,
				Max:        maxVal,
				Color:      t.Color,
				Count:      count,
				Percentage: pct,
			})
		}
		stat.Thresholds = thresholds
		dimensions = append(dimensions, stat)
	}

	return &DimensionSummary{
		QuestionnaireTitle:    questionnaire.Title,
		ScoringMethod:         evalConfig.ScoringMethod,
		TotalEvaluated:        totalEval,
		TotalAssigned:         len(allAssignments),
		Dimensions:            dimensions,
		GeneralInterpretation: evalConfig.GeneralInterpretation,
	}, nil
}

// DepartmentDimensions holds dimension breakdown by department
type DepartmentDimensions struct {
	Departments []DepartmentDimensionStat `json:"departments"`
}

// DepartmentDimensionStat holds dimension data for a single department
type DepartmentDimensionStat struct {
	Department    string                       `json:"department"`
	EmployeeCount int                          `json:"employee_count"`
	Evaluated     int                          `json:"evaluated"`
	Dimensions    []DepartmentDimensionDetail  `json:"dimensions"`
}

// DepartmentDimensionDetail holds a dimension's stats within a department
type DepartmentDimensionDetail struct {
	Code              string         `json:"code"`
	Name              string         `json:"name"`
	AvgScore          float64        `json:"avg_score"`
	PredominantLevel  string         `json:"predominant_level"`
	PredominantColor  string         `json:"predominant_color"`
	LevelDistribution map[string]int `json:"level_distribution"`
}

// GetDepartmentDimensions returns dimension breakdown by department
func (s *ReportService) GetDepartmentDimensions(ctx context.Context, cqID primitive.ObjectID, userID string, isSuperAdmin bool) (*DepartmentDimensions, error) {
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

	if questionnaire.EvaluationConfig == nil || !questionnaire.EvaluationConfig.Enabled {
		return &DepartmentDimensions{Departments: []DepartmentDimensionStat{}}, nil
	}

	evalConfig := questionnaire.EvaluationConfig

	// Get user departments
	users, err := s.userMetadataRepo.GetByCompanyID(ctx, cq.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	userDept := make(map[string]string)
	for _, u := range users {
		dept := u.Department
		if dept == "" {
			dept = "Sin departamento"
		}
		userDept[u.ID] = dept
	}

	// Get evaluated assignments
	allAssignments, err := s.assignmentRepo.GetByCompanyQuestionnaireID(ctx, cqID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignments: %w", err)
	}

	// Accumulate per department per dimension
	type dimAccum struct {
		sum   float64
		count int
		levels map[string]int
	}
	type deptAccum struct {
		totalAssigned int
		evaluated     int
		dims          map[string]*dimAccum
	}
	deptData := make(map[string]*deptAccum)

	for _, a := range allAssignments {
		dept := userDept[a.UserID]
		if dept == "" {
			dept = "Sin departamento"
		}
		if _, ok := deptData[dept]; !ok {
			dims := make(map[string]*dimAccum)
			for _, dim := range evalConfig.Dimensions {
				dims[dim.Code] = &dimAccum{levels: make(map[string]int)}
			}
			deptData[dept] = &deptAccum{dims: dims}
		}
		deptData[dept].totalAssigned++

		if a.EvaluationResult == nil || len(a.EvaluationResult.DimensionScores) == 0 {
			continue
		}
		deptData[dept].evaluated++

		for _, ds := range a.EvaluationResult.DimensionScores {
			acc, ok := deptData[dept].dims[ds.DimensionCode]
			if !ok {
				continue
			}
			acc.sum += ds.RawScore
			acc.count++
			acc.levels[ds.Level]++
		}
	}

	// Build dimension config lookup for colors
	dimConfigMap := make(map[string]models.DimensionConfig)
	for _, dim := range evalConfig.Dimensions {
		dimConfigMap[dim.Code] = dim
	}

	departments := make([]DepartmentDimensionStat, 0, len(deptData))
	for dept, da := range deptData {
		dimDetails := make([]DepartmentDimensionDetail, 0, len(evalConfig.Dimensions))
		for _, dim := range evalConfig.Dimensions {
			acc := da.dims[dim.Code]
			avg := 0.0
			if acc.count > 0 {
				avg = math.Round(acc.sum/float64(acc.count)*10) / 10
			}

			// Find predominant level
			predominantLevel := ""
			predominantColor := ""
			maxCount := 0
			for lvl, cnt := range acc.levels {
				if cnt > maxCount {
					maxCount = cnt
					predominantLevel = lvl
				}
			}
			// Get color from thresholds
			for _, t := range dim.Thresholds {
				if t.Level == predominantLevel {
					predominantColor = t.Color
					break
				}
			}

			dimDetails = append(dimDetails, DepartmentDimensionDetail{
				Code:              dim.Code,
				Name:              dim.Name,
				AvgScore:          avg,
				PredominantLevel:  predominantLevel,
				PredominantColor:  predominantColor,
				LevelDistribution: acc.levels,
			})
		}

		departments = append(departments, DepartmentDimensionStat{
			Department:    dept,
			EmployeeCount: da.totalAssigned,
			Evaluated:     da.evaluated,
			Dimensions:    dimDetails,
		})
	}

	// Sort departments alphabetically
	sort.Slice(departments, func(i, j int) bool {
		return departments[i].Department < departments[j].Department
	})

	return &DepartmentDimensions{Departments: departments}, nil
}

// RiskAnalysisResult holds risk profile analysis for a company questionnaire
type RiskAnalysisResult struct {
	RiskProfiles []RiskProfileResult `json:"risk_profiles"`
}

// RiskProfileResult holds the result of evaluating a single risk profile
type RiskProfileResult struct {
	Name               string  `json:"name"`
	Description        string  `json:"description"`
	Severity           string  `json:"severity"`
	Color              string  `json:"color"`
	AffectedCount      int     `json:"affected_count"`
	AffectedPercentage float64 `json:"affected_percentage"`
}

// GetRiskAnalysis evaluates risk profiles defined in the questionnaire's config
func (s *ReportService) GetRiskAnalysis(ctx context.Context, cqID primitive.ObjectID, userID string, isSuperAdmin bool) (*RiskAnalysisResult, error) {
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

	if questionnaire.EvaluationConfig == nil || !questionnaire.EvaluationConfig.Enabled || len(questionnaire.EvaluationConfig.RiskProfiles) == 0 {
		return &RiskAnalysisResult{RiskProfiles: []RiskProfileResult{}}, nil
	}

	// Get evaluated assignments
	allAssignments, err := s.assignmentRepo.GetByCompanyQuestionnaireID(ctx, cqID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignments: %w", err)
	}

	var evaluated []*models.UserQuestionnaireAssignment
	for _, a := range allAssignments {
		if a.EvaluationResult != nil && len(a.EvaluationResult.DimensionScores) > 0 {
			evaluated = append(evaluated, a)
		}
	}

	totalEval := len(evaluated)
	results := make([]RiskProfileResult, 0, len(questionnaire.EvaluationConfig.RiskProfiles))

	for _, rp := range questionnaire.EvaluationConfig.RiskProfiles {
		affected := 0
		for _, a := range evaluated {
			if matchesRiskProfile(a.EvaluationResult, rp) {
				affected++
			}
		}

		pct := 0.0
		if totalEval > 0 {
			pct = math.Round(float64(affected)/float64(totalEval)*1000) / 10
		}

		results = append(results, RiskProfileResult{
			Name:               rp.Name,
			Description:        rp.Description,
			Severity:           rp.Severity,
			Color:              rp.Color,
			AffectedCount:      affected,
			AffectedPercentage: pct,
		})
	}

	return &RiskAnalysisResult{RiskProfiles: results}, nil
}

// matchesRiskProfile checks if an evaluation result matches a risk profile's conditions
func matchesRiskProfile(result *models.EvaluationResult, rp models.RiskProfile) bool {
	// Build lookup: dimension_code → DimensionScore
	scoreMap := make(map[string]models.DimensionScore)
	for _, ds := range result.DimensionScores {
		scoreMap[ds.DimensionCode] = ds
	}

	matchCount := 0
	for _, cond := range rp.Conditions {
		ds, ok := scoreMap[cond.DimensionCode]
		if !ok {
			if rp.Logic == "all" {
				return false
			}
			continue
		}

		condMet := false
		switch cond.Operator {
		case "level_in":
			for _, val := range cond.Values {
				if ds.Level == val {
					condMet = true
					break
				}
			}
		case "score_gt":
			if len(cond.Values) > 0 {
				if threshold, err := strconv.ParseFloat(cond.Values[0], 64); err == nil {
					condMet = ds.RawScore > threshold
				}
			}
		case "score_lt":
			if len(cond.Values) > 0 {
				if threshold, err := strconv.ParseFloat(cond.Values[0], 64); err == nil {
					condMet = ds.RawScore < threshold
				}
			}
		}

		if condMet {
			matchCount++
		}
		if !condMet && rp.Logic == "all" {
			return false
		}
	}

	if rp.Logic == "all" {
		return matchCount == len(rp.Conditions)
	}
	// "any" logic
	return matchCount > 0
}

// ScoreDistribution holds histogram data for scores across dimensions
type ScoreDistribution struct {
	Dimensions []DimensionHistogram `json:"dimensions"`
}

// DimensionHistogram holds histogram buckets for a single dimension
type DimensionHistogram struct {
	Code     string            `json:"code"`
	Name     string            `json:"name"`
	MaxScore int               `json:"max_score"`
	Buckets  []HistogramBucket `json:"buckets"`
}

// HistogramBucket represents a single bucket in the histogram
type HistogramBucket struct {
	RangeStart int `json:"range_start"`
	RangeEnd   int `json:"range_end"`
	Count      int `json:"count"`
}

// GetScoreDistribution returns score histograms per dimension
func (s *ReportService) GetScoreDistribution(ctx context.Context, cqID primitive.ObjectID, userID string, isSuperAdmin bool) (*ScoreDistribution, error) {
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

	if questionnaire.EvaluationConfig == nil || !questionnaire.EvaluationConfig.Enabled {
		return &ScoreDistribution{Dimensions: []DimensionHistogram{}}, nil
	}

	evalConfig := questionnaire.EvaluationConfig

	allAssignments, err := s.assignmentRepo.GetByCompanyQuestionnaireID(ctx, cqID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignments: %w", err)
	}

	// Collect scores per dimension
	dimScores := make(map[string][]float64)
	for _, dim := range evalConfig.Dimensions {
		dimScores[dim.Code] = []float64{}
	}
	for _, a := range allAssignments {
		if a.EvaluationResult == nil {
			continue
		}
		for _, ds := range a.EvaluationResult.DimensionScores {
			dimScores[ds.DimensionCode] = append(dimScores[ds.DimensionCode], ds.RawScore)
		}
	}

	dimensions := make([]DimensionHistogram, 0, len(evalConfig.Dimensions))
	for _, dim := range evalConfig.Dimensions {
		scores := dimScores[dim.Code]

		// Use thresholds as bucket boundaries for meaningful distribution
		buckets := make([]HistogramBucket, 0, len(dim.Thresholds))
		for _, t := range dim.Thresholds {
			minVal, maxVal := 0, dim.MaxScore
			if t.MinScore != nil {
				minVal = *t.MinScore
			}
			if t.MaxScore != nil {
				maxVal = *t.MaxScore
			}
			count := 0
			for _, sc := range scores {
				intScore := int(sc)
				if intScore >= minVal && intScore <= maxVal {
					count++
				}
			}
			buckets = append(buckets, HistogramBucket{
				RangeStart: minVal,
				RangeEnd:   maxVal,
				Count:      count,
			})
		}

		dimensions = append(dimensions, DimensionHistogram{
			Code:     dim.Code,
			Name:     dim.Name,
			MaxScore: dim.MaxScore,
			Buckets:  buckets,
		})
	}

	return &ScoreDistribution{Dimensions: dimensions}, nil
}

// SendReportEmail sends a report summary email for a company questionnaire via Resend
func (s *ReportService) SendReportEmail(ctx context.Context, cqID primitive.ObjectID, userID string, isSuperAdmin bool, recipients []string, subject string, customMessage string) error {
	// Get company questionnaire
	cq, err := s.companyQuestionnaireRepo.GetByID(ctx, cqID)
	if err != nil {
		return fmt.Errorf("company questionnaire not found: %w", err)
	}

	// Verify authorization
	if !isSuperAdmin {
		userMeta, err := s.userMetadataRepo.GetByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("user metadata not found: %w", err)
		}
		if cq.CompanyID != userMeta.CompanyID {
			return fmt.Errorf("unauthorized: cannot access reports from other companies")
		}
	}

	// Get company for branding
	company, err := s.companyRepo.GetByID(ctx, cq.CompanyID)
	if err != nil {
		return fmt.Errorf("company not found: %w", err)
	}

	// Get completion metrics
	metrics, err := s.GetCompletionMetrics(ctx, cqID, userID, isSuperAdmin)
	if err != nil {
		return fmt.Errorf("failed to get completion metrics: %w", err)
	}

	// Get dimension summary
	dimSummary, err := s.GetDimensionSummary(ctx, cqID, userID, isSuperAdmin)
	if err != nil {
		return fmt.Errorf("failed to get dimension summary: %w", err)
	}

	// Build HTML
	htmlBody := s.buildReportEmailHTML(company, metrics, dimSummary, customMessage)

	// Use questionnaire title as subject if not provided
	if subject == "" {
		subject = fmt.Sprintf("Reporte de %s", metrics.QuestionnaireTitle)
	}

	return s.sendViaResend(recipients, subject, htmlBody)
}

// sendViaResend sends an email via the Resend HTTP API
func (s *ReportService) sendViaResend(to []string, subject, htmlBody string) error {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY not configured")
	}

	payload := map[string]interface{}{
		"from":    "WeMoova <no-reply@wemoova.com>",
		"to":      to,
		"subject": subject,
		"html":    htmlBody,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend API error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// buildReportEmailHTML builds an HTML email with inline CSS for the report summary
func (s *ReportService) buildReportEmailHTML(company *models.Company, metrics *CompletionMetrics, dimSummary *DimensionSummary, customMessage string) string {
	// Determine branding — prefer secondary color (darker), fallback to primary, then WeMoova teal
	brandColor := "#3ba5cc"
	logoHTML := ""
	if company.Branding != nil {
		if company.Branding.SecondaryColor != "" {
			brandColor = company.Branding.SecondaryColor
		} else if company.Branding.PrimaryColor != "" {
			brandColor = company.Branding.PrimaryColor
		}
		if company.Branding.Logo != "" {
			// Use absolute URL for email clients; MinIO URLs should already be absolute
			logoHTML = fmt.Sprintf(`<div style="text-align:center;margin-bottom:16px;"><img src="%s" alt="%s" style="max-height:50px;max-width:180px;display:inline-block;" /></div>`, company.Branding.Logo, company.Name)
		}
	}
	primaryColor := brandColor

	// Completion rate formatted
	completionRate := fmt.Sprintf("%.1f%%", metrics.CompletionPercentage)
	avgTime := fmt.Sprintf("%.1f min", metrics.AvgTimeToComplete)

	// Build dimension rows
	dimensionRows := ""
	for _, dim := range dimSummary.Dimensions {
		// Find predominant level (highest count threshold)
		predominantLevel := ""
		predominantColor := "#888888"
		maxCount := 0
		for _, t := range dim.Thresholds {
			if t.Count > maxCount {
				maxCount = t.Count
				predominantLevel = t.Label
				if t.Color != "" {
					predominantColor = t.Color
				}
			}
		}

		dimensionRows += fmt.Sprintf(`<tr>
			<td style="padding:10px 12px;border-bottom:1px solid #eee;font-size:14px;">%s</td>
			<td style="padding:10px 12px;border-bottom:1px solid #eee;font-size:14px;text-align:center;">%.1f</td>
			<td style="padding:10px 12px;border-bottom:1px solid #eee;text-align:center;">
				<span style="display:inline-block;padding:4px 12px;border-radius:12px;font-size:12px;font-weight:600;color:#fff;background-color:%s;">%s</span>
			</td>
		</tr>`, dim.Name, dim.AvgScore, predominantColor, predominantLevel)
	}

	// Custom message block
	customMessageHTML := ""
	if customMessage != "" {
		customMessageHTML = fmt.Sprintf(`<div style="background-color:#f8f9fa;border-left:4px solid %s;padding:16px;margin:24px 0;border-radius:0 8px 8px 0;">
			<p style="margin:0;font-size:14px;color:#555;font-style:italic;">%s</p>
		</div>`, primaryColor, customMessage)
	}

	// Dimension table (only if there are dimensions)
	dimensionTableHTML := ""
	if len(dimSummary.Dimensions) > 0 {
		dimensionTableHTML = fmt.Sprintf(`<h2 style="font-size:18px;color:#333;margin:32px 0 16px 0;">Resumen por Dimensiones</h2>
		<table style="width:100%%;border-collapse:collapse;border-radius:8px;overflow:hidden;">
			<thead>
				<tr style="background-color:%s;">
					<th style="padding:12px;text-align:left;color:#fff;font-size:13px;font-weight:600;">Dimension</th>
					<th style="padding:12px;text-align:center;color:#fff;font-size:13px;font-weight:600;">Puntaje Promedio</th>
					<th style="padding:12px;text-align:center;color:#fff;font-size:13px;font-weight:600;">Nivel Predominante</th>
				</tr>
			</thead>
			<tbody>
				%s
			</tbody>
		</table>`, primaryColor, dimensionRows)
	}

	currentDate := time.Now().Format("02/01/2006")

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"></head>
<body style="margin:0;padding:0;background-color:#f4f4f7;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif;">
<div style="max-width:600px;margin:0 auto;padding:20px;">
	<div style="background-color:#ffffff;border-radius:12px;box-shadow:0 2px 8px rgba(0,0,0,0.08);overflow:hidden;">
		<!-- Header -->
		<div style="background-color:%s;padding:32px 24px;text-align:center;">
			%s
			<h1 style="margin:0;color:#ffffff;font-size:22px;font-weight:700;">Reporte de %s</h1>
			<p style="margin:8px 0 0 0;color:rgba(255,255,255,0.85);font-size:14px;">%s</p>
		</div>

		<!-- Body -->
		<div style="padding:32px 24px;">
			%s

			<!-- KPI Summary -->
			<h2 style="font-size:18px;color:#333;margin:0 0 16px 0;">Resumen General</h2>
			<table style="width:100%%;border-collapse:collapse;border-radius:8px;overflow:hidden;">
				<thead>
					<tr style="background-color:%s;">
						<th style="padding:12px;text-align:left;color:#fff;font-size:13px;font-weight:600;">Metrica</th>
						<th style="padding:12px;text-align:right;color:#fff;font-size:13px;font-weight:600;">Valor</th>
					</tr>
				</thead>
				<tbody>
					<tr><td style="padding:10px 12px;border-bottom:1px solid #eee;font-size:14px;">Asignados</td><td style="padding:10px 12px;border-bottom:1px solid #eee;font-size:14px;text-align:right;font-weight:600;">%d</td></tr>
					<tr><td style="padding:10px 12px;border-bottom:1px solid #eee;font-size:14px;">Completados</td><td style="padding:10px 12px;border-bottom:1px solid #eee;font-size:14px;text-align:right;font-weight:600;">%d</td></tr>
					<tr><td style="padding:10px 12px;border-bottom:1px solid #eee;font-size:14px;">Tasa de Completitud</td><td style="padding:10px 12px;border-bottom:1px solid #eee;font-size:14px;text-align:right;font-weight:600;">%s</td></tr>
					<tr><td style="padding:10px 12px;font-size:14px;">Tiempo Promedio</td><td style="padding:10px 12px;font-size:14px;text-align:right;font-weight:600;">%s</td></tr>
				</tbody>
			</table>

			%s

			<!-- CTA Button -->
			<div style="text-align:center;margin:32px 0 16px 0;">
				<a href="https://qa.admin.wemoova.com/dashboard/reports" style="display:inline-block;padding:14px 32px;background-color:%s;color:#ffffff;text-decoration:none;border-radius:8px;font-size:15px;font-weight:600;">Ver Reporte Completo</a>
			</div>
		</div>

		<!-- Footer -->
		<div style="background-color:#f8f9fa;padding:20px 24px;text-align:center;border-top:1px solid #eee;">
			<p style="margin:0;font-size:12px;color:#999;">Generado por WeMoova &mdash; %s</p>
		</div>
	</div>
</div>
</body>
</html>`,
		primaryColor,
		logoHTML,
		metrics.QuestionnaireTitle,
		metrics.CompanyName,
		customMessageHTML,
		primaryColor,
		metrics.Assigned,
		metrics.Completed,
		completionRate,
		avgTime,
		dimensionTableHTML,
		primaryColor,
		currentDate,
	)

	return html
}
