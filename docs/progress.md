# RouterLens — Progress & Readiness

_Last updated: 2026-07-01. Living document — update as plans get written and executed._

## Current phase

**Implementation, nearly complete.** All eight backend/infra plans (01–06, 08) are executed, reviewed, and merged to `dev`. Frontend Plans 01–03 (shell/design system, auth, projects/API-keys/pricing CRUD) are executed. **FE Plan 04 (Request Logs + Analytics screens) is the only remaining v0.1 gap** — it's in progress in a separate session. Plan 08's Task 4 (full end-to-end Definition-of-Done walkthrough through the real UI) is written but deliberately deferred until FE Plan 04 lands, since it exercises those screens directly.

A post-v0.1 feature, **Pricing Suggestions** (OpenRouter-backed reference prices shown while creating a Pricing Rule), was scoped and started outside the original 8-plan roadmap — its backend half is executed; its frontend half is not yet built.

---

## Snapshot

| Area | Status |
|------|--------|
| Design spec | ✅ Done |
| Project rules (`CLAUDE.md`) | ✅ Done (local-only, gitignored) |
| Domain glossary (`CONTEXT.md`) | ✅ Done (local-only, gitignored) |
| `README.md` + `.gitignore` | ✅ Done |
| Implementation plans | ✅ All 8 written and executed (01–08); + a 9th, unplanned "Pricing Suggestions" plan (backend done, frontend pending) |
| Backend source code | ✅ Plans 01–06 + 08 (Tasks 1–3) executed: foundation, shared kit, auth, CRUD, event ingestion, analytics, embed/deploy/create-admin. Plan 08 Task 4 (DoD verification) pending FE Plan 04 |
| Frontend source code | 🟡 FE Plans 01–03 executed (shell + design system, auth screens, projects/api-keys/pricing CRUD); **FE Plan 04 (logs, analytics) in progress** — `/logs` still a placeholder, `/analytics` route not yet created |
| Migrations applied | ✅ 001–006 defined and applied via goose on boot |
| Tests run | ✅ Per-plan unit/integration tests pass (backend verified against real Postgres each plan); no full-suite CI yet |
| Git repository | ✅ On branch `dev`, pushed to `origin/dev` through Plan 08 |

---

## Documents in place

- [x] `docs/superpowers/specs/2026-06-29-routerlens-design.md` — full design (architecture, data model, API, flows, quality gates, decisions, DoD)
- [x] `docs/superpowers/specs/2026-07-01-pricing-suggestions-design.md` — design spec for the post-v0.1 Pricing Suggestions feature
- [x] `docs/adr/0001-pricing-suggestions-openrouter.md` — ADR for the OpenRouter outbound-call decision
- [x] `CLAUDE.md` — operative project rules + canonical decisions (local-only)
- [x] `CONTEXT.md` — domain glossary (Event, Project, Session, observed/priced/unpriced, …) (local-only)
- [x] `README.md` — public-facing overview, quickstart, ingestion example
- [x] `.gitignore`
- [x] `docs/superpowers/plans/2026-06-29-routerlens-01-foundation.md` — executed
- [x] `docs/superpowers/plans/2026-06-29-routerlens-02-shared-kit.md` — executed
- [x] `docs/superpowers/plans/2026-06-29-routerlens-03-auth.md` — executed
- [x] `docs/superpowers/plans/2026-06-29-routerlens-04-crud.md` — executed
- [x] `docs/superpowers/plans/2026-06-29-routerlens-05-events.md` — executed
- [x] `docs/superpowers/plans/2026-07-01-routerlens-06-analytics.md` — executed
- [x] `docs/superpowers/plans/2026-07-01-routerlens-08-embed-deploy.md` — Tasks 1–3 executed; Task 4 pending FE Plan 04
- [x] `docs/superpowers/plans/2026-06-30-routerlens-fe-01-foundation.md` — executed
- [x] `docs/superpowers/plans/2026-07-01-routerlens-fe-03-crud.md` — executed
- [x] `docs/superpowers/plans/2026-07-01-routerlens-fe-04-logs-analytics.md` — written; **in progress**
- [x] `docs/superpowers/plans/2026-07-01-pricing-suggestions-implementation.md` — 2 tasks (backend + frontend); backend executed, frontend pending

