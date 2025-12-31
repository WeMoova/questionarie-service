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
)

// Question represents an embedded question within a questionnaire
type Question struct {
	QuestionID   string                 `bson:"question_id" json:"question_id"`
	QuestionText string                 `bson:"question_text" json:"question_text" validate:"required,min=5"`
	QuestionType QuestionType           `bson:"question_type" json:"question_type" validate:"required,oneof=multiple_choice likert_scale free_text yes_no"`
	Options      map[string]interface{} `bson:"options,omitempty" json:"options,omitempty"`
	OrderIndex   int                    `bson:"order_index" json:"order_index" validate:"min=0"`
	IsRequired   bool                   `bson:"is_required" json:"is_required"`
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
