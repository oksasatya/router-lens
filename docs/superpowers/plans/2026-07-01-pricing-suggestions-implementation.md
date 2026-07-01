# Pricing Suggestions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let an admin create a Pricing Rule by picking a model from a searchable, OpenRouter-sourced list (grouped by provider, with logos) instead of typing everything by hand — selecting an entry pre-fills the existing create-rule form, which still requires a normal save.

**Architecture:** Backend adds one new outbound integration (OpenRouter's public models list), cached in memory for 1 hour with singleflight de-duplication, exposed via one new session-protected endpoint. Frontend adds a searchable picker dialog (shadcn `Command`) that feeds the *existing* `PricingFormDialog` via a new optional `defaultValues` prop — no new form, no new persistence path.

**Tech Stack:** Go stdlib `net/http` + `golang.org/x/sync/singleflight` (already an indirect dependency — this promotes it to direct); shadcn `command`; `simple-icons` npm package for provider logos (exact, correctly-licensed brand SVG data — not hand-copied paths).

## Global Constraints

- See `docs/superpowers/specs/2026-07-01-pricing-suggestions-design.md` and `docs/adr/0001-pricing-suggestions-openrouter.md` for full rationale — this plan implements them verbatim.
- **Naming:** the feature is "Pricing Suggestion" everywhere (code, i18n, UI copy) — never "catalog" (CONTEXT.md explicitly rules out a stored model catalog; see the spec's Terminology section).
- **Layering (HARD):** the `SuggestionSource` port lives in `internal/usecase/pricing`, NOT `internal/domain/pricing`. Domain stays untouched by this feature.
- **Config gate:** `PRICING_SUGGESTIONS_ENABLED` env var, default `true` (only `"false"` turns it off).
- Tailwind v4 only; Base UI `render={<Component />}` composition, never `asChild`.
- Sonar-Go: `go:S107` ≤7 params, `go:S3776` cognitive ≤15, `go:S1192` const for 3×-duplicated literals, errcheck, gosec.
- Sonar-TS/React: readonly props, stable list keys (never index/derived), `globalThis` not `window`.
- i18n: every new user-facing string via `t("...")`, added to both `en.json` and `id.json` in the same step.
- Nothing in this plan touches `apps/backend/internal/domain/event`, `usecase/event`, or any file under active work by the concurrent Plan 05/06 effort — verify with `git status` before each commit that only this plan's files are staged (this repo has had two commits polluted by a concurrent session's staged files earlier in this same day; commit by explicit pathspec, never bare `git add -A`).

---

## Task 1: Backend — Pricing Suggestions endpoint

**Files:**
- Modify: `apps/backend/internal/platform/config/config.go`
- Modify: `apps/backend/internal/usecase/pricing/pricing.go`
- Modify: `apps/backend/internal/usecase/pricing/pricing_test.go`
- Create: `apps/backend/internal/adapter/openrouter/client.go`
- Create: `apps/backend/internal/adapter/openrouter/client_test.go`
- Modify: `apps/backend/internal/adapter/http/dto/pricing.go`
- Modify: `apps/backend/internal/adapter/http/handler/pricing_handler.go`
- Modify: `apps/backend/internal/platform/bootstrap/bootstrap.go`
- Modify: `apps/backend/.env.example`
- Modify: `apps/backend/go.mod` (promote `golang.org/x/sync` from indirect to direct)

**Interfaces:**
- Produces: `pricingapp.PriceSuggestion{Provider, Model string; InputPricePer1M, OutputPricePer1M decimal.Decimal}`, `pricingapp.SuggestionSource` interface (`List(ctx) ([]PriceSuggestion, error)`), `pricingapp.ErrSuggestionsDisabled` sentinel, `pricingapp.Service.ListSuggestions(ctx) ([]PriceSuggestion, error)`, `openrouter.NewClient(cfg config.Config) *openrouter.Client` implementing `SuggestionSource`, `GET /api/v1/pricing/suggestions` (200/404/502).

- [ ] **Step 1: Config flag**

In `apps/backend/internal/platform/config/config.go`, add the field and its parse line:

```go
type Config struct {
	AppEnv                     string
	AppPort                    string
	DatabaseURL                string
	SessionSecret              string
	CookieCrossSite            bool
	MaxBackdateDays            int
	RetentionDays              int
	LogLevel                   string
	PricingSuggestionsEnabled  bool
}
```

In `parse`, add (matching the file's existing `orDefault`/direct-comparison style — this flag defaults to `true`, so it's compared against the disabling value rather than the enabling one, unlike `CookieCrossSite`):

```go
		LogLevel:                  orDefault(get("LOG_LEVEL"), "info"),
		PricingSuggestionsEnabled: get("PRICING_SUGGESTIONS_ENABLED") != "false",
```

Run: `cd apps/backend && go build ./...`
Expected: clean (no other file references the `Config` struct literal exhaustively, so adding a field is non-breaking).

Add to `apps/backend/.env.example` (after `RETENTION_DAYS`):
```
PRICING_SUGGESTIONS_ENABLED=true
```

- [ ] **Step 2: `PriceSuggestion` type + `SuggestionSource` port + `ListSuggestions` on the existing Service**

In `apps/backend/internal/usecase/pricing/pricing.go`, add near the top (after the `Input` struct, before `Service`):

```go
// PriceSuggestion is a third-party reference price for a (provider, model) pair,
// offered while filling in a Pricing Rule form. It is never persisted — see
// CONTEXT.md's "Pricing Suggestion" glossary note. Not a domain type: this is a
// UI convenience backed by an external integration, not a domain rule.
type PriceSuggestion struct {
	Provider         string
	Model            string
	InputPricePer1M  decimal.Decimal
	OutputPricePer1M decimal.Decimal
}

// SuggestionSource is the application-level port to a third-party price
// reference (OpenRouter). It lives here, not in internal/domain/pricing.
type SuggestionSource interface {
	List(ctx context.Context) ([]PriceSuggestion, error)
}

// ErrSuggestionsDisabled is returned by a SuggestionSource when
// PRICING_SUGGESTIONS_ENABLED=false. The HTTP layer maps this to 404.
var ErrSuggestionsDisabled = errors.New("pricing: suggestions disabled")
```

Change the `Service` struct and constructor:

```go
type Service struct {
	repo   pricingdomain.PricingRepository
	source SuggestionSource
}

func NewService(repo pricingdomain.PricingRepository, source SuggestionSource) *Service {
	return &Service{repo: repo, source: source}
}
```

Add a new method (near `List`):

```go
// ListSuggestions returns third-party reference prices, or ErrSuggestionsDisabled
// if the feature is turned off by config (source is a nil-safe check away from
// a panic if a caller ever constructs a Service without one — production wiring
// always provides one via Fx).
func (s *Service) ListSuggestions(ctx context.Context) ([]PriceSuggestion, error) {
	if s.source == nil {
		return nil, ErrSuggestionsDisabled
	}
	return s.source.List(ctx)
}
```

- [ ] **Step 2b: Fix the 3 existing `NewService(f)` call sites**

`apps/backend/internal/usecase/pricing/pricing_test.go` calls `NewService(f)` with one argument in three places (in `TestUpsert`'s three subtests). Update each to `NewService(f, nil)` — these tests never call `ListSuggestions`, so `nil` is safe (per Step 2's nil-check).

Run: `cd apps/backend && go build ./... && go test ./internal/usecase/pricing/...`
Expected: builds clean, existing 3 subtests still pass (they don't exercise the new code path).

- [ ] **Step 3: OpenRouter adapter — fetch, filter, transform (TDD: yes — write the test first)**

First, verify the live response shape (OpenRouter's endpoint is public, no auth needed):
```bash
curl -s https://openrouter.ai/api/v1/models | head -c 2000
```
Expected: a JSON object with a top-level `"data"` array; each entry has `id` (e.g. `"openai/gpt-4o"`), and a `pricing` object with string fields including `prompt` and `completion` (price per token, e.g. `"0.0000025"`). If the real shape differs from what's coded below, adjust the `openRouterModel` struct's JSON tags to match — this is the one part of the plan sourced from external, version-able API docs rather than this repo's own code.

Write the test file `apps/backend/internal/adapter/openrouter/client_test.go`:

```go
package openrouter

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestTransform(t *testing.T) {
	t.Run("splits provider/model and converts price to per-1M", func(t *testing.T) {
		raw := []openRouterModel{
			{ID: "openai/gpt-4o", Pricing: openRouterPricing{Prompt: "0.0000025", Completion: "0.00001"}},
		}
		got := transform(raw)
		if len(got) != 1 {
			t.Fatalf("want 1 suggestion, got %d", len(got))
		}
		want := decimal.NewFromFloat(2.5)
		if got[0].Provider != "openai" || got[0].Model != "gpt-4o" {
			t.Fatalf("provider/model split wrong: %+v", got[0])
		}
		if !got[0].InputPricePer1M.Equal(want) {
			t.Fatalf("input price = %s, want %s", got[0].InputPricePer1M, want)
		}
	})

	t.Run("skips entries with unknown (-1) pricing", func(t *testing.T) {
		raw := []openRouterModel{
			{ID: "some/model", Pricing: openRouterPricing{Prompt: "-1", Completion: "-1"}},
		}
		if got := transform(raw); len(got) != 0 {
			t.Fatalf("want 0 suggestions for unknown pricing, got %d", len(got))
		}
	})

	t.Run("skips aliases and malformed ids", func(t *testing.T) {
		raw := []openRouterModel{
			{ID: "~openai/gpt-4o-alias", Pricing: openRouterPricing{Prompt: "0.000001", Completion: "0.000002"}},
			{ID: "no-slash-here", Pricing: openRouterPricing{Prompt: "0.000001", Completion: "0.000002"}},
			{ID: "too/many/slashes", Pricing: openRouterPricing{Prompt: "0.000001", Completion: "0.000002"}},
		}
		if got := transform(raw); len(got) != 0 {
			t.Fatalf("want 0 suggestions for alias/malformed ids, got %d", len(got))
		}
	})

	t.Run("skips entries with non-numeric pricing", func(t *testing.T) {
		raw := []openRouterModel{
			{ID: "openai/gpt-4o", Pricing: openRouterPricing{Prompt: "not-a-number", Completion: "0.00001"}},
		}
		if got := transform(raw); len(got) != 0 {
			t.Fatalf("want 0 suggestions for unparseable pricing, got %d", len(got))
		}
	})

	t.Run("skips oversized provider/model strings", func(t *testing.T) {
		longModel := ""
		for range 250 {
			longModel += "x"
		}
		raw := []openRouterModel{
			{ID: "openai/" + longModel, Pricing: openRouterPricing{Prompt: "0.000001", Completion: "0.000002"}},
		}
		if got := transform(raw); len(got) != 0 {
			t.Fatalf("want 0 suggestions for oversized model string, got %d", len(got))
		}
	})
}
```

Run: `cd apps/backend && go test ./internal/adapter/openrouter/... -run TestTransform -v`
Expected: FAIL (package `openrouter` and `transform`/`openRouterModel`/`openRouterPricing` don't exist yet).

Now write `apps/backend/internal/adapter/openrouter/client.go`:

```go
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
	ID      string             `json:"id"`
	Pricing openRouterPricing  `json:"pricing"`
}

type openRouterResponse struct {
	Data []openRouterModel `json:"data"`
}

// Client implements pricingapp.SuggestionSource.
type Client struct {
	enabled    bool
	httpClient *http.Client
	group      singleflight.Group

	mu        sync.Mutex
	cached    []pricingapp.PriceSuggestion
	cachedAt  time.Time
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
```

Run: `cd apps/backend && go test ./internal/adapter/openrouter/... -run TestTransform -v`
Expected: PASS (5/5 subtests).

- [ ] **Step 4: `go.mod` — promote `golang.org/x/sync`**

```bash
cd apps/backend && go mod tidy
```
Expected: `golang.org/x/sync` moves from the `// indirect` block to the main `require` block in `go.mod` (it's already in `go.sum` at v0.21.0 — no new download, just a direct-dependency declaration since `client.go` now imports it directly).

- [ ] **Step 5: DTO + handler + route**

In `apps/backend/internal/adapter/http/dto/pricing.go`, add:

```go
// PriceSuggestionResponse is the wire shape of a third-party reference price.
type PriceSuggestionResponse struct {
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	InputPrice1M  string `json:"input_price_per_1m"`
	OutputPrice1M string `json:"output_price_per_1m"`
}

// FromPriceSuggestion maps a usecase-layer suggestion to its response shape.
func FromPriceSuggestion(s pricingapp.PriceSuggestion) PriceSuggestionResponse {
	return PriceSuggestionResponse{
		Provider:      s.Provider,
		Model:         s.Model,
		InputPrice1M:  s.InputPricePer1M.String(),
		OutputPrice1M: s.OutputPricePer1M.String(),
	}
}
```

Add the import at the top of the file: `pricingapp "router-lens/internal/usecase/pricing"`.

In `apps/backend/internal/adapter/http/handler/pricing_handler.go`, register the route (add to `Register`):

```go
func (h *PricingHandler) Register(api *echo.Group, session echo.MiddlewareFunc) {
	api.GET("/pricing", h.list, session)
	api.POST("/pricing", h.upsert, session)
	api.PUT("/pricing/:id", h.update, session)
	api.DELETE("/pricing/:id", h.delete, session)
	api.GET("/pricing/suggestions", h.suggestions, session)
}
```

Add the handler method (near `list`):

```go
func (h *PricingHandler) suggestions(c echo.Context) error {
	suggestions, err := h.svc.ListSuggestions(c.Request().Context())
	if err != nil {
		if errors.Is(err, pricingapp.ErrSuggestionsDisabled) {
			return echo.NewHTTPError(http.StatusNotFound)
		}
		return echo.NewHTTPError(http.StatusBadGateway, "pricing suggestions unavailable")
	}
	dtos := make([]dto.PriceSuggestionResponse, 0, len(suggestions))
	for _, s := range suggestions {
		dtos = append(dtos, dto.FromPriceSuggestion(s))
	}
	return response.Data(c, http.StatusOK, dtos)
}
```

Add `"errors"` to the file's imports — `pricingapp "router-lens/internal/usecase/pricing"` is already imported (used by the existing `toPricingInput` for `pricingapp.Input`).

Run: `cd apps/backend && go build ./...`
Expected: clean.

- [ ] **Step 6: Bootstrap wiring**

In `apps/backend/internal/platform/bootstrap/bootstrap.go`, update `pricingModule`:

```go
var pricingModule = fx.Module("pricing",
	fx.Provide(
		fx.Annotate(postgres.NewPricingRepository, fx.As(new(pricing.PricingRepository))),
		fx.Annotate(openrouter.NewClient, fx.As(new(pricingapp.SuggestionSource))),
		pricingapp.NewService,
		handler.NewPricingHandler,
	),
	fx.Invoke(registerPricingRoutes),
)
```

Add the import: `"router-lens/internal/adapter/openrouter"`.

Run: `cd apps/backend && go build ./... && go vet ./...`
Expected: clean — Fx resolves `openrouter.NewClient`'s `config.Config` parameter from `coreModule` (already provided), and `pricingapp.NewService`'s second parameter from the newly-provided `pricingapp.SuggestionSource`.

- [ ] **Step 7: Full verification + commit**

```bash
cd apps/backend
gofmt -l . && go vet ./... && golangci-lint run && go test -race -cover ./...
```
Expected: `gofmt -l .` empty (no unformatted files), `go vet` clean, lint clean, all tests pass including the 5 new `TestTransform` subtests and the updated `TestUpsert` (still 3/3).

```bash
git status --short
```
Confirm ONLY this task's files are listed as modified/new (per Global Constraints — do not stage anything under `usecase/event`, `domain/event`, or any file not listed in this task's Files section).

```bash
git commit \
  apps/backend/internal/platform/config/config.go \
  apps/backend/internal/usecase/pricing/pricing.go \
  apps/backend/internal/usecase/pricing/pricing_test.go \
  apps/backend/internal/adapter/openrouter/client.go \
  apps/backend/internal/adapter/openrouter/client_test.go \
  apps/backend/internal/adapter/http/dto/pricing.go \
  apps/backend/internal/adapter/http/handler/pricing_handler.go \
  apps/backend/internal/platform/bootstrap/bootstrap.go \
  apps/backend/.env.example \
  apps/backend/go.mod apps/backend/go.sum \
  -m "feat(backend): Pricing Suggestions — OpenRouter-backed price reference endpoint"
```

---

## Task 2: Frontend — Model Suggestion Picker

**Files:**
- Create: `apps/frontend/src/services/pricingSuggestionsService.ts`
- Create: `apps/frontend/src/lib/pricingSuggestions.ts`
- Create: `apps/frontend/src/lib/providerLogos.tsx`
- Create: `apps/frontend/src/components/pricing/ModelSuggestionPicker.tsx`
- Modify: `apps/frontend/src/components/pricing/PricingFormDialog.tsx`
- Modify: `apps/frontend/src/routes/_app.pricing.tsx`
- Modify: `apps/frontend/src/i18n/en.json`, `apps/frontend/src/i18n/id.json`
- Modify: `apps/frontend/package.json` (via CLI/package manager, no manual edit)

**Interfaces:**
- Consumes: Task 1's `GET /api/v1/pricing/suggestions` (200 `{data: PriceSuggestionResponse[]}` / 404 disabled / 502 unavailable).
- Produces: `PriceSuggestion` type + `listPricingSuggestions()` (`services/pricingSuggestionsService.ts`); `pricingSuggestionsQueryOptions` (`lib/pricingSuggestions.ts`); `ModelSuggestionPicker` component with `{ open, onOpenChange, onSelect: (s: PriceSuggestion) => void }`; `PricingFormDialog` gains an optional `defaultValues` prop.

- [ ] **Step 1: shadcn `command` + `simple-icons`**

```bash
cd apps/frontend
bunx --bun shadcn@latest add command
bun add simple-icons
```

Expected: `src/components/ui/command.tsx` created (Base UI/cmdk-based, mirrors this project's existing shadcn primitives — if the generated file's exported component names differ from `Command`/`CommandInput`/`CommandList`/`CommandEmpty`/`CommandGroup`/`CommandItem`, adjust Step 4's usage to match what was actually generated, same escape hatch used elsewhere in this project's plans for CLI-generated files). `simple-icons` added to `dependencies`.

Verify the import shape before using it — `simple-icons` has changed its per-icon export pattern across major versions:
```bash
node -e "const {siOpenai} = require('simple-icons'); console.log(siOpenai)" 2>&1 || \
node -e "const siOpenai = require('simple-icons/icons/openai'); console.log(siOpenai)" 2>&1
```
Expected: one of these prints an object with `path` (SVG path data string) and `hex` (brand color) fields. Use whichever import form actually works with the installed version in Step 3.

- [ ] **Step 2: i18n keys**

`en.json`, add to the existing `"pricing"` object:

```jsonc
"suggestions": {
  "browse": "New from suggestion",
  "title": "Pick a model",
  "searchPlaceholder": "Search provider or model…",
  "empty": "No suggestions available.",
  "unavailable": "Suggestions unavailable right now — enter the rule manually.",
  "caption": "Reference prices from OpenRouter — confirm the provider/model text matches exactly what your router reports before saving."
}
```

`id.json`, add:

```jsonc
"suggestions": {
  "browse": "Baru dari saran",
  "title": "Pilih model",
  "searchPlaceholder": "Cari provider atau model…",
  "empty": "Belum ada saran tersedia.",
  "unavailable": "Saran sedang tidak tersedia — isi aturan secara manual.",
  "caption": "Harga referensi dari OpenRouter — pastikan teks provider/model persis sama seperti yang dilaporkan router Anda sebelum menyimpan."
}
```

- [ ] **Step 3: `pricingSuggestionsService.ts` + `lib/pricingSuggestions.ts`**

`apps/frontend/src/services/pricingSuggestionsService.ts`:

```ts
import { z } from "zod";
import { api, ApiError } from "@/lib/api";

const priceSuggestionSchema = z.object({
  provider: z.string(),
  model: z.string(),
  input_price_per_1m: z.string(),
  output_price_per_1m: z.string(),
});

export type PriceSuggestion = z.infer<typeof priceSuggestionSchema>;

/**
 * GET /pricing/suggestions. Resolves to [] when the feature is disabled
 * (404) so callers can just render nothing rather than an error state;
 * any other failure (502, network) still rejects.
 */
export async function listPricingSuggestions(): Promise<PriceSuggestion[]> {
  try {
    const res = await api.get("/pricing/suggestions");
    return z.array(priceSuggestionSchema).parse(res.data);
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return [];
    throw err;
  }
}
```

`apps/frontend/src/lib/pricingSuggestions.ts`:

```ts
import { queryOptions } from "@tanstack/react-query";
import { listPricingSuggestions } from "@/services/pricingSuggestionsService";

export const pricingSuggestionsQueryOptions = queryOptions({
  queryKey: ["pricing", "suggestions"],
  queryFn: listPricingSuggestions,
  staleTime: 60 * 60 * 1000, // 1h — matches the backend's own cache window
  retry: false,
});
```

- [ ] **Step 4: `providerLogos.tsx`**

```tsx
// Provider brand icons for the suggestion picker. Sourced from `simple-icons`
// (CC0-licensed, exact brand SVG data) rather than hand-copied paths — see
// docs/adr/0001-pricing-suggestions-openrouter.md.
import { Bot } from "lucide-react";
import type { ReactNode } from "react";

// NOTE: adjust these imports to match whichever form Step 1 verified works
// against the installed simple-icons version.
import { siOpenai, siAnthropic, siGooglegemini, siMeta, siMistralai, siX, siCohere, siDeepseek, siQwen } from "simple-icons";

interface BrandIcon {
  readonly path: string;
  readonly hex: string;
}

const PROVIDER_ICONS: Record<string, BrandIcon> = {
  openai: siOpenai,
  anthropic: siAnthropic,
  google: siGooglegemini,
  "meta-llama": siMeta,
  mistralai: siMistralai,
  "x-ai": siX,
  cohere: siCohere,
  deepseek: siDeepseek,
  qwen: siQwen,
};

/** Renders a provider's brand icon, or a generic fallback for unlisted providers. */
export function ProviderLogo({ provider, className }: { readonly provider: string; readonly className?: string }): ReactNode {
  const icon = PROVIDER_ICONS[provider.toLowerCase()];
  if (!icon) return <Bot className={className} aria-hidden />;
  return (
    <svg role="img" viewBox="0 0 24 24" className={className} fill={`#${icon.hex}`} aria-hidden>
      <path d={icon.path} />
    </svg>
  );
}
```

- [ ] **Step 5: `ModelSuggestionPicker.tsx`**

```tsx
import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { formatUSD } from "@/lib/money";
import { pricingSuggestionsQueryOptions } from "@/lib/pricingSuggestions";
import { ProviderLogo } from "@/lib/providerLogos";
import type { PriceSuggestion } from "@/services/pricingSuggestionsService";

interface ModelSuggestionPickerProps {
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
  readonly onSelect: (suggestion: PriceSuggestion) => void;
}

export function ModelSuggestionPicker({ open, onOpenChange, onSelect }: ModelSuggestionPickerProps) {
  const { t } = useTranslation();
  const [search, setSearch] = useState("");
  const query = useQuery({ ...pricingSuggestionsQueryOptions, enabled: open });

  const groups = useMemo(() => {
    const byProvider = new Map<string, PriceSuggestion[]>();
    for (const s of query.data ?? []) {
      const list = byProvider.get(s.provider) ?? [];
      list.push(s);
      byProvider.set(s.provider, list);
    }
    return [...byProvider.entries()].sort(([a], [b]) => a.localeCompare(b));
  }, [query.data]);

  function handleOpenChange(next: boolean) {
    if (!next) setSearch("");
    onOpenChange(next);
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("pricing.suggestions.title")}</DialogTitle>
          <DialogDescription>{t("pricing.suggestions.caption")}</DialogDescription>
        </DialogHeader>
        <Command shouldFilter value={search} onValueChange={setSearch}>
          <CommandInput placeholder={t("pricing.suggestions.searchPlaceholder")} />
          <CommandList>
            <CommandEmpty>{t("pricing.suggestions.empty")}</CommandEmpty>
            {groups.map(([provider, items]) => (
              <CommandGroup
                key={provider}
                heading={
                  <span className="flex items-center gap-1.5">
                    <ProviderLogo provider={provider} className="size-3.5" />
                    {provider}
                  </span>
                }
              >
                {items.map((s) => (
                  <CommandItem
                    key={`${s.provider}/${s.model}`}
                    value={`${s.provider} ${s.model}`}
                    onSelect={() => onSelect(s)}
                    className="justify-between"
                  >
                    <span className="font-mono text-xs">{s.model}</span>
                    <span className="text-xs text-muted-foreground">
                      {formatUSD(s.input_price_per_1m)} / {formatUSD(s.output_price_per_1m)}
                    </span>
                  </CommandItem>
                ))}
              </CommandGroup>
            ))}
          </CommandList>
        </Command>
      </DialogContent>
    </Dialog>
  );
}
```

(Verify `Command`'s `shouldFilter`/`value`/`onValueChange` prop names against the actually-generated `command.tsx` from Step 1 — `cmdk`-based shadcn commands commonly expose exactly these, but confirm before treating a mismatch as your own bug.)

- [ ] **Step 6: `PricingFormDialog` — optional `defaultValues` prop**

In `apps/frontend/src/components/pricing/PricingFormDialog.tsx`, change the props interface and the `useEffect` reset:

```tsx
interface PricingFormDialogProps {
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
  readonly rule: PricingRule | null;
  readonly defaultValues?: {
    provider: string;
    model: string;
    input_price_per_1m: string;
    output_price_per_1m: string;
  } | null;
}
```

```tsx
export function PricingFormDialog({ open, onOpenChange, rule, defaultValues = null }: PricingFormDialogProps) {
```

```tsx
  useEffect(() => {
    if (open) {
      form.reset({
        provider: rule?.provider ?? defaultValues?.provider ?? "",
        model: rule?.model ?? defaultValues?.model ?? "",
        input_price_per_1m: rule?.input_price_per_1m ?? defaultValues?.input_price_per_1m ?? "",
        output_price_per_1m: rule?.output_price_per_1m ?? defaultValues?.output_price_per_1m ?? "",
      });
    }
  }, [open, rule, defaultValues, form]);
```

`isEdit` stays `!!rule` unchanged — a suggestion-prefilled dialog with `rule={null}` is still a create, exactly as a blank one is.

- [ ] **Step 7: Wire into `routes/_app.pricing.tsx`**

Add state + the picker + the new button, next to the existing `"New rule"` button:

```tsx
import { Sparkles } from "lucide-react"; // add to the existing lucide-react import line
```

```tsx
  const [pickerOpen, setPickerOpen] = useState(false);
  const [suggestionDefaults, setSuggestionDefaults] = useState<{
    provider: string;
    model: string;
    input_price_per_1m: string;
    output_price_per_1m: string;
  } | null>(null);
```

In the header's button group (next to the existing "New rule" `<Button>`):

```tsx
        <div className="flex gap-2">
          <Button
            variant="outline"
            onClick={() => setPickerOpen(true)}
          >
            <Sparkles className="size-4" />
            {t("pricing.suggestions.browse")}
          </Button>
          <Button
            onClick={() => {
              setEditing(null);
              setFormOpen(true);
            }}
          >
            <Plus className="size-4" />
            {t("pricing.new")}
          </Button>
        </div>
```

(This replaces the single `<Button>` currently there — wrap both in the `<div className="flex gap-2">` shown above, inside the existing `<div className="flex items-center justify-between">` header row.)

Add the picker + update `PricingFormDialog`'s usage near the bottom of the JSX:

```tsx
      <ModelSuggestionPicker
        open={pickerOpen}
        onOpenChange={setPickerOpen}
        onSelect={(s) => {
          setPickerOpen(false);
          setEditing(null);
          setSuggestionDefaults(s);
          setFormOpen(true);
        }}
      />

      <PricingFormDialog
        open={formOpen}
        onOpenChange={(o) => {
          setFormOpen(o);
          if (!o) setSuggestionDefaults(null);
        }}
        rule={editing}
        defaultValues={suggestionDefaults}
      />
```

(This replaces the existing bare `<PricingFormDialog open={formOpen} onOpenChange={setFormOpen} rule={editing} />` — the `onOpenChange` now also clears `suggestionDefaults` on close so reopening via "New rule" doesn't carry over stale suggestion data.)

Add the import: `import { ModelSuggestionPicker } from "@/components/pricing/ModelSuggestionPicker";`

- [ ] **Step 8: Verify + commit**

```bash
cd apps/frontend
bun run build
bun run test
bun run lint
```
Expected: build/test/lint all clean (no new unit tests expected for this task — TDD verdict is "no", same as the rest of the Pricing screen; verify by running per Step 9).

- [ ] **Step 9: Manual verification (live, per the `verify` skill — build/lint passing is not sufficient)**

With the backend + frontend dev servers running and a logged-in session: navigate to `/pricing`, click **"New from suggestion"** — confirm the picker opens, lists real OpenRouter models grouped by provider with logos, and search filters as you type. Pick one — confirm it closes the picker, opens the create-rule dialog with provider/model/prices already filled in, and saving creates the rule normally (appears in the pricing table). Also verify: with `PRICING_SUGGESTIONS_ENABLED=false` set and the backend restarted, the "New from suggestion" button does not appear at all, while "New rule" (manual) still works.

```bash
git status --short
```
Confirm only this task's files are listed.

```bash
git commit \
  apps/frontend/package.json apps/frontend/bun.lock \
  apps/frontend/src/components/ui/command.tsx \
  apps/frontend/src/services/pricingSuggestionsService.ts \
  apps/frontend/src/lib/pricingSuggestions.ts \
  apps/frontend/src/lib/providerLogos.tsx \
  apps/frontend/src/components/pricing/ModelSuggestionPicker.tsx \
  apps/frontend/src/components/pricing/PricingFormDialog.tsx \
  apps/frontend/src/routes/_app.pricing.tsx \
  apps/frontend/src/i18n/en.json apps/frontend/src/i18n/id.json \
  -m "feat(frontend): Model Suggestion Picker for Pricing — OpenRouter-backed"
```

---

## Definition of Done

`/pricing` has a "New from suggestion" button that opens a searchable, provider-grouped picker of real OpenRouter models with logos; picking one pre-fills the existing create-rule dialog (still editable, still a normal save). The backend endpoint gracefully degrades (404 when disabled via config, 502 with a manual-entry fallback when OpenRouter is unreachable) — no other feature is affected by OpenRouter's availability. `PRICING_SUGGESTIONS_ENABLED=false` fully disables the feature. Both tasks verified live in a running browser, not just via build/lint/test (per this session's own `_app.projects.$projectId` Outlet lesson). Committed on `dev`, one commit per task, each commit scoped by explicit pathspec to avoid mixing with the concurrent Plan 05/06 session's staged files.
