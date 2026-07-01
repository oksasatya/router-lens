# RouterLens Plan 08 — Embed, Deploy, and Definition of Done

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the two-binary dev setup (Go backend + a separately-run Vite dev server) into the single-deployable monolith the design spec promises: the built frontend embedded into the Go binary via `embed.FS` with an SPA fallback, a real multi-stage Dockerfile that builds both halves, a working `-create-admin` CLI fallback (already documented in the README but never implemented), optional seed data, and a final end-to-end Definition-of-Done verification.

**Architecture:** No new bounded context — this plan touches `internal/web` (the embed + SPA fallback, currently a placeholder), `cmd/server/main.go` (one new flag), `internal/platform/bootstrap` (one new non-Fx entry point mirroring the existing `MigrateAndExit`), the root `Dockerfile`/`Makefile`/`.env.example`, and adds one small seed script. **Tasks 1–3 do not depend on Plan 07 (the FE logs/analytics screens) being finished** — the embed mechanism, the Dockerfile, and the CLI flag only need *a* buildable frontend to exist, not a *complete* one; the current frontend (even mid-Plan-07) builds to a valid `dist/` today. **Task 4 (the full DoD walkthrough) is the one task that must wait** until Plan 07 lands, since it exercises the real Logs and Analytics screens through the browser.

**Tech Stack:** Go 1.26 `embed.FS`, Docker multi-stage builds, Bun (frontend build), goose (migrations, unchanged).

## Global Constraints

- **`go:embed` requires the target directory to exist at compile time (HARD).** `apps/frontend/dist/` is git-ignored (build output) — `internal/web`'s embed source must be a path that's always present in a fresh checkout. Task 1 solves this with a tracked placeholder directory + a `web-build` Makefile step that copies real output into it before `go build` — never make `go build`/`go vet`/`go test` fail on a clean clone that hasn't run a frontend build yet.
- **Anti-duplication (HARD):** `-create-admin` (Task 2) calls the SAME `auth.Service.Setup(...)` the HTTP setup wizard already uses — it must not reimplement admin creation, password hashing, or the "already initialized" check as a second code path.
- **No new migration:** the `users`/`is_initialized` semantics are unchanged (`COUNT(users) = 0`, decision 5) — `-create-admin` is just a second caller of the existing `Setup` use case, not a new admin-creation mechanism.
- **Dockerfile build context stays the repo root** (already true — `docker-compose.yml`'s `app` service already sets `context: .`) so a single multi-stage build can see both `apps/backend/` and `apps/frontend/`.
- **Don't touch Plan 05/06 code.** This plan is additive at the infra/embed layer; no changes to `domain/`, `usecase/`, or `adapter/postgres/`.

### Sonar guardrails — write compliant from the first commit

```
Go:
- go:S107 — <=7 params (<=5 preferred).
- go:S3776 — cognitive complexity <=15.
- go:S1192 — const for any string literal duplicated 3+ times.
- errcheck — handle every returned error; never `_ = fallible()`.
- gosec — no hardcoded secrets; `-create-admin`'s password comes from a flag/env the operator controls, never a default value baked into the binary.
```

### Skill brief for implementer subagents (every task)

> Invoke `golang-expert` first — hub skill, auto-chains go-patterns/go-review/go-test/go-error-handling + senior-backend + senior-security + algorithmic-complexity. Follow its Auto-chain. Task 3 (Dockerfile/Makefile) is DevOps/config, not application Go code — apply `senior-backend`'s deployment-hygiene judgment (multi-stage build, minimal final image, no build tools in the runtime layer) rather than the full Go application discipline. Apply `ponytail`: the seed script is a handful of INSERT statements, not a framework; the SPA fallback's "not built yet" message is a plain string, not a templated error page.

### Algorithmic complexity

Nothing data-heavy in this plan — a `-create-admin` flag runs the existing O(1) `Setup` use case once; the SPA fallback handler does an O(1) `fs.Stat`-equivalent lookup per request (embedded FS reads are in-memory, not disk I/O); the seed script inserts a fixed, small number of rows once.

### TDD fit (per §16)

**TDD: no** for this entire plan — Dockerfile/Makefile/config/embed wiring is exactly the "DI/wiring, config/dotfile, migration/SQL-only" category this project's TDD-fit rule marks as test-after (or, for a seed script and a Dockerfile, not unit-testable at all — verified by running them). The one piece of new application logic (`-create-admin`'s flag parsing + calling `Setup`) is a thin CLI wrapper around an ALREADY-TESTED use case (`auth.Service.Setup` has its own tests from Plan 03) — verify by actually running the binary with the flag against a real Postgres, not a new unit test.

