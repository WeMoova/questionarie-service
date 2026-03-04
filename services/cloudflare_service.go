package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// CloudflareService manages DNS records via the Cloudflare API
type CloudflareService struct {
	apiToken string
	zoneID   string
	serverIP string
	domain   string
}

// reservedSlugs lists subdomains that must not be used by companies
var reservedSlugs = map[string]bool{
	"admin":    true,
	"api":      true,
	"app":      true,
	"argocd":   true,
	"auth":     true,
	"docs":     true,
	"grafana":  true,
	"mail":     true,
	"minio":    true,
	"qa":       true,
	"services": true,
	"www":      true,
}

var slugRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// NewCloudflareService creates a CloudflareService from environment variables.
// Returns nil if CLOUDFLARE_API_TOKEN or CLOUDFLARE_ZONE_ID are not configured.
func NewCloudflareService() *CloudflareService {
	apiToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	zoneID := os.Getenv("CLOUDFLARE_ZONE_ID")
	if apiToken == "" || zoneID == "" {
		slog.Warn("Cloudflare DNS integration disabled: CLOUDFLARE_API_TOKEN or CLOUDFLARE_ZONE_ID not set")
		return nil
	}

	serverIP := os.Getenv("CLOUDFLARE_SERVER_IP")
	if serverIP == "" {
		serverIP = "72.62.137.166"
	}
	domain := os.Getenv("CLOUDFLARE_DOMAIN")
	if domain == "" {
		domain = "wemoova.com"
	}

	slog.Info("Cloudflare DNS integration enabled", "domain", domain, "server_ip", serverIP)
	return &CloudflareService{
		apiToken: apiToken,
		zoneID:   zoneID,
		serverIP: serverIP,
		domain:   domain,
	}
}

// ValidateSlug checks that a slug is valid for use as a subdomain.
// Returns an error describing the problem, or nil if the slug is valid.
func (s *CloudflareService) ValidateSlug(slug string) error {
	if slug == "" {
		return nil // empty slug is allowed (means no subdomain)
	}
	if len(slug) < 2 {
		return fmt.Errorf("invalid slug: el subdominio debe tener al menos 2 caracteres")
	}
	if len(slug) > 63 {
		return fmt.Errorf("invalid slug: el subdominio no puede tener más de 63 caracteres")
	}
	if !slugRegex.MatchString(slug) {
		return fmt.Errorf("invalid slug: el subdominio solo puede contener letras minúsculas, números y guiones")
	}
	if reservedSlugs[slug] {
		return fmt.Errorf("invalid slug: el subdominio '%s' está reservado", slug)
	}
	return nil
}

// cloudflareResponse is the envelope returned by the Cloudflare API
type cloudflareResponse struct {
	Success bool              `json:"success"`
	Errors  []cloudflareError `json:"errors"`
	Result  json.RawMessage   `json:"result"`
}

type cloudflareError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type dnsRecord struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	Proxied bool   `json:"proxied"`
	TTL     int    `json:"ttl"`
}

// fqdn returns the fully qualified domain name for a slug
func (s *CloudflareService) fqdn(slug string) string {
	return slug + "." + s.domain
}

// EnsureDNSRecord creates or updates an A record for {slug}.{domain} → serverIP.
// If the record already exists with the correct IP, it is left unchanged.
func (s *CloudflareService) EnsureDNSRecord(slug string) error {
	if err := s.ValidateSlug(slug); err != nil {
		return err
	}

	name := s.fqdn(slug)

	// Check if record already exists
	existing, err := s.findRecord(name)
	if err != nil {
		return fmt.Errorf("failed to check existing DNS record: %w", err)
	}

	if existing != nil {
		// Record exists — update only if IP differs
		if existing.Content == s.serverIP {
			slog.Info("DNS record already exists and is correct", "name", name)
			return nil
		}
		return s.updateRecord(existing.ID, name)
	}

	// Create new record
	return s.createRecord(name)
}

// DeleteDNSRecord removes the A record for {slug}.{domain} if it exists.
func (s *CloudflareService) DeleteDNSRecord(slug string) error {
	if slug == "" {
		return nil
	}
	name := s.fqdn(slug)
	existing, err := s.findRecord(name)
	if err != nil {
		return fmt.Errorf("failed to find DNS record for deletion: %w", err)
	}
	if existing == nil {
		return nil // nothing to delete
	}

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", s.zoneID, existing.ID)
	req, _ := http.NewRequest("DELETE", url, nil)
	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete DNS record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cloudflare API error %d: %s", resp.StatusCode, string(body))
	}

	slog.Info("DNS record deleted", "name", name)
	return nil
}

// findRecord searches for an existing A record by fully qualified name.
func (s *CloudflareService) findRecord(name string) (*dnsRecord, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?name=%s&type=A",
		s.zoneID, name)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cloudflare API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("cloudflare API error %d: %s", resp.StatusCode, string(body))
	}

	var cfResp struct {
		Success bool        `json:"success"`
		Result  []dnsRecord `json:"result"`
	}
	if err := json.Unmarshal(body, &cfResp); err != nil {
		return nil, fmt.Errorf("failed to parse cloudflare response: %w", err)
	}

	// Find exact match (case-insensitive)
	for _, r := range cfResp.Result {
		if strings.EqualFold(r.Name, name) && r.Type == "A" {
			return &r, nil
		}
	}
	return nil, nil
}

// createRecord creates a new A record
func (s *CloudflareService) createRecord(name string) error {
	payload := map[string]interface{}{
		"type":    "A",
		"name":    name,
		"content": s.serverIP,
		"ttl":     1, // automatic
		"proxied": false,
	}
	body, _ := json.Marshal(payload)

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", s.zoneID)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create DNS record: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("cloudflare API error %d: %s", resp.StatusCode, string(respBody))
	}

	slog.Info("DNS record created", "name", name, "ip", s.serverIP)
	return nil
}

// updateRecord updates an existing A record with the correct IP
func (s *CloudflareService) updateRecord(recordID, name string) error {
	payload := map[string]interface{}{
		"type":    "A",
		"name":    name,
		"content": s.serverIP,
		"ttl":     1,
		"proxied": false,
	}
	body, _ := json.Marshal(payload)

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", s.zoneID, recordID)
	req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update DNS record: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("cloudflare API error %d: %s", resp.StatusCode, string(respBody))
	}

	slog.Info("DNS record updated", "name", name, "ip", s.serverIP)
	return nil
}
