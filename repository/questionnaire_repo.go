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

// QuestionnaireRepository handles questionnaire data operations
type QuestionnaireRepository struct {
	collection *mongo.Collection
}

// NewQuestionnaireRepository creates a new QuestionnaireRepository
func NewQuestionnaireRepository(db *mongo.Database) *QuestionnaireRepository {
	return &QuestionnaireRepository{
		collection: db.Collection("questionnaires"),
	}
}

// Create creates a new questionnaire
func (r *QuestionnaireRepository) Create(ctx context.Context, questionnaire *models.Questionnaire) error {
	_, err := r.collection.InsertOne(ctx, questionnaire)
	if err != nil {
		return fmt.Errorf("failed to create questionnaire: %w", err)
	}
	return nil
}

// GetByID retrieves a questionnaire by ID
func (r *QuestionnaireRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Questionnaire, error) {
	var questionnaire models.Questionnaire
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&questionnaire)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("questionnaire not found")
		}
		return nil, fmt.Errorf("failed to get questionnaire: %w", err)
	}
	return &questionnaire, nil
}

// GetAll retrieves all questionnaires with pagination
func (r *QuestionnaireRepository) GetAll(ctx context.Context, page, pageSize int64, activeOnly bool) ([]*models.Questionnaire, error) {
	skip := (page - 1) * pageSize
	filter := bson.M{}
	if activeOnly {
		filter["is_active"] = true
	}

	opts := options.Find().
		SetSkip(skip).
		SetLimit(pageSize).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get questionnaires: %w", err)
	}
	defer cursor.Close(ctx)

	var questionnaires []*models.Questionnaire
	if err = cursor.All(ctx, &questionnaires); err != nil {
		return nil, fmt.Errorf("failed to decode questionnaires: %w", err)
	}

	return questionnaires, nil
}

// GetByCreator retrieves questionnaires created by a specific user
func (r *QuestionnaireRepository) GetByCreator(ctx context.Context, creatorID string) ([]*models.Questionnaire, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"created_by": creatorID})
	if err != nil {
		return nil, fmt.Errorf("failed to get questionnaires by creator: %w", err)
	}
	defer cursor.Close(ctx)

	var questionnaires []*models.Questionnaire
	if err = cursor.All(ctx, &questionnaires); err != nil {
		return nil, fmt.Errorf("failed to decode questionnaires: %w", err)
	}

	return questionnaires, nil
}

// Update updates a questionnaire
func (r *QuestionnaireRepository) Update(ctx context.Context, id primitive.ObjectID, questionnaire *models.Questionnaire) error {
	questionnaire.UpdatedAt = time.Now()
	update := bson.M{
		"$set": bson.M{
			"title":       questionnaire.Title,
			"description": questionnaire.Description,
			"is_active":   questionnaire.IsActive,
			"questions":   questionnaire.Questions,
			"updated_at":  questionnaire.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to update questionnaire: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("questionnaire not found")
	}

	return nil
}

// Deactivate deactivates a questionnaire (soft delete)
func (r *QuestionnaireRepository) Deactivate(ctx context.Context, id primitive.ObjectID) error {
	update := bson.M{
		"$set": bson.M{
			"is_active":  false,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to deactivate questionnaire: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("questionnaire not found")
	}

	return nil
}

// AddQuestion adds a question to a questionnaire
func (r *QuestionnaireRepository) AddQuestion(ctx context.Context, id primitive.ObjectID, question models.Question) error {
	update := bson.M{
		"$push": bson.M{"questions": question},
		"$set":  bson.M{"updated_at": time.Now()},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to add question: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("questionnaire not found")
	}

	return nil
}

// UpdateQuestion updates a specific question within a questionnaire
func (r *QuestionnaireRepository) UpdateQuestion(ctx context.Context, questionnaireID primitive.ObjectID, questionID string, question models.Question) error {
	filter := bson.M{
		"_id":                   questionnaireID,
		"questions.question_id": questionID,
	}

	update := bson.M{
		"$set": bson.M{
			"questions.$":  question,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update question: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("questionnaire or question not found")
	}

	return nil
}

// RemoveQuestion removes a question from a questionnaire
func (r *QuestionnaireRepository) RemoveQuestion(ctx context.Context, questionnaireID primitive.ObjectID, questionID string) error {
	filter := bson.M{"_id": questionnaireID}
	update := bson.M{
		"$pull": bson.M{"questions": bson.M{"question_id": questionID}},
		"$set":  bson.M{"updated_at": time.Now()},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to remove question: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("questionnaire not found")
	}

	return nil
}

// Count returns the total number of questionnaires
func (r *QuestionnaireRepository) Count(ctx context.Context, activeOnly bool) (int64, error) {
	filter := bson.M{}
	if activeOnly {
		filter["is_active"] = true
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count questionnaires: %w", err)
	}
	return count, nil
}
