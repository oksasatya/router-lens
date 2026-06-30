# RouterLens — Design Spec v0.1 (MVP)

- **Status:** approved for planning
- **Date:** 2026-06-29
- **Tagline:** Open-source observability dashboard for LLM routers and AI coding agents.
- **Companion docs:** `CLAUDE.md` (operative project rules), `CONTEXT.md` (domain glossary).

---

## 1. Overview & goals

RouterLens is a **self-hosted observability dashboard** for LLM routers, AI gateways, and AI
coding agents (9router, LiteLLM, OpenRouter proxy, Claude Code proxy, Agent Zero, custom
gateways). It ingests one record per LLM call (an **Event**) and turns the stream into request
logs, token/cost/latency/error analytics, and provider/model breakdowns.

RouterLens is **not** a router and contains **no** routing logic. It only observes.

**Primary goal of v0.1:** a developer can run `docker compose up`, complete first-run setup,
create a Project and an API Key, point a router/agent at the ingestion endpoint, and immediately
see logs and basic analytics with auto-computed estimated cost.

---

## 2. Scope

### In scope (v0.1)
User auth (session cookie) · first-run setup · Project CRUD · API Key management · Event
ingestion · request logs · token/cost/latency/error analytics · provider/model breakdown ·
pricing configuration · estimated-cost auto-calculation · CSV export of logs.

### Out of scope / non-goals (deliberate — YAGNI)
Redis · model routing · multi-user authorization · pre-aggregation (rollups / materialized
views / TimescaleDB) · realtime streaming · alerting/notifications · full per-key rate-limiting
(baseline hardening still ships, §12) · GIN index on metadata · signup/register flow ·
auto-prune job · multi-currency.

---

## 3. Glossary

