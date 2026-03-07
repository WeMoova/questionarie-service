package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AdminTheme holds visual customization for the admin platform
type AdminTheme struct {
	Logo           string   `bson:"logo,omitempty" json:"logo,omitempty"`
	LogoIcon       string   `bson:"logo_icon,omitempty" json:"logo_icon,omitempty"`
	LoginImages    []string `bson:"login_images,omitempty" json:"login_images,omitempty"`
	PrimaryColor   string   `bson:"primary_color,omitempty" json:"primary_color,omitempty"`
	SecondaryColor string   `bson:"secondary_color,omitempty" json:"secondary_color,omitempty"`
	AccentColor    string   `bson:"accent_color,omitempty" json:"accent_color,omitempty"`
	WelcomeMessage string   `bson:"welcome_message,omitempty" json:"welcome_message,omitempty"`
}

// AdminThemePreset represents a saved theme preset with optional scheduling
type AdminThemePreset struct {
	ID            string     `bson:"id" json:"id"`
	Name          string     `bson:"name" json:"name"`
	Theme         AdminTheme `bson:"theme" json:"theme"`
	ScheduleStart *string    `bson:"schedule_start,omitempty" json:"schedule_start,omitempty"`
	ScheduleEnd   *string    `bson:"schedule_end,omitempty" json:"schedule_end,omitempty"`
	CreatedAt     time.Time  `bson:"created_at" json:"created_at"`
}

// AdminSettings holds platform-wide admin settings including branding
type AdminSettings struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	ActiveTheme AdminTheme         `bson:"active_theme" json:"active_theme"`
	Presets     []AdminThemePreset `bson:"presets" json:"presets"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
	UpdatedBy   string             `bson:"updated_by" json:"updated_by"`
}
