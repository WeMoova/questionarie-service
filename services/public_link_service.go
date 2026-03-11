package services

import (
	"context"
	"fmt"
	"math"
	"questionarie-service/middleware"
	"questionarie-service/models"
	"questionarie-service/repository"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// slugRegex is declared in cloudflare_service.go (same package).

// PublicLinkService handles business logic for public links and the anonymous questionnaire flow.
type PublicLinkService struct {
	linkRepo          *repository.PublicLinkRepository
	assignmentRepo    *repository.AssignmentRepository
	cqRepo            *repository.CompanyQuestionnaireRepository
	questionnaireRepo *repository.QuestionnaireRepository
	companyRepo       *repository.CompanyRepository
}

// NewPublicLinkService creates a new PublicLinkService.
func NewPublicLinkService(
	linkRepo *repository.PublicLinkRepository,
	assignmentRepo *repository.AssignmentRepository,
	cqRepo *repository.CompanyQuestionnaireRepository,
	questionnaireRepo *repository.QuestionnaireRepository,
	companyRepo *repository.CompanyRepository,
) *PublicLinkService {
	return &PublicLinkService{
		linkRepo:          linkRepo,
		assignmentRepo:    assignmentRepo,
		cqRepo:            cqRepo,
		questionnaireRepo: questionnaireRepo,
		companyRepo:       companyRepo,
	}
}

// ---------------------------------------------------------------------------
// CRUD
// ---------------------------------------------------------------------------

// ValidateSlug checks that a slug meets format requirements and is unique.
// If excludeID is non-zero, that link is excluded from the uniqueness check
// (used during updates).
func (s *PublicLinkService) ValidateSlug(ctx context.Context, slug string, excludeID primitive.ObjectID) error {
	if len(slug) < 3 || len(slug) > 64 {
		return fmt.Errorf("slug must be between 3 and 64 characters")
	}
	if !slugRegex.MatchString(slug) {
		return fmt.Errorf("slug must contain only lowercase letters, numbers, and hyphens, and must start and end with a letter or number")
	}

	exists, err := s.linkRepo.SlugExists(ctx, slug)
	if err != nil {
		return fmt.Errorf("failed to check slug uniqueness: %w", err)
	}
	if exists {
		// If we are updating an existing link, the current slug is OK.
		if !excludeID.IsZero() {
			existing, err := s.linkRepo.GetBySlug(ctx, slug)
			if err == nil && existing.ID == excludeID {
				return nil
			}
		}
		return fmt.Errorf("slug '%s' is already in use", slug)
	}

	return nil
}

// CreateLink validates and persists a new public link.
func (s *PublicLinkService) CreateLink(ctx context.Context, cqID primitive.ObjectID, link *models.PublicLink) error {
	// Validate slug
	if err := s.ValidateSlug(ctx, link.Slug, primitive.NilObjectID); err != nil {
		return err
	}

	// Verify the company questionnaire exists
	cq, err := s.cqRepo.GetByID(ctx, cqID)
	if err != nil {
		return fmt.Errorf("company questionnaire not found: %w", err)
	}

	link.CompanyQuestionnaireID = cqID
	link.CompanyID = cq.CompanyID
	link.ResponseCount = 0

	return s.linkRepo.Create(ctx, link)
}

// GetLinksByCQ returns all public links for a company questionnaire.
func (s *PublicLinkService) GetLinksByCQ(ctx context.Context, cqID primitive.ObjectID) ([]models.PublicLink, error) {
	return s.linkRepo.GetByCompanyQuestionnaire(ctx, cqID)
}

// UpdateLink applies a partial update to a public link.
func (s *PublicLinkService) UpdateLink(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	// If slug is being changed, validate it.
	if slug, ok := update["slug"]; ok {
		if err := s.ValidateSlug(ctx, slug.(string), id); err != nil {
			return err
		}
	}
	return s.linkRepo.Update(ctx, id, update)
}

// DeleteLink removes a public link.
func (s *PublicLinkService) DeleteLink(ctx context.Context, id primitive.ObjectID) error {
	return s.linkRepo.Delete(ctx, id)
}

// ---------------------------------------------------------------------------
// Anonymous flow
// ---------------------------------------------------------------------------

// GetPublicLinkInfo returns public-facing metadata for a link identified by slug.
func (s *PublicLinkService) GetPublicLinkInfo(ctx context.Context, slug string) (map[string]any, error) {
	link, err := s.linkRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("link not found: %w", err)
	}

	if !link.IsActive {
		return nil, fmt.Errorf("this link is no longer active")
	}

	cq, err := s.cqRepo.GetByID(ctx, link.CompanyQuestionnaireID)
	if err != nil {
		return nil, fmt.Errorf("company questionnaire not found: %w", err)
	}

	q, err := s.questionnaireRepo.GetByID(ctx, cq.QuestionnaireID)
	if err != nil {
		return nil, fmt.Errorf("questionnaire not found: %w", err)
	}

	company, err := s.companyRepo.GetByID(ctx, cq.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("company not found: %w", err)
	}

	displayMode := string(cq.DisplayMode)
	if displayMode == "" {
		displayMode = "step_by_step"
	}

	result := map[string]any{
		"link_id":                   link.ID,
		"slug":                      link.Slug,
		"questionnaire_title":       q.Title,
		"questionnaire_description": q.Description,
		"demographic_fields":        link.DemographicFields,
		"display_mode":              displayMode,
		"show_instructions":         cq.ShowInstructions,
		"company_name":              company.Name,
	}

	if company.Branding != nil {
		result["branding"] = company.Branding
	}

	return result, nil
}

