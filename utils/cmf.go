package utils

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// CMF API response structures
type cmfResponse struct {
	UFs []cmfUF `json:"UFs"`
}

type cmfUF struct {
	Valor string `json:"Valor"`
	Fecha string `json:"Fecha"`
}

// UFValue represents a UF value with its date
type UFValue struct {
	Value float64 `json:"value"`
	Date  string  `json:"date"`
	Source string  `json:"source"`
}

// In-memory cache for UF value
var (
	cachedUF     *UFValue
	cachedUFMu   sync.RWMutex
	cacheTTL     = 1 * time.Hour
	cacheExpires time.Time
)

// FetchCurrentUF fetches the current UF value from CMF Chile API.
// Uses an in-memory cache with 1-hour TTL to avoid excessive API calls.
func FetchCurrentUF() (*UFValue, error) {
	// Check cache first
	cachedUFMu.RLock()
	if cachedUF != nil && time.Now().Before(cacheExpires) {
		val := *cachedUF
		cachedUFMu.RUnlock()
		return &val, nil
	}
	cachedUFMu.RUnlock()

	// Fetch from CMF API
	apiKey := os.Getenv("CMF_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("CMF_API_KEY not configured")
	}

	url := fmt.Sprintf(
		"https://api.cmfchile.cl/api-sbifv3/recursos_api/uf?apikey=%s&formato=json",
		apiKey,
	)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("CMF API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CMF API returned status %d", resp.StatusCode)
	}

	var cmfResp cmfResponse
	if err := json.NewDecoder(resp.Body).Decode(&cmfResp); err != nil {
		return nil, fmt.Errorf("failed to parse CMF response: %w", err)
	}

	if len(cmfResp.UFs) == 0 {
		return nil, fmt.Errorf("CMF API returned no UF data")
	}

	// Parse the value: "39.841,72" → 39841.72
	rawValue := cmfResp.UFs[0].Valor
	parsed, err := parseCLPNumber(rawValue)
	if err != nil {
		return nil, fmt.Errorf("failed to parse UF value %q: %w", rawValue, err)
	}

	uf := &UFValue{
		Value:  parsed,
		Date:   cmfResp.UFs[0].Fecha,
		Source: "CMF Chile",
	}

	// Update cache
	cachedUFMu.Lock()
	cachedUF = uf
	cacheExpires = time.Now().Add(cacheTTL)
	cachedUFMu.Unlock()

	slog.Info("UF value fetched from CMF", "value", uf.Value, "date", uf.Date)
	return uf, nil
}

// parseCLPNumber parses a Chilean formatted number like "39.841,72" to 39841.72
func parseCLPNumber(s string) (float64, error) {
	s = strings.TrimSpace(s)
	// Remove dots (thousands separator)
	s = strings.ReplaceAll(s, ".", "")
	// Replace comma with dot (decimal separator)
	s = strings.ReplaceAll(s, ",", ".")
	return strconv.ParseFloat(s, 64)
}
