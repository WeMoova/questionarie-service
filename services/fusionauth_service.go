package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"questionarie-service/models"
	"strings"
	"time"
)

// FusionAuthService handles FusionAuth multi-tenant operations
type FusionAuthService struct {
	apiKey  string
	baseURL string
	apiBase string // Public API base for image URLs (e.g. https://qa.services.wemoova.com/questionarie-service)
}

// NewFusionAuthService creates a new FusionAuthService
func NewFusionAuthService() *FusionAuthService {
	baseURL := os.Getenv("FUSIONAUTH_URL")
	if baseURL == "" {
		baseURL = "https://auth.wemoova.com"
	}
	apiKey := os.Getenv("FUSIONAUTH_API_KEY")
	apiBase := os.Getenv("API_BASE_URL")
	if apiBase == "" {
		apiBase = "https://qa.services.wemoova.com/questionarie-service"
	}

	return &FusionAuthService{
		apiKey:  apiKey,
		baseURL: baseURL,
		apiBase: apiBase,
	}
}

// TenantResult holds the IDs returned after creating a tenant + application
type TenantResult struct {
	TenantID      string
	ApplicationID string
	ClientID      string
	ClientSecret  string
}

// CreateTenantForCompany creates a FusionAuth tenant and application for the given company.
func (s *FusionAuthService) CreateTenantForCompany(ctx context.Context, company *models.Company) (*TenantResult, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("FUSIONAUTH_API_KEY not configured")
	}

	slug := ""
	if company.CustomDomain != nil {
		slug = company.CustomDomain.Slug
	}
	if slug == "" {
		return nil, fmt.Errorf("company must have a slug to create a tenant")
	}

	// 1. Fetch default tenant SMTP config to copy
	smtpConfig, err := s.fetchDefaultTenantSMTP(ctx)
	if err != nil {
		slog.Warn("failed to fetch default tenant SMTP, proceeding without email config", "error", err)
		smtpConfig = map[string]interface{}{}
	}

	// Build tenant.data with branding
	tenantData := s.buildTenantData(company)

	// Merge SMTP config with template IDs
	emailConfig := s.mergeEmailConfig(smtpConfig)
	// Override fromName with company name for white-label emails
	emailConfig["defaultFromName"] = company.Name

	// 2. Create tenant
	tenantBody := map[string]interface{}{
		"tenant": map[string]interface{}{
			"name":   company.Name,
			"issuer": slug + ".wemoova.com",
			"data":   tenantData,
			"emailConfiguration": emailConfig,
			"externalIdentifierConfiguration": map[string]interface{}{
				"changePasswordIdTimeToLiveInSeconds":            5184000,
				"setupPasswordIdTimeToLiveInSeconds":             5184000,
				"emailVerificationIdTimeToLiveInSeconds":         5184000,
				"emailVerificationOneTimeCodeTimeToLiveInSeconds": 5184000,
			},
		},
	}

	respBody, err := s.doRequest(ctx, "POST", "/api/tenant", nil, tenantBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	tenantResp, ok := respBody["tenant"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected tenant creation response")
	}
	tenantID, _ := tenantResp["id"].(string)
	if tenantID == "" {
		return nil, fmt.Errorf("tenant ID not returned")
	}

	slog.Info("fusionauth tenant created", "tenant_id", tenantID, "company", company.Name)

	// 3. Create application inside the tenant
	appBody := map[string]interface{}{
		"application": map[string]interface{}{
			"name":     company.Name + " - Employee App",
			"tenantId": tenantID,
			"oauthConfiguration": map[string]interface{}{
				"authorizedRedirectURLs":    []string{"https://" + slug + ".wemoova.com/api/auth/callback"},
				"enabledGrants":             []string{"password", "authorization_code", "refresh_token"},
				"generateRefreshTokens":     true,
				"requireClientAuthentication": true,
			},
			"loginConfiguration": map[string]interface{}{
				"generateRefreshTokens": true,
			},
			"registrationConfiguration": map[string]interface{}{
				"enabled": true,
			},
			"roles": []map[string]interface{}{
				{"name": "employee", "description": "Empleado"},
				{"name": "supervisor", "description": "Supervisor"},
				{"name": "company_admin", "description": "Admin de empresa"},
			},
		},
	}

	headers := map[string]string{"X-FusionAuth-TenantId": tenantID}
	appResp, err := s.doRequest(ctx, "POST", "/api/application", headers, appBody)
	if err != nil {
		// Rollback: delete the tenant
		_ = s.DeleteTenant(ctx, tenantID)
		return nil, fmt.Errorf("failed to create application: %w", err)
	}

	app, ok := appResp["application"].(map[string]interface{})
	if !ok {
		_ = s.DeleteTenant(ctx, tenantID)
		return nil, fmt.Errorf("unexpected application creation response")
	}

	appID, _ := app["id"].(string)
	oauthConfig, _ := app["oauthConfiguration"].(map[string]interface{})
	clientID, _ := oauthConfig["clientId"].(string)
	clientSecret, _ := oauthConfig["clientSecret"].(string)

	slog.Info("fusionauth application created", "app_id", appID, "tenant_id", tenantID)

	return &TenantResult{
		TenantID:      tenantID,
		ApplicationID: appID,
		ClientID:      clientID,
		ClientSecret:  clientSecret,
	}, nil
}

