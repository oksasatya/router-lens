# RouterLens — Plan 01: Foundation & Persistence

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up a runnable RouterLens skeleton — `docker compose up` boots PostgreSQL and the Go/Echo app, migrations 001–006 apply, and `/api/v1/healthz` + `/readyz` respond.

**Architecture:** Single Go monolith (Echo) talking to PostgreSQL via pgx/v5. Hexagonal layering starts here: `cmd/server` wires `infrastructure` (postgres pool, echo http) on top of `shared` cross-cutting packages. The frontend is embedded later (Plan 07); this plan serves an SPA-fallback placeholder.

**Tech Stack:** Go 1.26, Echo v4, pgx/v5 (`pgxpool` + `stdlib`), goose v3 (embedded migrations), godotenv (dev `.env`), PostgreSQL.

## Global Constraints

Applies to every task in this plan and all sibling plans:

- **Monorepo layout:** all Go code lives under `apps/backend/` (go.mod there); the frontend under `apps/frontend/`. The `Files:` paths below (`cmd/server`, `internal/...`, `migrations/`) are **relative to `apps/backend/`**. The Go module is `router-lens` and import paths stay `router-lens/internal/...` — the module name is independent of the folder.
- **Go 1.26**, module `router-lens`. Import paths: `router-lens/internal/...`.
- **No Redis.** PostgreSQL is the only datastore. **DI via Uber Fx** (`go.uber.org/fx`) — per-module provider constructors + `fx.Lifecycle` `OnStart`/`OnStop` hooks in `cmd/server/main.go`. Constructors stay plain (unit-testable without Fx); `fx.Run` handles SIGINT/SIGTERM graceful shutdown.
- **Hexagonal layering (HARD):** `domain/` imports nothing from Echo/`database/sql`/pgx/`infrastructure/`. Handlers parse→usecase→response only. Repository interfaces live in `domain/`, implementations in `infrastructure/postgres/`.
- **Money** = `NUMERIC` (never float). **Time** = `timestamptz`, UTC. **IDs** = UUID.
- **Single deployable, same-origin:** one binary serves API (`/api/v1/*`) and the embedded UI (`/*`).
- **golang-expert hub:** every Go task → "Invoke `golang-expert` first (hub — auto-chains go-patterns / go-review / go-test / go-error-handling / go-concurrency-patterns + senior-backend + senior-security + algorithmic-complexity)."
- **Sonar guardrails (write compliant on first commit):**
  ```
  Go:
  - go:S107 — ≤5 params (6+ = smell → Deps/Opts struct from the start).
  - go:S3776 — cognitive complexity ≤15 → extract helpers; tests use t.Run subtests.
  - go:S1192 — const for any string literal duplicated 3+ times.
  - errcheck — handle every returned error; never `_ = fallible()`. Wrap with %w; sentinel + errors.Is/As.
  - gosec — no hardcoded secrets, parameterized SQL only, crypto/rand for tokens.
  ```
- **Verification before "done":** `gofmt -l` (empty) → `go vet ./...` → `golangci-lint run` → `go test -race ./...` → `go build ./...`.

---

### Task 1: Module scaffold, directory skeleton, Makefile, `.env.example`

**TDD:** no — scaffolding/config only. Verify by `go build ./...` and `make` targets running.

**Files:**
- Modify: `go.mod` (add deps via `go get`)
- Create: `Makefile`, `.env.example`, `.gitkeep` placeholders as needed
- Create dirs: `cmd/server/`, `internal/app/`, `internal/domain/`, `internal/application/`, `internal/infrastructure/postgres/`, `internal/infrastructure/http/middleware/`, `internal/infrastructure/http/handler/`, `internal/shared/{response,errors,i18n,pagination,validator,security,datetime,csv}/`, `internal/web/`, `migrations/` (all under `apps/backend/`), and `apps/frontend/`

**Interfaces:**
- Produces: the module path `router-lens` and dependency set for all later tasks.

- [ ] **Step 1: Add dependencies**

```bash
go get github.com/labstack/echo/v4@latest
go get github.com/jackc/pgx/v5@latest
go get github.com/pressly/goose/v3@latest
go get github.com/joho/godotenv@latest
go get github.com/google/uuid@latest
go get go.uber.org/fx@latest
```

- [ ] **Step 2: Create `.env.example`**

