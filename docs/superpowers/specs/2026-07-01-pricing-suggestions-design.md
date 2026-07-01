# Pricing Suggestions — Design Spec

> Feature: help an admin fill in a Pricing Rule faster by suggesting real-world prices from
> OpenRouter's public model list. See `docs/adr/0001-pricing-suggestions-openrouter.md` for the
> architectural decision behind the outbound call; see `CONTEXT.md` for the "Pricing Suggestion"
> glossary note (explicitly *not* a domain concept).

## Goal

On the existing Pricing page, add a way to create a new Pricing Rule by picking a model from a
searchable list (grouped by provider, with provider logos) instead of typing everything by
hand. Picking an entry pre-fills the existing create-rule form with provider, model, and
suggested input/output prices — all still editable — and the rule is saved through the exact
same create flow that exists today. Nothing is persisted until the admin saves.

**Out of scope for v1:** editing an existing rule from a suggestion (only new-rule creation);
normalizing/matching suggested provider/model strings against real ingested Events (a known,
pre-existing limitation — see "Known limitation" below).

## Terminology

- **Pricing Suggestion** — a third-party reference price, sourced from OpenRouter, shown while
  filling in the Pricing Rule form. Ephemeral: fetched on demand, cached briefly server-side,
  never stored in RouterLens's database. Explicitly not a "model catalog" (RouterLens's own
  glossary rules that out — see `CONTEXT.md`).
- **Pricing Rule** — unchanged, existing domain concept (`CONTEXT.md`).

## Architecture

```
Browser                          RouterLens backend                    OpenRouter
   |                                    |                                   |
   |  GET /api/v1/pricing/suggestions   |                                   |
   |----------------------------------->|                                   |
   |                                    | usecase/pricing.ListSuggestions() |
   |                                    |   -> cache hit? return cached     |
   |                                    |   -> cache miss/expired:          |
   |                                    |      adapter/openrouter.Client    |
   |                                    |----------------------------------->|
   |                                    |   GET /api/v1/models              |
   |                                    |<-----------------------------------|
   |                                    |   filter + transform + cache      |
   |<-----------------------------------|                                   |
   |  { items: PriceSuggestion[] }      |                                   |
```

### Backend (`apps/backend/internal/`)

**Config** (`platform/config`): `PricingSuggestionsEnabled bool` env `PRICING_SUGGESTIONS_ENABLED`,
default `true`.

**Usecase layer** (`usecase/pricing/pricing.go`, extend existing `Service`):
- New port, defined in the usecase package (not domain — this is an application-level
  integration, not a domain rule):
  ```go
  type PriceSuggestion struct {
      Provider, Model                string
      InputPricePer1M, OutputPricePer1M decimal.Decimal
  }
  type SuggestionSource interface {
      List(ctx context.Context) ([]PriceSuggestion, error)
  }
  ```
- `Service.ListSuggestions(ctx) ([]PriceSuggestion, error)` — returns `nil, ErrSuggestionsDisabled`
  (a sentinel, mapped to 404) when config is off; otherwise delegates to the injected
  `SuggestionSource`.
- `internal/domain/pricing` is untouched by this feature — no new domain types, no new
  repository methods.

**Adapter** (new package `adapter/openrouter/client.go`):
- Implements `pricing.SuggestionSource`.
- HTTP client: 5s timeout, one retry on transient network error, response body capped (e.g. 5MB)
  to avoid an unbounded read.
- In-memory cache: `sync.Mutex` + `time.Time` last-fetch + 1-hour TTL. `singleflight.Group`
  (stdlib-adjacent, `golang.org/x/sync/singleflight`) so concurrent callers during a cache miss
  share one in-flight OpenRouter request rather than firing N.
- On fetch failure: if a previous successful fetch exists (even if stale), log a warning and
  serve it. If there has never been a successful fetch, return the error (caller maps to 502).
- **Filtering + transform** (the part Codex's review specifically flagged as under-specified):
  - Skip entries whose `id` doesn't match `provider/model` shape (drop `~`-prefixed aliases and
    anything without exactly one `/`).
  - Skip entries where `pricing.prompt` or `pricing.completion` is `"-1"`, missing, or fails to
    parse as a non-negative decimal.
  - Multiply per-token price by 1,000,000 to get price-per-1M-tokens (matching RouterLens's
    existing `pricing_rules` unit).
  - Truncate/skip provider or model strings exceeding RouterLens's existing validation limits
    (provider max 100 chars, model max 200 chars — same limits `PricingRequest` already
    enforces) rather than letting an oversized string reach the frontend only to fail on save.

**HTTP** (`adapter/http/handler/pricing_handler.go`, extend existing `PricingHandler`):
- `GET /pricing/suggestions` (session-protected, registered alongside the other pricing routes).
- 200 with `{ items: PriceSuggestionResponse[] }` (plain array, no pagination — same shape
  precedent as the existing `GET /pricing`).
