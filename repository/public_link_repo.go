package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"questionarie-service/models"
)

type PublicLinkRepository struct {
	collection *mongo.Collection
}

func NewPublicLinkRepository(db *mongo.Database) *PublicLinkRepository {
	return &PublicLinkRepository{collection: db.Collection("public_links")}
}

func (r *PublicLinkRepository) Create(ctx context.Context, link *models.PublicLink) error {
	link.CreatedAt = time.Now()
	link.UpdatedAt = time.Now()
	result, err := r.collection.InsertOne(ctx, link)
	if err != nil {
		return fmt.Errorf("failed to create public link: %w", err)
	}
	link.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *PublicLinkRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.PublicLink, error) {
	var link models.PublicLink
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&link)
	if err != nil {
		return nil, fmt.Errorf("failed to get public link: %w", err)
	}
	return &link, nil
}

func (r *PublicLinkRepository) GetBySlug(ctx context.Context, slug string) (*models.PublicLink, error) {
	var link models.PublicLink
	err := r.collection.FindOne(ctx, bson.M{"slug": slug}).Decode(&link)
	if err != nil {
		return nil, fmt.Errorf("failed to get public link by slug: %w", err)
	}
	return &link, nil
}

func (r *PublicLinkRepository) GetByCompanyQuestionnaire(ctx context.Context, cqID primitive.ObjectID) ([]models.PublicLink, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"company_questionnaire_id": cqID}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to list public links: %w", err)
	}
	defer cursor.Close(ctx)
	var links []models.PublicLink
	if err := cursor.All(ctx, &links); err != nil {
		return nil, fmt.Errorf("failed to decode public links: %w", err)
	}
	return links, nil
}

func (r *PublicLinkRepository) Update(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	update["updated_at"] = time.Now()
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	if err != nil {
		return fmt.Errorf("failed to update public link: %w", err)
	}
	return nil
}

func (r *PublicLinkRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete public link: %w", err)
	}
	return nil
}

// DeleteByCompanyQuestionnaireIDs removes all public links for the given CQ IDs.
func (r *PublicLinkRepository) DeleteByCompanyQuestionnaireIDs(ctx context.Context, cqIDs []primitive.ObjectID) (int64, error) {
	if len(cqIDs) == 0 {
		return 0, nil
	}
	result, err := r.collection.DeleteMany(ctx, bson.M{"company_questionnaire_id": bson.M{"$in": cqIDs}})
	if err != nil {
		return 0, fmt.Errorf("failed to delete public links by CQ IDs: %w", err)
	}
	return result.DeletedCount, nil
}

func (r *PublicLinkRepository) IncrementResponseCount(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$inc": bson.M{"response_count": 1}})
	if err != nil {
		return fmt.Errorf("failed to increment response count: %w", err)
	}
	return nil
}

func (r *PublicLinkRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"slug": slug})
	if err != nil {
		return false, fmt.Errorf("failed to check slug: %w", err)
	}
	return count > 0, nil
}

func (r *PublicLinkRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "slug", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

func (r *PublicLinkRepository) GetByCompanyID(ctx context.Context, companyID primitive.ObjectID) ([]models.PublicLink, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"company_id": companyID}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to list public links by company: %w", err)
	}
	defer cursor.Close(ctx)
	var links []models.PublicLink
	if err := cursor.All(ctx, &links); err != nil {
		return nil, fmt.Errorf("failed to decode public links: %w", err)
	}
	return links, nil
}
