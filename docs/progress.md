# RouterLens — Progress & Readiness

_Last updated: 2026-07-01. Living document — update as plans get written and executed._

## Current phase

**Implementation.** Design settled, execution underway on branch `dev`. Backend: foundation, shared kit, auth, and CRUD are in place and executed (Plans 01–04), then refactored to the adapter/platform layout (slog+tint logging, Echo-native server, Go 1.26 idioms). Frontend is a Vite + React SPA: app shell + design system (FE Plan 01), auth screens (FE Plan 02), and the projects/API-keys/pricing CRUD screens (FE Plan 03) are built and verified. Remaining dashboard screens (logs, settings) are still route placeholders.

---

## Snapshot

| Area | Status |
|------|--------|
| Design spec | ✅ Done |
| Project rules (`CLAUDE.md`) | ✅ Done (local-only, gitignored) |
| Domain glossary (`CONTEXT.md`) | ✅ Done (local-only, gitignored) |
| `README.md` + `.gitignore` | ✅ Done |
| Implementation plans | 🟡 5 of 8 written (01–05); 06 (analytics) and the FE CRUD-screens plan not yet written |
| Backend source code | 🟡 Plans 01–04 executed (foundation, shared kit, auth, CRUD); Plan 05 (event ingestion) written but not executed |
| Frontend source code | 🟡 FE Plans 01–03 executed (shell + design system, auth screens, projects/api-keys/pricing CRUD); logs/settings are still `<Placeholder>` routes |
| Migrations applied | ✅ 001–006 defined and applied via goose on boot |
| Tests run | 🟡 Per-plan unit/integration tests pass; no full-suite CI yet |
| Git repository | ✅ Initialized, on branch `dev` |

---

## Documents in place

- [x] `docs/superpowers/specs/2026-06-29-routerlens-design.md` — full design (architecture, data model, API, flows, quality gates, decisions, DoD)
- [x] `CLAUDE.md` — operative project rules + 18 canonical decisions (local-only)
- [x] `CONTEXT.md` — domain glossary (Event, Project, Session, observed/priced/unpriced, …) (local-only)
- [x] `README.md` — public-facing overview, quickstart, ingestion example
- [x] `.gitignore`
- [x] `docs/superpowers/plans/2026-06-29-routerlens-01-foundation.md`
- [x] `docs/superpowers/plans/2026-06-29-routerlens-02-shared-kit.md`
- [x] `docs/superpowers/plans/2026-06-29-routerlens-03-auth.md`
- [x] `docs/superpowers/plans/2026-06-29-routerlens-04-crud.md`
- [x] `docs/superpowers/plans/2026-06-29-routerlens-05-events.md` (written, not yet executed)
- [x] `docs/superpowers/plans/2026-06-30-routerlens-fe-01-foundation.md`
- [x] `docs/superpowers/plans/2026-07-01-routerlens-fe-03-crud.md`

---

## Plan roadmap (8 plans)

Each plan produces working, testable software on its own. Plans 04–08 are scoped but not yet written.

| Plan | Scope | Status | Delivers | Depends on |
|------|-------|--------|----------|------------|
| 01 | Foundation & Persistence | ✅ Executed | `docker compose up` boots Postgres + app, migrations 001–006 apply, `/healthz`+`/readyz`, server skeleton + SPA stub | — |
| 02 | Shared kit + security + cost calculator | ✅ Executed | argon2id, session token + API key, session cookie, i18n (EN/ID), validator (v10 + translator), pagination (offset+keyset), date-range, CSV, pure cost calculator — all TDD | 01 |
| 03 | Auth + first-run setup | ✅ Executed | setup-status → setup → login (httpOnly cookie) → me → logout, race-safe admin, localized errors | 01, 02 |
| 04 | Projects + API Keys + Pricing CRUD | ✅ Executed | CRUD behind the session middleware; the pricing repo Plan 05 reads for cost | 02, 03 |
| 05 | Event ingestion + logs + CSV | 🟡 Written, not executed | `POST /events` (Bearer API key, validated, idempotent, cost at ingest), keyset logs, streamed CSV | 02, 04 |
| 06 | Analytics endpoints | ⬜ Not written | overview / tokens / cost / latency (P95) / errors / providers / models, bounded date range | 05 |
| 07 | Frontend (now split into FE Plans 01–0N, Vite + React SPA) | 🟡 In progress | FE-01 ✅ shell + design system; FE-02 ✅ auth (setup/login/guard/user menu); FE-03 ✅ projects/api-keys/pricing CRUD screens; then logs → analytics screens | 03–06 |
| 08 | Embed + DoD | ⬜ Planned | `internal/web` `embed.FS` of the built frontend, final Dockerfile, seed, full verification | 07 |

