package models

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Section represents a section within a questionnaire (e.g., "Sección A: Desgaste Profesional")
type Section struct {
	ID          string `bson:"id" json:"id"`
	Title       string `bson:"title" json:"title"`
	Description string `bson:"description,omitempty" json:"description,omitempty"`
	OrderIndex  int    `bson:"order_index" json:"order_index"`
}

// ScoreThreshold defines a scoring level (e.g., ALTO: > 27 points)
type ScoreThreshold struct {
	Level       string `bson:"level" json:"level"`                                 // "alto", "medio", "bajo"
	Label       string `bson:"label" json:"label"`                                 // "ALTO", "MEDIO", "BAJO"
	MinScore    *int   `bson:"min_score,omitempty" json:"min_score,omitempty"`
	MaxScore    *int   `bson:"max_score,omitempty" json:"max_score,omitempty"`
	Color       string `bson:"color,omitempty" json:"color,omitempty"`
	Description string `bson:"description,omitempty" json:"description,omitempty"`
}

// DimensionConfig defines a scoring dimension (e.g., AE = Agotamiento Emocional)
type DimensionConfig struct {
	Code             string           `bson:"code" json:"code"`
	Name             string           `bson:"name" json:"name"`
	Description      string           `bson:"description,omitempty" json:"description,omitempty"`
	ScoringDirection string           `bson:"scoring_direction" json:"scoring_direction"` // "direct" or "inverse"
	MaxScore         int              `bson:"max_score" json:"max_score"`
	Thresholds       []ScoreThreshold `bson:"thresholds" json:"thresholds"`
}

// RiskCondition defines a single condition for evaluating a risk profile
type RiskCondition struct {
	DimensionCode string   `bson:"dimension_code" json:"dimension_code"`
	Operator      string   `bson:"operator" json:"operator"` // "level_in", "score_gt", "score_lt"
	Values        []string `bson:"values" json:"values"`
}

// RiskProfile defines a configurable risk pattern across multiple dimensions
type RiskProfile struct {
	Name             string          `bson:"name" json:"name"`
	Description      string          `bson:"description" json:"description"`
	Severity         string          `bson:"severity" json:"severity"` // "critical", "warning", "info"
	Color            string          `bson:"color" json:"color"`
	Logic            string          `bson:"logic" json:"logic"` // "all" (AND) or "any" (OR)
	Conditions       []RiskCondition `bson:"conditions" json:"conditions"`
}

// EvaluationConfig holds the scoring methodology for a questionnaire
type EvaluationConfig struct {
	Enabled               bool              `bson:"enabled" json:"enabled"`
	ScoringMethod         string            `bson:"scoring_method" json:"scoring_method"` // "sum" or "average"
	Dimensions            []DimensionConfig `bson:"dimensions" json:"dimensions"`
	RiskProfiles          []RiskProfile     `bson:"risk_profiles,omitempty" json:"risk_profiles,omitempty"`
	GeneralInterpretation string            `bson:"general_interpretation,omitempty" json:"general_interpretation,omitempty"`
}