---

## Plan roadmap (8 plans, all written; 7 of 8 fully executed)

| Plan | Scope | Status | Delivers | Depends on |
|------|-------|--------|----------|------------|
| 01 | Foundation & Persistence | ✅ Executed | `docker compose up` boots Postgres + app, migrations 001–006 apply, `/healthz`+`/readyz`, server skeleton | — |
| 02 | Shared kit + security + cost calculator | ✅ Executed | argon2id, session token + API key, session cookie, i18n (EN/ID), validator (v10 + translator), pagination (offset+keyset), date-range, CSV, pure cost calculator | 01 |
| 03 | Auth + first-run setup | ✅ Executed | setup-status → setup → login (httpOnly cookie) → me → logout, race-safe admin, localized errors | 01, 02 |
| 04 | Projects + API Keys + Pricing CRUD | ✅ Executed | CRUD behind the session middleware | 02, 03 |
| 05 | Event ingestion + logs + CSV | ✅ Executed | `POST /events` (Bearer API key, validated, idempotent, cost at ingest), keyset logs, streamed CSV | 02, 04 |
| 06 | Analytics endpoints | ✅ Executed | overview / tokens / cost / latency (P95) / errors / providers / models, bounded date range, 5 queries serve 7 endpoints | 05 |
| 07 | Frontend (split into FE Plans 01–04, Vite + React SPA) | 🟡 In progress | FE-01 ✅ shell + design system; FE-02 ✅ auth; FE-03 ✅ projects/api-keys/pricing CRUD; **FE-04 🟡 logs + analytics screens, in progress in a separate session** | 03–06 |
| 08 | Embed + DoD | 🟡 Tasks 1–3 executed | Task 1 ✅ `internal/web` `embed.FS` + SPA fallback; Task 2 ✅ `-create-admin` CLI; Task 3 ✅ multi-stage Dockerfile (44.4MB final image) + seed script; **Task 4 ⬜ full DoD walkthrough, blocked on FE-04** | 07 |

> **Frontend pivot:** the FE is a **Vite + React SPA** (TanStack Router + Query, shadcn/ui on Base UI, Tailwind v4), tracked as its own `FE Plan NN` series (not TanStack Start).

### Unplanned addition: Pricing Suggestions (post-v0.1, outside the 8-plan roadmap)

A separate 2-task plan (`2026-07-01-pricing-suggestions-implementation.md`) adds an OpenRouter-backed
"pick a model, pre-fill the price" flow to the existing Pricing screen. **Backend (Task 1) is executed**
(`usecase/pricing.ListSuggestions`, `adapter/openrouter` client, `GET /api/v1/pricing/suggestions`
handler + DTO). **Frontend (Task 2) is not yet built.** This is additive to v0.1, not a blocker for it.

---

## Implementation status

