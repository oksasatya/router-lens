# RouterLens — Progress & Readiness

_Last updated: 2026-06-30. Living document — update as plans get written and executed._

## Current phase

**Planning.** The design is settled and the first three implementation plans are written. **No source code exists yet** — the repository holds documentation and plans only. The project is ready to begin executing Plan 01.

---

## Snapshot

| Area | Status |
|------|--------|
| Design spec | ✅ Done |
| Project rules (`CLAUDE.md`) | ✅ Done (local-only, gitignored) |
| Domain glossary (`CONTEXT.md`) | ✅ Done (local-only, gitignored) |
| `README.md` + `.gitignore` | ✅ Done |
| Implementation plans | 🟡 3 of 8 written (01–03) |
| Backend source code | ⬜ Not started |
| Frontend source code | ⬜ Not started |
| Migrations applied | ⬜ Not yet (defined in Plan 01) |
| Tests run | ⬜ Not yet (defined per plan) |
| Git repository | ⬜ Not initialized (owner will run `git init`) |

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

---

## Plan roadmap (8 plans)

Each plan produces working, testable software on its own. Plans 04–08 are scoped but not yet written.

| Plan | Scope | Status | Delivers | Depends on |
|------|-------|--------|----------|------------|
| 01 | Foundation & Persistence | ✅ Written | `docker compose up` boots Postgres + app, migrations 001–006 apply, `/healthz`+`/readyz`, server skeleton + SPA stub | — |
| 02 | Shared kit + security + cost calculator | ✅ Written | argon2id, session token + API key, session cookie, i18n (EN/ID), validator (v10 + translator), pagination (offset+keyset), date-range, CSV, pure cost calculator — all TDD | 01 |
| 03 | Auth + first-run setup | ✅ Written | setup-status → setup → login (httpOnly cookie) → me → logout, race-safe admin, localized errors | 01, 02 |
| 04 | Projects + API Keys + Pricing CRUD | ⬜ Planned | CRUD behind the session middleware; the pricing repo Plan 05 reads for cost | 02, 03 |
| 05 | Event ingestion + logs + CSV | ⬜ Planned | `POST /events` (Bearer API key, validated, idempotent, cost at ingest), keyset logs, streamed CSV | 02, 04 |
| 06 | Analytics endpoints | ⬜ Planned | overview / tokens / cost / latency (P95) / errors / providers / models, bounded date range | 05 |
| 07 | Frontend (TanStack Start + shadcn/ui) | ⬜ Planned | setup/login + dashboard/logs/analytics/projects/keys/pricing/settings, api client + query hooks | 03–06 |
| 08 | Embed + DoD | ⬜ Planned | `internal/web` `embed.FS` of the built frontend, final Dockerfile, seed, full verification | 07 |

---

## Implementation status

Nothing built yet. The plans contain complete, runnable code for every step; executing them produces the code.

- **Backend packages:** 0 of (config, shared/{response,errors,i18n,pagination,validator,security,datetime,csv}, domain/{user,project,apikey,event,pricing}, application/*, infrastructure/{postgres,http}).
- **Frontend:** 0 (apps/web not scaffolded).
- **Migrations:** 6 defined in Plan 01, 0 applied.
- **Tests:** defined across Plans 01–03, 0 run.

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

- **Toolchain:** Go 1.26, Docker + Docker Compose, Node + pnpm (frontend, Plan 07).
- **`git init`** — owner runs this (the workspace is not yet a git repo; plans commit per task).
- **Go dependencies** (`go get`, listed in the plans): Echo v4, pgx/v5, goose v3, godotenv, google/uuid, x/crypto, shopspring/decimal, go-playground validator/v10 + universal-translator + locales.
- **`.env`** from `.env.example` (Plan 01 Task 1).

---

## Deferred to v0.2 (explicit non-goals now)

Multi-user authorization (membership) · CSRF token (double-submit) · per-key ingestion rate-limiting · pre-aggregated rollups / materialized views · alerting · CORS config · GIN index on `metadata` · auto-prune job · multi-currency · model routing (never — out of product scope).

---

## Next actions

1. Owner runs `git init` (+ initial commit of docs).
2. Execute **Plan 01** via `superpowers:subagent-driven-development` (fresh subagent per task, review between tasks), then **02**, then **03**.
3. Write **Plans 04–08** — recommended just-in-time (after the prior phase's real code exists) to minimize interface drift.