```dotenv
APP_ENV=development
APP_PORT=8080
DATABASE_URL=postgres://routerlens:routerlens@localhost:5432/routerlens?sslmode=disable
SESSION_SECRET=change-me-to-a-long-random-string
COOKIE_CROSS_SITE=false
MAX_BACKDATE_DAYS=7
RETENTION_DAYS=0
```

- [ ] **Step 3: Create `Makefile`**

```makefile
.PHONY: dev build test lint migrate create-admin tidy

dev:
	go run ./cmd/server

build:
	go build -o bin/routerlens ./cmd/server

test:
	go test -race -cover ./...

lint:
	golangci-lint run

tidy:
	go mod tidy

migrate:
	go run ./cmd/server -migrate-only

create-admin:
	go run ./cmd/server -create-admin
```

- [ ] **Step 4: Verify build skeleton**

Run: `go build ./...`
Expected: succeeds (no packages yet → no error), `go.mod` lists the new deps.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum Makefile .env.example
git commit -m "chore: scaffold module, makefile, env example"
```

---

### Task 2: Config loader

**TDD:** yes — parsing has logic (defaults, int/bool parse, required-field validation). Red test first.

**Files:**
- Create: `internal/app/config.go`
- Test: `internal/app/config_test.go`

**Interfaces:**
- Produces:
  ```go
  type Config struct {
      AppEnv          string
      AppPort         string
      DatabaseURL     string
      SessionSecret   string
      CookieCrossSite bool
      MaxBackdateDays int
      RetentionDays   int
  }
  func (c Config) IsProduction() bool
  func Load() (Config, error)            // reads os.Getenv
  ```

- [ ] **Step 1: Write the failing test**

```go
package app

import "testing"