---

## Task 1: Real frontend embed + SPA fallback

Replaces the placeholder `internal/web.SPAHandler` with a real `embed.FS` of the built frontend, plus a `web-build` step that gets a real `dist/` into the embeddable location.

**Files:**
- Modify: `apps/backend/internal/web/web.go`
- Create: `apps/backend/internal/web/dist/.gitkeep`
- Modify: `Makefile` (fix `pnpm`→`bun`, add `web-build` copy step, add a top-level `build` that chains web-build + go build)
- Modify: `.gitignore` (ensure `apps/backend/internal/web/dist/*` is ignored EXCEPT `.gitkeep` — the built assets must never be committed, only the placeholder)

**Interfaces:**
- Consumes: nothing new (stdlib `embed`, `io/fs`, `net/http`, echo).
- Produces: `web.SPAHandler() echo.HandlerFunc` (same exported name/signature as today — no caller changes needed in `adapter/http/server.go`, wherever it's currently registered).

- [ ] **Step 1: Read the current server wiring**

Read `apps/backend/internal/adapter/http/*.go` to find exactly where `web.SPAHandler()` is currently registered (likely a catch-all route in `server.go` or `bootstrap.go`) — this plan does NOT change the registration call, only what the handler does internally. Confirm the exact call site before editing `web.go` so you don't accidentally change the route pattern.

- [ ] **Step 2: Ensure the embeddable directory always exists**

```bash
mkdir -p apps/backend/internal/web/dist
touch apps/backend/internal/web/dist/.gitkeep
```
Add to `.gitignore` (find the right section, likely near other build-output ignores):
```
apps/backend/internal/web/dist/*
!apps/backend/internal/web/dist/.gitkeep
```

- [ ] **Step 3: Rewrite `web.go` with a real embed + SPA fallback**

`apps/backend/internal/web/web.go`:
```go
// Package web serves the embedded frontend. dist/ is populated by `make
// web-build` (or the Dockerfile's frontend build stage) before `go build`;
// in a fresh checkout it holds only .gitkeep, and SPAHandler serves a plain
// "not built yet" message instead of failing to compile or panicking.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/labstack/echo/v4"
)

//go:embed all:dist
var embedded embed.FS

const distDir = "dist"

// distFS strips the "dist" prefix so paths match what the browser requests
// ("/assets/index.js", not "/dist/assets/index.js").
func distFS() fs.FS {
	sub, err := fs.Sub(embedded, distDir)
	if err != nil {
		return embedded // unreachable in practice (distDir is a compile-time constant that always exists)
	}
	return sub
}

// SPAHandler serves the built frontend: a real file if the request path
// matches one, otherwise index.html (client-side routing fallback). Returns
// a plain message if the frontend hasn't been built yet (dist/ has only
// .gitkeep) rather than a broken 404 loop.
func SPAHandler() echo.HandlerFunc {
	assets := distFS()
	return func(c echo.Context) error {
		reqPath := strings.TrimPrefix(c.Request().URL.Path, "/")
		if reqPath == "" {
			reqPath = "index.html"
		}
		if data, err := fs.ReadFile(assets, reqPath); err == nil {
			return c.Blob(http.StatusOK, contentType(reqPath), data)
		}
		// Not a real asset path -> SPA fallback to index.html.
		if data, err := fs.ReadFile(assets, "index.html"); err == nil {
			return c.HTMLBlob(http.StatusOK, data)
		}
		return c.String(http.StatusOK, "RouterLens — frontend not built yet (run `make web-build`)")
	}
}

// contentType maps a handful of known static-asset extensions; anything else
// falls back to octet-stream (the SPA's own asset filenames are always one
// of these, generated by Vite's build).
func contentType(p string) string {
	switch path.Ext(p) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".js":
		return "text/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".json":
		return "application/json; charset=utf-8"
	case ".woff2":
		return "font/woff2"
	case ".ico":
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
}
```
NOTE: `//go:embed all:dist` (the `all:` prefix) is required because Vite build output can include dotfiles/underscore-prefixed chunk names that plain `//go:embed dist` would silently skip.

- [ ] **Step 4: Fix the Makefile's frontend targets + add a chained `build`**

