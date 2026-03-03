package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"questionarie-service/middleware"
	"questionarie-service/models"
	"questionarie-service/repository"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// APITokenService handles business logic for API tokens
type APITokenService struct {
	tokenRepo   *repository.APITokenRepository
	companyRepo *repository.CompanyRepository
}

// NewAPITokenService creates a new APITokenService
func NewAPITokenService(tokenRepo *repository.APITokenRepository, companyRepo *repository.CompanyRepository) *APITokenService {
	return &APITokenService{
		tokenRepo:   tokenRepo,
		companyRepo: companyRepo,
	}
}

// GenerateToken creates a new API token for a company
func (s *APITokenService) GenerateToken(ctx context.Context, companyID primitive.ObjectID, name, createdBy string) (*models.APIToken, error) {
	// Verify company exists and is active
	company, err := s.companyRepo.GetByID(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("company not found: %w", err)
	}
	if !company.IsActive {
		return nil, fmt.Errorf("cannot create token for inactive company")
	}

	// Generate random token: wm_live_ + 40 hex chars
	randomBytes := make([]byte, 20)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}
	rawToken := "wm_live_" + hex.EncodeToString(randomBytes)

	// Hash the token for storage
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	// Store prefix for display (first 16 chars)
	tokenPrefix := rawToken[:16] + "..."

	token := &models.APIToken{
		ID:          primitive.NewObjectID(),
		CompanyID:   companyID,
		TokenHash:   tokenHash,
		TokenPrefix: tokenPrefix,
		Name:        name,
		IsActive:    true,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
	}

	if err := s.tokenRepo.Create(ctx, token); err != nil {
		return nil, err
	}

	// Set raw token for one-time display
	token.RawToken = rawToken
	token.CompanyName = company.Name
	return token, nil
}

// ListByCompany returns all tokens for a company
func (s *APITokenService) ListByCompany(ctx context.Context, companyID primitive.ObjectID) ([]*models.APIToken, error) {
	// Verify company exists
	company, err := s.companyRepo.GetByID(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("company not found: %w", err)
	}

	tokens, err := s.tokenRepo.GetByCompanyID(ctx, companyID)
	if err != nil {
		return nil, err
	}

	// Enrich with company name
	for _, t := range tokens {
		t.CompanyName = company.Name
	}

	return tokens, nil
}

// RevokeToken permanently revokes a token
func (s *APITokenService) RevokeToken(ctx context.Context, tokenID primitive.ObjectID) error {
	return s.tokenRepo.Revoke(ctx, tokenID)
}

// ToggleToken toggles a token's active status
func (s *APITokenService) ToggleToken(ctx context.Context, tokenID primitive.ObjectID) error {
	token, err := s.tokenRepo.GetByID(ctx, tokenID)
	if err != nil {
		return err
	}
	if token.RevokedAt != nil {
		return fmt.Errorf("cannot toggle a revoked token")
	}
	return s.tokenRepo.ToggleActive(ctx, tokenID, !token.IsActive)
}

// ValidateTokenRaw validates a raw API token and returns company info for the middleware.
// Implements middleware.APITokenValidator.
func (s *APITokenService) ValidateTokenRaw(ctx context.Context, rawToken string) (*middleware.APICompanyInfo, error) {
	token, err := s.ValidateToken(ctx, rawToken)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, nil
	}
	return &middleware.APICompanyInfo{
		CompanyID:   token.CompanyID,
		CompanyName: token.CompanyName,
		TokenID:     token.ID,
		TokenName:   token.Name,
	}, nil
}

// ValidateToken validates a raw API token and returns the associated token info.
// It hashes the raw token and looks it up in the database. If valid, it updates last_used_at.
func (s *APITokenService) ValidateToken(ctx context.Context, rawToken string) (*models.APIToken, error) {
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	token, err := s.tokenRepo.FindByHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}
	if token == nil {
		return nil, nil
	}

	// Update last_used_at in background (don't block the response)
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.tokenRepo.UpdateLastUsed(bgCtx, token.ID)
	}()

	// Enrich with company name
	company, err := s.companyRepo.GetByID(ctx, token.CompanyID)
	if err == nil {
		token.CompanyName = company.Name
	}

	return token, nil
}