// GetPublicQuestions returns the questions and sections for the questionnaire
// behind the given slug.
func (s *PublicLinkService) GetPublicQuestions(ctx context.Context, slug string) (map[string]any, error) {
	link, err := s.linkRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("link not found: %w", err)
	}

	if !link.IsActive {
		return nil, fmt.Errorf("this link is no longer active")
	}

	cq, err := s.cqRepo.GetByID(ctx, link.CompanyQuestionnaireID)
	if err != nil {
		return nil, fmt.Errorf("company questionnaire not found: %w", err)
	}

	q, err := s.questionnaireRepo.GetByID(ctx, cq.QuestionnaireID)
	if err != nil {
		return nil, fmt.Errorf("questionnaire not found: %w", err)
	}

	result := map[string]any{
		"questions": q.Questions,
	}
	if len(q.Sections) > 0 {
		result["sections"] = q.Sections
	}

	return result, nil
}

// StartAnonymousSession validates demographic data, creates an anonymous
// assignment, and returns a session token.
func (s *PublicLinkService) StartAnonymousSession(ctx context.Context, slug string, demographicData map[string]string) (map[string]any, error) {
	link, err := s.linkRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("link not found: %w", err)
	}

	if !link.IsActive {
		return nil, fmt.Errorf("this link is no longer active")
	}

	// Validate required demographic fields.
	for _, field := range link.DemographicFields {
		if field.Required {
			val, ok := demographicData[field.Key]
			if !ok || val == "" {
				return nil, fmt.Errorf("required demographic field '%s' is missing", field.Label)
			}
		}
	}

	// Create an anonymous assignment.
	now := time.Now()
	assignment := &models.UserQuestionnaireAssignment{
		ID:                     primitive.NewObjectID(),
		CompanyQuestionnaireID: link.CompanyQuestionnaireID,
		UserID:                 "anonymous",
		AssignedBy:             "public_link",
		AssignedAt:             now,
		Status:                 models.AssignmentStatusInProgress,
		StartedAt:              &now,
		Responses:              []models.Response{},
		IsAnonymous:            true,
		PublicLinkID:           link.ID,
		DemographicData:        demographicData,
	}

	if err := s.assignmentRepo.Create(ctx, assignment); err != nil {
		return nil, fmt.Errorf("failed to create anonymous assignment: %w", err)
	}

	// Generate session token.
	token, err := middleware.GenerateSessionToken(assignment.ID, link.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}

	return map[string]any{
		"assignment_id": assignment.ID,
		"session_token": token,
	}, nil
}