// SyncBrandingToTenant updates the tenant.data with the latest company branding
func (s *FusionAuthService) SyncBrandingToTenant(ctx context.Context, company *models.Company) error {
	if company.FusionAuthTenantID == "" {
		return nil // no tenant to sync
	}

	tenantData := s.buildTenantData(company)

	body := map[string]interface{}{
		"tenant": map[string]interface{}{
			"name": company.Name,
			"data": tenantData,
			"emailConfiguration": map[string]interface{}{
				"defaultFromName": company.Name,
			},
		},
	}

	_, err := s.doRequest(ctx, "PATCH", "/api/tenant/"+company.FusionAuthTenantID, nil, body)
	if err != nil {
		return fmt.Errorf("failed to sync branding to tenant: %w", err)
	}

	slog.Info("fusionauth tenant synced", "tenant_id", company.FusionAuthTenantID, "company", company.Name)
	return nil
}

// DeleteTenant deletes a FusionAuth tenant
func (s *FusionAuthService) DeleteTenant(ctx context.Context, tenantID string) error {
	_, err := s.doRequest(ctx, "DELETE", "/api/tenant/"+tenantID, nil, nil)
	return err
}

// DeleteUser deletes a user from FusionAuth by user ID
func (s *FusionAuthService) DeleteUser(ctx context.Context, userID string) error {
	_, err := s.doRequest(ctx, "DELETE", "/api/user/"+userID+"?hardDelete=true", nil, nil)
	return err
}

// buildTenantData creates the tenant.data map from company branding
func (s *FusionAuthService) buildTenantData(company *models.Company) map[string]interface{} {
	data := map[string]interface{}{
		"companyName": company.Name,
	}

	if company.CustomDomain != nil {
		data["slug"] = company.CustomDomain.Slug
	}

	if company.Branding != nil {
		b := company.Branding
		if b.PrimaryColor != "" {
			data["primaryColor"] = b.PrimaryColor
		}
		if b.SecondaryColor != "" {
			data["secondaryColor"] = b.SecondaryColor
		}
		if b.AccentColor != "" {
			data["accentColor"] = b.AccentColor
		}
		if b.Logo != "" && !strings.HasSuffix(strings.ToLower(b.Logo), ".svg") {
			data["logoUrl"] = s.apiBase + "/api/v1/images/" + b.Logo
		}
		if b.LogoIcon != "" && !strings.HasSuffix(strings.ToLower(b.LogoIcon), ".svg") {
			data["logoIconUrl"] = s.apiBase + "/api/v1/images/" + b.LogoIcon
		}
	}

	return data
}

// mergeEmailConfig builds emailConfiguration with SMTP settings from default tenant + template IDs
func (s *FusionAuthService) mergeEmailConfig(smtpConfig map[string]interface{}) map[string]interface{} {
	// Start with the SMTP config from default tenant
	config := smtpConfig

	// Assign email template IDs (only templates available without Threat Detection license)
	config["setPasswordEmailTemplateId"] = "5cb4c04f-da04-4568-8bad-b1f640afdd80"
	config["forgotPasswordEmailTemplateId"] = "fb178f2d-2f61-41a0-baa8-16e69bb403a7"
	config["verifyEmailTemplateId"] = "0dc276b0-a0ce-48c1-8042-db46b25fe090"
	config["verificationEmailTemplateId"] = "f52265f4-ac41-403b-a780-d7e4c96101e8"
	config["passwordlessEmailTemplateId"] = "2b4c751d-b81d-4bde-80a8-7aa7ef012262"
	// NOTE: loginNewDevice, passwordResetSuccess, twoFactorMethodAdd/Remove require Threat Detection license

	return config
}

// fetchDefaultTenantSMTP fetches the email configuration from the default tenant
func (s *FusionAuthService) fetchDefaultTenantSMTP(ctx context.Context) (map[string]interface{}, error) {
	defaultTenantID := "0f5461b3-1d9a-7272-dbea-94f0944fd890"

	respBody, err := s.doRequest(ctx, "GET", "/api/tenant/"+defaultTenantID, nil, nil)
	if err != nil {
		return nil, err
	}

	tenant, ok := respBody["tenant"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected tenant response format")
	}

	emailConfig, ok := tenant["emailConfiguration"].(map[string]interface{})
	if !ok {
		return map[string]interface{}{}, nil
	}

	// Extract only SMTP-related fields (not template IDs which we'll set ourselves)
	smtp := map[string]interface{}{}
	smtpFields := []string{
		"host", "port", "username", "password", "security",
		"defaultFromEmail", "defaultFromName",
		"additionalHeaders",
	}
	for _, field := range smtpFields {
		if v, exists := emailConfig[field]; exists {
			smtp[field] = v
		}
	}

	return smtp, nil
}

// doRequest executes an HTTP request against the FusionAuth API
func (s *FusionAuthService) doRequest(ctx context.Context, method, path string, extraHeaders map[string]string, body interface{}) (map[string]interface{}, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", s.apiKey)
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("FusionAuth API error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	// For DELETE or empty responses
	if len(respBytes) == 0 {
		return map[string]interface{}{}, nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}
