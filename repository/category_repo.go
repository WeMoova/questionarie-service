package repository

import (
	"context"
	"fmt"
	"questionarie-service/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CategoryRepository handles questionnaire category operations
type CategoryRepository struct {
	collection *mongo.Collection
}

// NewCategoryRepository creates a new CategoryRepository
func NewCategoryRepository(db *mongo.Database) *CategoryRepository {
	return &CategoryRepository{
		collection: db.Collection("questionnaire_categories"),
	}
}

// Create creates a new category
func (r *CategoryRepository) Create(ctx context.Context, category *models.QuestionnaireCategory) error {
	_, err := r.collection.InsertOne(ctx, category)
	if err != nil {
		return fmt.Errorf("failed to create category: %w", err)
	}
	return nil
}

// GetByID retrieves a category by ID
func (r *CategoryRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.QuestionnaireCategory, error) {
	var category models.QuestionnaireCategory
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&category)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("category not found")
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}
	return &category, nil
}

// GetAll retrieves all categories with optional active filter
func (r *CategoryRepository) GetAll(ctx context.Context, activeOnly bool) ([]*models.QuestionnaireCategory, error) {
	filter := bson.M{}
	if activeOnly {
		filter["is_active"] = true
	}

	opts := options.Find().SetSort(bson.D{{Key: "name", Value: 1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	defer cursor.Close(ctx)

	var categories []*models.QuestionnaireCategory
	if err = cursor.All(ctx, &categories); err != nil {
		return nil, fmt.Errorf("failed to decode categories: %w", err)
	}

	return categories, nil
}

// Update updates a category
func (r *CategoryRepository) Update(ctx context.Context, id primitive.ObjectID, category *models.QuestionnaireCategory) error {
	update := bson.M{
		"$set": bson.M{
			"name":        category.Name,
			"description": category.Description,
			"is_active":   category.IsActive,
			"updated_at":  category.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("category not found")
	}

	return nil
}

// Delete permanently removes a category
func (r *CategoryRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("category not found")
	}
	return nil
}