// SaveAnonymousResponses merges the provided responses into the anonymous
// assignment identified by the session claims.
func (s *PublicLinkService) SaveAnonymousResponses(ctx context.Context, claims *middleware.SessionClaims, responses []models.Response) error {
	assignmentID, err := primitive.ObjectIDFromHex(claims.AssignmentID)
	if err != nil {
		return fmt.Errorf("invalid assignment ID in token: %w", err)
	}

	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return fmt.Errorf("assignment not found: %w", err)
	}

	if !assignment.IsAnonymous {
		return fmt.Errorf("this endpoint is only for anonymous assignments")
	}

	if assignment.Status == models.AssignmentStatusCompleted {
		return fmt.Errorf("cannot modify a completed assignment")
	}

	// Merge: index existing responses by question_id, then overlay new ones.
	existingMap := make(map[string]models.Response, len(assignment.Responses))
	for _, r := range assignment.Responses {
		existingMap[r.QuestionID] = r
	}
	for _, r := range responses {
		existingMap[r.QuestionID] = r
	}

	merged := make([]models.Response, 0, len(existingMap))
	for _, r := range existingMap {
		merged = append(merged, r)
	}

	return s.assignmentRepo.SetAllResponses(ctx, assignmentID, merged)
}

// SubmitAnonymous marks the anonymous assignment as completed, increments the
// link's response count, and — if the result mode is "score" — calculates and
// returns the score with its matching range.
func (s *PublicLinkService) SubmitAnonymous(ctx context.Context, claims *middleware.SessionClaims) (map[string]any, error) {
	assignmentID, err := primitive.ObjectIDFromHex(claims.AssignmentID)
	if err != nil {
		return nil, fmt.Errorf("invalid assignment ID in token: %w", err)
	}

	linkID, err := primitive.ObjectIDFromHex(claims.LinkID)
	if err != nil {
		return nil, fmt.Errorf("invalid link ID in token: %w", err)
	}

	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, fmt.Errorf("assignment not found: %w", err)
	}

	if !assignment.IsAnonymous {
		return nil, fmt.Errorf("this endpoint is only for anonymous assignments")
	}

	if assignment.Status == models.AssignmentStatusCompleted {
		return nil, fmt.Errorf("assignment already completed")
	}

	// Mark completed.
	if err := s.assignmentRepo.UpdateStatus(ctx, assignmentID, models.AssignmentStatusCompleted); err != nil {
		return nil, fmt.Errorf("failed to mark assignment as completed: %w", err)
	}

	// Increment response count on the link.
	if err := s.linkRepo.IncrementResponseCount(ctx, linkID); err != nil {
		// Non-fatal — log but don't fail the submission.
		_ = err
	}

	// Fetch the link to get result config.
	link, err := s.linkRepo.GetByID(ctx, linkID)
	if err != nil {
		return nil, fmt.Errorf("link not found: %w", err)
	}

	result := map[string]any{
		"result_config": link.ResultConfig,
	}

	if link.ResultConfig.Mode == "score" {
		score := s.calculateScore(assignment.Responses)
		result["computed_score"] = math.Round(score*100) / 100

		if matchingRange := s.findMatchingRange(score, link.ResultConfig.ScoreRanges); matchingRange != nil {
			result["matching_range"] = matchingRange
		}
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// Scoring helpers
// ---------------------------------------------------------------------------

// calculateScore computes the average of all numeric response values.
// Each response's numeric value is expected at response_value["value"].
func (s *PublicLinkService) calculateScore(responses []models.Response) float64 {
	var sum float64
	var count int

	for _, r := range responses {
		if r.ResponseValue == nil {
			continue
		}
		raw, ok := r.ResponseValue["value"]
		if !ok {
			continue
		}

		var val float64
		switch v := raw.(type) {
		case float64:
			val = v
		case float32:
			val = float64(v)
		case int:
			val = float64(v)
		case int32:
			val = float64(v)
		case int64:
			val = float64(v)
		default:
			continue // skip non-numeric values
		}

		sum += val
		count++
	}

	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// findMatchingRange returns the first ScoreRange where score >= min && score < max.
// For the last range (highest max), it uses score <= max as the upper bound so
// the maximum possible score is included.
func (s *PublicLinkService) findMatchingRange(score float64, ranges []models.ScoreRange) *models.ScoreRange {
	for i := range ranges {
		r := &ranges[i]
		if i == len(ranges)-1 {
			// Last range: inclusive upper bound.
			if score >= r.Min && score <= r.Max {
				return r
			}
		} else {
			if score >= r.Min && score < r.Max {
				return r
			}
		}
	}
	return nil
}
