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

// GamificationRepository handles all gamification data operations
type GamificationRepository struct {
	pointRules       *mongo.Collection
	userPoints       *mongo.Collection
	badges           *mongo.Collection
	userBadges       *mongo.Collection
	userStreaks       *mongo.Collection
	achievements     *mongo.Collection
	userAchievements *mongo.Collection
}

// NewGamificationRepository creates a new GamificationRepository
func NewGamificationRepository(db *mongo.Database) *GamificationRepository {
	return &GamificationRepository{
		pointRules:       db.Collection("point_rules"),
		userPoints:       db.Collection("user_points"),
		badges:           db.Collection("badges"),
		userBadges:       db.Collection("user_badges"),
		userStreaks:       db.Collection("user_streaks"),
		achievements:     db.Collection("achievements"),
		userAchievements: db.Collection("user_achievements"),
	}
}

// ── Point Rules ─────────────────────────────────────────────────────────────

func (r *GamificationRepository) CreatePointRule(ctx context.Context, rule *models.PointRule) error {
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	_, err := r.pointRules.InsertOne(ctx, rule)
	return err
}

func (r *GamificationRepository) GetPointRules(ctx context.Context) ([]*models.PointRule, error) {
	cursor, err := r.pointRules.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rules []*models.PointRule
	if err := cursor.All(ctx, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (r *GamificationRepository) GetActivePointRules(ctx context.Context) ([]*models.PointRule, error) {
	cursor, err := r.pointRules.Find(ctx, bson.M{"is_active": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rules []*models.PointRule
	if err := cursor.All(ctx, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (r *GamificationRepository) UpdatePointRule(ctx context.Context, id primitive.ObjectID, rule *models.PointRule) error {
	rule.UpdatedAt = time.Now()
	result, err := r.pointRules.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"name":        rule.Name,
			"description": rule.Description,
			"points":      rule.Points,
			"is_active":   rule.IsActive,
			"updated_at":  rule.UpdatedAt,
		},
	})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("point rule not found")
	}
	return nil
}

// ── User Points ─────────────────────────────────────────────────────────────

func (r *GamificationRepository) GetUserPoints(ctx context.Context, userID string) (*models.UserPoints, error) {
	var up models.UserPoints
	err := r.userPoints.FindOne(ctx, bson.M{"user_id": userID}).Decode(&up)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &up, nil
}

func (r *GamificationRepository) IncrementPoints(ctx context.Context, userID string, companyID primitive.ObjectID, transaction models.PointTransaction) error {
	filter := bson.M{"user_id": userID}
	newLevel := models.CalculateLevel(transaction.Points) // will be recalculated

	update := bson.M{
		"$inc": bson.M{"total_points": transaction.Points},
		"$push": bson.M{
			"transactions": bson.M{
				"$each":  []models.PointTransaction{transaction},
				"$slice": -1000, // keep last 1000 transactions
			},
		},
		"$set": bson.M{
			"updated_at": time.Now(),
			"company_id": companyID,
		},
		"$setOnInsert": bson.M{
			"user_id":    userID,
			"created_at": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.userPoints.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}

	// Recalculate level based on actual total
	up, err := r.GetUserPoints(ctx, userID)
	if err != nil || up == nil {
		return err
	}
	newLevel = models.CalculateLevel(up.TotalPoints)
	if newLevel != up.Level {
		_, err = r.userPoints.UpdateOne(ctx, filter, bson.M{"$set": bson.M{"level": newLevel}})
	}
	return err
}

func (r *GamificationRepository) GetLeaderboard(ctx context.Context, companyID *primitive.ObjectID, limit int) ([]models.LeaderboardEntry, error) {
	filter := bson.M{}
	if companyID != nil {
		filter["company_id"] = *companyID
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "total_points", Value: -1}}).
		SetLimit(int64(limit)).
		SetProjection(bson.M{"user_id": 1, "total_points": 1, "level": 1})

	cursor, err := r.userPoints.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var entries []models.LeaderboardEntry
	rank := 1
	for cursor.Next(ctx) {
		var up models.UserPoints
		if err := cursor.Decode(&up); err != nil {
			continue
		}
		entries = append(entries, models.LeaderboardEntry{
			UserID:      up.UserID,
			TotalPoints: up.TotalPoints,
			Level:       up.Level,
			Rank:        rank,
		})
		rank++
	}
	return entries, nil
}

func (r *GamificationRepository) GetUserRank(ctx context.Context, userID string, companyID *primitive.ObjectID) (int, error) {
	up, err := r.GetUserPoints(ctx, userID)
	if err != nil || up == nil {
		return 0, err
	}

	filter := bson.M{"total_points": bson.M{"$gt": up.TotalPoints}}
	if companyID != nil {
		filter["company_id"] = *companyID
	}

	count, err := r.userPoints.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	return int(count) + 1, nil
}

// GetCompletionCount returns total completed questionnaires for a user
func (r *GamificationRepository) GetCompletionCount(ctx context.Context, userID string) (int, error) {
	up, err := r.GetUserPoints(ctx, userID)
	if err != nil || up == nil {
		return 0, err
	}
	count := 0
	for _, t := range up.Transactions {
		if t.EventType == models.PointEventQuestionnaireCompleted {
			count++
		}
	}
	return count, nil
}

// GetOnTimeCompletionCount returns on-time completions for a user
func (r *GamificationRepository) GetOnTimeCompletionCount(ctx context.Context, userID string) (int, error) {
	up, err := r.GetUserPoints(ctx, userID)
	if err != nil || up == nil {
		return 0, err
	}
	count := 0
	for _, t := range up.Transactions {
		if t.EventType == models.PointEventOnTimeCompletion {
			count++
		}
	}
	return count, nil
}

// ── Badges ──────────────────────────────────────────────────────────────────

func (r *GamificationRepository) CreateBadge(ctx context.Context, badge *models.Badge) error {
	badge.CreatedAt = time.Now()
	badge.UpdatedAt = time.Now()
	_, err := r.badges.InsertOne(ctx, badge)
	return err
}

func (r *GamificationRepository) GetBadges(ctx context.Context) ([]*models.Badge, error) {
	cursor, err := r.badges.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var badges []*models.Badge
	if err := cursor.All(ctx, &badges); err != nil {
		return nil, err
	}
	return badges, nil
}

func (r *GamificationRepository) GetActiveBadges(ctx context.Context) ([]*models.Badge, error) {
	cursor, err := r.badges.Find(ctx, bson.M{"is_active": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var badges []*models.Badge
	if err := cursor.All(ctx, &badges); err != nil {
		return nil, err
	}
	return badges, nil
}

func (r *GamificationRepository) GetBadgeByID(ctx context.Context, id primitive.ObjectID) (*models.Badge, error) {
	var badge models.Badge
	err := r.badges.FindOne(ctx, bson.M{"_id": id}).Decode(&badge)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("badge not found")
		}
		return nil, err
	}
	return &badge, nil
}

func (r *GamificationRepository) UpdateBadge(ctx context.Context, id primitive.ObjectID, badge *models.Badge) error {
	badge.UpdatedAt = time.Now()
	result, err := r.badges.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"name":        badge.Name,
			"description": badge.Description,
			"icon_url":    badge.IconURL,
			"criteria":    badge.Criteria,
			"is_active":   badge.IsActive,
			"updated_at":  badge.UpdatedAt,
		},
	})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("badge not found")
	}
	return nil
}