Canonical domain language lives in `CONTEXT.md`. Key terms: **Event** (the atomic observed LLM
call; entity `llm_events`), **Request Logs** (the UI view of Events, not a separate entity),
**Project** (namespace owning Events + API Keys), **User**/**Session** (dashboard human auth),
**API Key** (machine ingestion auth, scoped to one Project), **Provider/Model/Route Source/Agent**
(observed dimensions), **observed/priced/unpriced** (model state), **Pricing Rule**,
**Estimated Cost**, **Price Snapshot**.

---

## 4. Architecture

Hexagonal + DDD + Clean Architecture. Dependencies point **inward only**:

```
cmd/server (main.go — Uber Fx: per-module providers + fx.Lifecycle hooks)
  → infrastructure  (postgres repo impls; echo http: handlers, middleware, router)
    → application   (use cases: orchestrate domain + ports; no HTTP knowledge)
      → domain      (entities, value objects, repository INTERFACES, domain rules, cost calculator)
```

**Hard layering rules**
- `domain/` imports nothing from Echo, `database/sql`, pgx, or `infrastructure/`. Pure Go + stdlib.
- Repository interfaces are defined in `domain/<aggregate>/repository.go`; implementations live in `infrastructure/postgres/`.
- Handlers only parse → call use case → write response. No business logic, no DB access in handlers.
- Use cases never touch `echo.Context` or HTTP status codes; they take/return domain types or DTOs.
- Wiring uses **Uber Fx** in `cmd/server/main.go`: per-module provider constructors + `fx.Lifecycle` (`OnStart`/`OnStop`) for server start/stop and pool cleanup; `fx.Run` handles signals. Constructors stay plain so they remain unit-testable without Fx.

### Folder layout

```
apps/backend/                  Go module `router-lens` (go.mod here; imports stay router-lens/internal/...)
  cmd/server/main.go           Fx app entrypoint
  go.mod, go.sum, .env.example, Dockerfile
  internal/
    app/          config.go (+ bootstrap)
    domain/
      user/         entity.go, repository.go
      project/      entity.go, repository.go
      apikey/       entity.go, repository.go
      event/        entity.go, value_object.go, repository.go
      pricing/      entity.go, repository.go, calculator.go
    application/
      auth/         setup_usecase.go, login_usecase.go, logout_usecase.go (me via handler)
      project/      create_, list_, get_, update_, delete_project_usecase.go
      apikey/       create_, list_, revoke_apikey_usecase.go
      event/        ingest_, list_, get_, export_event_usecase.go, analytics_usecase.go
      pricing/      list_, upsert_, update_, delete_pricing_usecase.go, calculate_cost.go
    infrastructure/
      postgres/     db.go, migrate.go, user_repository.go, project_repository.go, apikey_repository.go,
                    event_repository.go, pricing_repository.go
      http/         server.go, router.go,
                    middleware/  session_middleware.go, apikey_middleware.go,
                                 error_middleware.go, lang_middleware.go
                    handler/     setup_handler.go, auth_handler.go, project_handler.go,
                                 apikey_handler.go, event_handler.go, analytics_handler.go,
                                 pricing_handler.go
    shared/
      response/     response.go
      errors/       errors.go
      pagination/   offset.go, keyset.go
      i18n/         i18n.go (Lang, Resolve, error-code catalog)
      validator/    validator.go (go-playground v10 + EN/ID translator)
      security/     password.go, token.go (session + API key), cookie.go
      datetime/     range.go
      csv/          exporter.go
    web/            embed.go (//go:embed the built frontend) + SPA fallback handler
  migrations/       NNN_<name>.sql (goose single-file: -- +goose Up / -- +goose Down)
apps/frontend/      TanStack Start app (built to static, embedded into the Go binary)
docker-compose.yml, Makefile, README.md, docs/
```

---

## 5. Data model

UUID primary keys. `NUMERIC` for money. `timestamptz` (UTC) for time. Every migration ships
`.up.sql` and `.down.sql`.

```
users
  id              uuid pk
  email           text unique not null
  password_hash   text not null            -- argon2id
  name            text
  created_at      timestamptz not null default now()
  updated_at      timestamptz not null default now()

sessions
  id              uuid pk
  user_id         uuid not null references users(id) on delete cascade
  token_hash      text unique not null     -- sha256 of the opaque cookie token
  expires_at      timestamptz not null
  user_agent      text
  ip              inet
  created_at      timestamptz not null default now()
  index (token_hash), index (expires_at)

projects
  id              uuid pk
  owner_user_id   uuid not null references users(id)   -- stamped now; shared visibility v0.1
  name            text not null
  slug            text not null
  description     text
  created_at      timestamptz not null default now()
  updated_at      timestamptz not null default now()
  unique (owner_user_id, slug)

api_keys
  id              uuid pk
  project_id      uuid not null references projects(id) on delete cascade
  name            text not null
  key_hash        text unique not null     -- sha256 of the secret; plaintext shown once
  key_prefix      text not null            -- e.g. "rl_live_ab12" for display
  last_used_at    timestamptz
  revoked_at      timestamptz              -- soft delete
  created_at      timestamptz not null default now()
  index (key_hash)

pricing_rules
  id                    uuid pk
  provider              text not null
  model                 text not null
  input_price_per_1m    numeric not null   -- USD per 1,000,000 input tokens
  output_price_per_1m   numeric not null
  currency              text not null default 'USD'
  created_at            timestamptz not null default now()
  updated_at            timestamptz not null default now()
  unique (provider, model)

llm_events
  id                    uuid pk
  project_id            uuid not null references projects(id) on delete cascade
  event_id              text                -- optional client idempotency key
  provider              text not null
  model                 text not null
  route_source          text
  agent                 text
  input_tokens          bigint not null     -- >= 0
  output_tokens         bigint not null     -- >= 0
  cost_usd              numeric             -- NULL when unpriced (never 0)
  input_price_1m        numeric             -- price snapshot; NULL when unpriced
  output_price_1m       numeric             -- price snapshot; NULL when unpriced
  latency_ms            integer             -- >= 0
  status_code           integer             -- 100..599
  is_error              boolean not null    -- computed at ingest, stored
  error_message         text
  request_started_at    timestamptz not null   -- client-reported
  request_finished_at   timestamptz
  received_at           timestamptz not null default now()  -- server-authoritative
  metadata              jsonb               -- capped ~8KB
  created_at            timestamptz not null default now()
  unique (project_id, event_id)             -- partial: where event_id is not null
  index (project_id, request_started_at desc, id desc)   -- analytics range + keyset paging
```

**Indexing note (§8):** the single composite index `(project_id, request_started_at desc, id desc)`
serves both bounded-range analytics and keyset log pagination. Additional indexes (e.g. on
`is_error`, `provider`, `model`) are added only when a query proves slow against real volume —
not preemptively.

---

## 6. Domain model

- **User** — identity for dashboard auth. Rule: password verified via argon2id.
- **Project** — owns API Keys and Events; carries `owner_user_id`.
- **APIKey** — value: only the hash is stored; plaintext is generated once. Rule: revoked keys cannot ingest.
- **Event** — immutable observed call. Value object **TokenUsage** (`input_tokens`, `output_tokens`). Rule: `is_error` derived once at construction.
- **PricingRule** — price per `(provider, model)`.
- **Cost calculator** (`domain/pricing/calculator.go`) — pure function:
  `CalculateCost(usage TokenUsage, rule *PricingRule) (cost *Money, snapshot *PriceSnapshot)`.
  Returns `nil` cost + `nil` snapshot when `rule == nil` (unpriced). No HTTP, no DB, no I/O.

Repository interfaces (in `domain/`): `UserRepository`, `ProjectRepository`, `APIKeyRepository`,
`EventRepository` (incl. keyset list + analytics aggregate queries), `PricingRepository`.

---

## 7. Use cases (application layer)

- **auth:** `Setup` (race-safe first admin), `Login`, `Logout`, `Me`.
- **project:** `Create`, `List`, `Get`, `Update`, `Delete`.
- **apikey:** `Create` (returns plaintext once), `List`, `Revoke`.
- **event:** `Ingest` (validate → look up pricing → calculate cost → insert idempotently),
  `List` (keyset), `Get`, `Export` (stream), `Analytics` (overview/tokens/cost/latency/errors/providers/models).
- **pricing:** `List`, `Upsert`, `Update`, `Delete`.

Use cases depend only on domain repository interfaces + the cost calculator. No HTTP, no SQL.

---

## 8. HTTP API

Base path `/api/v1`. JSON envelope carries a `meta` block on every response:

```json
// success
{ "data": { }, "meta": { "lang": "en", "request_id": "…", "timestamp": "2026-06-29T10:00:00Z" } }
// error
{ "error": { "code": "validation_failed", "message": "<localized>", "details": { "email": "<localized>" } },
  "meta": { "lang": "id", "request_id": "…", "timestamp": "…" } }
```

`message` and validation `details` are **localized** (EN default + ID) by the error `code` through
the `i18n` catalog (and the go-playground/validator/v10 translator for field errors). Language is
resolved from the `Accept-Language` header (RFC 7231 content negotiation; no query/cookie override
in v0.1), falling back to EN.

Two auth boundaries: **session cookie** for dashboard routes, **API Key header** for ingestion only.

| Method | Path | Auth | Notes |
|--------|------|------|-------|
| POST | `/setup` | none | Creates first admin; **403** once any user exists; race-safe insert |
| POST | `/auth/login` | none | email+password → sets session cookie |
| POST | `/auth/logout` | session | deletes session row + clears cookie |
| GET | `/auth/me` | session | current user |
| POST | `/projects` | session | |
| GET | `/projects` | session | offset pagination |
| GET | `/projects/:id` | session | |
| PUT | `/projects/:id` | session | |
| DELETE | `/projects/:id` | session | |
| POST | `/projects/:projectId/api-keys` | session | returns plaintext key once |
| GET | `/projects/:projectId/api-keys` | session | prefix only |
| DELETE | `/api-keys/:id` | session | soft revoke |
| POST | `/events` | **API Key** | ingestion; **no body `project_id`** (derived from key); 202 / 202+deduplicated |
| GET | `/events` | session | keyset pagination; filters: project, date range, provider, model, status |
| GET | `/events/:id` | session | |
| GET | `/events/export.csv` | session | streamed, range-bounded, formula-injection-safe |
| GET | `/analytics/overview` | session | date range required |
| GET | `/analytics/{tokens,cost,latency,errors,providers,models}` | session | date range required |
| GET | `/pricing` | session | |
| POST | `/pricing` | session | upsert by (provider, model) |
| PUT | `/pricing/:id` | session | |
| DELETE | `/pricing/:id` | session | |

### Ingestion payload (`POST /events`)
```json
{
  "event_id": "optional-idempotency-key",
  "provider": "anthropic",
  "model": "claude-sonnet-4-5",
  "route_source": "9router",
  "agent": "claude-code",
  "input_tokens": 12000,
  "output_tokens": 1800,
  "latency_ms": 8420,
  "status_code": 200,
  "error_message": null,
  "request_started_at": "2026-06-29T10:00:00Z",
  "request_finished_at": "2026-06-29T10:00:08Z",
  "metadata": { "session_id": "optional", "workspace": "nuvora", "fallback_from": null }
}
```
`project_id` is **derived from the API Key** and must not appear in the body.

### Analytics responses (shape)
- `overview` — total requests, total input/output tokens, total estimated cost, avg latency, P95 latency, error rate, most-used provider/model, most-expensive model, top projects by usage.
- `tokens` / `cost` — time series bucketed by `date_trunc` over the range.
- `latency` — avg + P95 (`percentile_cont(0.95)`), optionally bucketed.
- `errors` — error timeline + error rate.
- `providers` / `models` — distribution (count, tokens, cost) grouped by dimension.

All analytics endpoints **require a bounded date range**: default = last 24h, hard max = 90 days.

---

## 9. Cross-cutting flows

**First-run setup.** On load, the app checks whether a User exists (`COUNT(users)=0`). If none,
the frontend routes to `/setup`; `POST /api/v1/setup` creates the admin inside a transaction
using `INSERT ... SELECT ... WHERE NOT EXISTS (SELECT 1 FROM users)` plus the `users.email`
unique constraint, so concurrent setup requests cannot create two admins. Once a User exists the
endpoint returns 403 and `/setup` redirects to login.

**Auth / Session.** Login verifies argon2id, generates an opaque random token (`crypto/rand`),
stores `sha256(token)` in `sessions`, and sets the cookie: `HttpOnly` always, `Secure` in
production, `SameSite=Lax` (same-domain) or `SameSite=None; Secure` (cross-domain), driven by
config. `me` looks up the session by hash and checks expiry. `logout` deletes the row and clears
the cookie. Tokens are never placed in localStorage/sessionStorage.

**Ingestion.** `apikey_middleware` hashes the presented key, looks it up, rejects if missing or
revoked, updates `last_used_at`, and injects `project_id`. The use case validates the payload
(§ validation below), looks up the Pricing Rule for `(provider, model)`, computes Estimated Cost
+ Price Snapshot (or leaves them NULL when unpriced), derives `is_error`, and inserts with
`ON CONFLICT (project_id, event_id) DO NOTHING`. New → `202`; duplicate → `202` with
`{ "deduplicated": true }`.

**Validation (at ingest boundary).** Reject: `input_tokens`/`output_tokens` negative or beyond a
sane max; `latency_ms < 0`; `status_code` outside 100–599; `request_started_at` in the future or
older than `MAX_BACKDATE` (config, default 7 days); over-long `provider`/`model`/`agent`/
`route_source`; `metadata` larger than ~8 KB. Validation errors use the consistent error envelope.

**Analytics.** Raw SQL over `llm_events`, always `WHERE project_id = $1 AND request_started_at
BETWEEN $2 AND $3`, with `GROUP BY date_trunc(...)` and `percentile_cont(0.95)` for P95, riding
the composite index. The bounded range is mandatory (§8 API).

**Logs (keyset).** `WHERE project_id=$1 AND (request_started_at, id) < ($cursorTime, $cursorId)
ORDER BY request_started_at DESC, id DESC LIMIT $n`. The cursor encodes `(request_started_at, id)`.

**CSV export.** Streams `csv.Writer` rows directly to the response, bounded by the date range.
Any cell beginning with `= + - @` is prefixed with `'` to prevent spreadsheet formula injection.

---

## 10. Reusable shared packages

`response` (envelope + `meta` builder + localization) · `errors` (`AppError` with i18n `Code`,
sentinels, central Echo error handler) · `i18n` (`Lang` type, `Resolve`, error-code → `{en,id}`
catalog; pure) · `pagination` (`offset` for CRUD, `keyset` for events incl. cursor encode/decode) ·
`validator` (go-playground/validator/v10 + universal-translator, EN + ID; localized field→message) ·
`security` (`password` argon2id, `session_token`, `apikey` generate+hash, `cookie` builder) ·
`datetime` (range parser: `from`/`to` ISO or `preset=24h|7d|30d`, applies
default + max-window) · `csv` (streaming, injection-safe exporter). No duplicated logic for any of
these concerns anywhere else.

---

## 11. Frontend (apps/web — TanStack Start)

**Routes:** `__root.tsx`, `setup.tsx`, `login.tsx`, `dashboard.tsx` (overview), `logs.tsx`,
`providers.tsx`, `models.tsx`, `projects.tsx`, `api-keys.tsx`, `pricing.tsx`, `settings.tsx`.

**Data layer (no duplication):**
- One typed API client `lib/api.ts` (`fetch` wrapper, `credentials: "include"`, parses the
  `{data}`/`{error}` envelope). Every request goes through it.
- Per-domain service (`services/authService.ts`, `projectService.ts`, `eventService.ts`,
  `analyticsService.ts`, `pricingService.ts`) calling the client — endpoints defined once here.
- TanStack Query hooks (`hooks/useAuth.ts`, `useProjects.ts`, `useEvents.ts`, `useAnalytics.ts`,
  `usePricing.ts`) own caching/invalidation. Components never call `fetch` directly.
- Formatting helpers reusable: `lib/money.ts`, `lib/token.ts`, `lib/date.ts`, `lib/format.ts`.

**Component system:** **shadcn/ui** — Radix primitives + Tailwind v4 + CSS-variable theming
(`:root`/`.dark`) + the `cn()`/`cva` pattern, with `components.json` at `apps/web/`. Built on top of
it: `<DataTable>` (filter + keyset paging), `<DateRangePicker>`, `<StatCard>`, `<ChartCard>`, plus
`ui/` (shadcn primitives), `layout/`, `dashboard/`, `logs/`, `charts/`, `forms/`. Charts use the
shadcn Chart component (Recharts-based).

**Auth state** comes from `GET /api/v1/auth/me` via TanStack Query — never from storage.
Unpriced models/Events render an explicit "unpriced" badge rather than `$0`.

**Serving:** the frontend is built to static assets and **embedded into the Go binary** (`internal/web`
via `embed.FS`) with an SPA fallback (any non-`/api`, non-asset path serves `index.html`). UI and API
are therefore **same-origin** in the default deployment. In development the frontend may run on its own
Vite dev server proxying `/api` to the Go server for hot reload.

**Aesthetic:** clean, modern, simple developer-tool dashboard. Behind-auth, so the performance
gate applies; the SEO gate does not.

---

## 12. Security

- **Passwords:** argon2id. **Session tokens & API keys:** `crypto/rand`, stored only as `sha256`.
- **CSRF (SameSite, no token in v0.1):** the deployment is same-origin, so the session cookie uses
  `SameSite=Lax` and every state change uses POST/PUT/DELETE (never GET). `SameSite=Lax` keeps the
  session cookie off cross-site mutating requests, which covers CSRF for this self-hosted threat
  model; a double-submit CSRF token is deferred to v0.2 (split-origin or untrusted multi-user only).
  The API-key ingestion route is cookie-exempt.
- **Baseline hardening (v0.1):** Echo `BodyLimit` (≈64 KB on ingestion), server read/write
  timeouts, bounded DB connection pool. Full per-key rate-limiting deferred to v0.2.
- **No CORS:** the monolith is same-origin; the dev frontend reaches the API through the Vite proxy.
  CORS is not configured (re-add a small middleware later only if a split-origin deploy is needed).
- **Secrets:** never log API keys, passwords, session tokens, or raw `metadata`. `gitleaks` clean.
- **SQL:** parameterized only (`$1`...). No string-built queries.

---

## 13. Quality gates

### Sonar guardrails (write compliant from the first commit)
```
Go:
- go:S107 — ≤7 params (project preference ≤5; 6+ = smell → Deps/Opts struct from the start).
- go:S3776 — cognitive complexity ≤15 → extract helpers; tests use t.Run subtests to reset the budget.
- go:S1192 — const for any string literal duplicated 3+ times (error messages, path segments, keys).
- errcheck — handle every returned error; never `_ = fallible()`. Wrap with %w; sentinel + errors.Is/As.
- gosec — no hardcoded secrets, parameterized SQL only, crypto/rand for tokens.

TypeScript / React:
- typescript:S6759 — React props readonly.
- typescript:S7764 — `globalThis` not `window` (with `?.` for SSR).
- typescript:S3358 — no nested ternaries.
- typescript:S3923 — no identical-branch ternaries.
- typescript:S4624 — no nested template literals.
- typescript:S6582 — prefer optional chaining `?.`.
- typescript:S7755 — `arr.at(-1)` over `arr[arr.length - 1]`.
- typescript:S6819 — real elements over ARIA roles.
- typescript:S1874 — no deprecated APIs (zod v4: `z.email(...)`).
- typescript:S6606 — prefer `??=`.
- typescript:S6479 — stable list keys (never array index).

When fixing one instance, check sibling files for the same anti-pattern and fix-forward.
Review the diff against this list BEFORE marking compliant.
```

### Algorithmic complexity (§8)
The hot path is `llm_events`. Every analytics/log query is bounded by a date range and rides the
`(project_id, request_started_at desc, id desc)` index → range scan is `O(log n + k)` where `k` is
rows in range; P95 sort within range is `O(k log k)`; keyset paging is `O(log n + limit)`. The
mandatory range cap bounds `k`. No N+1, no SELECT-in-loop, no offset paging on events.
User-facing complexity explanations are written in Bahasa Indonesia.

### TDD fit (verdict per work area)
- **TDD = YES (test first):** cost calculator, time-range parser, keyset cursor encode/decode,
  ingestion validators, argon2id hash/verify + session-token + API-key hashing. Clear
  input→output contracts; money/security/parsers.
- **TDD = NO (tests after + normal-path regression test):** SQL analytics aggregations (integration
  tests against real Postgres / testcontainers), HTTP handlers (integration), all frontend
  visual/layout work (verify by running + `react-doctor`).

### Verification routine
Backend: `gofmt → go vet → golangci-lint → go test -race -cover → govulncheck`.
Frontend: build + `react-doctor` (lint/a11y/bundle) before commit.

---

## 14. Decisions & rationale (ADR-worthy log)

1. **Cost computed at ingest, with a price snapshot on the Event.** *Alternatives:* compute at
   query time by joining `pricing_rules`. *Chosen* because analytics becomes plain `SUM(cost_usd)`
   with no join (simpler, faster at scale) and historical cost stays explainable even after a
   Pricing Rule changes. *Trade-off:* a wrong price is not retroactively corrected — it needs a
   one-off backfill recompute. Unpriced pairs store `NULL` cost (never `0`).
2. **Raw analytics, no pre-aggregation.** *Alternatives:* rollup tables / materialized views /
   TimescaleDB. *Chosen* because raw `GROUP BY`/`percentile_cont` over an indexed, range-bounded
   table is correct and simple for MVP volumes. *Trade-off:* very large ranges over very large
   tables will eventually need pre-aggregation — deferred until a real query proves slow.
3. **DB-backed sessions over stateless JWT.** Real server-side revocation (logout, leaked cookie)
   for a few-user self-hosted tool, at the cost of one indexed lookup per request. No Redis needed.
4. **Single-user now, multi-user-shaped schema.** `owner_user_id` is stamped from day 1 so a
   future `project_members` table is additive — avoiding an ownership backfill migration later.
5. **API Key derives `project_id`; body `project_id` forbidden.** Removes a cross-project write
   footgun for zero MVP benefit.
6. **First-run setup page (no signup).** Avoids a stale-env "default admin recreated" failure mode;
   the endpoint locks after the first User. Race-safe via conditional insert + unique email.

Separate `docs/adr/` files are intentionally not created for v0.1 — this log is the record. Extract
to formal ADRs if/when the project grows.

---

## 15. Definition of Done

`docker compose up` brings up Postgres + Go/Echo backend + TanStack Start frontend; migrations
apply (on boot and via `make migrate`). First-run setup creates the admin; login/logout works via
httpOnly cookie. A User can create a Project, create an API Key (plaintext shown once), ingest an
Event with that key (validated, idempotent, estimated cost auto-computed from pricing with a price
snapshot, or marked unpriced), see it in Request Logs (keyset paging), view basic analytics
(overview/tokens/cost/latency/errors/providers/models with a date range), and configure Pricing
Rules. CSV export streams safely. No duplicated logic for the same concern; reusable
helpers/services are extracted per §10/§11. `.env.example`, a curl example for ingestion, optional
seed data, and a clear README are present.

---

## 16. Build order (for writing-plans)

1. Monorepo scaffold + `go.mod` + `docker-compose.yml` (Postgres) + `.env.example` + Makefile.
2. Migrations 001–00N (users, sessions, projects, api_keys, pricing_rules, llm_events) + DB connection.
3. Shared packages (response, errors, pagination, validator, security, datetime, csv) — TDD where YES.
4. Domain + repositories (entities, value objects, repo interfaces, cost calculator) — TDD the calculator.
5. Auth + first-run setup (session middleware, session cookie) — TDD security helpers.
6. Project + API Key use cases, handlers, repos.
7. Event ingestion (API-key middleware, validation, cost calc, idempotent insert) — TDD validators.
8. Request logs (keyset list, get, CSV export).
9. Analytics use cases + raw SQL (overview/tokens/cost/latency/errors/providers/models) — integration tests.
10. Pricing CRUD.
11. Frontend: api client + services + query hooks → setup/login → dashboard → logs → analytics → projects/keys/pricing/settings.
12. Refactor pass: confirm zero duplicated logic; README + seed + curl example; full verification routine.