Read the current root `Makefile` first, then apply these changes (it currently says `pnpm dev`/`pnpm build` under the `web`/`web-build` targets — this project uses Bun, confirmed by `apps/frontend/bun.lock`):
```makefile
# ---- frontend (Vite + React lives in ./apps/frontend) ----
web:
	cd apps/frontend && bun install && bun run dev

web-build:
	cd apps/frontend && bun install && bun run build
	rm -rf apps/backend/internal/web/dist
	mkdir -p apps/backend/internal/web/dist
	cp -r apps/frontend/dist/. apps/backend/internal/web/dist/
```
Add a new top-level target that chains both halves (place near `build`):
```makefile
build-all: web-build
	cd apps/backend && go build -o bin/routerlens ./cmd/server
```
Leave the existing backend-only `build` target as-is (it's still useful for backend-only iteration without touching the frontend).

- [ ] **Step 5: Verify**

```bash
cd apps/backend && go build ./... && go vet ./...
```
Expected: clean (the placeholder `dist/.gitkeep`-only directory is enough for `//go:embed all:dist` to compile).

```bash
make web-build && make build-all
```
Expected: frontend builds, its output lands in `apps/backend/internal/web/dist/`, and the final Go binary embeds it (confirm by running the binary locally and loading `http://localhost:8080/` in a browser — it should serve the real dashboard shell, not the placeholder string).

Do NOT git commit — this project commits once per plan, at the end (see `.superpowers/sdd/progress.md`).

---

## Task 2: `-create-admin` CLI fallback

Implements the flag the README already documents (`make create-admin`) but that doesn't exist in code today — reusing the existing `auth.Service.Setup` use case, not a new admin-creation path.

**Files:**
- Modify: `apps/backend/cmd/server/main.go`
- Modify: `apps/backend/internal/platform/bootstrap/bootstrap.go`
- Modify: `Makefile` (wire the `create-admin` target to the new flags)

**Interfaces:**
- Consumes: `auth.NewService(users, sessions) *auth.Service` and `(*auth.Service).Setup(ctx, email, password, name string) error` (both already exist, Plan 03, unmodified).
- Produces: `bootstrap.CreateAdminAndExit(email, password, name string) error`.

- [ ] **Step 1: Add the non-Fx entry point in `bootstrap.go`**

Read the existing `MigrateAndExit` function in `apps/backend/internal/platform/bootstrap/bootstrap.go` first — mirror its exact shape (load config, open a pool, defer close, no Fx graph). Append:
```go
// CreateAdminAndExit is the non-Fx path for the `-create-admin` flag: load
// config, open a pool, and call the SAME auth.Service.Setup the HTTP setup
// wizard uses — this is a second caller of that use case, not a second
// admin-creation mechanism. Setup itself already enforces "locked after the
// first user" (decision 5), so a second run against an initialized instance
// fails the same way the web wizard would.
func CreateAdminAndExit(email, password, name string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logging.New(cfg.IsProduction(), cfg.LogLevel)
	pool, err := postgres.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	users := postgres.NewUserRepository(pool)
	sessions := postgres.NewSessionRepository(pool)
	return auth.NewService(users, sessions).Setup(context.Background(), email, password, name)
}
```
Add `"router-lens/internal/usecase/auth"` to `bootstrap.go`'s import block if it isn't already there (check first — `authModule` already references `auth.NewService`, so the import likely already exists under a plain `auth` name; reuse it, don't alias-duplicate).

- [ ] **Step 2: Add the flag in `main.go`**

Read the current `apps/backend/cmd/server/main.go` first (it's currently ~12 lines with just `-migrate-only`) — extend it:
```go
package main

import (
	"flag"
	"log"

	"router-lens/internal/platform/bootstrap"
)

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "apply migrations then exit")
	createAdmin := flag.Bool("create-admin", false, "create the first admin user then exit")
	email := flag.String("email", "", "admin email (required with -create-admin)")
	password := flag.String("password", "", "admin password (required with -create-admin)")
	name := flag.String("name", "", "admin display name (optional)")
	flag.Parse()

	if *migrateOnly {
		if err := bootstrap.MigrateAndExit(); err != nil {
			log.Fatalf("migrate: %v", err)
		}
		return
	}
	if *createAdmin {
		if *email == "" || *password == "" {
			log.Fatal("create-admin: -email and -password are required")
		}
		if err := bootstrap.CreateAdminAndExit(*email, *password, *name); err != nil {
			log.Fatalf("create-admin: %v", err)
		}
		return
	}
	bootstrap.New().Run()
}
```

- [ ] **Step 3: Wire the Makefile target**

Read the current `create-admin` target in the root `Makefile` first (it likely calls a non-existent `-create-admin` flag already, per the pre-existing stale reference) — fix it to actually work, taking email/password from the invoker's environment or interactive input:
```makefile
create-admin:
	cd apps/backend && go run ./cmd/server -create-admin -email="$(EMAIL)" -password="$(PASSWORD)" -name="$(NAME)"
```
(Usage: `make create-admin EMAIL=admin@example.com PASSWORD=changeme NAME=Admin` — documented in the README's existing Configuration/Development section if it isn't already.)

- [ ] **Step 4: Verify**

```bash
cd apps/backend && gofmt -l . && go vet ./... && go build ./...
```
Expected: clean.

Manual verification against a real Postgres (reuse this session's established pattern — a throwaway container on a non-conflicting host port if 5432 is taken locally):
```bash
go run ./cmd/server -migrate-only
go run ./cmd/server -create-admin -email=admin@test.local -password=testpass123 -name=Admin
go run ./cmd/server -create-admin -email=admin@test.local -password=testpass123 -name=Admin   # second run
```
Expected: first run succeeds silently (exit 0); second run fails with the same "setup already locked" error the web wizard produces (proving it reuses `Setup`, not a bypass).

Do NOT git commit (see Task 1's note).

---

## Task 3: Multi-stage Dockerfile, seed data, and `.env.example` completeness

Rewrites the Dockerfile to build both halves into one image, adds a small optional seed script, and closes a couple of documentation-reality gaps.

**Files:**
- Modify: `apps/backend/Dockerfile`
- Create: `scripts/seed.sql`
- Modify: `Makefile` (add a `seed` target)
- Modify: `apps/backend/.env.example` (add `LOG_LEVEL`)

**Interfaces:** none (build/config files only).

- [ ] **Step 1: Rewrite the Dockerfile as a 3-stage build**

Read the current `apps/backend/Dockerfile` first (it currently only builds the Go binary, with a comment already anticipating this: *"Build context is the repo root (so a later stage can also build apps/frontend and embed it)"*). Replace it:
```dockerfile
# syntax=docker/dockerfile:1
# Build context is the repo root — this stage builds apps/frontend, the next
# stage copies its output into the Go module's embed source before building.

FROM oven/bun:1 AS frontend-build
WORKDIR /src
COPY apps/frontend/package.json apps/frontend/bun.lock ./
RUN bun install --frozen-lockfile
COPY apps/frontend/ .
RUN bun run build

FROM golang:1.26 AS build
WORKDIR /src
COPY apps/backend/go.mod apps/backend/go.sum ./
RUN go mod download
COPY apps/backend/ .
COPY --from=frontend-build /src/dist ./internal/web/dist
RUN CGO_ENABLED=0 go build -o /out/routerlens ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/routerlens /routerlens
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/routerlens"]
```
NOTE: the `COPY --from=frontend-build /src/dist ./internal/web/dist` step OVERWRITES the placeholder `.gitkeep`-only directory inside the build stage's filesystem (never touches the actual git-tracked file on disk) — this is exactly the same "populate before `go build`" pattern as the local `make web-build` target, just inside Docker's layer graph instead of the host filesystem.

- [ ] **Step 2: Write a minimal seed script**

`scripts/seed.sql` (repo root — a NEW top-level `scripts/` directory is fine, this is the only file in it):
```sql
-- Optional demo data for local exploration. Run AFTER first-run setup has
-- created at least one user (this script reads the first user as owner).
-- Usage: make seed  (or: psql "$DATABASE_URL" -f scripts/seed.sql)

INSERT INTO projects (owner_user_id, name, slug, description)
SELECT id, 'Demo Project', 'demo-project', 'Seeded via scripts/seed.sql'
FROM users ORDER BY created_at LIMIT 1
ON CONFLICT (owner_user_id, slug) DO NOTHING;

INSERT INTO pricing_rules (provider, model, input_price_per_1m, output_price_per_1m, currency)
VALUES
	('anthropic', 'claude-sonnet-4-5', 3.00, 15.00, 'USD'),
	('openai', 'gpt-4o', 2.50, 10.00, 'USD')
ON CONFLICT (provider, model) DO NOTHING;
```
NOTE: no seed `llm_events` rows here deliberately (ponytail) — an event needs a real API key (which needs the plaintext-shown-once flow), so seeding events meaningfully requires either the dashboard or a curl call per the README's existing ingestion example, not a raw SQL INSERT that would have to fake a key hash. Seeding a project + pricing rules is enough to make the dashboard non-empty on first login; a user who wants seeded log/analytics data can follow the README's curl example once a key exists.

Add to the root `Makefile`:
```makefile
seed:
	psql "$$(grep DATABASE_URL apps/backend/.env | cut -d= -f2-)" -f scripts/seed.sql
```
(Falls back gracefully with a clear psql connection error if `.env` doesn't exist yet or `DATABASE_URL` isn't set — no special-casing needed beyond what psql already reports.)

- [ ] **Step 3: Complete `.env.example`**

Read the current `apps/backend/.env.example` first, then add the one documented-but-missing variable (`config.go` already has a `LOG_LEVEL` field defaulting to `"info"` — `.env.example` should show it for discoverability even though it's optional):
```
LOG_LEVEL=info
```

- [ ] **Step 4: Verify**

```bash
docker compose build app
```
Expected: the multi-stage build succeeds (frontend stage runs `bun install`+`bun run build`, backend stage embeds the result and compiles). This is the first real proof the embed mechanism works end-to-end outside local `make` targets.

Do NOT git commit (see Task 1's note).

---

## Task 4: Full Definition-of-Done verification (BLOCKED until Plan 07 / FE-04 is merged)

**Do not start this task until the Request Logs and Analytics screens (RouterLens FE Plan 04) are real, not placeholders** — this task's whole point is exercising them through the browser. Tasks 1–3 above are already complete and merged by the time this runs; this task adds no new source files, it's a verification pass.

**Files:** none (verification only — if it finds a bug, file it as a follow-up fix, don't silently patch scope into this task).

- [ ] **Step 1: Full-stack boot from scratch**

```bash
docker compose down -v   # clean slate — confirm with the user before running if a shared dev volume might matter
docker compose up --build
```
Expected: Postgres + the single `app` container start; migrations apply on boot (check logs for "applying database migrations").

- [ ] **Step 2: Walk the golden path via a real browser**

Against `http://localhost:8080`:
1. First load routes to `/setup` (no users yet) — create the admin.
2. Login with those credentials — httpOnly cookie set, redirected to the dashboard shell.
3. Create a Project.
4. Create an API Key for it — plaintext shown exactly once, copy it.
5. From a terminal, `curl` the README's ingestion example against `POST /api/v1/events` with that key — confirm `202 {"data":{"deduplicated":false,...}}`.
6. Repeat the same curl body (same `event_id`) — confirm `202 {"data":{"deduplicated":true}}`.
7. In the dashboard, open **Logs** — the ingested event appears; filters (project/provider/model/errors-only/date-range) work; CSV export downloads a file.
8. Open **Analytics** — overview stat cards reflect the one ingested event; switching the date-range preset doesn't error even when a narrower window has zero matching events (empty-state charts, not a crash).
9. Open **Pricing** — configure a rule for the ingested event's `(provider, model)`; re-ingest (a new `event_id`) and confirm the new event's cost is no longer unpriced.
10. Logout — cookie cleared, redirected to `/login`.

- [ ] **Step 3: Record the result**

Update `docs/progress.md`'s Snapshot table and Plan roadmap table: mark Plan 08 executed, update "Frontend source code" and "Backend source code" rows to reflect the finished state, and remove/resolve the "Next actions" section (the roadmap is now fully executed through v0.1's Definition of Done).

Do NOT git commit the verification itself, but DO commit the `docs/progress.md` update as this plan's final commit (single commit, matching the established one-commit-per-plan cadence) once the walkthrough passes.

---

## Plan-level Definition of Done

- `docker compose up --build` boots Postgres + one `app` container serving both the API and the real embedded frontend (no separate frontend dev server needed in production).
- `internal/web` serves real static assets + SPA fallback from an embedded `dist/`; a fresh clone that hasn't run a frontend build still compiles (`.gitkeep` placeholder) and serves a clear "not built yet" message instead of crashing.
- `make create-admin EMAIL=... PASSWORD=... NAME=...` works and reuses the existing `auth.Service.Setup` — the README's documented command is no longer a broken promise.
- The full v0.1 Definition of Done (design spec §15) is manually walked end-to-end through the real UI: setup → login → project → API key → ingest (idempotent) → see it in Logs → see it in Analytics → configure pricing → re-ingest priced → CSV export → logout.
- `docs/progress.md` reflects the fully-executed roadmap.
