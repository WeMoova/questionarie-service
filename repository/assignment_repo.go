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

// AssignmentRepository handles user questionnaire assignments
type AssignmentRepository struct {
	collection *mongo.Collection
}

// NewAssignmentRepository creates a new AssignmentRepository
func NewAssignmentRepository(db *mongo.Database) *AssignmentRepository {
	return &AssignmentRepository{
		collection: db.Collection("user_questionnaire_assignments"),
	}
}

// Create creates a new assignment
func (r *AssignmentRepository) Create(ctx context.Context, assignment *models.UserQuestionnaireAssignment) error {
	_, err := r.collection.InsertOne(ctx, assignment)
	if err != nil {
		return fmt.Errorf("failed to create assignment: %w", err)
	}
	return nil
}

// GetByID retrieves an assignment by ID
func (r *AssignmentRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.UserQuestionnaireAssignment, error) {
	var assignment models.UserQuestionnaireAssignment
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&assignment)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("assignment not found")
		}
		return nil, fmt.Errorf("failed to get assignment: %w", err)
	}
	return &assignment, nil
}

// GetByUserID retrieves all assignments for a specific user
func (r *AssignmentRepository) GetByUserID(ctx context.Context, userID string, status *models.AssignmentStatus) ([]*models.UserQuestionnaireAssignment, error) {
	filter := bson.M{"user_id": userID}
	if status != nil {
		filter["status"] = *status
	}

	opts := options.Find().SetSort(bson.D{{Key: "assigned_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get user assignments: %w", err)
	}
	defer cursor.Close(ctx)

	var assignments []*models.UserQuestionnaireAssignment
	if err = cursor.All(ctx, &assignments); err != nil {
		return nil, fmt.Errorf("failed to decode assignments: %w", err)
	}

	return assignments, nil
}

// GetByCompanyQuestionnaireID retrieves all assignments for a company questionnaire
func (r *AssignmentRepository) GetByCompanyQuestionnaireID(ctx context.Context, cqID primitive.ObjectID) ([]*models.UserQuestionnaireAssignment, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"company_questionnaire_id": cqID})
	if err != nil {
		return nil, fmt.Errorf("failed to get assignments: %w", err)
	}
	defer cursor.Close(ctx)

	var assignments []*models.UserQuestionnaireAssignment
	if err = cursor.All(ctx, &assignments); err != nil {
		return nil, fmt.Errorf("failed to decode assignments: %w", err)
	}

	return assignments, nil
}

// Update updates an assignment
func (r *AssignmentRepository) Update(ctx context.Context, id primitive.ObjectID, assignment *models.UserQuestionnaireAssignment) error {
	update := bson.M{
		"$set": bson.M{
			"status":       assignment.Status,
			"started_at":   assignment.StartedAt,
			"completed_at": assignment.CompletedAt,
			"responses":    assignment.Responses,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to update assignment: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("assignment not found")
	}

	return nil
}

// UpdateStatus updates only the status of an assignment
func (r *AssignmentRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.AssignmentStatus) error {
	update := bson.M{
		"$set": bson.M{
			"status": status,
		},
	}

	if status == models.AssignmentStatusInProgress {
		update["$set"].(bson.M)["started_at"] = time.Now()
	} else if status == models.AssignmentStatusCompleted {
		update["$set"].(bson.M)["completed_at"] = time.Now()
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to update assignment status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("assignment not found")
	}

	return nil
}

// AddOrUpdateResponse adds or updates a response in an assignment
func (r *AssignmentRepository) AddOrUpdateResponse(ctx context.Context, assignmentID primitive.ObjectID, response models.Response) error {
	// First, try to update existing response
	filter := bson.M{
		"_id":                   assignmentID,
		"responses.question_id": response.QuestionID,
	}

	update := bson.M{
		"$set": bson.M{
			"responses.$": response,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update response: %w", err)
	}

	// If no existing response found, add new one
	if result.MatchedCount == 0 {
		filter = bson.M{"_id": assignmentID}
		update = bson.M{
			"$push": bson.M{"responses": response},
		}

		result, err = r.collection.UpdateOne(ctx, filter, update)
		if err != nil {
			return fmt.Errorf("failed to add response: %w", err)
		}

		if result.MatchedCount == 0 {
			return fmt.Errorf("assignment not found")
		}

		// Auto-start assignment if this is first response
		r.UpdateStatus(ctx, assignmentID, models.AssignmentStatusInProgress)
	}

	return nil
}

// GetCompletionStats retrieves completion statistics for a company questionnaire
func (r *AssignmentRepository) GetCompletionStats(ctx context.Context, cqID primitive.ObjectID) (map[string]int64, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"company_questionnaire_id": cqID}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$status",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get completion stats: %w", err)
	}
	defer cursor.Close(ctx)

	stats := make(map[string]int64)
	stats["pending"] = 0
	stats["in_progress"] = 0
	stats["completed"] = 0

	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode stats: %w", err)
		}
		stats[result.ID] = result.Count
	}

	return stats, nil
}

// GetAverageCompletionTime calculates average time to complete for a company questionnaire
func (r *AssignmentRepository) GetAverageCompletionTime(ctx context.Context, cqID primitive.ObjectID) (float64, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"company_questionnaire_id": cqID,
			"status":                   models.AssignmentStatusCompleted,
			"started_at":               bson.M{"$exists": true},
			"completed_at":             bson.M{"$exists": true},
		}}},
		{{Key: "$project", Value: bson.M{
			"duration": bson.M{
				"$subtract": []interface{}{"$completed_at", "$started_at"},
			},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id": nil,
			"avgDuration": bson.M{"$avg": "$duration"},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate average completion time: %w", err)
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		var result struct {
			AvgDuration float64 `bson:"avgDuration"`
		}
		if err := cursor.Decode(&result); err != nil {
			return 0, fmt.Errorf("failed to decode average time: %w", err)
		}
		// Convert milliseconds to minutes
		return result.AvgDuration / (1000 * 60), nil
	}

	return 0, nil
}

// Delete deletes an assignment
func (r *AssignmentRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete assignment: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("assignment not found")
	}

	return nil
}

// CheckDuplicate checks if a user already has an assignment for a company questionnaire
func (r *AssignmentRepository) CheckDuplicate(ctx context.Context, userID string, cqID primitive.ObjectID) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{
		"user_id":                  userID,
		"company_questionnaire_id": cqID,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check duplicate assignment: %w", err)
	}

	return count > 0, nil
}
