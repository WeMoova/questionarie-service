package services

import (
	"context"
	"log"
	"questionarie-service/models"
	"questionarie-service/repository"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GamificationService handles gamification business logic
type GamificationService struct {
	repo             *repository.GamificationRepository
	userMetadataRepo *repository.UserMetadataRepository
}

// NewGamificationService creates a new GamificationService
func NewGamificationService(
	repo *repository.GamificationRepository,
	userMetadataRepo *repository.UserMetadataRepository,
) *GamificationService {
	return &GamificationService{
		repo:             repo,
		userMetadataRepo: userMetadataRepo,
	}
}

// AwardPointsForCompletion awards points when a user completes a questionnaire
func (s *GamificationService) AwardPointsForCompletion(ctx context.Context, userID, assignmentID string) {
	// Get user metadata for company ID
	userMeta, err := s.userMetadataRepo.GetByID(ctx, userID)
	if err != nil {
		log.Printf("gamification: failed to get user metadata for %s: %v", userID, err)
		return
	}

	rules, err := s.repo.GetActivePointRules(ctx)
	if err != nil {
		log.Printf("gamification: failed to get point rules: %v", err)
		return
	}

	for _, rule := range rules {
		switch rule.EventType {
		case models.PointEventQuestionnaireCompleted:
			s.awardPoints(ctx, userID, userMeta.CompanyID, rule, assignmentID)

		case models.PointEventFirstCompletion:
			count, _ := s.repo.GetCompletionCount(ctx, userID)
			if count == 0 { // this is their first
				s.awardPoints(ctx, userID, userMeta.CompanyID, rule, assignmentID)
			}

		case models.PointEventOnTimeCompletion:
			// Always award on-time bonus (assignment validation already ensures it's within period)
			s.awardPoints(ctx, userID, userMeta.CompanyID, rule, assignmentID)
		}
	}
}

func (s *GamificationService) awardPoints(ctx context.Context, userID string, companyID primitive.ObjectID, rule *models.PointRule, referenceID string) {
	transaction := models.PointTransaction{
		EventType:   rule.EventType,
		Points:      rule.Points,
		Description: rule.Name,
		ReferenceID: referenceID,
		AwardedAt:   time.Now(),
	}

	if err := s.repo.IncrementPoints(ctx, userID, companyID, transaction); err != nil {
		log.Printf("gamification: failed to award points to %s: %v", userID, err)
	}
}

// UpdateStreak updates the user's streak after completing a questionnaire
func (s *GamificationService) UpdateStreak(ctx context.Context, userID string) {
	userMeta, err := s.userMetadataRepo.GetByID(ctx, userID)
	if err != nil {
		log.Printf("gamification: failed to get user metadata for streak: %v", err)
		return
	}

	streak, err := s.repo.GetUserStreak(ctx, userID)
	if err != nil {
		log.Printf("gamification: failed to get user streak: %v", err)
		return
	}

	now := time.Now()
	currentStreak := 1
	longestStreak := 1

	if streak != nil {
		lastActivity := streak.LastActivityAt
		daysSince := int(now.Sub(lastActivity).Hours() / 24)

		if daysSince == 0 {
			// Already counted today
			return
		} else if daysSince == 1 {
			// Consecutive day
			currentStreak = streak.CurrentStreak + 1
		}
		// daysSince > 1: streak broken, reset to 1

		longestStreak = streak.LongestStreak
		if currentStreak > longestStreak {
			longestStreak = currentStreak
		}
	}

	if err := s.repo.UpdateStreak(ctx, userID, userMeta.CompanyID, currentStreak, longestStreak); err != nil {
		log.Printf("gamification: failed to update streak: %v", err)
		return
	}

	// Check for streak bonus (every 7 days)
	if currentStreak > 0 && currentStreak%7 == 0 {
		rules, _ := s.repo.GetActivePointRules(ctx)
		for _, rule := range rules {
			if rule.EventType == models.PointEventStreakBonus {
				s.awardPoints(ctx, userID, userMeta.CompanyID, rule, "")
				break
			}
		}
	}
}

// CheckAndAwardBadges evaluates all badges for a user
func (s *GamificationService) CheckAndAwardBadges(ctx context.Context, userID string) {
	badges, err := s.repo.GetActiveBadges(ctx)
	if err != nil {
		log.Printf("gamification: failed to get badges: %v", err)
		return
	}

	for _, badge := range badges {
		has, _ := s.repo.HasBadge(ctx, userID, badge.ID)
		if has {
			continue
		}

		earned := false
		switch badge.Criteria.Type {
		case models.BadgeCriteriaCompletions:
			count, _ := s.repo.GetCompletionCount(ctx, userID)
			earned = count >= badge.Criteria.Threshold

		case models.BadgeCriteriaStreak:
			streak, _ := s.repo.GetUserStreak(ctx, userID)
			if streak != nil {
				earned = streak.LongestStreak >= badge.Criteria.Threshold
			}

		case models.BadgeCriteriaPoints:
			up, _ := s.repo.GetUserPoints(ctx, userID)
			if up != nil {
				earned = up.TotalPoints >= badge.Criteria.Threshold
			}

		case models.BadgeCriteriaOnTime:
			count, _ := s.repo.GetOnTimeCompletionCount(ctx, userID)
			earned = count >= badge.Criteria.Threshold
		}

		if earned {
			if err := s.repo.AwardBadge(ctx, userID, badge.ID); err != nil {
				log.Printf("gamification: failed to award badge %s to %s: %v", badge.Name, userID, err)
			}
		}
	}
}

// CheckAndAwardAchievements evaluates all achievements for a user
func (s *GamificationService) CheckAndAwardAchievements(ctx context.Context, userID string) {
	achievements, err := s.repo.GetActiveAchievements(ctx)
	if err != nil {
		log.Printf("gamification: failed to get achievements: %v", err)
		return
	}

	userMeta, err := s.userMetadataRepo.GetByID(ctx, userID)
	if err != nil {
		return
	}

	for _, achievement := range achievements {
		has, _ := s.repo.HasAchievement(ctx, userID, achievement.ID)
		if has {
			continue
		}

		earned := false
		switch achievement.Category {
		case models.AchievementCategoryCompletions:
			count, _ := s.repo.GetCompletionCount(ctx, userID)
			earned = count >= achievement.Threshold

		case models.AchievementCategoryStreak:
			streak, _ := s.repo.GetUserStreak(ctx, userID)
			if streak != nil {
				earned = streak.LongestStreak >= achievement.Threshold
			}

		case models.AchievementCategoryPoints:
			up, _ := s.repo.GetUserPoints(ctx, userID)
			if up != nil {
				earned = up.TotalPoints >= achievement.Threshold
			}
		}

		if earned {
			if err := s.repo.AwardAchievement(ctx, userID, achievement.ID); err != nil {
				log.Printf("gamification: failed to award achievement %s to %s: %v", achievement.Name, userID, err)
				continue
			}

			// Award bonus points for achievement
			if achievement.PointReward > 0 {
				transaction := models.PointTransaction{
					EventType:   models.PointEventPerfectScore,
					Points:      achievement.PointReward,
					Description: "Logro: " + achievement.Name,
					AwardedAt:   time.Now(),
				}
				s.repo.IncrementPoints(ctx, userID, userMeta.CompanyID, transaction)
			}
		}
	}
}

// ── Read Operations (for handlers) ──────────────────────────────────────────

func (s *GamificationService) GetPointRules(ctx context.Context) ([]*models.PointRule, error) {
	return s.repo.GetPointRules(ctx)
}

func (s *GamificationService) UpdatePointRule(ctx context.Context, id primitive.ObjectID, rule *models.PointRule) error {
	return s.repo.UpdatePointRule(ctx, id, rule)
}

func (s *GamificationService) GetBadges(ctx context.Context) ([]*models.Badge, error) {
	return s.repo.GetBadges(ctx)
}

func (s *GamificationService) CreateBadge(ctx context.Context, badge *models.Badge) error {
	return s.repo.CreateBadge(ctx, badge)
}

func (s *GamificationService) UpdateBadge(ctx context.Context, id primitive.ObjectID, badge *models.Badge) error {
	return s.repo.UpdateBadge(ctx, id, badge)
}

func (s *GamificationService) GetAchievements(ctx context.Context) ([]*models.Achievement, error) {
	return s.repo.GetAchievements(ctx)
}

func (s *GamificationService) CreateAchievement(ctx context.Context, achievement *models.Achievement) error {
	return s.repo.CreateAchievement(ctx, achievement)
}

func (s *GamificationService) UpdateAchievement(ctx context.Context, id primitive.ObjectID, achievement *models.Achievement) error {
	return s.repo.UpdateAchievement(ctx, id, achievement)
}

func (s *GamificationService) GetLeaderboard(ctx context.Context, companyID *primitive.ObjectID, limit int) ([]models.LeaderboardEntry, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.GetLeaderboard(ctx, companyID, limit)
}

func (s *GamificationService) GetUserProfile(ctx context.Context, userID string) (*models.UserGamificationProfile, error) {
	up, err := s.repo.GetUserPoints(ctx, userID)
	if err != nil {
		return nil, err
	}

	profile := &models.UserGamificationProfile{
		UserID: userID,
	}

	if up != nil {
		profile.TotalPoints = up.TotalPoints
		profile.Level = up.Level
		// Get last 10 transactions
		if len(up.Transactions) > 10 {
			profile.RecentTransactions = up.Transactions[len(up.Transactions)-10:]
		} else {
			profile.RecentTransactions = up.Transactions
		}
	}

	rank, _ := s.repo.GetUserRank(ctx, userID, nil)
	profile.Rank = rank

	streak, _ := s.repo.GetUserStreak(ctx, userID)
	if streak != nil {
		profile.CurrentStreak = streak.CurrentStreak
		profile.LongestStreak = streak.LongestStreak
	}

	// Enrich badges with badge details
	userBadges, _ := s.repo.GetUserBadges(ctx, userID)
	for i, ub := range userBadges {
		badge, err := s.repo.GetBadgeByID(ctx, ub.BadgeID)
		if err == nil {
			userBadges[i].BadgeName = badge.Name
			userBadges[i].BadgeIcon = badge.IconURL
			userBadges[i].BadgeDesc = badge.Description
		}
	}
	profile.Badges = userBadges

	// Enrich achievements
	userAchievements, _ := s.repo.GetUserAchievements(ctx, userID)
	profile.Achievements = userAchievements

	return profile, nil
}

// SeedDefaults seeds default gamification data
func (s *GamificationService) SeedDefaults(ctx context.Context) error {
	return s.repo.SeedDefaultData(ctx)
}
