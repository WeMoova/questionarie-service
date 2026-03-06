package repository

import (
	"context"
	"fmt"
	"questionarie-service/models"
	"regexp"

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
	setFields := bson.M{
		"name":       company.Name,
		"is_active":  company.IsActive,
		"updated_at": company.UpdatedAt,
	}
	if company.Branding != nil {
		setFields["branding"] = company.Branding
	}
	if company.CustomDomain != nil {
		setFields["custom_domain"] = company.CustomDomain
	}
	if company.Settings != nil {
		setFields["settings"] = company.Settings
	}
	if company.FusionAuthTenantID != "" {
		setFields["fusionauth_tenant_id"] = company.FusionAuthTenantID
	}
	if company.FusionAuthApplicationID != "" {
		setFields["fusionauth_application_id"] = company.FusionAuthApplicationID
	}
	if company.FusionAuthClientID != "" {
		setFields["fusionauth_client_id"] = company.FusionAuthClientID
	}
	if company.FusionAuthClientSecret != "" {
		setFields["fusionauth_client_secret"] = company.FusionAuthClientSecret
	}
	update := bson.M{"$set": setFields}

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

// GetBySlug finds a company by its custom domain slug
func (r *CompanyRepository) GetBySlug(ctx context.Context, slug string) (*models.Company, error) {
	var company models.Company
	err := r.collection.FindOne(ctx, bson.M{"custom_domain.slug": slug}).Decode(&company)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find company by slug: %w", err)
	}
	return &company, nil
}

// SearchByName searches companies by name (case-insensitive)
func (r *CompanyRepository) SearchByName(ctx context.Context, name string) ([]*models.Company, error) {
	filter := bson.M{
		"name": bson.M{
			"$regex":   regexp.QuoteMeta(name),
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
