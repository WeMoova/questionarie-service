package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ── Point Configuration ─────────────────────────────────────────────────────

// PointEventType represents the type of event that awards points
type PointEventType string

const (
	PointEventQuestionnaireCompleted PointEventType = "questionnaire_completed"
	PointEventStreakBonus            PointEventType = "streak_bonus"
	PointEventFirstCompletion        PointEventType = "first_completion"
	PointEventOnTimeCompletion       PointEventType = "on_time_completion"
	PointEventPerfectScore           PointEventType = "perfect_score"
)

// PointRule defines how points are awarded for a specific event
type PointRule struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	EventType   PointEventType     `bson:"event_type" json:"event_type"`
	Points      int                `bson:"points" json:"points"`
	IsActive    bool               `bson:"is_active" json:"is_active"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// ── User Points Ledger ──────────────────────────────────────────────────────

// UserPoints tracks a user's total points and transaction history
type UserPoints struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserID       string             `bson:"user_id" json:"user_id"`
	CompanyID    primitive.ObjectID `bson:"company_id" json:"company_id"`
	TotalPoints  int                `bson:"total_points" json:"total_points"`
	Level        int                `bson:"level" json:"level"`
	Transactions []PointTransaction `bson:"transactions" json:"transactions,omitempty"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

// PointTransaction records a single point award or deduction
type PointTransaction struct {
	EventType   PointEventType `bson:"event_type" json:"event_type"`
	Points      int            `bson:"points" json:"points"`
	Description string         `bson:"description" json:"description"`
	ReferenceID string         `bson:"reference_id,omitempty" json:"reference_id,omitempty"`
	AwardedAt   time.Time      `bson:"awarded_at" json:"awarded_at"`
}

// CalculateLevel returns the level based on total points (every 100 points = 1 level)
func CalculateLevel(totalPoints int) int {
	if totalPoints <= 0 {
		return 1
	}
	return (totalPoints / 100) + 1
}

// ── Badges ──────────────────────────────────────────────────────────────────

// BadgeCriteriaType defines how a badge is earned
type BadgeCriteriaType string

const (
	BadgeCriteriaCompletions BadgeCriteriaType = "total_completions"
	BadgeCriteriaStreak      BadgeCriteriaType = "streak_days"
	BadgeCriteriaPoints      BadgeCriteriaType = "total_points"
	BadgeCriteriaOnTime      BadgeCriteriaType = "on_time_completions"
)

// BadgeCriteria defines the requirements to earn a badge
type BadgeCriteria struct {
	Type      BadgeCriteriaType `bson:"type" json:"type"`
	Threshold int               `bson:"threshold" json:"threshold"`
}

// Badge defines an earnable badge
type Badge struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name        string             `bson:"name" json:"name" validate:"required"`
	Description string             `bson:"description" json:"description"`
	IconURL     string             `bson:"icon_url,omitempty" json:"icon_url,omitempty"`
	Criteria    BadgeCriteria      `bson:"criteria" json:"criteria"`
	IsActive    bool               `bson:"is_active" json:"is_active"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// UserBadge records a badge earned by a user
type UserBadge struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserID   string             `bson:"user_id" json:"user_id"`
	BadgeID  primitive.ObjectID `bson:"badge_id" json:"badge_id"`
	EarnedAt time.Time          `bson:"earned_at" json:"earned_at"`
	// Transient fields
	BadgeName string `bson:"-" json:"badge_name,omitempty"`
	BadgeIcon string `bson:"-" json:"badge_icon,omitempty"`
	BadgeDesc string `bson:"-" json:"badge_description,omitempty"`
}

// ── Streaks ─────────────────────────────────────────────────────────────────

// UserStreak tracks a user's completion streak
type UserStreak struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserID          string             `bson:"user_id" json:"user_id"`
	CompanyID       primitive.ObjectID `bson:"company_id" json:"company_id"`
	CurrentStreak   int                `bson:"current_streak" json:"current_streak"`
	LongestStreak   int                `bson:"longest_streak" json:"longest_streak"`
	LastActivityAt  time.Time          `bson:"last_activity_at" json:"last_activity_at"`
	StreakStartedAt time.Time          `bson:"streak_started_at" json:"streak_started_at"`
}

// ── Achievements ────────────────────────────────────────────────────────────

// AchievementCategory groups achievements
type AchievementCategory string

const (
	AchievementCategoryCompletions AchievementCategory = "completions"
	AchievementCategoryStreak      AchievementCategory = "streak"
	AchievementCategoryPoints      AchievementCategory = "points"
	AchievementCategorySocial      AchievementCategory = "social"
)

// Achievement represents a milestone achievement
type Achievement struct {
	ID          primitive.ObjectID  `bson:"_id,omitempty" json:"id,omitempty"`
	Name        string              `bson:"name" json:"name" validate:"required"`
	Description string              `bson:"description" json:"description"`
	IconURL     string              `bson:"icon_url,omitempty" json:"icon_url,omitempty"`
	Category    AchievementCategory `bson:"category" json:"category"`
	Threshold   int                 `bson:"threshold" json:"threshold"`
	PointReward int                 `bson:"point_reward" json:"point_reward"`
	IsActive    bool                `bson:"is_active" json:"is_active"`
	CreatedAt   time.Time           `bson:"created_at" json:"created_at"`
}

// UserAchievement records an achievement unlocked by a user
type UserAchievement struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserID          string             `bson:"user_id" json:"user_id"`
	AchievementID   primitive.ObjectID `bson:"achievement_id" json:"achievement_id"`
	UnlockedAt      time.Time          `bson:"unlocked_at" json:"unlocked_at"`
	// Transient fields
	AchievementName string `bson:"-" json:"achievement_name,omitempty"`
	AchievementIcon string `bson:"-" json:"achievement_icon,omitempty"`
}

// ── Leaderboard ─────────────────────────────────────────────────────────────

// LeaderboardEntry represents a single entry in the leaderboard
type LeaderboardEntry struct {
	UserID      string `json:"user_id"`
	TotalPoints int    `json:"total_points"`
	Level       int    `json:"level"`
	Rank        int    `json:"rank"`
}

// UserGamificationProfile contains a user's full gamification state
type UserGamificationProfile struct {
	UserID             string            `json:"user_id"`
	TotalPoints        int               `json:"total_points"`
	Level              int               `json:"level"`
	Rank               int               `json:"rank"`
	CurrentStreak      int               `json:"current_streak"`
	LongestStreak      int               `json:"longest_streak"`
	Badges             []UserBadge       `json:"badges"`
	Achievements       []UserAchievement `json:"achievements"`
	RecentTransactions []PointTransaction `json:"recent_transactions"`
}
