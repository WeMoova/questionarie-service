package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AssignmentStatus represents the status of a user assignment
type AssignmentStatus string

const (
	AssignmentStatusPending    AssignmentStatus = "pending"
	AssignmentStatusInProgress AssignmentStatus = "in_progress"
	AssignmentStatusCompleted  AssignmentStatus = "completed"
)

// UserQuestionnaireAssignment represents a questionnaire assigned to a user with embedded responses
type UserQuestionnaireAssignment struct {
	ID                     primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CompanyQuestionnaireID primitive.ObjectID `bson:"company_questionnaire_id" json:"company_questionnaire_id" validate:"required"`
	UserID                 string             `bson:"user_id" json:"user_id" validate:"required"` // FusionAuth user ID
	AssignedBy             string             `bson:"assigned_by" json:"assigned_by"`             // FusionAuth user ID
	AssignedAt             time.Time          `bson:"assigned_at" json:"assigned_at"`
	Status                 AssignmentStatus   `bson:"status" json:"status"`
	StartedAt              *time.Time         `bson:"started_at,omitempty" json:"started_at,omitempty"`
	CompletedAt            *time.Time         `bson:"completed_at,omitempty" json:"completed_at,omitempty"`
	Responses              []Response         `bson:"responses" json:"responses"`
}

// NewUserQuestionnaireAssignment creates a new assignment
func NewUserQuestionnaireAssignment(companyQuestionnaireID primitive.ObjectID, userID, assignedBy string) *UserQuestionnaireAssignment {
	return &UserQuestionnaireAssignment{
		ID:                     primitive.NewObjectID(),
		CompanyQuestionnaireID: companyQuestionnaireID,
		UserID:                 userID,
		AssignedBy:             assignedBy,
		AssignedAt:             time.Now(),
		Status:                 AssignmentStatusPending,
		Responses:              []Response{},
	}
}

// Start marks the assignment as in progress
func (a *UserQuestionnaireAssignment) Start() {
	if a.Status == AssignmentStatusPending {
		now := time.Now()
		a.StartedAt = &now
		a.Status = AssignmentStatusInProgress
	}
}

// Complete marks the assignment as completed
func (a *UserQuestionnaireAssignment) Complete() {
	if a.Status == AssignmentStatusInProgress {
		now := time.Now()
		a.CompletedAt = &now
		a.Status = AssignmentStatusCompleted
	}
}

// AddResponse adds or updates a response for a specific question
func (a *UserQuestionnaireAssignment) AddResponse(response Response) {
	// Check if response already exists for this question
	for i, r := range a.Responses {
		if r.QuestionID == response.QuestionID {
			// Update existing response
			a.Responses[i] = response
			return
		}
	}
	// Add new response
	a.Responses = append(a.Responses, response)

	// Auto-start if this is the first response
	if a.Status == AssignmentStatusPending {
		a.Start()
	}
}

// GetResponse retrieves a response by question ID
func (a *UserQuestionnaireAssignment) GetResponse(questionID string) *Response {
	for _, response := range a.Responses {
		if response.QuestionID == questionID {
			return &response
		}
	}
	return nil
}

// GetProgress calculates the progress of the assignment
func (a *UserQuestionnaireAssignment) GetProgress(totalQuestions int) (answered int, total int, percentage float64) {
	answered = len(a.Responses)
	total = totalQuestions
	if total > 0 {
		percentage = (float64(answered) / float64(total)) * 100
	}
	return answered, total, percentage
}

// IsComplete checks if all required questions are answered
func (a *UserQuestionnaireAssignment) IsComplete(totalRequiredQuestions int) bool {
	return len(a.Responses) >= totalRequiredQuestions
}

// GetTimeToComplete returns the duration taken to complete the assignment
func (a *UserQuestionnaireAssignment) GetTimeToComplete() *time.Duration {
	if a.StartedAt != nil && a.CompletedAt != nil {
		duration := a.CompletedAt.Sub(*a.StartedAt)
		return &duration
	}
	return nil
}
