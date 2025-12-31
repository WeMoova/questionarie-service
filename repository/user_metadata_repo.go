package repository

import (
	"context"
	"fmt"
	"questionarie-service/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UserMetadataRepository handles user metadata operations
type UserMetadataRepository struct {
	collection *mongo.Collection
}

// NewUserMetadataRepository creates a new UserMetadataRepository
func NewUserMetadataRepository(db *mongo.Database) *UserMetadataRepository {
	return &UserMetadataRepository{
		collection: db.Collection("users_metadata"),
	}
}

// Create creates a new user metadata
func (r *UserMetadataRepository) Create(ctx context.Context, metadata *models.UserMetadata) error {
	_, err := r.collection.InsertOne(ctx, metadata)
	if err != nil {
		return fmt.Errorf("failed to create user metadata: %w", err)
	}
	return nil
}

// GetByID retrieves user metadata by user ID (FusionAuth ID)
func (r *UserMetadataRepository) GetByID(ctx context.Context, userID string) (*models.UserMetadata, error) {
	var metadata models.UserMetadata
	err := r.collection.FindOne(ctx, bson.M{"_id": userID}).Decode(&metadata)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user metadata not found")
		}
		return nil, fmt.Errorf("failed to get user metadata: %w", err)
	}
	return &metadata, nil
}

// GetByCompanyID retrieves all users metadata for a company
func (r *UserMetadataRepository) GetByCompanyID(ctx context.Context, companyID primitive.ObjectID) ([]*models.UserMetadata, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"company_id": companyID})
	if err != nil {
		return nil, fmt.Errorf("failed to get users by company: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*models.UserMetadata
	if err = cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}

	return users, nil
}

// GetBySupervisorID retrieves all users supervised by a specific supervisor
func (r *UserMetadataRepository) GetBySupervisorID(ctx context.Context, supervisorID string) ([]*models.UserMetadata, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"supervisor_id": supervisorID})
	if err != nil {
		return nil, fmt.Errorf("failed to get users by supervisor: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*models.UserMetadata
	if err = cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}

	return users, nil
}

// GetByCompanyAndDepartment retrieves users by company and department
func (r *UserMetadataRepository) GetByCompanyAndDepartment(ctx context.Context, companyID primitive.ObjectID, department string) ([]*models.UserMetadata, error) {
	filter := bson.M{
		"company_id": companyID,
		"department": department,
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by department: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*models.UserMetadata
	if err = cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}

	return users, nil
}

// Update updates user metadata
func (r *UserMetadataRepository) Update(ctx context.Context, userID string, metadata *models.UserMetadata) error {
	metadata.UpdatedAt = time.Now()
	update := bson.M{
		"$set": bson.M{
			"company_id":    metadata.CompanyID,
			"supervisor_id": metadata.SupervisorID,
			"department":    metadata.Department,
			"updated_at":    metadata.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": userID}, update)
	if err != nil {
		return fmt.Errorf("failed to update user metadata: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user metadata not found")
	}

	return nil
}

// UpdateCompany updates only the company for a user
func (r *UserMetadataRepository) UpdateCompany(ctx context.Context, userID string, companyID primitive.ObjectID) error {
	update := bson.M{
		"$set": bson.M{
			"company_id": companyID,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": userID}, update)
	if err != nil {
		return fmt.Errorf("failed to update user company: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user metadata not found")
	}

	return nil
}

// UpdateSupervisor updates only the supervisor for a user
func (r *UserMetadataRepository) UpdateSupervisor(ctx context.Context, userID, supervisorID string) error {
	update := bson.M{
		"$set": bson.M{
			"supervisor_id": supervisorID,
			"updated_at":    time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": userID}, update)
	if err != nil {
		return fmt.Errorf("failed to update user supervisor: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user metadata not found")
	}

	return nil
}

// Delete deletes user metadata
func (r *UserMetadataRepository) Delete(ctx context.Context, userID string) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": userID})
	if err != nil {
		return fmt.Errorf("failed to delete user metadata: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("user metadata not found")
	}

	return nil
}

// Exists checks if user metadata exists
func (r *UserMetadataRepository) Exists(ctx context.Context, userID string) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"_id": userID})
	if err != nil {
		return false, fmt.Errorf("failed to check user metadata existence: %w", err)
	}
	return count > 0, nil
}

// GetByIDs retrieves multiple users metadata by their IDs
func (r *UserMetadataRepository) GetByIDs(ctx context.Context, userIDs []string) ([]*models.UserMetadata, error) {
	filter := bson.M{
		"_id": bson.M{"$in": userIDs},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by IDs: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*models.UserMetadata
	if err = cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}

	return users, nil
}

// CountByCompany returns the total number of users in a company
func (r *UserMetadataRepository) CountByCompany(ctx context.Context, companyID primitive.ObjectID) (int64, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"company_id": companyID})
	if err != nil {
		return 0, fmt.Errorf("failed to count users by company: %w", err)
	}
	return count, nil
}

// GetDepartmentsByCompany retrieves all unique departments for a company
func (r *UserMetadataRepository) GetDepartmentsByCompany(ctx context.Context, companyID primitive.ObjectID) ([]string, error) {
	filter := bson.M{
		"company_id": companyID,
		"department": bson.M{"$exists": true, "$ne": ""},
	}

	departments, err := r.collection.Distinct(ctx, "department", filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get departments: %w", err)
	}

	result := make([]string, len(departments))
	for i, dept := range departments {
		result[i] = dept.(string)
	}

	return result, nil
}

// GetUsersByCompanyWithPagination retrieves users with pagination
func (r *UserMetadataRepository) GetUsersByCompanyWithPagination(ctx context.Context, companyID primitive.ObjectID, page, pageSize int64) ([]*models.UserMetadata, error) {
	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSkip(skip).
		SetLimit(pageSize).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{"company_id": companyID}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*models.UserMetadata
	if err = cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}

	return users, nil
}