- **Backend packages (`apps/backend/internal/`):** `shared/{response,errors,i18n,pagination,validator,security,datetime,csv}` ✅; `domain/{user,project,apikey,pricing,event}` ✅; `usecase/{auth,project,apikey,pricing,event}` ✅ (event usecase covers both ingestion and analytics); `adapter/{postgres,http/{handler,middleware,dto},openrouter}` ✅; `platform/{config,logging,bootstrap,web}` ✅ (Fx composition root, slog+tint, real frontend embed). `go build ./...` / `go vet ./...` / `gofmt -l .` clean; full suite green against real Postgres.
- **Frontend (`apps/frontend/src/`):** shell/routing/i18n/design-system + auth routes ✅. Projects, API keys, and pricing are real CRUD screens ✅ — `routes/_app.projects.tsx`, `routes/_app.projects.$projectId.tsx`, `routes/_app.pricing.tsx`, sharing `<DataTable>`/`<OffsetPagination>`/`<ConfirmDialog>`. Events/analytics data layer (`services/eventService.ts`, `services/analyticsService.ts`, `lib/events.ts`, `lib/analytics.ts`, `DateRangeFilter`, `CursorPagination`) is built and independently reviewed clean — **the screens that consume it (`routes/_app.logs.tsx`, `routes/_app.analytics.tsx`) are not yet built**; `/logs` is still a `<Placeholder>`, `/analytics` doesn't exist as a route file yet.
- **Migrations:** 6 defined in Plan 01, all applied via goose on boot.
- **Tests:** unit + integration suites pass across all executed backend plans, verified against a real Postgres each time (not just mocked); FE build (`bun run build`) verified clean through the data-layer work.
- **Deployment:** `docker compose build app` produces a working 3-stage image (bun frontend build → Go backend build with the frontend embedded → distroless final layer, 44.4MB). `make create-admin`, `make web-build`, `make seed` all work.

---

## Locked decisions (summary — authoritative copy in `CLAUDE.md` / the spec)

- **Single deployable monolith:** one Go binary serves the API + embedded Vite + React SPA frontend; same-origin. Postgres only — **no Redis**.
- **Auth:** dashboard = DB-backed **httpOnly session cookie** (`SameSite=Lax`, `Secure` in prod, revocable); ingestion = **Bearer API key** scoped to one project.
- **CSRF:** SameSite=Lax + no-state-change-on-GET (no token in v0.1). **No CORS** (same-origin; dev uses the Vite proxy).
- **i18n:** EN (default) + ID, resolved from `Accept-Language` only; error envelope carries `meta {lang, request_id, timestamp}`; validation via go-playground/validator/v10 translator.
- **Cost:** computed at ingest, stored with a price snapshot; unpriced `(provider, model)` → `cost_usd = NULL` (never `0`).
- **Analytics:** raw SQL over `llm_events`, bounded date range, `percentile_cont` P95, composite index — no rollups. 5 queries serve all 7 endpoints.
- **Logs:** keyset pagination. **Ingestion:** idempotent via optional `event_id`.
- **Tenancy:** single admin in v0.1; `owner_user_id` stamped for a clean multi-user upgrade later.

---

## Prerequisites to execute

- **Toolchain:** Go 1.26, Docker + Docker Compose, Bun (frontend).
- **Go dependencies** (already in `go.mod`): Echo v4, pgx/v5, goose v3, godotenv, google/uuid, **go.uber.org/fx**, x/crypto, shopspring/decimal, go-playground validator/v10 + universal-translator + locales.
- **`.env`** from `.env.example` (includes `LOG_LEVEL` as of Plan 08).

---

## Deferred to v0.2 (explicit non-goals now)

Multi-user authorization (membership) · CSRF token (double-submit) · per-key ingestion rate-limiting · pre-aggregated rollups / materialized views · alerting · CORS config · GIN index on `metadata` · auto-prune job · multi-currency · model routing (never — out of product scope).

---

## Next actions

1. **Finish FE Plan 04** (Request Logs + Analytics screens) — in progress in a separate session. Data layer (schemas/services/query-hooks/`DateRangeFilter`/`CursorPagination`) is already built and reviewed; remaining work is the two route components themselves (`_app.logs.tsx`, `_app.analytics.tsx`).
2. **Run Plan 08 Task 4** once FE Plan 04 lands: `docker compose up --build`, walk the full golden path (setup → login → project → API key → ingest → see it in Logs → see it in Analytics → configure pricing → CSV export → logout), then update this document's final status.
3. Optionally finish the Pricing Suggestions frontend task (Task 2 of its own plan) — additive, not a v0.1 blocker.