- 404 when suggestions are disabled by config.
- 502 when OpenRouter is unreachable and there is no cache to fall back on.

### Frontend (`apps/frontend/src/`)

- `services/pricingSuggestionsService.ts` — `listPricingSuggestions()`, returns `[]` gracefully
  on a 404 (disabled) so the UI can just hide the entry point; surfaces other errors normally.
- `lib/pricingSuggestions.ts` — `pricingSuggestionsQueryOptions` (`staleTime` ~1 hour, matching
  the backend's own cache window — no reason to refetch more often).
- `lib/providerLogos.ts` — a small map of provider slug → Simple Icons SVG (bundled, not
  fetched), covering the common providers OpenRouter lists (openai, anthropic, google,
  meta-llama, mistralai, x-ai, cohere, deepseek, qwen, and a handful more); unlisted providers
  fall back to a generic icon (lucide `Bot`).
- `components/pricing/ModelSuggestionPicker.tsx` — shadcn `Command` dialog: search-as-you-type,
  `CommandGroup` per provider (logo in the group label), each item shows model name + a compact
  price preview (e.g. "$2.50 / $10.00 per 1M"). Selecting an entry calls `onSelect(suggestion)`
  and closes.
- **Entry point:** on the Pricing page, a `"New from suggestion"` button sits next to the
  existing `"New rule"` button (not a generic "Browse models" — the CTA names exactly what it
  does, and only appears next to rule *creation*, matching the new-rule-only scope). If
  suggestions are unavailable (disabled or fetch failed), this button is hidden entirely —
  `"New rule"` (manual entry) always works regardless.
- Selecting a suggestion opens the *existing* `PricingFormDialog` in create mode, with `provider`
  /`model`/`input_price_per_1m`/`output_price_per_1m` pre-filled from the suggestion. The dialog
  and its save flow are otherwise unchanged — a prefilled create is still a normal create.
- The picker shows a persistent, quiet caption: *"Reference prices from OpenRouter — confirm the
  provider/model text matches exactly what your router reports before saving."* (addresses the
  known limitation below without gating the flow behind a dismissable modal).

## Known limitation (pre-existing, not solved here)

Provider/Model on a real ingested Event are whatever the upstream router reports verbatim — no
normalization. A rule created from a suggestion (e.g. `openai` / `gpt-4o`) will not match an
Event whose gateway reports different casing or a dated model version (e.g. `OpenAI` /
`gpt-4o-2024-08-06`), and that Event stays Unpriced. This already happens today with fully
manual entry — Pricing Suggestions doesn't create the problem, but it can make it more
surprising ("the app knew this model, why is it still unpriced?"), hence the caption above.

## Error handling summary

| Condition | Backend | Frontend |
|---|---|---|
| Suggestions disabled by config | 404 | Hide "New from suggestion" entirely |
| OpenRouter unreachable, stale cache exists | 200 (stale data, logged warning server-side) | Renders normally |
| OpenRouter unreachable, no cache at all | 502 | Hide the button, `"New rule"` still works |
| Malformed OpenRouter entry | Filtered out server-side, never reaches the response | — |

## TDD verdict (§16)

- OpenRouter response filtering/transform (id-splitting, price×1M conversion, validation) —
  **TDD: yes.** Pure function, clear input→output, exactly the kind of boundary-condition logic
  (the `-1`/alias/oversized-string cases) that's easy to get subtly wrong.
- Cache TTL + singleflight behavior — **TDD: yes**, with an injected clock so expiry is testable
  without real sleeps.
- HTTP handler wiring, frontend picker/dialog — **TDD: no** (thin wiring + visual composition,
  matches the rest of FE-03's precedent); verified by running the app, per the `verify` skill's
  standard (build/lint passing is not sufficient evidence — this plan learned that lesson
  directly from the `_app.projects.$projectId` Outlet bug found in this same session).

## Global constraints carried from the project

- Tailwind v4 only; Base UI `render={<Component />}` composition, never `asChild`.
- Sonar-Go: `go:S107` ≤7 params, `go:S3776` cognitive ≤15, `go:S1192` const for 3×-duplicated
  literals, errcheck, gosec (no hardcoded secrets — `PRICING_SUGGESTIONS_ENABLED` and any future
  OpenRouter base-URL override are config, not literals).
- Sonar-TS/React: readonly props, stable list keys, `globalThis` not `window`.
- No new frontend runtime dependency beyond the bundled Simple Icons SVGs (static assets, not a
  new package — pick the specific icons needed rather than pulling in the whole icon library).
- Backend gets exactly one new third-party Go dependency if `golang.org/x/sync/singleflight`
  isn't already indirectly available — verify before adding; it's a `golang.org/x` module
  (same trust tier as `x/crypto`, already a dependency).