func TestParseConfig(t *testing.T) {
	t.Run("defaults applied and required present", func(t *testing.T) {
		env := map[string]string{
			"DATABASE_URL":   "postgres://x",
			"SESSION_SECRET": "secret",
		}
		cfg, err := parse(func(k string) string { return env[k] })
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.AppPort != "8080" || cfg.AppEnv != "development" {
			t.Fatalf("bad defaults: %+v", cfg)
		}
		if cfg.MaxBackdateDays != 7 || cfg.RetentionDays != 0 || cfg.CookieCrossSite {
			t.Fatalf("bad numeric/bool defaults: %+v", cfg)
		}
	})

	t.Run("missing required fails", func(t *testing.T) {
		if _, err := parse(func(string) string { return "" }); err == nil {
			t.Fatal("expected error for missing DATABASE_URL/SESSION_SECRET")
		}
	})

	t.Run("overrides parsed", func(t *testing.T) {
		env := map[string]string{
			"DATABASE_URL": "u", "SESSION_SECRET": "s",
			"APP_ENV": "production", "COOKIE_CROSS_SITE": "true", "MAX_BACKDATE_DAYS": "30",
		}
		cfg, _ := parse(func(k string) string { return env[k] })
		if !cfg.IsProduction() || !cfg.CookieCrossSite || cfg.MaxBackdateDays != 30 {
			t.Fatalf("overrides not applied: %+v", cfg)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/ -run TestParseConfig -v`
Expected: FAIL — `parse` / `Config` undefined.

- [ ] **Step 3: Write minimal implementation**

```go
package app

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

const (
	envProduction = "production"
	defaultPort   = "8080"
)

type Config struct {
	AppEnv          string
	AppPort         string
	DatabaseURL     string
	SessionSecret   string
	CookieCrossSite bool
	MaxBackdateDays int
	RetentionDays   int
}

func (c Config) IsProduction() bool { return c.AppEnv == envProduction }

// Load reads configuration from the environment, loading a local .env first if present.
func Load() (Config, error) {
	_ = godotenv.Load() // best-effort in dev; ignored in prod where vars are set directly
	return parse(os.Getenv)
}

func parse(get func(string) string) (Config, error) {
	cfg := Config{
		AppEnv:          orDefault(get("APP_ENV"), "development"),
		AppPort:         orDefault(get("APP_PORT"), defaultPort),
		DatabaseURL:     get("DATABASE_URL"),
		SessionSecret:   get("SESSION_SECRET"),
		CookieCrossSite: get("COOKIE_CROSS_SITE") == "true",
		MaxBackdateDays: atoiOr(get("MAX_BACKDATE_DAYS"), 7),
		RetentionDays:   atoiOr(get("RETENTION_DAYS"), 0),
	}
	if cfg.DatabaseURL == "" || cfg.SessionSecret == "" {
		return Config{}, fmt.Errorf("config: DATABASE_URL and SESSION_SECRET are required")
	}
	return cfg, nil
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func atoiOr(v string, def int) int {
	if n, err := strconv.Atoi(v); err == nil {
		return n
	}
	return def
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/app/ -run TestParseConfig -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/app/config.go internal/app/config_test.go
git commit -m "feat: config loader with env defaults and validation"
```

---

### Task 3: Shared error type (`shared/errors`)

**TDD:** yes — kind→status mapping and Unwrap have behavior worth pinning.

**Files:**
- Create: `internal/shared/errors/errors.go`
- Test: `internal/shared/errors/errors_test.go`

**Interfaces:**
- Produces:
  ```go
  type Kind string
  const (KindValidation, KindUnauthorized, KindForbidden, KindNotFound, KindConflict, KindInternal Kind)
  type AppError struct { /* Kind, Code, Message string; Details any; wrapped error */ }
  func (e *AppError) Error() string
  func (e *AppError) Unwrap() error
  func New(kind Kind, code, message string) *AppError
  func Wrap(kind Kind, code, message string, err error) *AppError
  func (e *AppError) WithDetails(d any) *AppError
  func HTTPStatus(kind Kind) int
  func As(err error) (*AppError, bool)
  ```
  Domain packages must NOT import this (it is application/infra level); domain returns sentinel errors, the application layer maps them to `AppError`.

- [ ] **Step 1: Write the failing test**

```go
package errors

import (
	stderrors "errors"
	"net/http"
	"testing"
)

func TestAppError(t *testing.T) {
	t.Run("status mapping", func(t *testing.T) {
		cases := map[Kind]int{
			KindValidation:   http.StatusBadRequest,
			KindUnauthorized: http.StatusUnauthorized,
			KindForbidden:    http.StatusForbidden,
			KindNotFound:     http.StatusNotFound,
			KindConflict:     http.StatusConflict,
			KindInternal:     http.StatusInternalServerError,
		}
		for k, want := range cases {
			if got := HTTPStatus(k); got != want {
				t.Errorf("kind %s: got %d want %d", k, got, want)
			}
		}
	})

	t.Run("wrap and unwrap", func(t *testing.T) {
		base := stderrors.New("boom")
		e := Wrap(KindInternal, "internal_error", "failed", base)
		if !stderrors.Is(e, base) {
			t.Fatal("expected Is to find wrapped error")
		}
		if got, ok := As(e); !ok || got.Code != "internal_error" {
			t.Fatalf("As failed: %+v %v", got, ok)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/shared/errors/ -v`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Write minimal implementation**

```go
// Package errors defines the application-level error type and its HTTP mapping.
package errors

import (
	stderrors "errors"
	"net/http"
)

type Kind string

const (
	KindValidation   Kind = "validation"
	KindUnauthorized Kind = "unauthorized"
	KindForbidden    Kind = "forbidden"
	KindNotFound     Kind = "not_found"
	KindConflict     Kind = "conflict"
	KindInternal     Kind = "internal"
)

type AppError struct {
	Kind    Kind
	Code    string
	Message string
	Details any
	wrapped error
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.wrapped }

func (e *AppError) WithDetails(d any) *AppError {
	e.Details = d
	return e
}

func New(kind Kind, code, message string) *AppError {
	return &AppError{Kind: kind, Code: code, Message: message}
}

func Wrap(kind Kind, code, message string, err error) *AppError {
	return &AppError{Kind: kind, Code: code, Message: message, wrapped: err}
}

func As(err error) (*AppError, bool) {
	var ae *AppError
	if stderrors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}

func HTTPStatus(kind Kind) int {
	switch kind {
	case KindValidation:
		return http.StatusBadRequest
	case KindUnauthorized:
		return http.StatusUnauthorized
	case KindForbidden:
		return http.StatusForbidden
	case KindNotFound:
		return http.StatusNotFound
	case KindConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/shared/errors/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shared/errors/
git commit -m "feat: shared AppError type with HTTP status mapping"
```

---

### Task 4: i18n catalog + response envelope (meta + localization)

**TDD:** yes for `i18n` (Resolve precedence + Message fallback have logic); the response wrappers get one test asserting the localized error + `meta` shape.

**Files:**
- Create: `internal/shared/i18n/i18n.go`
- Test: `internal/shared/i18n/i18n_test.go`
- Create: `internal/shared/response/response.go`
- Test: `internal/shared/response/response_test.go`

**Interfaces:**
- Consumes: `shared/errors` (`AppError`, `HTTPStatus`, `As`).
- Produces:
  ```go
  // i18n (pure — no Echo import)
  type Lang string
  const ( EN Lang = "en"; ID Lang = "id" )
  const Default = EN
  const ContextKey = "lang"                          // echo context key the middleware sets
  func Resolve(acceptLanguage string) Lang            // first supported Accept-Language tag, else Default
  func Message(code string, lang Lang, fallback string) string // catalog lookup, fallback to EN then `fallback`

  // response
  type Meta struct { Lang, RequestID, Timestamp string }
  func Data(c echo.Context, status int, data any) error
  func Created(c echo.Context, data any) error
  func NoContent(c echo.Context) error
  func Error(c echo.Context, err error) error   // localizes message by code+lang; adds meta
  ```
  Later plans extend the `i18n` catalog with their own error codes (one map entry per code).

- [ ] **Step 1: Write the failing i18n test**

```go
package i18n

import "testing"

func TestResolve(t *testing.T) {
	cases := []struct {
		accept string
		want   Lang
	}{
		{"", EN},
		{"id-ID,id;q=0.9,en;q=0.8", ID},
		{"en-US,en;q=0.9", EN},
		{"fr-FR", EN}, // unsupported falls back to default
	}
	for _, tc := range cases {
		if got := Resolve(tc.accept); got != tc.want {
			t.Errorf("Resolve(%q)=%q want %q", tc.accept, got, tc.want)
		}
	}
}

func TestMessage(t *testing.T) {
	if Message("validation_failed", ID, "x") != "Validasi gagal" {
		t.Errorf("ID validation message wrong: %q", Message("validation_failed", ID, "x"))
	}
	if Message("unknown_code", ID, "fallback msg") != "fallback msg" {
		t.Error("unknown code should use fallback")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/shared/i18n/ -v`
Expected: FAIL — undefined.

- [ ] **Step 3: Write `i18n.go`**

```go
// Package i18n resolves the request language and maps error codes to localized
// messages. It is pure (no Echo import) so domain/shared code can depend on it.
package i18n

import "strings"

type Lang string

const (
	EN Lang = "en"
	ID Lang = "id"
)

const (
	Default    = EN
	ContextKey = "lang"
)

func supported(l Lang) bool { return l == EN || l == ID }

// Resolve picks the language from the Accept-Language header — the first
// supported tag wins, otherwise the default. Header-driven content negotiation
// per RFC 7231; no query/cookie override in v0.1.
func Resolve(acceptLanguage string) Lang {
	for _, part := range strings.Split(acceptLanguage, ",") {
		tag := strings.ToLower(strings.TrimSpace(part))
		if i := strings.IndexAny(tag, ";-"); i >= 0 {
			tag = tag[:i]
		}
		if l := Lang(tag); supported(l) {
			return l
		}
	}
	return Default
}

// catalog maps an error code to its localized messages. Feature plans append.
var catalog = map[string]map[Lang]string{
	"internal_error":    {EN: "Internal server error", ID: "Terjadi kesalahan pada server"},
	"validation_failed": {EN: "Validation failed", ID: "Validasi gagal"},
	"unauthorized":      {EN: "Authentication required", ID: "Perlu autentikasi"},
	"forbidden":         {EN: "Access denied", ID: "Akses ditolak"},
	"not_found":         {EN: "Resource not found", ID: "Data tidak ditemukan"},
}

// Message returns the localized message for code, falling back to the default
// language and finally to the provided fallback string.
func Message(code string, lang Lang, fallback string) string {
	byLang, ok := catalog[code]
	if !ok {
		return fallback
	}
	if msg, ok := byLang[lang]; ok {
		return msg
	}
	if msg, ok := byLang[Default]; ok {
		return msg
	}
	return fallback
}
```

- [ ] **Step 4: Run i18n test to verify it passes**

Run: `go test ./internal/shared/i18n/ -v`
Expected: PASS.

- [ ] **Step 5: Write the failing response test**

```go
package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
)

func TestError_LocalizesWithMeta(t *testing.T) {
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
	c.Set(i18n.ContextKey, i18n.ID)

	_ = Error(c, apperrors.New(apperrors.KindValidation, "validation_failed", "validation failed"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d want 400", rec.Code)
	}
	var body struct {
		Error struct{ Code, Message string } `json:"error"`
		Meta  struct{ Lang, Timestamp string } `json:"meta"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error.Message != "Validasi gagal" {
		t.Fatalf("expected ID message, got %q", body.Error.Message)
	}
	if body.Meta.Lang != "id" || body.Meta.Timestamp == "" {
		t.Fatalf("meta wrong: %+v", body.Meta)
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./internal/shared/response/ -v`
Expected: FAIL — undefined `Error`.

- [ ] **Step 7: Write `response.go`**

```go
// Package response writes the canonical JSON envelope (with meta + localized
// error messages) for all handlers.
package response

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
)

type Meta struct {
	Lang      string `json:"lang"`
	RequestID string `json:"request_id,omitempty"`
	Timestamp string `json:"timestamp"`
}

type envelope struct {
	Data  any        `json:"data,omitempty"`
	Error *errorBody `json:"error,omitempty"`
	Meta  Meta       `json:"meta"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

const codeInternal = "internal_error"

// LangOf returns the language resolved by the Lang middleware, or the default.
func LangOf(c echo.Context) i18n.Lang {
	if l, ok := c.Get(i18n.ContextKey).(i18n.Lang); ok && l != "" {
		return l
	}
	return i18n.Default
}

func meta(c echo.Context) Meta {
	return Meta{
		Lang:      string(LangOf(c)),
		RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func Data(c echo.Context, status int, data any) error {
	return c.JSON(status, envelope{Data: data, Meta: meta(c)})
}

func Created(c echo.Context, data any) error {
	return c.JSON(http.StatusCreated, envelope{Data: data, Meta: meta(c)})
}

func NoContent(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func Error(c echo.Context, err error) error {
	lang := LangOf(c)
	if ae, ok := apperrors.As(err); ok {
		return c.JSON(apperrors.HTTPStatus(ae.Kind), envelope{
			Error: &errorBody{Code: ae.Code, Message: i18n.Message(ae.Code, lang, ae.Message), Details: ae.Details},
			Meta:  meta(c),
		})
	}
	return c.JSON(http.StatusInternalServerError, envelope{
		Error: &errorBody{Code: codeInternal, Message: i18n.Message(codeInternal, lang, "internal server error")},
		Meta:  meta(c),
	})
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `go test ./internal/shared/response/ -v`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/shared/i18n/ internal/shared/response/
git commit -m "feat: i18n catalog and localized response envelope with meta"
```

---

### Task 5: PostgreSQL connection pool

**TDD:** no — I/O wiring. Verified by the boot integration in Task 8.

**Files:**
- Create: `internal/infrastructure/postgres/db.go`

**Interfaces:**
- Consumes: `Config.DatabaseURL`.
- Produces:
  ```go
  func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error)
  ```

- [ ] **Step 1: Write implementation**

```go
// Package postgres holds the pgx connection pool and repository implementations.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool opens a bounded pgx pool and verifies connectivity.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse config: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: new pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}
	return pool, nil
}
```

- [ ] **Step 2: Verify it builds**

Run: `go build ./internal/infrastructure/postgres/`
Expected: succeeds.

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/postgres/db.go
git commit -m "feat: pgx connection pool with bounded limits"
```

---

### Task 6: Migrations 001–006 + embedded goose runner

**TDD:** no — SQL + runner. Verified by Task 8 boot (`select` against tables) and a goose dry parse.

**Files:**
- Create: `migrations/embed.go`
- Create: `migrations/001_create_users.up.sql` + `.down.sql` … through `006_create_llm_events.{up,down}.sql`
- Create: `internal/infrastructure/postgres/migrate.go`

**Interfaces:**
- Consumes: `*pgxpool.Pool`.
- Produces: `func Migrate(ctx context.Context, pool *pgxpool.Pool) error`

- [ ] **Step 1: Create `migrations/embed.go`**

```go
// Package migrations embeds the SQL migration files for goose.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
```

- [ ] **Step 2: Create `001_create_users.up.sql`**

```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    name          text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
```

`001_create_users.down.sql`:

```sql
DROP TABLE IF EXISTS users;
```

- [ ] **Step 3: Create `002_create_sessions.up.sql`**

```sql
CREATE TABLE sessions (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  text NOT NULL UNIQUE,
    expires_at  timestamptz NOT NULL,
    user_agent  text,
    ip          inet,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);
```

`002_create_sessions.down.sql`:

```sql
DROP TABLE IF EXISTS sessions;
```

- [ ] **Step 4: Create `003_create_projects.up.sql`**

```sql
CREATE TABLE projects (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id uuid NOT NULL REFERENCES users(id),
    name          text NOT NULL,
    slug          text NOT NULL,
    description   text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (owner_user_id, slug)
);
```

`003_create_projects.down.sql`:

```sql
DROP TABLE IF EXISTS projects;
```

- [ ] **Step 5: Create `004_create_api_keys.up.sql`**

```sql
CREATE TABLE api_keys (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name         text NOT NULL,
    key_hash     text NOT NULL UNIQUE,
    key_prefix   text NOT NULL,
    last_used_at timestamptz,
    revoked_at   timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_api_keys_project_id ON api_keys (project_id);
```

`004_create_api_keys.down.sql`:

```sql
DROP TABLE IF EXISTS api_keys;
```

- [ ] **Step 6: Create `005_create_pricing_rules.up.sql`**

```sql
CREATE TABLE pricing_rules (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    provider            text NOT NULL,
    model               text NOT NULL,
    input_price_per_1m  numeric NOT NULL,
    output_price_per_1m numeric NOT NULL,
    currency            text NOT NULL DEFAULT 'USD',
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, model)
);
```

`005_create_pricing_rules.down.sql`:

```sql
DROP TABLE IF EXISTS pricing_rules;
```

- [ ] **Step 7: Create `006_create_llm_events.up.sql`**

```sql
CREATE TABLE llm_events (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    event_id            text,
    provider            text NOT NULL,
    model               text NOT NULL,
    route_source        text,
    agent               text,
    input_tokens        bigint NOT NULL,
    output_tokens       bigint NOT NULL,
    cost_usd            numeric,
    input_price_1m      numeric,
    output_price_1m     numeric,
    latency_ms          integer,
    status_code         integer,
    is_error            boolean NOT NULL,
    error_message       text,
    request_started_at  timestamptz NOT NULL,
    request_finished_at timestamptz,
    received_at         timestamptz NOT NULL DEFAULT now(),
    metadata            jsonb,
    created_at          timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_llm_events_idempotency
    ON llm_events (project_id, event_id) WHERE event_id IS NOT NULL;
CREATE INDEX idx_llm_events_project_time
    ON llm_events (project_id, request_started_at DESC, id DESC);
```

`006_create_llm_events.down.sql`:

```sql
DROP TABLE IF EXISTS llm_events;
```

- [ ] **Step 8: Create `internal/infrastructure/postgres/migrate.go`**

```go
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"router-lens/migrations"
)

// Migrate applies all up migrations using goose over a database/sql handle
// derived from the pgx pool.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("postgres: set dialect: %w", err)
	}
	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf("postgres: migrate up: %w", err)
	}
	return nil
}
```

- [ ] **Step 9: Verify build + goose can parse the FS**

Run: `go build ./...`
Expected: succeeds. (Full apply is exercised in Task 8 against a live DB.)

- [ ] **Step 10: Commit**

```bash
git add migrations/ internal/infrastructure/postgres/migrate.go
git commit -m "feat: migrations 001-006 and embedded goose runner"
```

---

### Task 7: Echo server, error middleware, health endpoints, SPA-fallback stub, main wiring

**TDD:** no — wiring. A small httptest covers `/healthz`.

**Files:**
- Create: `internal/infrastructure/http/server.go`
- Create: `internal/infrastructure/http/middleware/error_middleware.go`
- Create: `internal/infrastructure/http/middleware/lang_middleware.go`
- Create: `internal/web/web.go` (SPA-fallback stub; real embed lands in Plan 07)
- Create: `cmd/server/main.go`
- Test: `internal/infrastructure/http/server_test.go`

**Interfaces:**
- Consumes: `Config`, `shared/response`, `shared/errors`.
- Produces:
  ```go
  // http
  func NewServer(cfg app.Config) *echo.Echo               // mounts /api/v1 group + SPA fallback
  func RegisterHealth(g *echo.Group, ready func() bool)   // /healthz, /readyz
  // middleware
  func Lang(next echo.HandlerFunc) echo.HandlerFunc        // resolves i18n.Lang into the context
  // web
  func SPAHandler() echo.HandlerFunc                       // stub returns 200 "RouterLens" until Plan 07
  ```

- [ ] **Step 1: Write the failing test**

```go
package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"router-lens/internal/app"
)

func TestHealthz(t *testing.T) {
	e := NewServer(app.Config{AppEnv: "development", AppPort: "8080"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz: got %d want 200", rec.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/infrastructure/http/ -run TestHealthz -v`
Expected: FAIL — `NewServer` undefined.

- [ ] **Step 3: Write the error middleware**

```go
// Package middleware holds Echo middleware shared across routes.
package middleware

import (
	"github.com/labstack/echo/v4"

	"router-lens/internal/shared/response"
)

// ErrorHandler is Echo's central HTTPErrorHandler: it routes every handler
// error through the canonical response envelope.
func ErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	_ = response.Error(c, err)
}
```

- [ ] **Step 3b: Write the language middleware (header-driven)**

```go
package middleware

import (
	"github.com/labstack/echo/v4"

	"router-lens/internal/shared/i18n"
)

// Lang resolves the request language from the Accept-Language header and stores
// it in the Echo context for response localization.
func Lang(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Set(i18n.ContextKey, i18n.Resolve(c.Request().Header.Get("Accept-Language")))
		return next(c)
	}
}
```

- [ ] **Step 4: Write the SPA-fallback stub**

```go
// Package web serves the embedded frontend. Until Plan 07 embeds the real
// build, it returns a placeholder so the route exists.
package web

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func SPAHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.String(http.StatusOK, "RouterLens — frontend not yet embedded")
	}
}
```

- [ ] **Step 5: Write the server**

```go
// Package http builds the Echo application.
package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	emw "github.com/labstack/echo/v4/middleware"

	"router-lens/internal/app"
	"router-lens/internal/infrastructure/http/middleware"
	"router-lens/internal/web"
)

const ingestionBodyLimit = "64KB"

// NewServer constructs the Echo instance with shared middleware, the
// /api/v1 group, and the SPA fallback for all other paths.
func NewServer(cfg app.Config) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.Debug = !cfg.IsProduction()
	e.HTTPErrorHandler = middleware.ErrorHandler

	e.Use(emw.Recover())
	e.Use(emw.Logger())
	e.Use(emw.RequestID())
	e.Use(emw.BodyLimit(ingestionBodyLimit))
	e.Use(middleware.Lang)

	api := e.Group("/api/v1")
	RegisterHealth(api, func() bool { return true })

	// SPA fallback: anything not under /api/v1 serves the frontend.
	e.GET("/*", web.SPAHandler())
	return e
}

// RegisterHealth mounts liveness and readiness probes.
func RegisterHealth(g *echo.Group, ready func() bool) {
	g.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	g.GET("/readyz", func(c echo.Context) error {
		if !ready() {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "not ready"})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
	})
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./internal/infrastructure/http/ -run TestHealthz -v`
Expected: PASS.

- [ ] **Step 7: Write `cmd/server/main.go` (Fx app + lifecycle; `-migrate-only` non-Fx path)**

```go
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"go.uber.org/fx"

	"router-lens/internal/app"
	infrahttp "router-lens/internal/infrastructure/http"
	"router-lens/internal/infrastructure/postgres"
)

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "apply migrations then exit")
	flag.Parse()

	if *migrateOnly {
		if err := migrateAndExit(); err != nil {
			log.Fatalf("migrate: %v", err)
		}
		return
	}

	fx.New(
		fx.Provide(
			app.Load,            // () -> (app.Config, error)
			providePool,         // (fx.Lifecycle, app.Config) -> (*pgxpool.Pool, error)
			infrahttp.NewServer, // (app.Config) -> *echo.Echo
		),
		fx.Invoke(runMigrations), // runs during startup, before the server listens
		fx.Invoke(startServer),   // binds the HTTP server to the lifecycle
	).Run()
}

// providePool opens the pool and ties its lifetime to the fx lifecycle.
func providePool(lc fx.Lifecycle, cfg app.Config) (*pgxpool.Pool, error) {
	pool, err := postgres.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{OnStop: func(context.Context) error { pool.Close(); return nil }})
	return pool, nil
}

// runMigrations applies migrations during startup, before the server listens.
func runMigrations(pool *pgxpool.Pool) error {
	return postgres.Migrate(context.Background(), pool)
}

// startServer binds the HTTP server to the fx lifecycle. fx.Run handles SIGINT/SIGTERM.
func startServer(lc fx.Lifecycle, cfg app.Config, e *echo.Echo) {
	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      e,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Fatalf("listen: %v", err)
				}
			}()
			log.Printf("RouterLens listening on :%s", cfg.AppPort)
			return nil
		},
		OnStop: func(ctx context.Context) error { return srv.Shutdown(ctx) },
	})
}

// migrateAndExit is the non-Fx path for `-migrate-only`.
func migrateAndExit() error {
	cfg, err := app.Load()
	if err != nil {
		return err
	}
	pool, err := postgres.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	return postgres.Migrate(context.Background(), pool)
}
```

- [ ] **Step 8: Verify build + test**

Run: `go build ./... && go test ./internal/infrastructure/http/ -v`
Expected: build succeeds, `TestHealthz` PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/infrastructure/http/ internal/web/ cmd/server/main.go
git commit -m "feat: echo server, central error handler, health probes, graceful shutdown"
```

---

### Task 8: Docker Compose + multi-stage Dockerfile + boot verification

**TDD:** no — infra. Verified by `docker compose up` and curling `/healthz`.

**Files:**
- Create: `docker-compose.yml`
- Create: `Dockerfile`
- Create: `.dockerignore`

**Interfaces:**
- Consumes: the `make`/binary entrypoints and env from Task 1.
- Produces: a `postgres` + `app` topology answering on `:8080`.

- [ ] **Step 1: Create `.dockerignore`**

```
.git
bin
node_modules
apps/web/node_modules
apps/web/dist
*.md
.env
```

- [ ] **Step 2: Create `Dockerfile` (backend only for now; FE embed added in Plan 07)**

```dockerfile
# syntax=docker/dockerfile:1
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/routerlens ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/routerlens /routerlens
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/routerlens"]
```

- [ ] **Step 3: Create `docker-compose.yml`**

```yaml
services:
  postgres:
    image: postgres:17
    environment:
      POSTGRES_USER: routerlens
      POSTGRES_PASSWORD: routerlens
      POSTGRES_DB: routerlens
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U routerlens"]
      interval: 5s
      timeout: 3s
      retries: 10

  app:
    build: .
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      APP_ENV: production
      APP_PORT: "8080"
      DATABASE_URL: postgres://routerlens:routerlens@postgres:5432/routerlens?sslmode=disable
      SESSION_SECRET: change-me-in-production
      COOKIE_CROSS_SITE: "false"
      MAX_BACKDATE_DAYS: "7"
      RETENTION_DAYS: "0"
    ports:
      - "8080:8080"

volumes:
  pgdata:
```

- [ ] **Step 4: Boot and verify**

Run: `docker compose up --build -d && sleep 8 && curl -fsS http://localhost:8080/api/v1/healthz`
Expected: `{"status":"ok"}`. App logs show migrations applied.

- [ ] **Step 5: Verify migrations created the tables**

Run: `docker compose exec postgres psql -U routerlens -d routerlens -c "\dt"`
Expected: lists `users, sessions, projects, api_keys, pricing_rules, llm_events, goose_db_version`.

- [ ] **Step 6: Tear down + commit**

```bash
docker compose down
git add Dockerfile docker-compose.yml .dockerignore
git commit -m "feat: docker compose (postgres + app) and multi-stage dockerfile"
```

---

## Self-Review

- **Spec coverage (Plan 01 scope):** module scaffold ✓, config ✓, Postgres ✓, all 6 migrations with the partial idempotency index + composite analytics index ✓, error/response shared kit ✓, Echo server + central error handler + health probes ✓, SPA-fallback stub ✓, docker-compose + Dockerfile ✓. Auth/projects/events/analytics intentionally deferred to later plans.
- **Placeholder scan:** none — every step carries runnable code or an exact command.
- **Type consistency:** `app.Config` fields used by `NewServer`, `NewPool`, `Migrate` match Task 2's struct. `response.Error` consumes `errors.AppError`/`HTTPStatus` from Task 3. `Migrate` consumes `*pgxpool.Pool` from Task 5. `migrations.FS` consumed by `Migrate`.

## Next

Validated 8-plan roadmap (Codex decomposition pass applied — Plan 05 split into ingestion/logs/CSV + a separate analytics plan):

- **02** Shared kit + security + cost calculator (TDD-first)
- **03** Auth + first-run setup (sets the httpOnly session cookie on login, `SameSite=Lax`; no CSRF token — decision 13)
- **04** Projects + API Keys + **Pricing CRUD** (the pricing repository Plan 05 ingestion depends on)
- **05** Event ingestion + logs + CSV (depends on Plan 04 pricing repo for cost lookup and api_keys repo for the ingestion middleware)
- **06** Analytics endpoints (overview/tokens/cost/latency/errors/providers/models)
- **07** Frontend (pages + api layer + components + react-doctor)
- **08** Embed + DoD (real `internal/web` embed.FS, final Dockerfile that builds FE then embeds it — the Plan 01 Dockerfile is an interim backend-only stage, seed, full verification)
