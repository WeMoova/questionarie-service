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

// APITokenRepository handles API token data operations
type APITokenRepository struct {
	collection *mongo.Collection
}

// NewAPITokenRepository creates a new APITokenRepository
func NewAPITokenRepository(db *mongo.Database) *APITokenRepository {
	repo := &APITokenRepository{
		collection: db.Collection("company_api_tokens"),
	}
	repo.ensureIndexes()
	return repo
}

func (r *APITokenRepository) ensureIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "token_hash", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "company_id", Value: 1}},
		},
	})
}

// Create inserts a new API token
func (r *APITokenRepository) Create(ctx context.Context, token *models.APIToken) error {
	_, err := r.collection.InsertOne(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to create API token: %w", err)
	}
	return nil
}

// GetByCompanyID returns all tokens for a company
func (r *APITokenRepository) GetByCompanyID(ctx context.Context, companyID primitive.ObjectID) ([]*models.APIToken, error) {
	filter := bson.M{"company_id": companyID}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get API tokens: %w", err)
	}
	defer cursor.Close(ctx)

	var tokens []*models.APIToken
	if err = cursor.All(ctx, &tokens); err != nil {
		return nil, fmt.Errorf("failed to decode API tokens: %w", err)
	}
	return tokens, nil
}

// GetByID retrieves a single token by ID
func (r *APITokenRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.APIToken, error) {
	var token models.APIToken
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&token)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("API token not found")
		}
		return nil, fmt.Errorf("failed to get API token: %w", err)
	}
	return &token, nil
}

// FindByHash finds an active, non-revoked token by its hash
func (r *APITokenRepository) FindByHash(ctx context.Context, hash string) (*models.APIToken, error) {
	filter := bson.M{
		"token_hash": hash,
		"is_active":  true,
		"revoked_at": bson.M{"$eq": nil},
	}
	var token models.APIToken
	err := r.collection.FindOne(ctx, filter).Decode(&token)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find API token: %w", err)
	}
	return &token, nil
}

// Revoke marks a token as revoked
func (r *APITokenRepository) Revoke(ctx context.Context, id primitive.ObjectID) error {
	now := time.Now()
	update := bson.M{"$set": bson.M{"revoked_at": now, "is_active": false}}
	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to revoke API token: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("API token not found")
	}
	return nil
}

// ToggleActive toggles the is_active flag
func (r *APITokenRepository) ToggleActive(ctx context.Context, id primitive.ObjectID, active bool) error {
	update := bson.M{"$set": bson.M{"is_active": active}}
	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to toggle API token: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("API token not found")
	}
	return nil
}

// UpdateLastUsed updates the last_used_at timestamp
func (r *APITokenRepository) UpdateLastUsed(ctx context.Context, id primitive.ObjectID) error {
	now := time.Now()
	update := bson.M{"$set": bson.M{"last_used_at": now}}
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}
