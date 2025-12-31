package models

import (
	"time"
)

// Response represents an embedded response within an assignment
type Response struct {
	QuestionID    string                 `bson:"question_id" json:"question_id" validate:"required"`
	ResponseValue map[string]interface{} `bson:"response_value" json:"response_value"`
	AnsweredAt    time.Time              `bson:"answered_at" json:"answered_at"`
}

// NewResponse creates a new Response
func NewResponse(questionID string, value interface{}) *Response {
	return &Response{
		QuestionID: questionID,
		ResponseValue: map[string]interface{}{
			"value": value,
		},
		AnsweredAt: time.Now(),
	}
}

// SetTextResponse sets a text response value
func (r *Response) SetTextResponse(text string) {
	r.ResponseValue = map[string]interface{}{
		"value": text,
		"type":  "text",
	}
	r.AnsweredAt = time.Now()
}

// SetNumericResponse sets a numeric response value (for Likert scale)
func (r *Response) SetNumericResponse(value int) {
	r.ResponseValue = map[string]interface{}{
		"value": value,
		"type":  "numeric",
	}
	r.AnsweredAt = time.Now()
}

// SetBooleanResponse sets a boolean response value (for yes/no)
func (r *Response) SetBooleanResponse(value bool) {
	r.ResponseValue = map[string]interface{}{
		"value": value,
		"type":  "boolean",
	}
	r.AnsweredAt = time.Now()
}

// SetMultipleChoiceResponse sets a multiple choice response value
func (r *Response) SetMultipleChoiceResponse(selected string) {
	r.ResponseValue = map[string]interface{}{
		"value": selected,
		"type":  "multiple_choice",
	}
	r.AnsweredAt = time.Now()
}

// GetValue retrieves the response value
func (r *Response) GetValue() interface{} {
	if r.ResponseValue == nil {
		return nil
	}
	return r.ResponseValue["value"]
}
