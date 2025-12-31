package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Questionnaire represents a questionnaire with embedded questions
type Questionnaire struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Title       string             `bson:"title" json:"title" validate:"required,min=5,max=200"`
	Description string             `bson:"description" json:"description"`
	CreatedBy   string             `bson:"created_by" json:"created_by"` // FusionAuth user ID
	IsActive    bool               `bson:"is_active" json:"is_active"`
	Questions   []Question         `bson:"questions" json:"questions"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// NewQuestionnaire creates a new Questionnaire with timestamps
func NewQuestionnaire(title, description, createdBy string) *Questionnaire {
	now := time.Now()
	return &Questionnaire{
		ID:          primitive.NewObjectID(),
		Title:       title,
		Description: description,
		CreatedBy:   createdBy,
		IsActive:    true,
		Questions:   []Question{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// AddQuestion adds a question to the questionnaire
func (q *Questionnaire) AddQuestion(question Question) {
	q.Questions = append(q.Questions, question)
	q.UpdatedAt = time.Now()
}

// UpdateQuestion updates a question by ID
func (q *Questionnaire) UpdateQuestion(questionID string, updatedQuestion Question) bool {
	for i, question := range q.Questions {
		if question.QuestionID == questionID {
			q.Questions[i] = updatedQuestion
			q.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// RemoveQuestion removes a question by ID
func (q *Questionnaire) RemoveQuestion(questionID string) bool {
	for i, question := range q.Questions {
		if question.QuestionID == questionID {
			q.Questions = append(q.Questions[:i], q.Questions[i+1:]...)
			q.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// GetQuestionByID retrieves a question by ID
func (q *Questionnaire) GetQuestionByID(questionID string) *Question {
	for _, question := range q.Questions {
		if question.QuestionID == questionID {
			return &question
		}
	}
	return nil
}

// CompanyQuestionnaire represents a questionnaire assigned to a company
type CompanyQuestionnaire struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CompanyID       primitive.ObjectID `bson:"company_id" json:"company_id" validate:"required"`
	QuestionnaireID primitive.ObjectID `bson:"questionnaire_id" json:"questionnaire_id" validate:"required"`
	AssignedBy      string             `bson:"assigned_by" json:"assigned_by"`   // FusionAuth user ID
	AssignedAt      time.Time          `bson:"assigned_at" json:"assigned_at"`
	PeriodStart     time.Time          `bson:"period_start" json:"period_start"`
	PeriodEnd       time.Time          `bson:"period_end" json:"period_end"`
	IsActive        bool               `bson:"is_active" json:"is_active"`
}

// NewCompanyQuestionnaire creates a new company questionnaire assignment
func NewCompanyQuestionnaire(companyID, questionnaireID primitive.ObjectID, assignedBy string, periodStart, periodEnd time.Time) *CompanyQuestionnaire {
	return &CompanyQuestionnaire{
		ID:              primitive.NewObjectID(),
		CompanyID:       companyID,
		QuestionnaireID: questionnaireID,
		AssignedBy:      assignedBy,
		AssignedAt:      time.Now(),
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		IsActive:        true,
	}
}

// IsWithinPeriod checks if the current time is within the assignment period
func (cq *CompanyQuestionnaire) IsWithinPeriod() bool {
	now := time.Now()
	return now.After(cq.PeriodStart) && now.Before(cq.PeriodEnd)
}