> **Frontend pivot:** the FE is a **Vite + React SPA** (TanStack Router + Query, shadcn/ui on Base UI, Tailwind v4), tracked as its own `FE Plan NN` series (not TanStack Start). FE-01 (foundation/shell) and FE-02 (auth) are done; remaining screens follow.

---

## Implementation status

- **Backend packages (`apps/backend/internal/`):** `shared/{response,errors,i18n,pagination,validator,security,datetime,csv}` ✅; `domain/{user,project,apikey,pricing}` ✅, `domain/event` ⬜ (Plan 05); `usecase/{auth,project,apikey,pricing}` ✅, `usecase/event` ⬜ (Plan 05); `adapter/{postgres,http/{handler,middleware,dto}}` ✅ for the above, no event handler yet; `platform/{config,logging,bootstrap}` ✅ (Fx composition root, slog+tint). `go build ./...` clean.
- **Frontend (`apps/frontend/src/`):** shell/routing/i18n/design-system + `services/authService.ts` + auth routes (`setup`, `login`, `_app` guard) ✅. Projects, API keys, and pricing are real CRUD screens ✅ — `routes/_app.projects.tsx` (list/create/edit/delete/paginate), `routes/_app.projects.$projectId.tsx` (detail + per-project API key create/revoke with one-time plaintext reveal), `routes/_app.pricing.tsx` (list/create/edit/delete), sharing one `<DataTable>`/`<OffsetPagination>`/`<ConfirmDialog>`. The flat `/api-keys` placeholder and its nav entry are removed (keys are only ever managed from their owning project, matching the locked API surface). `routes/_app.{logs,settings}.tsx` remain `<Placeholder>` stubs.
- **Migrations:** 6 defined in Plan 01, all applied via goose on boot.
- **Tests:** defined and passing across Plans 01–04 (backend) and FE Plans 01–03 (frontend); no `llm_events`-path tests yet (Plan 05 not executed).

---

## Locked decisions (summary — authoritative copy in `CLAUDE.md` / the spec)

- **Single deployable monolith:** one Go binary serves the API + embedded TanStack Start frontend; same-origin. Postgres only — **no Redis**.
- **Auth:** dashboard = DB-backed **httpOnly session cookie** (`SameSite=Lax`, `Secure` in prod, revocable); ingestion = **Bearer API key** scoped to one project.
- **CSRF:** SameSite=Lax + no-state-change-on-GET (no token in v0.1). **No CORS** (same-origin; dev uses the Vite proxy).
- **i18n:** EN (default) + ID, resolved from `Accept-Language` only; error envelope carries `meta {lang, request_id, timestamp}`; validation via go-playground/validator/v10 translator.
- **Cost:** computed at ingest, stored with a price snapshot; unpriced `(provider, model)` → `cost_usd = NULL` (never `0`).
- **Analytics:** raw SQL over `llm_events`, bounded date range, `percentile_cont` P95, composite index — no rollups.
- **Logs:** keyset pagination. **Ingestion:** idempotent via optional `event_id`.
- **Tenancy:** single admin in v0.1; `owner_user_id` stamped for a clean multi-user upgrade later.

---

## Prerequisites to execute

- **Toolchain:** Go 1.26, Docker + Docker Compose, Node/Bun (frontend).
- **Go dependencies** (already in `go.mod`): Echo v4, pgx/v5, goose v3, godotenv, google/uuid, **go.uber.org/fx**, x/crypto, shopspring/decimal, go-playground validator/v10 + universal-translator + locales.
- **`.env`** from `.env.example` (Plan 01 Task 1).

---

## Deferred to v0.2 (explicit non-goals now)

Multi-user authorization (membership) · CSRF token (double-submit) · per-key ingestion rate-limiting · pre-aggregated rollups / materialized views · alerting · CORS config · GIN index on `metadata` · auto-prune job · multi-currency · model routing (never — out of product scope).

---

## Next actions

1. Execute **Plan 05** (event ingestion + logs + CSV) via `superpowers:subagent-driven-development` — already written, unblocks the FE logs page and Plan 06.
2. Write and execute **Plan 06** (analytics endpoints) — depends on 05.
3. ~~Write and execute an FE CRUD-screens plan (projects, api-keys, pricing)~~ — done as **FE Plan 03**, executed.
4. Then FE logs + analytics screens (depends on Plans 05/06 landing), then **Plan 08** (embed + DoD).
