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

// CompanyRepository handles company data operations
type CompanyRepository struct {
	collection *mongo.Collection
}

// NewCompanyRepository creates a new CompanyRepository
func NewCompanyRepository(db *mongo.Database) *CompanyRepository {
	return &CompanyRepository{
		collection: db.Collection("companies"),
	}
}

// Create creates a new company
func (r *CompanyRepository) Create(ctx context.Context, company *models.Company) error {
	_, err := r.collection.InsertOne(ctx, company)
	if err != nil {
		return fmt.Errorf("failed to create company: %w", err)
	}
	return nil
}

// GetByID retrieves a company by ID
func (r *CompanyRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Company, error) {
	var company models.Company
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&company)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("company not found")
		}
		return nil, fmt.Errorf("failed to get company: %w", err)
	}
	return &company, nil
}

// GetAll retrieves all companies with pagination
func (r *CompanyRepository) GetAll(ctx context.Context, page, pageSize int64) ([]*models.Company, error) {
	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSkip(skip).
		SetLimit(pageSize).
		SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get companies: %w", err)
	}
	defer cursor.Close(ctx)

	var companies []*models.Company
	if err = cursor.All(ctx, &companies); err != nil {
		return nil, fmt.Errorf("failed to decode companies: %w", err)
	}

	return companies, nil
}

// Update updates a company
func (r *CompanyRepository) Update(ctx context.Context, id primitive.ObjectID, company *models.Company) error {
	update := bson.M{
		"$set": bson.M{
			"name":       company.Name,
			"updated_at": company.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to update company: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("company not found")
	}

	return nil
}

// Delete deletes a company (soft delete - could be extended)
func (r *CompanyRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete company: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("company not found")
	}

	return nil
}

// Count returns the total number of companies
func (r *CompanyRepository) Count(ctx context.Context) (int64, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, fmt.Errorf("failed to count companies: %w", err)
	}
	return count, nil
}

// SearchByName searches companies by name (case-insensitive)
func (r *CompanyRepository) SearchByName(ctx context.Context, name string) ([]*models.Company, error) {
	filter := bson.M{
		"name": bson.M{
			"$regex":   name,
			"$options": "i", // case-insensitive
		},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to search companies: %w", err)
	}
	defer cursor.Close(ctx)

	var companies []*models.Company
	if err = cursor.All(ctx, &companies); err != nil {
		return nil, fmt.Errorf("failed to decode companies: %w", err)
	}

	return companies, nil
}