// ── User Badges ─────────────────────────────────────────────────────────────

func (r *GamificationRepository) GetUserBadges(ctx context.Context, userID string) ([]models.UserBadge, error) {
	cursor, err := r.userBadges.Find(ctx, bson.M{"user_id": userID}, options.Find().SetSort(bson.D{{Key: "earned_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var badges []models.UserBadge
	if err := cursor.All(ctx, &badges); err != nil {
		return nil, err
	}
	return badges, nil
}

func (r *GamificationRepository) HasBadge(ctx context.Context, userID string, badgeID primitive.ObjectID) (bool, error) {
	count, err := r.userBadges.CountDocuments(ctx, bson.M{"user_id": userID, "badge_id": badgeID})
	return count > 0, err
}

func (r *GamificationRepository) AwardBadge(ctx context.Context, userID string, badgeID primitive.ObjectID) error {
	_, err := r.userBadges.InsertOne(ctx, models.UserBadge{
		ID:       primitive.NewObjectID(),
		UserID:   userID,
		BadgeID:  badgeID,
		EarnedAt: time.Now(),
	})
	return err
}

// ── Streaks ─────────────────────────────────────────────────────────────────

func (r *GamificationRepository) GetUserStreak(ctx context.Context, userID string) (*models.UserStreak, error) {
	var streak models.UserStreak
	err := r.userStreaks.FindOne(ctx, bson.M{"user_id": userID}).Decode(&streak)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &streak, nil
}

func (r *GamificationRepository) UpdateStreak(ctx context.Context, userID string, companyID primitive.ObjectID, currentStreak, longestStreak int) error {
	now := time.Now()
	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$set": bson.M{
			"current_streak":   currentStreak,
			"longest_streak":   longestStreak,
			"last_activity_at": now,
			"company_id":       companyID,
		},
		"$setOnInsert": bson.M{
			"user_id":          userID,
			"streak_started_at": now,
		},
	}

	if currentStreak == 1 {
		update["$set"].(bson.M)["streak_started_at"] = now
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.userStreaks.UpdateOne(ctx, filter, update, opts)
	return err
}

// ── Achievements ────────────────────────────────────────────────────────────

func (r *GamificationRepository) CreateAchievement(ctx context.Context, achievement *models.Achievement) error {
	achievement.CreatedAt = time.Now()
	_, err := r.achievements.InsertOne(ctx, achievement)
	return err
}

func (r *GamificationRepository) GetAchievements(ctx context.Context) ([]*models.Achievement, error) {
	cursor, err := r.achievements.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var achievements []*models.Achievement
	if err := cursor.All(ctx, &achievements); err != nil {
		return nil, err
	}
	return achievements, nil
}

func (r *GamificationRepository) GetActiveAchievements(ctx context.Context) ([]*models.Achievement, error) {
	cursor, err := r.achievements.Find(ctx, bson.M{"is_active": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var achievements []*models.Achievement
	if err := cursor.All(ctx, &achievements); err != nil {
		return nil, err
	}
	return achievements, nil
}

func (r *GamificationRepository) UpdateAchievement(ctx context.Context, id primitive.ObjectID, achievement *models.Achievement) error {
	result, err := r.achievements.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"name":         achievement.Name,
			"description":  achievement.Description,
			"icon_url":     achievement.IconURL,
			"category":     achievement.Category,
			"threshold":    achievement.Threshold,
			"point_reward": achievement.PointReward,
			"is_active":    achievement.IsActive,
		},
	})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("achievement not found")
	}
	return nil
}

// ── User Achievements ───────────────────────────────────────────────────────

func (r *GamificationRepository) GetUserAchievements(ctx context.Context, userID string) ([]models.UserAchievement, error) {
	cursor, err := r.userAchievements.Find(ctx, bson.M{"user_id": userID}, options.Find().SetSort(bson.D{{Key: "unlocked_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var achievements []models.UserAchievement
	if err := cursor.All(ctx, &achievements); err != nil {
		return nil, err
	}
	return achievements, nil
}

func (r *GamificationRepository) HasAchievement(ctx context.Context, userID string, achievementID primitive.ObjectID) (bool, error) {
	count, err := r.userAchievements.CountDocuments(ctx, bson.M{"user_id": userID, "achievement_id": achievementID})
	return count > 0, err
}

func (r *GamificationRepository) AwardAchievement(ctx context.Context, userID string, achievementID primitive.ObjectID) error {
	_, err := r.userAchievements.InsertOne(ctx, models.UserAchievement{
		ID:            primitive.NewObjectID(),
		UserID:        userID,
		AchievementID: achievementID,
		UnlockedAt:    time.Now(),
	})
	return err
}

// ── Seed Data ───────────────────────────────────────────────────────────────

// SeedDefaultData creates default point rules, badges, and achievements if they don't exist
func (r *GamificationRepository) SeedDefaultData(ctx context.Context) error {
	// Seed point rules
	count, _ := r.pointRules.CountDocuments(ctx, bson.M{})
	if count == 0 {
		rules := []interface{}{
			&models.PointRule{ID: primitive.NewObjectID(), Name: "Cuestionario completado", Description: "Puntos por completar un cuestionario", EventType: models.PointEventQuestionnaireCompleted, Points: 100, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			&models.PointRule{ID: primitive.NewObjectID(), Name: "Primera vez", Description: "Bonus por completar tu primer cuestionario", EventType: models.PointEventFirstCompletion, Points: 50, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			&models.PointRule{ID: primitive.NewObjectID(), Name: "A tiempo", Description: "Bonus por completar antes de la fecha límite", EventType: models.PointEventOnTimeCompletion, Points: 25, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			&models.PointRule{ID: primitive.NewObjectID(), Name: "Racha semanal", Description: "Bonus por mantener racha de 7 días", EventType: models.PointEventStreakBonus, Points: 50, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}
		r.pointRules.InsertMany(ctx, rules)
	}

	// Seed badges
	count, _ = r.badges.CountDocuments(ctx, bson.M{})
	if count == 0 {
		badges := []interface{}{
			&models.Badge{ID: primitive.NewObjectID(), Name: "Primer Paso", Description: "Completaste tu primer cuestionario", Criteria: models.BadgeCriteria{Type: models.BadgeCriteriaCompletions, Threshold: 1}, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			&models.Badge{ID: primitive.NewObjectID(), Name: "Constante", Description: "Completaste 5 cuestionarios", Criteria: models.BadgeCriteria{Type: models.BadgeCriteriaCompletions, Threshold: 5}, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			&models.Badge{ID: primitive.NewObjectID(), Name: "Comprometido", Description: "Completaste 10 cuestionarios", Criteria: models.BadgeCriteria{Type: models.BadgeCriteriaCompletions, Threshold: 10}, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			&models.Badge{ID: primitive.NewObjectID(), Name: "Racha de 7", Description: "Mantuviste una racha de 7 días consecutivos", Criteria: models.BadgeCriteria{Type: models.BadgeCriteriaStreak, Threshold: 7}, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			&models.Badge{ID: primitive.NewObjectID(), Name: "Racha de 30", Description: "Mantuviste una racha de 30 días consecutivos", Criteria: models.BadgeCriteria{Type: models.BadgeCriteriaStreak, Threshold: 30}, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			&models.Badge{ID: primitive.NewObjectID(), Name: "Centurion", Description: "Acumulaste 100 puntos", Criteria: models.BadgeCriteria{Type: models.BadgeCriteriaPoints, Threshold: 100}, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			&models.Badge{ID: primitive.NewObjectID(), Name: "Maestro", Description: "Acumulaste 500 puntos", Criteria: models.BadgeCriteria{Type: models.BadgeCriteriaPoints, Threshold: 500}, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}
		r.badges.InsertMany(ctx, badges)
	}

	return nil
}

// ── Company Cleanup ─────────────────────────────────────────────────────────

// DeleteUserPointsByCompanyID deletes all user_points for a company
func (r *GamificationRepository) DeleteUserPointsByCompanyID(ctx context.Context, companyID primitive.ObjectID) (int64, error) {
	result, err := r.userPoints.DeleteMany(ctx, bson.M{"company_id": companyID})
	if err != nil {
		return 0, fmt.Errorf("failed to delete user points: %w", err)
	}
	return result.DeletedCount, nil
}

// DeleteUserBadgesByCompanyID deletes all user_badges for a company
func (r *GamificationRepository) DeleteUserBadgesByCompanyID(ctx context.Context, companyID primitive.ObjectID) (int64, error) {
	result, err := r.userBadges.DeleteMany(ctx, bson.M{"company_id": companyID})
	if err != nil {
		return 0, fmt.Errorf("failed to delete user badges: %w", err)
	}
	return result.DeletedCount, nil
}

// DeleteUserStreaksByCompanyID deletes all user_streaks for a company
func (r *GamificationRepository) DeleteUserStreaksByCompanyID(ctx context.Context, companyID primitive.ObjectID) (int64, error) {
	result, err := r.userStreaks.DeleteMany(ctx, bson.M{"company_id": companyID})
	if err != nil {
		return 0, fmt.Errorf("failed to delete user streaks: %w", err)
	}
	return result.DeletedCount, nil
}
