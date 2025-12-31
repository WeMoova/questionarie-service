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

// CompanyQuestionnaireRepository handles company questionnaire assignments
type CompanyQuestionnaireRepository struct {
	collection *mongo.Collection
}

// NewCompanyQuestionnaireRepository creates a new CompanyQuestionnaireRepository
func NewCompanyQuestionnaireRepository(db *mongo.Database) *CompanyQuestionnaireRepository {
	return &CompanyQuestionnaireRepository{
		collection: db.Collection("company_questionnaires"),
	}
}

// Create creates a new company questionnaire assignment
func (r *CompanyQuestionnaireRepository) Create(ctx context.Context, cq *models.CompanyQuestionnaire) error {
	_, err := r.collection.InsertOne(ctx, cq)
	if err != nil {
		return fmt.Errorf("failed to create company questionnaire: %w", err)
	}
	return nil
}

// GetByID retrieves a company questionnaire by ID
func (r *CompanyQuestionnaireRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.CompanyQuestionnaire, error) {
	var cq models.CompanyQuestionnaire
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&cq)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("company questionnaire not found")
		}
		return nil, fmt.Errorf("failed to get company questionnaire: %w", err)
	}
	return &cq, nil
}

// GetByCompanyID retrieves all questionnaires assigned to a company
func (r *CompanyQuestionnaireRepository) GetByCompanyID(ctx context.Context, companyID primitive.ObjectID, activeOnly bool) ([]*models.CompanyQuestionnaire, error) {
	filter := bson.M{"company_id": companyID}
	if activeOnly {
		filter["is_active"] = true
	}

	opts := options.Find().SetSort(bson.D{{Key: "assigned_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get company questionnaires: %w", err)
	}
	defer cursor.Close(ctx)

	var cqs []*models.CompanyQuestionnaire
	if err = cursor.All(ctx, &cqs); err != nil {
		return nil, fmt.Errorf("failed to decode company questionnaires: %w", err)
	}

	return cqs, nil
}

// GetByQuestionnaireID retrieves all companies assigned to a questionnaire
func (r *CompanyQuestionnaireRepository) GetByQuestionnaireID(ctx context.Context, questionnaireID primitive.ObjectID) ([]*models.CompanyQuestionnaire, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"questionnaire_id": questionnaireID})
	if err != nil {
		return nil, fmt.Errorf("failed to get company questionnaires: %w", err)
	}
	defer cursor.Close(ctx)

	var cqs []*models.CompanyQuestionnaire
	if err = cursor.All(ctx, &cqs); err != nil {
		return nil, fmt.Errorf("failed to decode company questionnaires: %w", err)
	}

	return cqs, nil
}

// GetActiveByCompanyAndPeriod retrieves active questionnaires for a company within current period
func (r *CompanyQuestionnaireRepository) GetActiveByCompanyAndPeriod(ctx context.Context, companyID primitive.ObjectID) ([]*models.CompanyQuestionnaire, error) {
	now := time.Now()
	filter := bson.M{
		"company_id": companyID,
		"is_active":  true,
		"period_start": bson.M{"$lte": now},
		"period_end":   bson.M{"$gte": now},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get active company questionnaires: %w", err)
	}
	defer cursor.Close(ctx)

	var cqs []*models.CompanyQuestionnaire
	if err = cursor.All(ctx, &cqs); err != nil {
		return nil, fmt.Errorf("failed to decode company questionnaires: %w", err)
	}

	return cqs, nil
}

// Update updates a company questionnaire
func (r *CompanyQuestionnaireRepository) Update(ctx context.Context, id primitive.ObjectID, cq *models.CompanyQuestionnaire) error {
	update := bson.M{
		"$set": bson.M{
			"period_start": cq.PeriodStart,
			"period_end":   cq.PeriodEnd,
			"is_active":    cq.IsActive,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to update company questionnaire: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("company questionnaire not found")
	}

	return nil
}

// Deactivate deactivates a company questionnaire
func (r *CompanyQuestionnaireRepository) Deactivate(ctx context.Context, id primitive.ObjectID) error {
	update := bson.M{
		"$set": bson.M{
			"is_active": false,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to deactivate company questionnaire: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("company questionnaire not found")
	}

	return nil
}

// Delete deletes a company questionnaire
func (r *CompanyQuestionnaireRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete company questionnaire: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("company questionnaire not found")
	}

	return nil
}

// CheckDuplicate checks if a questionnaire is already assigned to a company for the same period
func (r *CompanyQuestionnaireRepository) CheckDuplicate(ctx context.Context, companyID, questionnaireID primitive.ObjectID, periodStart, periodEnd time.Time) (bool, error) {
	filter := bson.M{
		"company_id":       companyID,
		"questionnaire_id": questionnaireID,
		"is_active":        true,
		"$or": []bson.M{
			{
				"period_start": bson.M{"$lte": periodEnd},
				"period_end":   bson.M{"$gte": periodStart},
			},
		},
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to check duplicate: %w", err)
	}

	return count > 0, nil
}
