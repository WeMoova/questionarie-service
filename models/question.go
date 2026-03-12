package models

import (
	"github.com/google/uuid"
)

// QuestionType represents the type of question
type QuestionType string

const (
	QuestionTypeMultipleChoice QuestionType = "multiple_choice"
	QuestionTypeLikertScale    QuestionType = "likert_scale"
	QuestionTypeFreeText       QuestionType = "free_text"
	QuestionTypeYesNo          QuestionType = "yes_no"
	QuestionTypeNumber         QuestionType = "number"
	QuestionTypeMultiSelect    QuestionType = "multiselect"
	QuestionTypeBMICalculator  QuestionType = "bmi_calculator"
)

// SkipRule defines a condition that hides subsequent questions based on an answer value.
// When the user selects IfValue for this question, the questions identified by
// SkipToSection or SkipQuestions are hidden.
type SkipRule struct {
	IfValue       string `bson:"if_value" json:"if_value"`                                   // answer value that triggers the skip
	SkipToSection string `bson:"skip_to_section,omitempty" json:"skip_to_section,omitempty"` // jump forward to this section (skip everything in between)
	SkipQuestions []int  `bson:"skip_questions,omitempty" json:"skip_questions,omitempty"`    // skip specific questions by order_index
}

// Question represents an embedded question within a questionnaire
type Question struct {
	QuestionID   string                 `bson:"question_id" json:"id"`
	QuestionText string                 `bson:"question_text" json:"question_text" validate:"required,min=5"`
	QuestionType QuestionType           `bson:"question_type" json:"question_type" validate:"required,oneof=multiple_choice likert_scale free_text yes_no number multiselect bmi_calculator"`
	ImageURL      string                 `bson:"image_url,omitempty" json:"image_url,omitempty"`
	Options       map[string]interface{} `bson:"options,omitempty" json:"options,omitempty"`
	OrderIndex    int                    `bson:"order_index" json:"order_index" validate:"min=0"`
	IsRequired    bool                   `bson:"is_required" json:"is_required"`
	SectionID     string                 `bson:"section_id,omitempty" json:"section_id,omitempty"`
	DimensionCode string                 `bson:"dimension_code,omitempty" json:"dimension_code,omitempty"`
	HasDetail     bool                   `bson:"has_detail,omitempty" json:"has_detail,omitempty"`
	DetailPrompt  string                 `bson:"detail_prompt,omitempty" json:"detail_prompt,omitempty"`
	SkipLogic     []SkipRule             `bson:"skip_logic,omitempty" json:"skip_logic,omitempty"`
}

// NewQuestion creates a new Question with a unique ID
func NewQuestion(text string, questionType QuestionType, orderIndex int, isRequired bool) *Question {
	return &Question{
		QuestionID:   uuid.New().String(),
		QuestionText: text,
		QuestionType: questionType,
		OrderIndex:   orderIndex,
		IsRequired:   isRequired,
		Options:      make(map[string]interface{}),
	}
}

// SetMultipleChoiceOptions sets options for multiple choice questions
func (q *Question) SetMultipleChoiceOptions(choices []string) {
	q.Options = map[string]interface{}{
		"choices": choices,
	}
}

// SetLikertScaleOptions sets options for Likert scale questions
func (q *Question) SetLikertScaleOptions(min, max int, labels []string) {
	q.Options = map[string]interface{}{
		"min":    min,
		"max":    max,
		"labels": labels,
	}
}

// VisibleQuestionIDs returns the set of question IDs that are visible given
// the current responses. Questions hidden by skip logic are excluded.
// responses maps questionID → answer value (string).
func VisibleQuestionIDs(questions []Question, responses map[string]string) map[string]struct{} {
	// Build order_index → section_id lookup and question_id → order_index
	orderToSection := make(map[int]string)
	idByOrder := make(map[int]string)
	orderByID := make(map[string]int)
	for _, q := range questions {
		orderToSection[q.OrderIndex] = q.SectionID
		idByOrder[q.OrderIndex] = q.QuestionID
		orderByID[q.QuestionID] = q.OrderIndex
	}

	// Collect all order_indexes that should be skipped
	skippedOrders := make(map[int]struct{})

	for _, q := range questions {
		if len(q.SkipLogic) == 0 {
			continue
		}
		answerVal, answered := responses[q.QuestionID]
		if !answered {
			continue
		}
		for _, rule := range q.SkipLogic {
			if answerVal != rule.IfValue {
				continue
			}
			// Skip specific questions by order_index
			for _, orderIdx := range rule.SkipQuestions {
				skippedOrders[orderIdx] = struct{}{}
			}
			// Skip to section: hide all questions between this question and the target section
			if rule.SkipToSection != "" {
				for _, other := range questions {
					if other.OrderIndex <= q.OrderIndex {
						continue
					}
					if other.SectionID == rule.SkipToSection {
						break
					}
					skippedOrders[other.OrderIndex] = struct{}{}
				}
			}
		}
	}

	visible := make(map[string]struct{})
	for _, q := range questions {
		if _, skipped := skippedOrders[q.OrderIndex]; !skipped {
			visible[q.QuestionID] = struct{}{}
		}
	}
	return visible
}
