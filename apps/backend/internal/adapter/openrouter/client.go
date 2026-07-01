// Package openrouter is RouterLens's first outbound network integration — see
// docs/adr/0001-pricing-suggestions-openrouter.md for why. It implements
// pricingapp.SuggestionSource by fetching OpenRouter's public model/price
// list, filtering out entries RouterLens must not turn into suggestions
// verbatim, and caching the result in memory.
package openrouter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"golang.org/x/sync/singleflight"

	"router-lens/internal/platform/config"
	pricingapp "router-lens/internal/usecase/pricing"
)

const (
	modelsURL         = "https://openrouter.ai/api/v1/models"
	fetchTimeout      = 5 * time.Second
	cacheTTL          = time.Hour
	maxResponseBytes  = 5 << 20 // 5MB
	maxProviderLength = 100     // mirrors dto.PricingRequest's Provider validation
	maxModelLength    = 200     // mirrors dto.PricingRequest's Model validation
)

type openRouterPricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
}

type openRouterModel struct {
	ID      string            `json:"id"`
	Pricing openRouterPricing `json:"pricing"`
}

type openRouterResponse struct {
	Data []openRouterModel `json:"data"`
}

// Client implements pricingapp.SuggestionSource.
type Client struct {
	enabled    bool
	httpClient *http.Client
	group      singleflight.Group

	mu       sync.Mutex
	cached   []pricingapp.PriceSuggestion
	cachedAt time.Time
}

// NewClient reads the enable flag from config; when disabled, List always
// returns pricingapp.ErrSuggestionsDisabled without ever reaching the network.
func NewClient(cfg config.Config) *Client {
	return &Client{
		enabled:    cfg.PricingSuggestionsEnabled,
		httpClient: &http.Client{Timeout: fetchTimeout},
	}
}

func (c *Client) List(ctx context.Context) ([]pricingapp.PriceSuggestion, error) {
	if !c.enabled {
		return nil, pricingapp.ErrSuggestionsDisabled
	}

	c.mu.Lock()
	fresh := time.Since(c.cachedAt) < cacheTTL
	cached := c.cached
	c.mu.Unlock()
	if fresh {
		return cached, nil
	}

	v, err, _ := c.group.Do("fetch", func() (any, error) {
		return c.fetch(ctx)
	})
	if err != nil {
		// Fall back to a stale cache rather than failing outright, if one exists.
		c.mu.Lock()
		hasStale := len(c.cached) > 0
		stale := c.cached
		c.mu.Unlock()
		if hasStale {
			return stale, nil
		}
		return nil, err
	}
	return v.([]pricingapp.PriceSuggestion), nil
}

func (c *Client) fetch(ctx context.Context) ([]pricingapp.PriceSuggestion, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, err
	}

	var parsed openRouterResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	suggestions := transform(parsed.Data)

	c.mu.Lock()
	c.cached = suggestions
	c.cachedAt = time.Now()
	c.mu.Unlock()

	return suggestions, nil
}

// transform filters and converts raw OpenRouter entries. Skipped: non-
// "provider/model"-shaped ids (aliases prefixed "~", anything without exactly
// one slash), unknown pricing ("-1", empty, or unparseable), and
// provider/model strings exceeding RouterLens's own field-length limits.
func transform(raw []openRouterModel) []pricingapp.PriceSuggestion {
	out := make([]pricingapp.PriceSuggestion, 0, len(raw))
	for _, m := range raw {
		provider, model, ok := splitProviderModel(m.ID)
		if !ok || len(provider) > maxProviderLength || len(model) > maxModelLength {
			continue
		}
		input, ok := parsePricePer1M(m.Pricing.Prompt)
		if !ok {
			continue
		}
		output, ok := parsePricePer1M(m.Pricing.Completion)
		if !ok {
			continue
		}
		out = append(out, pricingapp.PriceSuggestion{
			Provider:         provider,
			Model:            model,
			InputPricePer1M:  input,
			OutputPricePer1M: output,
		})
	}
	return out
}

func splitProviderModel(id string) (provider, model string, ok bool) {
	if strings.HasPrefix(id, "~") {
		return "", "", false
	}
	parts := strings.Split(id, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// parsePricePer1M converts a per-token price string to price-per-1M-tokens.
// Rejects unparseable strings and OpenRouter's "-1" unknown-price sentinel.
func parsePricePer1M(perToken string) (decimal.Decimal, bool) {
	d, err := decimal.NewFromString(perToken)
	if err != nil || d.IsNegative() {
		return decimal.Decimal{}, false
	}
	return d.Mul(decimal.NewFromInt(1_000_000)), true
}
