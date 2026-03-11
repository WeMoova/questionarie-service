package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DemographicField struct {
	Key      string `bson:"key" json:"key" validate:"required"`
	Label    string `bson:"label" json:"label" validate:"required"`
	Required bool   `bson:"required" json:"required"`
}

type ScoreRange struct {
	Min     float64 `bson:"min" json:"min"`
	Max     float64 `bson:"max" json:"max"`
	Label   string  `bson:"label" json:"label"`
	Message string  `bson:"message" json:"message"`
	Color   string  `bson:"color" json:"color"`
}

// ResultScreen holds the two-page result for a specific risk profile:
// Page 1 = risk result, Page 2 = action plan.
type ResultScreen struct {
	ProfileName   string `bson:"profile_name" json:"profile_name"`
	ResultTitle   string `bson:"result_title" json:"result_title"`
	ResultContent string `bson:"result_content" json:"result_content"` // markdown
	ResultImage   string `bson:"result_image,omitempty" json:"result_image,omitempty"`
	PlanTitle     string `bson:"plan_title" json:"plan_title"`
	PlanContent   string `bson:"plan_content" json:"plan_content"` // markdown
	PlanImage     string `bson:"plan_image,omitempty" json:"plan_image,omitempty"`
	Color         string `bson:"color,omitempty" json:"color,omitempty"`
}

type ResultConfig struct {
	Mode          string         `bson:"mode" json:"mode"` // "message", "score", "risk_profile"
	Title         string         `bson:"title" json:"title"`
	Message       string         `bson:"message" json:"message"`
	ImageURL      string         `bson:"image_url,omitempty" json:"image_url,omitempty"`
	RedirectURL   string         `bson:"redirect_url,omitempty" json:"redirect_url,omitempty"`
	RedirectDelay int            `bson:"redirect_delay" json:"redirect_delay"`
	ScoreRanges   []ScoreRange   `bson:"score_ranges,omitempty" json:"score_ranges,omitempty"`
	ResultScreens []ResultScreen `bson:"result_screens,omitempty" json:"result_screens,omitempty"`
}

type PublicLink struct {
	ID                     primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Slug                   string             `bson:"slug" json:"slug" validate:"required,min=3,max=64"`
	CompanyID              primitive.ObjectID `bson:"company_id" json:"company_id" validate:"required"`
	CompanyQuestionnaireID primitive.ObjectID `bson:"company_questionnaire_id" json:"company_questionnaire_id" validate:"required"`
	Label                  string             `bson:"label" json:"label"`
	IsActive               bool               `bson:"is_active" json:"is_active"`
	DemographicFields      []DemographicField `bson:"demographic_fields" json:"demographic_fields"`
	ResultConfig           ResultConfig       `bson:"result_config" json:"result_config"`
	ResponseCount          int                `bson:"response_count" json:"response_count"`
	CreatedAt              time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt              time.Time          `bson:"updated_at" json:"updated_at"`
	// Transient fields
	QuestionnaireTitle string `bson:"-" json:"questionnaire_title,omitempty"`
	CompanyName        string `bson:"-" json:"company_name,omitempty"`
}
