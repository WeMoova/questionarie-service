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

	// Compute visible questions based on skip logic + responses
	responseStrings := make(map[string]string)
	for _, r := range assignment.Responses {
		responseStrings[r.QuestionID] = r.GetStringValue()
	}
	visibleIDs := models.VisibleQuestionIDs(questionnaire.Questions, responseStrings)

	// Build map: questionID → Question (for option lookups), only visible questions
	questionMap := make(map[string]models.Question)
	questionDimension := make(map[string]string)
	for _, q := range questionnaire.Questions {
		if _, visible := visibleIDs[q.QuestionID]; !visible {
			continue
		}
		questionMap[q.QuestionID] = q
		if q.DimensionCode != "" {
			questionDimension[q.QuestionID] = q.DimensionCode
		}
	}

	// Build map: questionID → numeric response value
	// Supports: scalar numbers, multiple_choice with per-option scores, multiselect arrays
	responseValues := make(map[string]float64)
	for _, r := range assignment.Responses {
		val := r.GetValue()
		if val == nil {
			continue
		}

		q := questionMap[r.QuestionID]

		switch v := val.(type) {
		case float64:
			// For multiple_choice/calculator types with choices that have scores, look up the score
			if q.QuestionType == models.QuestionTypeMultipleChoice || q.QuestionType == models.QuestionTypeBMICalculator || q.QuestionType == models.QuestionTypePackYearsCalculator {
				if score, ok := lookupChoiceScore(q.Options, v, ""); ok {
					responseValues[r.QuestionID] = score
					continue
				}
			}
			responseValues[r.QuestionID] = v
		case int:
			responseValues[r.QuestionID] = float64(v)
		case int32:
			responseValues[r.QuestionID] = float64(v)
		case int64:
			responseValues[r.QuestionID] = float64(v)
		case string:
			// multiple_choice with string value — look up score from choices
			if score, ok := lookupChoiceScore(q.Options, 0, v); ok {
				responseValues[r.QuestionID] = score
			}
		case []interface{}:
			// multiselect — sum scores of all selected values
			var total float64
			for _, item := range v {
				if str, ok := item.(string); ok {
					if score, ok := lookupChoiceScore(q.Options, 0, str); ok {
						total += score
					}
				}
			}
			responseValues[r.QuestionID] = total
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

// lookupChoiceScore looks up the score for a selected choice in question options.
// Options format: {"choices": [{"value": "x", "label": "...", "score": 3}, ...]}
// If strVal is non-empty, matches by value string; otherwise matches by numVal.
// Returns (score, true) if found, (0, false) if choices don't have scores.
func lookupChoiceScore(options map[string]interface{}, numVal float64, strVal string) (float64, bool) {
	choicesRaw, ok := options["choices"]
	if !ok {
		return 0, false
	}
	choices, ok := choicesRaw.([]interface{})
	if !ok {
		// choices is a simple string array — no scores
		return 0, false
	}

	for _, c := range choices {
		choice, ok := c.(map[string]interface{})
		if !ok {
			// simple string choice — no scores
			return 0, false
		}
		scoreRaw, hasScore := choice["score"]
		if !hasScore {
			return 0, false
		}

		// Match by string value
		if strVal != "" {
			if v, ok := choice["value"].(string); ok && v == strVal {
				return toFloat64(scoreRaw), true
			}
			// Also match by label for backwards compatibility
			if l, ok := choice["label"].(string); ok && l == strVal {
				return toFloat64(scoreRaw), true
			}
			continue
		}

		// Match by numeric value
		if v := toFloat64(choice["value"]); v == numVal {
			return toFloat64(scoreRaw), true
		}
	}
	return 0, false
}

// toFloat64 converts an interface{} to float64 (handles int, float64, int32, int64)
func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}
