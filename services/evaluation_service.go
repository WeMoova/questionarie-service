package services

import (
	"context"
	"fmt"
	"questionarie-service/models"
	"questionarie-service/repository"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EvaluationService handles scoring and evaluation of completed questionnaires
type EvaluationService struct {
	questionnaireRepo *repository.QuestionnaireRepository
	assignmentRepo    *repository.AssignmentRepository
}

// NewEvaluationService creates a new EvaluationService
func NewEvaluationService(
	questionnaireRepo *repository.QuestionnaireRepository,
	assignmentRepo *repository.AssignmentRepository,
) *EvaluationService {
	return &EvaluationService{
		questionnaireRepo: questionnaireRepo,
		assignmentRepo:    assignmentRepo,
	}
}

// EvaluateAssignment calculates dimension scores for a completed assignment
// Returns nil if the questionnaire has no evaluation config or it is disabled
func (s *EvaluationService) EvaluateAssignment(ctx context.Context, assignmentID, questionnaireID primitive.ObjectID) (*models.EvaluationResult, error) {
	questionnaire, err := s.questionnaireRepo.GetByID(ctx, questionnaireID)
	if err != nil {
		return nil, fmt.Errorf("failed to get questionnaire: %w", err)
	}

	if questionnaire.EvaluationConfig == nil || !questionnaire.EvaluationConfig.Enabled {
		return nil, nil
	}

	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment: %w", err)
	}

	evalConfig := questionnaire.EvaluationConfig

	// Build map: questionID → dimensionCode
	questionDimension := make(map[string]string)
	for _, q := range questionnaire.Questions {
		if q.DimensionCode != "" {
			questionDimension[q.QuestionID] = q.DimensionCode
		}
	}

	// Build map: questionID → numeric response value
	responseValues := make(map[string]float64)
	for _, r := range assignment.Responses {
		val := r.GetValue()
		if val == nil {
			continue
		}
		switch v := val.(type) {
		case float64:
			responseValues[r.QuestionID] = v
		case int:
			responseValues[r.QuestionID] = float64(v)
		case int32:
			responseValues[r.QuestionID] = float64(v)
		case int64:
			responseValues[r.QuestionID] = float64(v)
		}
	}

	// Calculate scores per dimension
	dimensionScores := make([]models.DimensionScore, 0, len(evalConfig.Dimensions))
	for _, dim := range evalConfig.Dimensions {
		var sum float64
		count := 0

		for qID, dimCode := range questionDimension {
			if dimCode != dim.Code {
				continue
			}
			if val, ok := responseValues[qID]; ok {
				sum += val
				count++
			}
		}

		var rawScore float64
		if count > 0 {
			if evalConfig.ScoringMethod == "average" {
				rawScore = sum / float64(count)
			} else {
				rawScore = sum // default: "sum"
			}
		}

		// Determine level from thresholds
		level, label, color := determineLevel(rawScore, dim.Thresholds)

		dimensionScores = append(dimensionScores, models.DimensionScore{
			DimensionCode: dim.Code,
			DimensionName: dim.Name,
			RawScore:      rawScore,
			MaxScore:      dim.MaxScore,
			Level:         level,
			LevelLabel:    label,
			LevelColor:    color,
			Direction:     dim.ScoringDirection,
		})
	}

	result := &models.EvaluationResult{
		DimensionScores:       dimensionScores,
		GeneralInterpretation: evalConfig.GeneralInterpretation,
		EvaluatedAt:           time.Now(),
	}

	return result, nil
}

// determineLevel checks score against thresholds and returns the matching level
func determineLevel(score float64, thresholds []models.ScoreThreshold) (level, label, color string) {
	intScore := int(score)
	for _, t := range thresholds {
		matches := true
		if t.MinScore != nil && intScore < *t.MinScore {
			matches = false
		}
		if t.MaxScore != nil && intScore > *t.MaxScore {
			matches = false
		}
		if matches {
			return t.Level, t.Label, t.Color
		}
	}
	return "unknown", "N/A", "#6b7280"
}
