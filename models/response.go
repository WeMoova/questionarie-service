package models

import (
	"fmt"
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

// SetMultiSelectResponse sets a multiselect response value (array of selected values)
func (r *Response) SetMultiSelectResponse(selected []string) {
	r.ResponseValue = map[string]interface{}{
		"value": selected,
		"type":  "multiselect",
	}
	r.AnsweredAt = time.Now()
}

// SetNumberResponse sets a numeric input response value
func (r *Response) SetNumberResponse(value float64) {
	r.ResponseValue = map[string]interface{}{
		"value": value,
		"type":  "number",
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

// GetStringValue returns the response value as a string (for skip logic matching).
// For multiple_choice it returns the selected string value.
// For other types it attempts a string conversion via fmt.Sprintf.
func (r *Response) GetStringValue() string {
	v := r.GetValue()
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int(val)) {
			return fmt.Sprintf("%d", int(val))
		}
		return fmt.Sprintf("%g", val)
	case int:
		return fmt.Sprintf("%d", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", val)
	}
}
