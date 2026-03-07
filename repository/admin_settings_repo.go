package repository

import (
	"context"
	"fmt"
	"questionarie-service/models"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AdminSettingsRepository handles admin settings data operations
type AdminSettingsRepository struct {
	collection *mongo.Collection
}

// NewAdminSettingsRepository creates a new AdminSettingsRepository
func NewAdminSettingsRepository(db *mongo.Database) *AdminSettingsRepository {
	return &AdminSettingsRepository{
		collection: db.Collection("admin_settings"),
	}
}

// Get retrieves the admin settings document (singleton). If none exists, it seeds the default.
func (r *AdminSettingsRepository) Get(ctx context.Context) (*models.AdminSettings, error) {
	var settings models.AdminSettings
	err := r.collection.FindOne(ctx, bson.M{}).Decode(&settings)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Seed default and return
			defaultSettings := r.seedDefault()
			_, insertErr := r.collection.InsertOne(ctx, defaultSettings)
			if insertErr != nil {
				return nil, fmt.Errorf("failed to seed admin settings: %w", insertErr)
			}
			return defaultSettings, nil
		}
		return nil, fmt.Errorf("failed to get admin settings: %w", err)
	}
	return &settings, nil
}

// UpdateTheme updates the active theme
func (r *AdminSettingsRepository) UpdateTheme(ctx context.Context, theme models.AdminTheme, updatedBy string) error {
	update := bson.M{
		"$set": bson.M{
			"active_theme": theme,
			"updated_at":   time.Now(),
			"updated_by":   updatedBy,
		},
	}
	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, bson.M{}, update, opts)
	if err != nil {
		return fmt.Errorf("failed to update admin theme: %w", err)
	}
	return nil
}

// AddPreset adds a new preset to the presets array
func (r *AdminSettingsRepository) AddPreset(ctx context.Context, preset models.AdminThemePreset) error {
	update := bson.M{
		"$push": bson.M{
			"presets": preset,
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}
	_, err := r.collection.UpdateOne(ctx, bson.M{}, update)
	if err != nil {
		return fmt.Errorf("failed to add preset: %w", err)
	}
	return nil
}

// UpdatePreset updates an existing preset identified by its ID
func (r *AdminSettingsRepository) UpdatePreset(ctx context.Context, presetID string, preset models.AdminThemePreset) error {
	filter := bson.M{"presets.id": presetID}
	update := bson.M{
		"$set": bson.M{
			"presets.$":  preset,
			"updated_at": time.Now(),
		},
	}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update preset: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("preset not found")
	}
	return nil
}

// DeletePreset removes a preset by its ID
func (r *AdminSettingsRepository) DeletePreset(ctx context.Context, presetID string) error {
	update := bson.M{
		"$pull": bson.M{
			"presets": bson.M{"id": presetID},
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}
	result, err := r.collection.UpdateOne(ctx, bson.M{}, update)
	if err != nil {
		return fmt.Errorf("failed to delete preset: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("preset not found")
	}
	return nil
}

// seedDefault returns the default admin settings with sample presets
func (r *AdminSettingsRepository) seedDefault() *models.AdminSettings {
	now := time.Now()

	scheduleNavStart := "2026-12-01"
	scheduleNavEnd := "2026-12-31"
	scheduleHalStart := "2026-10-25"
	scheduleHalEnd := "2026-11-01"

	return &models.AdminSettings{
		ID: primitive.NewObjectID(),
		ActiveTheme: models.AdminTheme{
			PrimaryColor:   "#6C3FE4",
			SecondaryColor: "#3B82F6",
			AccentColor:    "#10B981",
			WelcomeMessage: "Bienvenido a WeMoova",
		},
		Presets: []models.AdminThemePreset{
			{
				ID:   uuid.New().String(),
				Name: "Navidad 2026",
				Theme: models.AdminTheme{
					PrimaryColor:   "#C41E3A",
					SecondaryColor: "#1A5E1A",
					AccentColor:    "#FFD700",
					WelcomeMessage: "Felices fiestas!",
				},
				ScheduleStart: &scheduleNavStart,
				ScheduleEnd:   &scheduleNavEnd,
				CreatedAt:     now,
			},
			{
				ID:   uuid.New().String(),
				Name: "Halloween 2026",
				Theme: models.AdminTheme{
					PrimaryColor:   "#FF6600",
					SecondaryColor: "#1A1A2E",
					AccentColor:    "#9B59B6",
					WelcomeMessage: "Happy Halloween",
				},
				ScheduleStart: &scheduleHalStart,
				ScheduleEnd:   &scheduleHalEnd,
				CreatedAt:     now,
			},
		},
		UpdatedAt: now,
		UpdatedBy: "system",
	}
}