// Questionnaire represents a questionnaire with embedded questions
type Questionnaire struct {
	ID               primitive.ObjectID  `bson:"_id,omitempty" json:"id,omitempty"`
	Title            string              `bson:"title" json:"title" validate:"required,min=5,max=200"`
	Description      string              `bson:"description" json:"description"`
	CoverImage       string              `bson:"cover_image,omitempty" json:"cover_image,omitempty"`
	CreatedBy        string              `bson:"created_by" json:"created_by"` // FusionAuth user ID
	IsActive         bool                `bson:"is_active" json:"is_active"`
	Questions        []Question          `bson:"questions" json:"questions"`
	Sections         []Section           `bson:"sections,omitempty" json:"sections,omitempty"`
	EvaluationConfig *EvaluationConfig   `bson:"evaluation_config,omitempty" json:"evaluation_config,omitempty"`
	Tags             []string            `bson:"tags" json:"tags"`
	CategoryID       *primitive.ObjectID `bson:"category_id,omitempty" json:"category_id,omitempty"`
	CreatedAt        time.Time           `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time           `bson:"updated_at" json:"updated_at"`
}

// ColorMode defines how colors are resolved for a company questionnaire
type ColorMode string

const (
	ColorModeDefault     ColorMode = "default"      // use base questionnaire colors
	ColorModeFromPrimary ColorMode = "from_primary"  // generate from company primary color
	ColorModeCustom      ColorMode = "custom"        // manually configured colors
)

// DimensionColorConfig holds threshold colors for a specific dimension
type DimensionColorConfig struct {
	DimensionCode   string            `bson:"dimension_code" json:"dimension_code"`
	ThresholdColors map[string]string `bson:"threshold_colors" json:"threshold_colors"` // level -> hex
}

// RiskProfileColorConfig holds color for a specific risk profile
type RiskProfileColorConfig struct {
	ProfileName string `bson:"profile_name" json:"profile_name"`
	Color       string `bson:"color" json:"color"`
}

// ColorConfig stores color configuration for a company questionnaire assignment
type ColorConfig struct {
	Mode              ColorMode                `bson:"mode" json:"mode"`
	PrimaryColor      string                   `bson:"primary_color,omitempty" json:"primary_color,omitempty"`
	DimensionColors   []DimensionColorConfig   `bson:"dimension_colors,omitempty" json:"dimension_colors,omitempty"`
	RiskProfileColors []RiskProfileColorConfig `bson:"risk_profile_colors,omitempty" json:"risk_profile_colors,omitempty"`
}

// CompanyQuestionnaireStatus represents the lifecycle state of a company questionnaire
type CompanyQuestionnaireStatus string

const (
	CQStatusDraft   CompanyQuestionnaireStatus = "draft"   // assigned but not yet started
	CQStatusActive  CompanyQuestionnaireStatus = "active"  // within period, receiving responses
	CQStatusPaused  CompanyQuestionnaireStatus = "paused"  // temporarily paused
	CQStatusClosed  CompanyQuestionnaireStatus = "closed"  // manually closed (terminal)
	CQStatusExpired CompanyQuestionnaireStatus = "expired" // period ended automatically
)

// DisplayMode controls how questions are presented to the employee
type DisplayMode string

const (
	DisplayModeStepByStep DisplayMode = "step_by_step"  // one question per page (default)
	DisplayModeAllAtOnce  DisplayMode = "all_at_once"   // all questions on one scrollable page
	DisplayModeBySection  DisplayMode = "by_section"    // one section per page, multiple questions
)

// StatusChange records a state transition in the lifecycle
type StatusChange struct {
	Status    CompanyQuestionnaireStatus `bson:"status" json:"status"`
	ChangedBy string                    `bson:"changed_by" json:"changed_by"`
	ChangedAt time.Time                 `bson:"changed_at" json:"changed_at"`
	Reason    string                    `bson:"reason,omitempty" json:"reason,omitempty"`
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
	ID              primitive.ObjectID         `bson:"_id,omitempty" json:"id,omitempty"`
	CompanyID       primitive.ObjectID         `bson:"company_id" json:"company_id" validate:"required"`
	QuestionnaireID primitive.ObjectID         `bson:"questionnaire_id" json:"questionnaire_id" validate:"required"`
	AssignedBy      string                     `bson:"assigned_by" json:"assigned_by"`   // FusionAuth user ID
	AssignedAt      time.Time                  `bson:"assigned_at" json:"assigned_at"`
	PeriodStart     time.Time                  `bson:"period_start" json:"period_start"`
	PeriodEnd       time.Time                  `bson:"period_end" json:"period_end"`
	IsActive        bool                       `bson:"is_active" json:"is_active"`
	Status          CompanyQuestionnaireStatus `bson:"status" json:"status"`
	StatusHistory   []StatusChange             `bson:"status_history" json:"status_history"`
	RemindersSent   int                        `bson:"reminders_sent" json:"reminders_sent"`
	LastReminderAt  *time.Time                 `bson:"last_reminder_at,omitempty" json:"last_reminder_at,omitempty"`
	DisplayMode     DisplayMode                `bson:"display_mode,omitempty" json:"display_mode,omitempty"`
	ColorConfig         *ColorConfig               `bson:"color_config,omitempty" json:"color_config,omitempty"`
	SupersetDashboardID string                     `bson:"superset_dashboard_id,omitempty" json:"superset_dashboard_id,omitempty"`
	// Transient fields — not persisted, populated by service layer
	QuestionnaireTitle       string `bson:"-" json:"questionnaire_title,omitempty"`
	QuestionnaireDescription string `bson:"-" json:"questionnaire_description,omitempty"`
}

// NewCompanyQuestionnaire creates a new company questionnaire assignment
func NewCompanyQuestionnaire(companyID, questionnaireID primitive.ObjectID, assignedBy string, periodStart, periodEnd time.Time) *CompanyQuestionnaire {
	now := time.Now()
	initialStatus := CQStatusActive
	return &CompanyQuestionnaire{
		ID:              primitive.NewObjectID(),
		CompanyID:       companyID,
		QuestionnaireID: questionnaireID,
		AssignedBy:      assignedBy,
		AssignedAt:      now,
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		IsActive:        true,
		Status:          initialStatus,
		StatusHistory: []StatusChange{
			{Status: initialStatus, ChangedBy: assignedBy, ChangedAt: now},
		},
	}
}

// IsWithinPeriod checks if the current time is within the assignment period
func (cq *CompanyQuestionnaire) IsWithinPeriod() bool {
	now := time.Now()
	return now.After(cq.PeriodStart) && now.Before(cq.PeriodEnd)
}

// TransitionStatus applies a status transition and records it in the history
// Returns an error if the transition is not allowed
func (cq *CompanyQuestionnaire) TransitionStatus(newStatus CompanyQuestionnaireStatus, changedBy, reason string) error {
	allowed := map[CompanyQuestionnaireStatus][]CompanyQuestionnaireStatus{
		CQStatusDraft:   {CQStatusActive},
		CQStatusActive:  {CQStatusPaused, CQStatusClosed},
		CQStatusPaused:  {CQStatusActive, CQStatusClosed},
		CQStatusClosed:  {},
		CQStatusExpired: {},
	}

	validNext, ok := allowed[cq.Status]
	if !ok {
		return fmt.Errorf("unknown current status: %s", cq.Status)
	}
	for _, s := range validNext {
		if s == newStatus {
			cq.StatusHistory = append(cq.StatusHistory, StatusChange{
				Status:    newStatus,
				ChangedBy: changedBy,
				ChangedAt: time.Now(),
				Reason:    reason,
			})
			cq.Status = newStatus
			cq.IsActive = newStatus == CQStatusActive
			return nil
		}
	}
	return fmt.Errorf("transition from %s to %s is not allowed", cq.Status, newStatus)
}
