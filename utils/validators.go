package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ParseRequestBody parses JSON request body into the provided structure
func ParseRequestBody(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	return nil
}

// ValidateObjectID validates and converts a string to MongoDB ObjectID
func ValidateObjectID(id string) (primitive.ObjectID, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("invalid ID format: %w", err)
	}
	return objectID, nil
}

// ValidateRequiredString validates that a string is not empty
func ValidateRequiredString(value, fieldName string) error {
	if value == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	return nil
}

// ValidateStringLength validates string length
func ValidateStringLength(value, fieldName string, min, max int) error {
	length := len(value)
	if length < min {
		return fmt.Errorf("%s must be at least %d characters long", fieldName, min)
	}
	if max > 0 && length > max {
		return fmt.Errorf("%s must not exceed %d characters", fieldName, max)
	}
	return nil
}

// ValidateDateRange validates that start date is before end date
func ValidateDateRange(start, end time.Time, fieldName string) error {
	if start.After(end) || start.Equal(end) {
		return fmt.Errorf("%s: start date must be before end date", fieldName)
	}
	return nil
}

// ValidateFutureDate validates that a date is in the future
func ValidateFutureDate(date time.Time, fieldName string) error {
	if date.Before(time.Now()) {
		return fmt.Errorf("%s must be in the future", fieldName)
	}
	return nil
}

// ValidatePastDate validates that a date is in the past
func ValidatePastDate(date time.Time, fieldName string) error {
	if date.After(time.Now()) {
		return fmt.Errorf("%s must be in the past", fieldName)
	}
	return nil
}

// ValidateEnum validates that a value is within allowed values
func ValidateEnum(value string, allowedValues []string, fieldName string) error {
	for _, allowed := range allowedValues {
		if value == allowed {
			return nil
		}
	}
	return fmt.Errorf("%s must be one of: %v", fieldName, allowedValues)
}

// ValidatePositiveInt validates that an integer is positive
func ValidatePositiveInt(value int, fieldName string) error {
	if value <= 0 {
		return fmt.Errorf("%s must be a positive number", fieldName)
	}
	return nil
}

// ValidateRange validates that a number is within a range
func ValidateRange(value, min, max int, fieldName string) error {
	if value < min || value > max {
		return fmt.Errorf("%s must be between %d and %d", fieldName, min, max)
	}
	return nil
}

// ValidateEmail validates basic email format
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}
	// Basic email validation (contains @ and .)
	hasAt := false
	hasDot := false
	for _, char := range email {
		if char == '@' {
			hasAt = true
		}
		if char == '.' && hasAt {
			hasDot = true
		}
	}
	if !hasAt || !hasDot {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// ValidatePagination validates and sets default pagination values
func ValidatePagination(page, pageSize *int64) error {
	if *page <= 0 {
		*page = 1
	}
	if *pageSize <= 0 {
		*pageSize = 10
	}
	if *pageSize > 100 {
		*pageSize = 100 // Max page size
	}
	return nil
}

// ValidateArrayNotEmpty validates that an array is not empty
func ValidateArrayNotEmpty(arr []interface{}, fieldName string) error {
	if len(arr) == 0 {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}
	return nil
}

// ValidateStringArrayNotEmpty validates that a string array is not empty
func ValidateStringArrayNotEmpty(arr []string, fieldName string) error {
	if len(arr) == 0 {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}
	return nil
}

// ValidateUniqueStrings validates that all strings in an array are unique
func ValidateUniqueStrings(arr []string, fieldName string) error {
	seen := make(map[string]bool)
	for _, item := range arr {
		if seen[item] {
			return fmt.Errorf("%s contains duplicate values: %s", fieldName, item)
		}
		seen[item] = true
	}
	return nil
}

// ValidateQuestionType validates question type
func ValidateQuestionType(qType string) error {
	allowedTypes := []string{"multiple_choice", "likert_scale", "free_text", "yes_no"}
	return ValidateEnum(qType, allowedTypes, "question_type")
}

// ValidateAssignmentStatus validates assignment status
func ValidateAssignmentStatus(status string) error {
	allowedStatuses := []string{"pending", "in_progress", "completed"}
	return ValidateEnum(status, allowedStatuses, "status")
}
