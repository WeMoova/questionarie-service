package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Branding holds visual customization for the employee-facing app
type Branding struct {
	Logo                 string `bson:"logo,omitempty" json:"logo,omitempty"`
	LogoIcon             string `bson:"logo_icon,omitempty" json:"logo_icon,omitempty"`
	Favicon              string `bson:"favicon,omitempty" json:"favicon,omitempty"`
	PrimaryColor         string `bson:"primary_color,omitempty" json:"primary_color,omitempty"`
	SecondaryColor       string `bson:"secondary_color,omitempty" json:"secondary_color,omitempty"`
	AccentColor          string `bson:"accent_color,omitempty" json:"accent_color,omitempty"`
	LoginBackgroundImage string `bson:"login_background_image,omitempty" json:"login_background_image,omitempty"`
	WelcomeMessage       string `bson:"welcome_message,omitempty" json:"welcome_message,omitempty"`
}

// CustomDomain holds URL configuration for the employee-facing app
type CustomDomain struct {
	Slug         string `bson:"slug,omitempty" json:"slug,omitempty"`
	CustomDomain string `bson:"custom_domain,omitempty" json:"custom_domain,omitempty"`
	IsVerified   bool   `bson:"is_verified" json:"is_verified"`
}

// Company represents a company entity
type Company struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name         string             `bson:"name" json:"name" validate:"required,min=3,max=200"`
	IsActive     bool               `bson:"is_active" json:"is_active"`
	Branding     *Branding          `bson:"branding,omitempty" json:"branding,omitempty"`
	CustomDomain *CustomDomain      `bson:"custom_domain,omitempty" json:"custom_domain,omitempty"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
	// Transient field — not persisted, populated by service layer
	QuestionnaireCount int `bson:"-" json:"questionnaire_count"`
}

// NewCompany creates a new Company with timestamps
func NewCompany(name string) *Company {
	now := time.Now()
	return &Company{
		ID:        primitive.NewObjectID(),
		Name:      name,
		IsActive:  true, // Default: true
		CreatedAt: now,
		UpdatedAt: now,
	}
}
