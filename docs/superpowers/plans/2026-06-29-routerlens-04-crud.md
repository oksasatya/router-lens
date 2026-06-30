# RouterLens Plan 04 — Projects + API Keys + Pricing CRUD

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the three session-authenticated CRUD bounded contexts — Projects, API Keys, Pricing Rules — as full hexagonal vertical slices (domain → application → postgres → http → Fx wiring), mirroring the auth slice shipped in Plan 03.

**Architecture:** Hexagonal + DDD. Each feature gets `domain/<ctx>/` (entity + repository port + pure rules), `application/<ctx>/` (use case service, HTTP-free), `infrastructure/postgres/<ctx>_repository.go` (port impl), `infrastructure/http/handler/<ctx>_handler.go` (parse → use case → response). Wiring is appended to the existing flat `fx.Provide` list in `cmd/server/main.go` plus one `fx.Invoke(registerXRoutes)` per feature. The session middleware is provided once via Fx and injected into every route registrar (removes the duplicated `middleware.Session(...)` construction).

**Tech Stack:** Go 1.26, Echo v4, Uber Fx, pgx/v5 (pgxpool), shopspring/decimal, goose migrations (already applied — schema 003/004/005 exist). New dependency: `github.com/jackc/pgx-shopspring-decimal` (numeric ↔ decimal codec, registered once on the pool — used by Pricing now and Events in Plan 05).

## Global Constraints

- **Layering (HARD):** `domain/` imports only stdlib + `shopspring/decimal`. Never Echo/pgx/SQL. Repository interfaces live in `domain/<ctx>/repository.go`; implementations in `infrastructure/postgres/`. Use cases never import `echo` or reference HTTP status codes. Handlers contain no business logic and run no SQL.
- **Naming clash:** the application package and the domain package share a name (`project`, `apikey`, `pricing`). In the application layer, alias the domain import: `projectdomain "router-lens/internal/domain/project"`, `apikeydomain`, `pricingdomain`. The application package keeps the short name (`package project`).
- **Error envelope + i18n:** handlers return errors; the central `middleware.ErrorHandler` + `response.Error` localize them. Every new error code is a `const` in `shared/i18n` with an EN+ID catalog entry, grouped by domain section. Cross-cutting kinds reuse the existing `apperrors.Kind*`. Domain-specific codes are namespaced (`project.not_found`, `apikey.not_found`, `pricing.not_found`, `pricing.duplicate`, `project.slug_taken`).
- **Anti-duplication:** the paginated list response shape is defined ONCE in `shared/response` (`response.Paginated`) and reused by every offset-paginated list. The unique-violation detector is defined ONCE in `infrastructure/postgres` (`isUniqueViolation`) and reused by the project and pricing repos.
- **Money:** prices are `decimal.Decimal` end to end; never float. Currency defaults to `"USD"` when the request omits it.
- **Ownership:** `projects.owner_user_id` is stamped from the authenticated user (`mw.CurrentUser(c).ID`) at create time. v0.1 has shared visibility — list/get/update/delete are NOT filtered by owner.
- **Slug:** derived from `name` at create (lowercase, ASCII-alnum runs joined by single hyphens), **immutable** after create. A per-owner slug collision returns `409 project.slug_taken`.

### Sonar guardrails — write compliant from the first commit

```
Go:
- go:S107 — ≤7 params (project preference ≤5; 6+ = smell → Deps/Opts struct).
- go:S3776 — cognitive complexity ≤15 → extract helpers; tests use t.Run subtests.
- go:S1192 — const for any string literal duplicated 3+ times (error codes, column lists).
- errcheck — handle every returned error; never `_ = fallible()`. Wrap with %w; sentinel + errors.Is/As.
- gosec — parameterized SQL only (no string-concat), crypto/rand for tokens (already in security pkg).

When fixing one instance, check sibling files for the same anti-pattern and fix-forward.
Review the diff against this list BEFORE marking compliant.
```

### Skill brief for implementer subagents (every task)

> Invoke `golang-expert` first — it is a hub skill and auto-chains the full Go discipline family (go-patterns / go-review / go-test / go-error-handling) + `senior-backend` + `senior-security` + `algorithmic-complexity`; follow its Auto-chain section. Apply `ponytail` (YAGNI): the simplest thing that works, reuse existing shared helpers before writing new ones, no speculative repository methods. Honor the Global Constraints and the Sonar block above.

### Algorithmic complexity (§8, Bahasa Indonesia)

- **Projects list:** `SELECT ... ORDER BY created_at DESC LIMIT $1 OFFSET $2` + `COUNT(*)`. Tabel `projects` kecil (puluhan baris milik satu admin) → sequential scan murah, **O(n)** untuk count + **O(limit)** untuk fetch, `n` = jumlah project. Offset pagination memang O(n) skip, tapi karena `n` kecil ini aman; kalau kelak project membengkak, baru pertimbangkan keyset. Tidak ada N+1: satu query list + satu query count, bukan query per baris.
- **API keys list:** `WHERE project_id=$1` naik ke `idx_api_keys_project_id` → **O(log n + k)**, `k` = kunci per project (sedikit).
- **Pricing list:** `SELECT * FROM pricing_rules` tanpa filter; tabel kecil (satu baris per provider/model) → **O(n)** seq scan, aman.
- Tidak ada loop bersarang, tidak ada query di dalam loop, tidak ada `.find()`-in-loop di seluruh plan ini.

### TDD verdicts (per §16)

- **Task 1 — Projects:** slug helper `TDD: yes` (pure parser, clear input→output). Application service `TDD: yes` (fake repo; tests the ErrNotFound/ErrSlugTaken → AppError mapping). Postgres repo + handler `TDD: no` (integration test against real Postgres after; verify-by running).
- **Task 2 — API Keys:** application service `TDD: yes` (fake repos; tests project-existence 404 + plaintext-returned-once). Postgres repo + handler `TDD: no` (integration after).
- **Task 3 — Pricing:** application service `TDD: yes` (fake repo; tests non-negative price validation + ErrNotFound/duplicate mapping). db.go decimal-codec registration + postgres repo + handler `TDD: no` (integration after).

---

## Task 1: Projects CRUD

**Files:**
- Create: `apps/backend/internal/domain/project/entity.go`
- Create: `apps/backend/internal/domain/project/repository.go`
- Create: `apps/backend/internal/domain/project/slug.go`
- Create: `apps/backend/internal/domain/project/slug_test.go`
- Create: `apps/backend/internal/application/project/project.go`
- Create: `apps/backend/internal/application/project/project_test.go`
- Modify: `apps/backend/internal/shared/response/response.go` (add `Paginated`)
- Create: `apps/backend/internal/shared/response/paginated_test.go`
- Create: `apps/backend/internal/infrastructure/postgres/pgerr.go` (shared `isUniqueViolation`)
- Create: `apps/backend/internal/infrastructure/postgres/project_repository.go`
- Create: `apps/backend/internal/infrastructure/postgres/project_repository_test.go`
- Create: `apps/backend/internal/infrastructure/http/handler/project_handler.go`
- Modify: `apps/backend/internal/infrastructure/http/middleware/session_middleware.go` (none — reference only)
- Modify: `apps/backend/cmd/server/main.go` (provide session middleware once; wire project)
- Modify: `apps/backend/internal/shared/i18n/i18n.go` (add `project.*` codes)

**Interfaces:**
- Consumes: `apperrors.{Kind*, New}`, `i18n` codes, `mw.CurrentUser`, `pagination.{ParseOffset, Offset}`, `response.{Data, Created, NoContent, LangOf}`, `validator.Validator`, the existing `bindAndValidate` helper in package `handler`.
- Produces (later tasks rely on these): `projectdomain.ProjectRepository` (API Keys uses `FindByID`), `project.NewService`, `handler.NewProjectHandler`, `response.Paginated`, `postgres.isUniqueViolation`, `postgres.NewProjectRepository`.

- [ ] **Step 1: Write the failing slug test**

`apps/backend/internal/domain/project/slug_test.go`:
```go
package project

import "testing"

func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"My Project", "my-project"},
		{"  Hello!!  World  ", "hello-world"},
		{"already-a-slug", "already-a-slug"},
		{"Trailing!!!", "trailing"},
		{"a---b", "a-b"},
		{"Café 123", "caf-123"},
		{"", "project"},
		{"!!!", "project"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := Slugify(c.in); got != c.want {
				t.Fatalf("Slugify(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd apps/backend && go test ./internal/domain/project/ -run TestSlugify`
Expected: FAIL — `undefined: Slugify`.

- [ ] **Step 3: Write the slug helper**

`apps/backend/internal/domain/project/slug.go`:
```go
package project

import "strings"

const fallbackSlug = "project"

// Slugify converts a display name to a URL-safe slug: lowercase, ASCII
// alphanumeric runs joined by single hyphens, no leading/trailing hyphen.
// Non-ASCII characters are dropped. Empty or all-punctuation input yields
// "project". ponytail: ASCII-only is sufficient for a dev tool; widen if a
// real consumer needs unicode slugs.
func Slugify(name string) string {
	var b strings.Builder
	pendingHyphen := false
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if pendingHyphen && b.Len() > 0 {
				b.WriteByte('-')
			}
			pendingHyphen = false
			b.WriteRune(r)
			continue
		}
		pendingHyphen = true
	}
	if b.Len() == 0 {
		return fallbackSlug
	}
	return b.String()
}
```

- [ ] **Step 4: Run it to verify it passes**

Run: `cd apps/backend && go test ./internal/domain/project/ -run TestSlugify`
Expected: PASS.

- [ ] **Step 5: Write the project domain entity + repository port**

`apps/backend/internal/domain/project/entity.go`:
```go
// Package project is the Project bounded context: the aggregate, its repository
// port, and the slug rule. Imports only stdlib (domain purity).
package project

import (
	"errors"
	"time"
)

// ErrNotFound is returned when a project row is absent.
var ErrNotFound = errors.New("project: not found")

// ErrSlugTaken is returned when (owner_user_id, slug) already exists.
var ErrSlugTaken = errors.New("project: slug already taken for owner")

type Project struct {
	ID          string
	OwnerUserID string
	Name        string
	Slug        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
```

`apps/backend/internal/domain/project/repository.go`:
```go
package project

import "context"

// ProjectRepository is the port for persisting and querying Project aggregates.
type ProjectRepository interface {
	// Create inserts p, setting ID/CreatedAt/UpdatedAt. Returns ErrSlugTaken on
	// a (owner_user_id, slug) unique violation.
	Create(ctx context.Context, p *Project) error
	List(ctx context.Context, limit, offset int) ([]*Project, error)
	Count(ctx context.Context) (int, error)
	FindByID(ctx context.Context, id string) (*Project, error)
	// Update changes name + description (slug is immutable). Returns ErrNotFound.
	Update(ctx context.Context, p *Project) error
	// Delete removes the row. Returns ErrNotFound when no row matched.
	Delete(ctx context.Context, id string) error
}
```

- [ ] **Step 6: Write the failing application-service test**

`apps/backend/internal/application/project/project_test.go`:
```go
package project

import (
	"context"
	"errors"
	"testing"

	projectdomain "router-lens/internal/domain/project"
	apperrors "router-lens/internal/shared/errors"
)

type fakeRepo struct {
	createErr error
	findErr   error
	updateErr error
	deleteErr error
	got       *projectdomain.Project
}

func (f *fakeRepo) Create(_ context.Context, p *projectdomain.Project) error {
	f.got = p
	if f.createErr != nil {
		return f.createErr
	}
	p.ID = "p1"
	return nil
}
func (f *fakeRepo) List(context.Context, int, int) ([]*projectdomain.Project, error) {
	return []*projectdomain.Project{{ID: "p1"}}, nil
}
func (f *fakeRepo) Count(context.Context) (int, error) { return 1, nil }
func (f *fakeRepo) FindByID(context.Context, string) (*projectdomain.Project, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	return &projectdomain.Project{ID: "p1", Name: "x"}, nil
}
func (f *fakeRepo) Update(_ context.Context, p *projectdomain.Project) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.got = p
	return nil
}
func (f *fakeRepo) Delete(context.Context, string) error { return f.deleteErr }

func TestCreate(t *testing.T) {
	t.Run("derives slug and stamps owner", func(t *testing.T) {
		f := &fakeRepo{}
		s := NewService(f)
		p, err := s.Create(context.Background(), "owner1", "My App", "desc")
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if p.Slug != "my-app" || f.got.OwnerUserID != "owner1" {
			t.Fatalf("slug=%q owner=%q", p.Slug, f.got.OwnerUserID)
		}
	})
	t.Run("maps slug collision to conflict AppError", func(t *testing.T) {
		f := &fakeRepo{createErr: projectdomain.ErrSlugTaken}
		_, err := NewService(f).Create(context.Background(), "o", "n", "")
		ae, ok := apperrors.As(err)
		if !ok || ae.Kind != apperrors.KindConflict {
			t.Fatalf("want conflict AppError, got %v", err)
		}
	})
}

func TestGetNotFound(t *testing.T) {
	f := &fakeRepo{findErr: projectdomain.ErrNotFound}
	_, err := NewService(f).Get(context.Background(), "missing")
	ae, ok := apperrors.As(err)
	if !ok || ae.Kind != apperrors.KindNotFound {
		t.Fatalf("want not_found AppError, got %v", err)
	}
}

func TestDeleteUnknownErrorPropagates(t *testing.T) {
	sentinel := errors.New("db down")
	f := &fakeRepo{deleteErr: sentinel}
	err := NewService(f).Delete(context.Background(), "p1")
	if !errors.Is(err, sentinel) {
		t.Fatalf("want raw error propagated, got %v", err)
	}
}
```

- [ ] **Step 7: Run it to verify it fails**

Run: `cd apps/backend && go test ./internal/application/project/`
Expected: FAIL — `undefined: NewService`.

- [ ] **Step 8: Write the application service**

`apps/backend/internal/application/project/project.go`:
```go
// Package project holds the Project CRUD use cases. Depends only on the domain
// port + shared/errors (no HTTP, no SQL).
package project

import (
	"context"
	"errors"

	projectdomain "router-lens/internal/domain/project"
	apperrors "router-lens/internal/shared/errors"
)

const (
	codeNotFound  = "project.not_found"
	codeSlugTaken = "project.slug_taken"
)

type Service struct{ repo projectdomain.ProjectRepository }

func NewService(repo projectdomain.ProjectRepository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, ownerUserID, name, description string) (*projectdomain.Project, error) {
	p := &projectdomain.Project{
		OwnerUserID: ownerUserID,
		Name:        name,
		Slug:        projectdomain.Slugify(name),
		Description: description,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		if errors.Is(err, projectdomain.ErrSlugTaken) {
			return nil, apperrors.New(apperrors.KindConflict, codeSlugTaken, "a project with this name already exists")
		}
		return nil, err
	}
	return p, nil
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]*projectdomain.Project, int, error) {
	items, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) Get(ctx context.Context, id string) (*projectdomain.Project, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, s.mapNotFound(err)
	}
	return p, nil
}

func (s *Service) Update(ctx context.Context, id, name, description string) (*projectdomain.Project, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, s.mapNotFound(err)
	}
	p.Name = name
	p.Description = description
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, s.mapNotFound(err)
	}
	return p, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return s.mapNotFound(err)
	}
	return nil
}

// mapNotFound translates the domain ErrNotFound sentinel to a 404 AppError and
// passes any other error through unchanged.
func (s *Service) mapNotFound(err error) error {
	if errors.Is(err, projectdomain.ErrNotFound) {
		return apperrors.New(apperrors.KindNotFound, codeNotFound, "project not found")
	}
	return err
}
```

- [ ] **Step 9: Run it to verify it passes**

Run: `cd apps/backend && go test ./internal/application/project/`
Expected: PASS.

- [ ] **Step 10: Write the failing `response.Paginated` test**

`apps/backend/internal/shared/response/paginated_test.go`:
```go
package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestPaginated(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := Paginated(c, http.StatusOK, []string{"a", "b"}, 2, 20, 41); err != nil {
		t.Fatalf("Paginated: %v", err)
	}
	var got struct {
		Data struct {
			Items      []string `json:"items"`
			Pagination struct {
				Page  int `json:"page"`
				Limit int `json:"limit"`
				Total int `json:"total"`
			} `json:"pagination"`
		} `json:"data"`
		Meta struct {
			Timestamp string `json:"timestamp"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Data.Items) != 2 || got.Data.Pagination.Total != 41 || got.Data.Pagination.Page != 2 {
		t.Fatalf("bad envelope: %+v", got)
	}
	if got.Meta.Timestamp == "" {
		t.Fatal("meta missing")
	}
}
```

- [ ] **Step 11: Run it to verify it fails**

Run: `cd apps/backend && go test ./internal/shared/response/ -run TestPaginated`
Expected: FAIL — `undefined: Paginated`.

- [ ] **Step 12: Add the `Paginated` helper**

Append to `apps/backend/internal/shared/response/response.go`:
```go
type pageMeta struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

type pageData struct {
	Items      any      `json:"items"`
	Pagination pageMeta `json:"pagination"`
}

// Paginated writes the canonical list envelope:
// { "data": { "items": [...], "pagination": {page, limit, total} }, "meta": {...} }.
// items must be a slice; nil renders as an empty array on the client via omitempty-free encoding.
func Paginated(c echo.Context, status int, items any, page, limit, total int) error {
	return c.JSON(status, envelope{
		Data: pageData{Items: items, Pagination: pageMeta{Page: page, Limit: limit, Total: total}},
		Meta: meta(c),
	})
}
```

- [ ] **Step 13: Run it to verify it passes**

Run: `cd apps/backend && go test ./internal/shared/response/`
Expected: PASS.

- [ ] **Step 14: Add the shared unique-violation detector**

`apps/backend/internal/infrastructure/postgres/pgerr.go`:
```go
package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// pgUniqueViolation is the SQLSTATE for a unique_violation.
const pgUniqueViolation = "23505"

// isUniqueViolation reports whether err is a Postgres unique-constraint violation.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation
}
```

- [ ] **Step 15: Write the project postgres repository**

`apps/backend/internal/infrastructure/postgres/project_repository.go`:
```go
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"router-lens/internal/domain/project"
)

type ProjectRepository struct{ pool *pgxpool.Pool }

func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository { return &ProjectRepository{pool: pool} }

var _ project.ProjectRepository = (*ProjectRepository)(nil)

const projectColumns = `id, owner_user_id, name, slug, COALESCE(description, ''), created_at, updated_at`

func (r *ProjectRepository) Create(ctx context.Context, p *project.Project) error {
	const q = `INSERT INTO projects (owner_user_id, name, slug, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, q, p.OwnerUserID, p.Name, p.Slug, p.Description).
		Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if isUniqueViolation(err) {
		return project.ErrSlugTaken
	}
	return err
}

func (r *ProjectRepository) List(ctx context.Context, limit, offset int) ([]*project.Project, error) {
	q := `SELECT ` + projectColumns + ` FROM projects ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*project.Project, 0, limit)
	for rows.Next() {
		var p project.Project
		if err := rows.Scan(&p.ID, &p.OwnerUserID, &p.Name, &p.Slug, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

func (r *ProjectRepository) Count(ctx context.Context) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM projects`).Scan(&n)
	return n, err
}

func (r *ProjectRepository) FindByID(ctx context.Context, id string) (*project.Project, error) {
	q := `SELECT ` + projectColumns + ` FROM projects WHERE id = $1`
	var p project.Project
	err := r.pool.QueryRow(ctx, q, id).
		Scan(&p.ID, &p.OwnerUserID, &p.Name, &p.Slug, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, project.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProjectRepository) Update(ctx context.Context, p *project.Project) error {
	const q = `UPDATE projects SET name = $2, description = $3, updated_at = now()
		WHERE id = $1
		RETURNING updated_at`
	err := r.pool.QueryRow(ctx, q, p.ID, p.Name, p.Description).Scan(&p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return project.ErrNotFound
	}
	return err
}

func (r *ProjectRepository) Delete(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return project.ErrNotFound
	}
	return nil
}
```

- [ ] **Step 16: Write the project integration test**

`apps/backend/internal/infrastructure/postgres/project_repository_test.go`. The existing `auth_repository_test.go` defines `testPool(t) context.Context` — a skip-guard that returns `context.Background()` and skips when `TEST_DATABASE_URL` is unset; it does NOT return a pool. This task adds a richer shared helper `setupTestDB(t) (context.Context, *pgxpool.Pool)` (open pool + migrate + TRUNCATE all tables for an isolated run) and uses it from every postgres integration test in this package (Tasks 2–3 + Plan 05 reuse it — define it ONCE here).
```go
package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"router-lens/internal/domain/project"
)

// setupTestDB opens a pool against TEST_DATABASE_URL (skips when unset), applies
// migrations, and truncates all tables so each integration test starts clean
// (re-running must not fail on unique constraints). Shared by every postgres
// integration test in this package.
func setupTestDB(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	ctx := context.Background()
	pool, err := NewPool(ctx, url)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE llm_events, api_keys, pricing_rules, sessions, projects, users CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return ctx, pool
}

func TestProjectRepository(t *testing.T) {
	ctx, pool := setupTestDB(t)
	repo := NewProjectRepository(pool)

	// An owner user is required by the FK. Insert one directly.
	var ownerID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3) RETURNING id`,
		"proj-owner@test.local", "x", "Owner").Scan(&ownerID); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	t.Run("create then find", func(t *testing.T) {
		p := &project.Project{OwnerUserID: ownerID, Name: "Alpha", Slug: "alpha", Description: "first"}
		if err := repo.Create(ctx, p); err != nil {
			t.Fatalf("create: %v", err)
		}
		if p.ID == "" {
			t.Fatal("id not set")
		}
		got, err := repo.FindByID(ctx, p.ID)
		if err != nil || got.Name != "Alpha" || got.Slug != "alpha" {
			t.Fatalf("find: %+v err=%v", got, err)
		}
	})

	t.Run("duplicate slug for same owner -> ErrSlugTaken", func(t *testing.T) {
		_ = repo.Create(ctx, &project.Project{OwnerUserID: ownerID, Name: "Beta", Slug: "dup", Description: ""})
		err := repo.Create(ctx, &project.Project{OwnerUserID: ownerID, Name: "Beta2", Slug: "dup", Description: ""})
		if err != project.ErrSlugTaken {
			t.Fatalf("want ErrSlugTaken, got %v", err)
		}
	})

	t.Run("update changes name, keeps slug", func(t *testing.T) {
		p := &project.Project{OwnerUserID: ownerID, Name: "Gamma", Slug: "gamma", Description: ""}
		_ = repo.Create(ctx, p)
		p.Name = "Gamma Renamed"
		if err := repo.Update(ctx, p); err != nil {
			t.Fatalf("update: %v", err)
		}
		got, _ := repo.FindByID(ctx, p.ID)
		if got.Name != "Gamma Renamed" || got.Slug != "gamma" {
			t.Fatalf("update result: %+v", got)
		}
	})

	t.Run("delete missing -> ErrNotFound", func(t *testing.T) {
		if err := repo.Delete(ctx, "00000000-0000-0000-0000-000000000000"); err != project.ErrNotFound {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})
}
```
NOTE: if `auth_repository_test.go`'s pool helper has a different name/signature, match it exactly. Do not introduce a second connection helper.

- [ ] **Step 17: Write the project HTTP handler**

`apps/backend/internal/infrastructure/http/handler/project_handler.go`:
```go
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	projectapp "router-lens/internal/application/project"
	projectdomain "router-lens/internal/domain/project"
	mw "router-lens/internal/infrastructure/http/middleware"
	"router-lens/internal/shared/pagination"
	"router-lens/internal/shared/response"
	"router-lens/internal/shared/validator"
)

type ProjectHandler struct {
	svc *projectapp.Service
	v   *validator.Validator
}

func NewProjectHandler(svc *projectapp.Service, v *validator.Validator) *ProjectHandler {
	return &ProjectHandler{svc: svc, v: v}
}

func (h *ProjectHandler) Register(api *echo.Group, session echo.MiddlewareFunc) {
	api.POST("/projects", h.create, session)
	api.GET("/projects", h.list, session)
	api.GET("/projects/:id", h.get, session)
	api.PUT("/projects/:id", h.update, session)
	api.DELETE("/projects/:id", h.delete, session)
}

type projectRequest struct {
	Name        string `json:"name" validate:"required,max=120"`
	Description string `json:"description" validate:"max=500"`
}

type projectDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func toProjectDTO(p *projectdomain.Project) projectDTO {
	return projectDTO{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		CreatedAt:   p.CreatedAt.UTC().Format(timeLayout),
		UpdatedAt:   p.UpdatedAt.UTC().Format(timeLayout),
	}
}

func (h *ProjectHandler) create(c echo.Context) error {
	var req projectRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	p, err := h.svc.Create(c.Request().Context(), mw.CurrentUser(c).ID, req.Name, req.Description)
	if err != nil {
		return err
	}
	return response.Created(c, toProjectDTO(p))
}

func (h *ProjectHandler) list(c echo.Context) error {
	off := pagination.ParseOffset(c.QueryParam("page"), c.QueryParam("limit"))
	items, total, err := h.svc.List(c.Request().Context(), off.Limit, off.SQLOffset())
	if err != nil {
		return err
	}
	dtos := make([]projectDTO, 0, len(items))
	for _, p := range items {
		dtos = append(dtos, toProjectDTO(p))
	}
	return response.Paginated(c, http.StatusOK, dtos, off.Page, off.Limit, total)
}

func (h *ProjectHandler) get(c echo.Context) error {
	p, err := h.svc.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, toProjectDTO(p))
}

func (h *ProjectHandler) update(c echo.Context) error {
	var req projectRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	p, err := h.svc.Update(c.Request().Context(), c.Param("id"), req.Name, req.Description)
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, toProjectDTO(p))
}

func (h *ProjectHandler) delete(c echo.Context) error {
	if err := h.svc.Delete(c.Request().Context(), c.Param("id")); err != nil {
		return err
	}
	return response.NoContent(c)
}
```
NOTE: `timeLayout` is a shared `const timeLayout = time.RFC3339` in package `handler`. If `auth_handler.go` does not already define it, add it once in a new file `apps/backend/internal/infrastructure/http/handler/handler.go`:
```go
package handler

import "time"

const timeLayout = time.RFC3339
```
Check first — if a time-format constant already exists in package `handler`, reuse it instead of adding a duplicate (S1192 / anti-dup).

- [ ] **Step 18: Add the `project.*` i18n codes**

In `apps/backend/internal/shared/i18n/i18n.go`, extend the code `const` block (after the auth section) and the `catalog` map:
```go
	// --- project ---
	CodeProjectNotFound  = "project.not_found"
	CodeProjectSlugTaken = "project.slug_taken"
```
```go
	// --- project ---
	CodeProjectNotFound:  {EN: "Project not found", ID: "Proyek tidak ditemukan"},
	CodeProjectSlugTaken: {EN: "A project with this name already exists", ID: "Proyek dengan nama ini sudah ada"},
```
Then change `application/project/project.go` to reference these consts instead of local string literals: replace `codeNotFound`/`codeSlugTaken` local consts with `i18n.CodeProjectNotFound` / `i18n.CodeProjectSlugTaken` (import `"router-lens/internal/shared/i18n"`). This keeps the code string defined once (S1192). Re-run `go test ./internal/application/project/`.

- [ ] **Step 19: Provide the session middleware once + wire Projects in Fx**

In `apps/backend/cmd/server/main.go`:

(a) Add a provider for the shared session middleware and register the project providers. Inside `fx.Provide(...)` add:
```go
		provideSessionMiddleware, // (user.SessionRepository, user.UserRepository) -> echo.MiddlewareFunc
		fx.Annotate(postgres.NewProjectRepository, fx.As(new(project.ProjectRepository))),
		projectapp.NewService,    // (project.ProjectRepository) -> *projectapp.Service
		handler.NewProjectHandler, // (*projectapp.Service, *validator.Validator) -> *ProjectHandler
```
with imports:
```go
	project "router-lens/internal/domain/project"
	projectapp "router-lens/internal/application/project"
```
(b) Add the provider + registrar functions:
```go
// provideSessionMiddleware builds the one shared session-auth middleware.
func provideSessionMiddleware(sessions user.SessionRepository, users user.UserRepository) echo.MiddlewareFunc {
	return middleware.Session(sessions, users)
}

// registerProjectRoutes mounts the project routes behind the session middleware.
func registerProjectRoutes(e *echo.Echo, h *handler.ProjectHandler, session echo.MiddlewareFunc) {
	h.Register(e.Group("/api/v1"), session)
}
```
(c) Refactor `registerAuthRoutes` to inject the shared middleware (remove the inline `middleware.Session(...)` construction):
```go
func registerAuthRoutes(e *echo.Echo, h *handler.AuthHandler, session echo.MiddlewareFunc) {
	h.Register(e.Group("/api/v1"), session)
}
```
(d) Add the invoke:
```go
		fx.Invoke(registerProjectRoutes),
```

- [ ] **Step 20: Verify the build, vet, full unit suite**

Run:
```
cd apps/backend && gofmt -l . && go vet ./... && go build ./... && go test ./internal/... -count=1
```
Expected: gofmt prints nothing; vet/build clean; all non-integration tests PASS (integration tests skip without a DB). If a local Postgres is available, run the project integration test against an isolated container (see Plan 03 notes — use port `55432` to avoid the host's `:5432`).

- [ ] **Step 21: Commit**

```bash
git add apps/backend
git commit -m "feat: projects CRUD (Plan 04 task 1)"
```

---

## Task 2: API Keys CRUD

**Files:**
- Create: `apps/backend/internal/domain/apikey/entity.go`
- Create: `apps/backend/internal/domain/apikey/repository.go`
- Create: `apps/backend/internal/application/apikey/apikey.go`
- Create: `apps/backend/internal/application/apikey/apikey_test.go`
- Create: `apps/backend/internal/infrastructure/postgres/apikey_repository.go`
- Create: `apps/backend/internal/infrastructure/postgres/apikey_repository_test.go`
- Create: `apps/backend/internal/infrastructure/http/handler/apikey_handler.go`
- Modify: `apps/backend/cmd/server/main.go` (wire apikey)
- Modify: `apps/backend/internal/shared/i18n/i18n.go` (add `apikey.*` codes)

**Interfaces:**
- Consumes: `projectdomain.ProjectRepository` (existence check), `security.GenerateAPIKey`, `apperrors`, `i18n`, `mw.CurrentUser` (auth gate only), `response`.
- Produces: `apikeydomain.APIKeyRepository`, `apikey.NewService`, `handler.NewAPIKeyHandler`, `postgres.NewAPIKeyRepository`.

- [ ] **Step 1: Write the api-key domain entity + repository port**

`apps/backend/internal/domain/apikey/entity.go`:
```go
// Package apikey is the API Key bounded context. Only the hash is persisted;
// the plaintext is generated once at creation. Imports only stdlib.
package apikey

import (
	"errors"
	"time"
)

// ErrNotFound is returned when an api_keys row is absent.
var ErrNotFound = errors.New("apikey: not found")

type APIKey struct {
	ID         string
	ProjectID  string
	Name       string
	KeyHash    string
	KeyPrefix  string
	LastUsedAt *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
}
```

`apps/backend/internal/domain/apikey/repository.go`:
```go
package apikey

import "context"

// APIKeyRepository is the port for persisting and querying API keys.
type APIKeyRepository interface {
	// Create inserts k, setting ID + CreatedAt.
	Create(ctx context.Context, k *APIKey) error
	ListByProject(ctx context.Context, projectID string) ([]*APIKey, error)
	// Revoke soft-deletes by setting revoked_at = now(). Returns ErrNotFound
	// when no row matched the id.
	Revoke(ctx context.Context, id string) error
}
```

- [ ] **Step 2: Write the failing application-service test**

`apps/backend/internal/application/apikey/apikey_test.go`:
```go
package apikey

import (
	"context"
	"strings"
	"testing"

	apikeydomain "router-lens/internal/domain/apikey"
	projectdomain "router-lens/internal/domain/project"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/security"
)

type fakeKeyRepo struct{ created *apikeydomain.APIKey }

func (f *fakeKeyRepo) Create(_ context.Context, k *apikeydomain.APIKey) error {
	k.ID = "k1"
	f.created = k
	return nil
}
func (f *fakeKeyRepo) ListByProject(context.Context, string) ([]*apikeydomain.APIKey, error) {
	return []*apikeydomain.APIKey{{ID: "k1", KeyPrefix: "rl_live_ab"}}, nil
}
func (f *fakeKeyRepo) Revoke(context.Context, string) error { return nil }

type fakeProjRepo struct{ exists bool }

func (f *fakeProjRepo) Create(context.Context, *projectdomain.Project) error { return nil }
func (f *fakeProjRepo) List(context.Context, int, int) ([]*projectdomain.Project, error) {
	return nil, nil
}
func (f *fakeProjRepo) Count(context.Context) (int, error) { return 0, nil }
func (f *fakeProjRepo) FindByID(context.Context, string) (*projectdomain.Project, error) {
	if f.exists {
		return &projectdomain.Project{ID: "p1"}, nil
	}
	return nil, projectdomain.ErrNotFound
}
func (f *fakeProjRepo) Update(context.Context, *projectdomain.Project) error { return nil }
func (f *fakeProjRepo) Delete(context.Context, string) error                 { return nil }

func TestCreate(t *testing.T) {
	t.Run("returns plaintext once and stores hash", func(t *testing.T) {
		kr := &fakeKeyRepo{}
		s := NewService(kr, &fakeProjRepo{exists: true})
		plaintext, k, err := s.Create(context.Background(), "p1", "ci")
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if !strings.HasPrefix(plaintext, security.APIKeyPrefix) {
			t.Fatalf("plaintext missing prefix: %q", plaintext)
		}
		if k.KeyHash == "" || k.KeyHash == plaintext {
			t.Fatal("hash must be set and differ from plaintext")
		}
		if security.HashAPIKey(plaintext) != kr.created.KeyHash {
			t.Fatal("stored hash must equal HashAPIKey(plaintext)")
		}
	})
	t.Run("unknown project -> not_found AppError", func(t *testing.T) {
		s := NewService(&fakeKeyRepo{}, &fakeProjRepo{exists: false})
		_, _, err := s.Create(context.Background(), "missing", "ci")
		ae, ok := apperrors.As(err)
		if !ok || ae.Kind != apperrors.KindNotFound {
			t.Fatalf("want not_found AppError, got %v", err)
		}
	})
}

func TestListUnknownProject(t *testing.T) {
	s := NewService(&fakeKeyRepo{}, &fakeProjRepo{exists: false})
	_, err := s.List(context.Background(), "missing")
	ae, ok := apperrors.As(err)
	if !ok || ae.Kind != apperrors.KindNotFound {
		t.Fatalf("want not_found AppError, got %v", err)
	}
}
```

- [ ] **Step 3: Run it to verify it fails**

Run: `cd apps/backend && go test ./internal/application/apikey/`
Expected: FAIL — `undefined: NewService`.

- [ ] **Step 4: Write the application service**

`apps/backend/internal/application/apikey/apikey.go`:
```go
// Package apikey holds the API Key use cases. Depends on the apikey + project
// domain ports + shared security/errors (no HTTP, no SQL).
package apikey

import (
	"context"
	"errors"

	apikeydomain "router-lens/internal/domain/apikey"
	projectdomain "router-lens/internal/domain/project"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
	"router-lens/internal/shared/security"
)

type Service struct {
	repo     apikeydomain.APIKeyRepository
	projects projectdomain.ProjectRepository
}

func NewService(repo apikeydomain.APIKeyRepository, projects projectdomain.ProjectRepository) *Service {
	return &Service{repo: repo, projects: projects}
}

// Create generates a key, returns the plaintext exactly once (never persisted),
// and stores only the hash. Verifies the project exists first.
func (s *Service) Create(ctx context.Context, projectID, name string) (plaintext string, key *apikeydomain.APIKey, err error) {
	if _, err = s.projects.FindByID(ctx, projectID); err != nil {
		if errors.Is(err, projectdomain.ErrNotFound) {
			return "", nil, apperrors.New(apperrors.KindNotFound, i18n.CodeProjectNotFound, "project not found")
		}
		return "", nil, err
	}
	plaintext, prefix, hash, err := security.GenerateAPIKey()
	if err != nil {
		return "", nil, err
	}
	k := &apikeydomain.APIKey{ProjectID: projectID, Name: name, KeyHash: hash, KeyPrefix: prefix}
	if err = s.repo.Create(ctx, k); err != nil {
		return "", nil, err
	}
	return plaintext, k, nil
}

func (s *Service) List(ctx context.Context, projectID string) ([]*apikeydomain.APIKey, error) {
	if _, err := s.projects.FindByID(ctx, projectID); err != nil {
		if errors.Is(err, projectdomain.ErrNotFound) {
			return nil, apperrors.New(apperrors.KindNotFound, i18n.CodeProjectNotFound, "project not found")
		}
		return nil, err
	}
	return s.repo.ListByProject(ctx, projectID)
}

func (s *Service) Revoke(ctx context.Context, id string) error {
	if err := s.repo.Revoke(ctx, id); err != nil {
		if errors.Is(err, apikeydomain.ErrNotFound) {
			return apperrors.New(apperrors.KindNotFound, i18n.CodeAPIKeyNotFound, "api key not found")
		}
		return err
	}
	return nil
}
```

- [ ] **Step 5: Add the `apikey.*` i18n code (so Step 4 compiles)**

In `apps/backend/internal/shared/i18n/i18n.go`, add to the const block + catalog:
```go
	// --- apikey ---
	CodeAPIKeyNotFound = "apikey.not_found"
```
```go
	// --- apikey ---
	CodeAPIKeyNotFound: {EN: "API key not found", ID: "Kunci API tidak ditemukan"},
```

- [ ] **Step 6: Run it to verify it passes**

Run: `cd apps/backend && go test ./internal/application/apikey/`
Expected: PASS.

- [ ] **Step 7: Write the api-key postgres repository**

`apps/backend/internal/infrastructure/postgres/apikey_repository.go`:
```go
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"router-lens/internal/domain/apikey"
)

type APIKeyRepository struct{ pool *pgxpool.Pool }

func NewAPIKeyRepository(pool *pgxpool.Pool) *APIKeyRepository { return &APIKeyRepository{pool: pool} }

var _ apikey.APIKeyRepository = (*APIKeyRepository)(nil)

func (r *APIKeyRepository) Create(ctx context.Context, k *apikey.APIKey) error {
	const q = `INSERT INTO api_keys (project_id, name, key_hash, key_prefix)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	return r.pool.QueryRow(ctx, q, k.ProjectID, k.Name, k.KeyHash, k.KeyPrefix).
		Scan(&k.ID, &k.CreatedAt)
}

func (r *APIKeyRepository) ListByProject(ctx context.Context, projectID string) ([]*apikey.APIKey, error) {
	const q = `SELECT id, project_id, name, key_prefix, last_used_at, revoked_at, created_at
		FROM api_keys WHERE project_id = $1 ORDER BY created_at DESC, id DESC`
	rows, err := r.pool.Query(ctx, q, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*apikey.APIKey, 0)
	for rows.Next() {
		var k apikey.APIKey
		if err := rows.Scan(&k.ID, &k.ProjectID, &k.Name, &k.KeyPrefix, &k.LastUsedAt, &k.RevokedAt, &k.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &k)
	}
	return out, rows.Err()
}

func (r *APIKeyRepository) Revoke(ctx context.Context, id string) error {
	const q = `UPDATE api_keys SET revoked_at = now() WHERE id = $1 RETURNING id`
	var got string
	err := r.pool.QueryRow(ctx, q, id).Scan(&got)
	if errors.Is(err, pgx.ErrNoRows) {
		return apikey.ErrNotFound
	}
	return err
}
```

- [ ] **Step 8: Write the api-key integration test**

`apps/backend/internal/infrastructure/postgres/apikey_repository_test.go` — reuse the same pool helper as the project test; seed a user + project, then exercise create/list/revoke:
```go
package postgres

import (
	"context"
	"testing"

	"router-lens/internal/domain/apikey"
	"router-lens/internal/domain/project"
)

func TestAPIKeyRepository(t *testing.T) {
	ctx, pool := setupTestDB(t)

	var ownerID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3) RETURNING id`,
		"key-owner@test.local", "x", "Owner").Scan(&ownerID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	proj := &project.Project{OwnerUserID: ownerID, Name: "Keys", Slug: "keys", Description: ""}
	if err := NewProjectRepository(pool).Create(ctx, proj); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	repo := NewAPIKeyRepository(pool)

	t.Run("create, list, revoke", func(t *testing.T) {
		k := &apikey.APIKey{ProjectID: proj.ID, Name: "ci", KeyHash: "hash-abc", KeyPrefix: "rl_live_ab"}
		if err := repo.Create(ctx, k); err != nil {
			t.Fatalf("create: %v", err)
		}
		list, err := repo.ListByProject(ctx, proj.ID)
		if err != nil || len(list) != 1 || list[0].RevokedAt != nil {
			t.Fatalf("list: %+v err=%v", list, err)
		}
		if err := repo.Revoke(ctx, k.ID); err != nil {
			t.Fatalf("revoke: %v", err)
		}
		list, _ = repo.ListByProject(ctx, proj.ID)
		if list[0].RevokedAt == nil {
			t.Fatal("revoked_at should be set")
		}
	})

	t.Run("revoke missing -> ErrNotFound", func(t *testing.T) {
		if err := repo.Revoke(ctx, "00000000-0000-0000-0000-000000000000"); err != apikey.ErrNotFound {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})
}
```

- [ ] **Step 9: Write the api-key HTTP handler**

`apps/backend/internal/infrastructure/http/handler/apikey_handler.go`:
```go
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	apikeyapp "router-lens/internal/application/apikey"
	apikeydomain "router-lens/internal/domain/apikey"
	"router-lens/internal/shared/response"
	"router-lens/internal/shared/validator"
)

type APIKeyHandler struct {
	svc *apikeyapp.Service
	v   *validator.Validator
}

func NewAPIKeyHandler(svc *apikeyapp.Service, v *validator.Validator) *APIKeyHandler {
	return &APIKeyHandler{svc: svc, v: v}
}

func (h *APIKeyHandler) Register(api *echo.Group, session echo.MiddlewareFunc) {
	api.POST("/projects/:projectId/api-keys", h.create, session)
	api.GET("/projects/:projectId/api-keys", h.list, session)
	api.DELETE("/api-keys/:id", h.revoke, session)
}

type apiKeyRequest struct {
	Name string `json:"name" validate:"required,max=120"`
}

type apiKeyDTO struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	KeyPrefix  string  `json:"key_prefix"`
	LastUsedAt *string `json:"last_used_at"`
	RevokedAt  *string `json:"revoked_at"`
	CreatedAt  string  `json:"created_at"`
}

// apiKeyCreatedDTO is returned ONCE on creation — it carries the plaintext key,
// which is never persisted and never returned again.
type apiKeyCreatedDTO struct {
	apiKeyDTO
	Key string `json:"key"`
}

func toAPIKeyDTO(k *apikeydomain.APIKey) apiKeyDTO {
	return apiKeyDTO{
		ID:         k.ID,
		Name:       k.Name,
		KeyPrefix:  k.KeyPrefix,
		LastUsedAt: formatNullableTime(k.LastUsedAt),
		RevokedAt:  formatNullableTime(k.RevokedAt),
		CreatedAt:  k.CreatedAt.UTC().Format(timeLayout),
	}
}

func (h *APIKeyHandler) create(c echo.Context) error {
	var req apiKeyRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	plaintext, k, err := h.svc.Create(c.Request().Context(), c.Param("projectId"), req.Name)
	if err != nil {
		return err
	}
	return response.Created(c, apiKeyCreatedDTO{apiKeyDTO: toAPIKeyDTO(k), Key: plaintext})
}

func (h *APIKeyHandler) list(c echo.Context) error {
	keys, err := h.svc.List(c.Request().Context(), c.Param("projectId"))
	if err != nil {
		return err
	}
	dtos := make([]apiKeyDTO, 0, len(keys))
	for _, k := range keys {
		dtos = append(dtos, toAPIKeyDTO(k))
	}
	return response.Data(c, http.StatusOK, dtos)
}

func (h *APIKeyHandler) revoke(c echo.Context) error {
	if err := h.svc.Revoke(c.Request().Context(), c.Param("id")); err != nil {
		return err
	}
	return response.NoContent(c)
}
```
NOTE: add the shared `formatNullableTime` helper to `apps/backend/internal/infrastructure/http/handler/handler.go` (the file created in Task 1 Step 17 for `timeLayout`):
```go
// formatNullableTime renders a *time.Time as a UTC RFC3339 *string, or nil.
func formatNullableTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(timeLayout)
	return &s
}
```

- [ ] **Step 10: Wire API Keys in Fx**

In `apps/backend/cmd/server/main.go` `fx.Provide(...)` add:
```go
		fx.Annotate(postgres.NewAPIKeyRepository, fx.As(new(apikey.APIKeyRepository))),
		apikeyapp.NewService,      // (apikey.APIKeyRepository, project.ProjectRepository) -> *apikeyapp.Service
		handler.NewAPIKeyHandler,  // (*apikeyapp.Service, *validator.Validator) -> *APIKeyHandler
```
imports:
```go
	apikey "router-lens/internal/domain/apikey"
	apikeyapp "router-lens/internal/application/apikey"
```
add the registrar + invoke:
```go
func registerAPIKeyRoutes(e *echo.Echo, h *handler.APIKeyHandler, session echo.MiddlewareFunc) {
	h.Register(e.Group("/api/v1"), session)
}
```
```go
		fx.Invoke(registerAPIKeyRoutes),
```

- [ ] **Step 11: Verify + Commit**

```
cd apps/backend && gofmt -l . && go vet ./... && go build ./... && go test ./internal/... -count=1
```
Expected: clean, all PASS (integration skips without DB).
```bash
git add apps/backend
git commit -m "feat: api-keys CRUD (Plan 04 task 2)"
```

---

## Task 3: Pricing CRUD

**Files:**
- Create: `apps/backend/internal/domain/pricing/entity.go`
- Create: `apps/backend/internal/domain/pricing/repository.go`
- Create: `apps/backend/internal/application/pricing/pricing.go`
- Create: `apps/backend/internal/application/pricing/pricing_test.go`
- Modify: `apps/backend/internal/infrastructure/postgres/db.go` (register decimal codec)
- Create: `apps/backend/internal/infrastructure/postgres/pricing_repository.go`
- Create: `apps/backend/internal/infrastructure/postgres/pricing_repository_test.go`
- Create: `apps/backend/internal/infrastructure/http/handler/pricing_handler.go`
- Modify: `apps/backend/cmd/server/main.go` (wire pricing)
- Modify: `apps/backend/internal/shared/i18n/i18n.go` (add `pricing.*` codes)
- Modify: `apps/backend/go.mod` / `go.sum` (add `github.com/jackc/pgx-shopspring-decimal`)

**Interfaces:**
- Consumes: `decimal.Decimal`, `apperrors`, `i18n`, `response`, `pricingdomain.Rule` (existing calculator VO).
- Produces: `pricingdomain.{PricingRule, PricingRepository}`, `pricing.NewService`, `handler.NewPricingHandler`, `postgres.NewPricingRepository`.

- [ ] **Step 1: Add the decimal-codec dependency**

Run (additive; do NOT run `go mod tidy` mid-scaffold — it prunes not-yet-imported deps):
```
cd apps/backend && go get github.com/jackc/pgx-shopspring-decimal@latest
```
Expected: `go.mod` gains the require line; `go.sum` updated.

- [ ] **Step 2: Register the decimal codec on the pool**

Modify `apps/backend/internal/infrastructure/postgres/db.go` — add the import and an `AfterConnect` hook so pgx scans `numeric` into `decimal.Decimal` (and encodes it back) for every pooled connection:
```go
import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	pgxdecimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5/pgxpool"
)
```
After `cfg.MaxConnIdleTime = 30 * time.Minute` and before `pgxpool.NewWithConfig`:
```go
	// Register the shopspring/decimal codec so numeric columns scan into
	// decimal.Decimal end to end (pricing now, event cost in Plan 05).
	cfg.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
		pgxdecimal.Register(conn.TypeMap())
		return nil
	}
```

- [ ] **Step 3: Write the pricing domain entity + repository port**

`apps/backend/internal/domain/pricing/entity.go` (the package already holds `calculator.go` with `Rule`/`Cost`/`TokenUsage`):
```go
package pricing

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

// ErrNotFound is returned when a pricing_rules row is absent.
var ErrNotFound = errors.New("pricing: not found")

// ErrConflict is returned when an update would collide with another row's
// (provider, model) unique pair.
var ErrConflict = errors.New("pricing: provider/model already exists")

// PricingRule is the full persisted rule. Its prices feed the cost calculator
// via Rule(); the calculator's value object stays the minimal Rule type.
type PricingRule struct {
	ID               string
	Provider         string
	Model            string
	InputPricePer1M  decimal.Decimal
	OutputPricePer1M decimal.Decimal
	Currency         string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Rule returns the value object the cost calculator consumes.
func (p *PricingRule) Rule() Rule {
	return Rule{InputPricePer1M: p.InputPricePer1M, OutputPricePer1M: p.OutputPricePer1M}
}
```

`apps/backend/internal/domain/pricing/repository.go`:
```go
package pricing

import "context"

// PricingRepository is the port for persisting and querying pricing rules.
type PricingRepository interface {
	List(ctx context.Context) ([]*PricingRule, error)
	FindByID(ctx context.Context, id string) (*PricingRule, error)
	// Upsert inserts r, or updates prices + currency on a (provider, model)
	// conflict. Sets ID/CreatedAt/UpdatedAt.
	Upsert(ctx context.Context, r *PricingRule) error
	// Update changes provider/model/prices/currency by id. Returns ErrNotFound
	// when the id is absent, ErrConflict when (provider, model) collides with
	// a different row.
	Update(ctx context.Context, r *PricingRule) error
	Delete(ctx context.Context, id string) error
}
```

- [ ] **Step 4: Write the failing application-service test**

`apps/backend/internal/application/pricing/pricing_test.go`:
```go
package pricing

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	pricingdomain "router-lens/internal/domain/pricing"
	apperrors "router-lens/internal/shared/errors"
)

type fakeRepo struct {
	upsertErr error
	updateErr error
	got       *pricingdomain.PricingRule
}

func (f *fakeRepo) List(context.Context) ([]*pricingdomain.PricingRule, error) { return nil, nil }
func (f *fakeRepo) FindByID(context.Context, string) (*pricingdomain.PricingRule, error) {
	return nil, nil
}
func (f *fakeRepo) Upsert(_ context.Context, r *pricingdomain.PricingRule) error {
	f.got = r
	if f.upsertErr != nil {
		return f.upsertErr
	}
	r.ID = "pr1"
	return nil
}
func (f *fakeRepo) Update(_ context.Context, r *pricingdomain.PricingRule) error { return f.updateErr }
func (f *fakeRepo) Delete(context.Context, string) error                         { return nil }

func TestUpsert(t *testing.T) {
	t.Run("defaults currency to USD", func(t *testing.T) {
		f := &fakeRepo{}
		_, err := NewService(f).Upsert(context.Background(), Input{
			Provider: "anthropic", Model: "claude", Input: decimal.NewFromInt(3), Output: decimal.NewFromInt(15),
		})
		if err != nil || f.got.Currency != "USD" {
			t.Fatalf("currency=%q err=%v", f.got.Currency, err)
		}
	})
	t.Run("rejects negative price as validation error", func(t *testing.T) {
		_, err := NewService(&fakeRepo{}).Upsert(context.Background(), Input{
			Provider: "p", Model: "m", Input: decimal.NewFromInt(-1), Output: decimal.NewFromInt(1),
		})
		ae, ok := apperrors.As(err)
		if !ok || ae.Kind != apperrors.KindValidation {
			t.Fatalf("want validation AppError, got %v", err)
		}
	})
	t.Run("maps update conflict to 409", func(t *testing.T) {
		f := &fakeRepo{updateErr: pricingdomain.ErrConflict}
		err := NewService(f).Update(context.Background(), "pr1", Input{
			Provider: "p", Model: "m", Input: decimal.NewFromInt(1), Output: decimal.NewFromInt(1),
		})
		ae, ok := apperrors.As(err)
		if !ok || ae.Kind != apperrors.KindConflict {
			t.Fatalf("want conflict AppError, got %v", err)
		}
	})
}
```

- [ ] **Step 5: Run it to verify it fails**

Run: `cd apps/backend && go test ./internal/application/pricing/`
Expected: FAIL — `undefined: NewService` / `undefined: Input`.

- [ ] **Step 6: Write the application service**

`apps/backend/internal/application/pricing/pricing.go`:
```go
// Package pricing holds the Pricing CRUD use cases. Depends on the pricing
// domain port + shared/errors (no HTTP, no SQL).
package pricing

import (
	"context"
	"errors"

	pricingdomain "router-lens/internal/domain/pricing"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"

	"github.com/shopspring/decimal"
)

const defaultCurrency = "USD"

// Input is the validated command for creating/updating a rule (a params object,
// keeping service methods under S107).
type Input struct {
	Provider string
	Model    string
	Input    decimal.Decimal
	Output   decimal.Decimal
	Currency string
}

type Service struct{ repo pricingdomain.PricingRepository }

func NewService(repo pricingdomain.PricingRepository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context) ([]*pricingdomain.PricingRule, error) {
	return s.repo.List(ctx)
}

func (s *Service) Upsert(ctx context.Context, in Input) (*pricingdomain.PricingRule, error) {
	r, err := s.build("", in)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Upsert(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Service) Update(ctx context.Context, id string, in Input) error {
	r, err := s.build(id, in)
	if err != nil {
		return err
	}
	if err := s.repo.Update(ctx, r); err != nil {
		return s.mapErr(err)
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return s.mapErr(err)
	}
	return nil
}

// build validates the input and assembles a PricingRule. Negative prices are a
// validation error; an empty currency defaults to USD.
func (s *Service) build(id string, in Input) (*pricingdomain.PricingRule, error) {
	if in.Input.IsNegative() || in.Output.IsNegative() {
		return nil, apperrors.New(apperrors.KindValidation, i18n.CodePricingInvalidPrice, "price must not be negative")
	}
	currency := in.Currency
	if currency == "" {
		currency = defaultCurrency
	}
	return &pricingdomain.PricingRule{
		ID:               id,
		Provider:         in.Provider,
		Model:            in.Model,
		InputPricePer1M:  in.Input,
		OutputPricePer1M: in.Output,
		Currency:         currency,
	}, nil
}

func (s *Service) mapErr(err error) error {
	switch {
	case errors.Is(err, pricingdomain.ErrNotFound):
		return apperrors.New(apperrors.KindNotFound, i18n.CodePricingNotFound, "pricing rule not found")
	case errors.Is(err, pricingdomain.ErrConflict):
		return apperrors.New(apperrors.KindConflict, i18n.CodePricingDuplicate, "a pricing rule for this provider/model already exists")
	default:
		return err
	}
}
```

- [ ] **Step 7: Add the `pricing.*` i18n codes (so Step 6 compiles)**

In `apps/backend/internal/shared/i18n/i18n.go`, add to the const block + catalog:
```go
	// --- pricing ---
	CodePricingNotFound     = "pricing.not_found"
	CodePricingDuplicate    = "pricing.duplicate"
	CodePricingInvalidPrice = "pricing.invalid_price"
```
```go
	// --- pricing ---
	CodePricingNotFound:     {EN: "Pricing rule not found", ID: "Aturan harga tidak ditemukan"},
	CodePricingDuplicate:    {EN: "A pricing rule for this provider/model already exists", ID: "Aturan harga untuk provider/model ini sudah ada"},
	CodePricingInvalidPrice: {EN: "Price must not be negative", ID: "Harga tidak boleh negatif"},
```

- [ ] **Step 8: Run it to verify it passes**

Run: `cd apps/backend && go test ./internal/application/pricing/`
Expected: PASS.

- [ ] **Step 9: Write the pricing postgres repository**

`apps/backend/internal/infrastructure/postgres/pricing_repository.go`:
```go
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"router-lens/internal/domain/pricing"
)

type PricingRepository struct{ pool *pgxpool.Pool }

func NewPricingRepository(pool *pgxpool.Pool) *PricingRepository { return &PricingRepository{pool: pool} }

var _ pricing.PricingRepository = (*PricingRepository)(nil)

const pricingColumns = `id, provider, model, input_price_per_1m, output_price_per_1m, currency, created_at, updated_at`

func scanRule(row pgx.Row, r *pricing.PricingRule) error {
	return row.Scan(&r.ID, &r.Provider, &r.Model, &r.InputPricePer1M, &r.OutputPricePer1M, &r.Currency, &r.CreatedAt, &r.UpdatedAt)
}

func (r *PricingRepository) List(ctx context.Context) ([]*pricing.PricingRule, error) {
	q := `SELECT ` + pricingColumns + ` FROM pricing_rules ORDER BY provider, model`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*pricing.PricingRule, 0)
	for rows.Next() {
		var rule pricing.PricingRule
		if err := scanRule(rows, &rule); err != nil {
			return nil, err
		}
		out = append(out, &rule)
	}
	return out, rows.Err()
}

func (r *PricingRepository) FindByID(ctx context.Context, id string) (*pricing.PricingRule, error) {
	q := `SELECT ` + pricingColumns + ` FROM pricing_rules WHERE id = $1`
	var rule pricing.PricingRule
	err := scanRule(r.pool.QueryRow(ctx, q, id), &rule)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, pricing.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *PricingRepository) Upsert(ctx context.Context, rule *pricing.PricingRule) error {
	const q = `INSERT INTO pricing_rules (provider, model, input_price_per_1m, output_price_per_1m, currency)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (provider, model) DO UPDATE SET
			input_price_per_1m = EXCLUDED.input_price_per_1m,
			output_price_per_1m = EXCLUDED.output_price_per_1m,
			currency = EXCLUDED.currency,
			updated_at = now()
		RETURNING id, created_at, updated_at`
	return r.pool.QueryRow(ctx, q, rule.Provider, rule.Model, rule.InputPricePer1M, rule.OutputPricePer1M, rule.Currency).
		Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
}

func (r *PricingRepository) Update(ctx context.Context, rule *pricing.PricingRule) error {
	const q = `UPDATE pricing_rules SET provider = $2, model = $3, input_price_per_1m = $4,
			output_price_per_1m = $5, currency = $6, updated_at = now()
		WHERE id = $1
		RETURNING created_at, updated_at`
	err := r.pool.QueryRow(ctx, q, rule.ID, rule.Provider, rule.Model, rule.InputPricePer1M, rule.OutputPricePer1M, rule.Currency).
		Scan(&rule.CreatedAt, &rule.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return pricing.ErrNotFound
	}
	if isUniqueViolation(err) {
		return pricing.ErrConflict
	}
	return err
}

func (r *PricingRepository) Delete(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM pricing_rules WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pricing.ErrNotFound
	}
	return nil
}
```

- [ ] **Step 10: Write the pricing integration test**

`apps/backend/internal/infrastructure/postgres/pricing_repository_test.go` — reuse the shared pool helper; pricing has no FK so no seeding needed:
```go
package postgres

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"router-lens/internal/domain/pricing"
)

func TestPricingRepository(t *testing.T) {
	ctx, pool := setupTestDB(t)
	repo := NewPricingRepository(pool)

	t.Run("upsert inserts then updates same pair", func(t *testing.T) {
		rule := &pricing.PricingRule{
			Provider: "anthropic", Model: "claude-test",
			InputPricePer1M: decimal.RequireFromString("3.00"), OutputPricePer1M: decimal.RequireFromString("15.00"),
			Currency: "USD",
		}
		if err := repo.Upsert(ctx, rule); err != nil {
			t.Fatalf("insert: %v", err)
		}
		firstID := rule.ID
		rule.InputPricePer1M = decimal.RequireFromString("4.50")
		if err := repo.Upsert(ctx, rule); err != nil {
			t.Fatalf("update upsert: %v", err)
		}
		got, _ := repo.FindByID(ctx, firstID)
		if !got.InputPricePer1M.Equal(decimal.RequireFromString("4.50")) {
			t.Fatalf("price not updated: %s", got.InputPricePer1M)
		}
	})

	t.Run("delete missing -> ErrNotFound", func(t *testing.T) {
		if err := repo.Delete(ctx, "00000000-0000-0000-0000-000000000000"); err != pricing.ErrNotFound {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})
}
```

- [ ] **Step 11: Write the pricing HTTP handler**

`apps/backend/internal/infrastructure/http/handler/pricing_handler.go`:
```go
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"

	pricingapp "router-lens/internal/application/pricing"
	pricingdomain "router-lens/internal/domain/pricing"
	"router-lens/internal/shared/response"
	"router-lens/internal/shared/validator"
)

type PricingHandler struct {
	svc *pricingapp.Service
	v   *validator.Validator
}

func NewPricingHandler(svc *pricingapp.Service, v *validator.Validator) *PricingHandler {
	return &PricingHandler{svc: svc, v: v}
}

func (h *PricingHandler) Register(api *echo.Group, session echo.MiddlewareFunc) {
	api.GET("/pricing", h.list, session)
	api.POST("/pricing", h.upsert, session)
	api.PUT("/pricing/:id", h.update, session)
	api.DELETE("/pricing/:id", h.delete, session)
}

type pricingRequest struct {
	Provider      string           `json:"provider" validate:"required,max=100"`
	Model         string           `json:"model" validate:"required,max=200"`
	InputPrice1M  *decimal.Decimal `json:"input_price_per_1m" validate:"required"`
	OutputPrice1M *decimal.Decimal `json:"output_price_per_1m" validate:"required"`
	Currency      string           `json:"currency" validate:"max=8"`
}

// toInput dereferences the price pointers — safe because `validate:"required"`
// rejects an omitted/null price (nil) at the boundary BEFORE this runs. A price
// explicitly set to 0 is allowed (free models); a missing price is a 400. This
// is the distinction Codex (GPT-5.5) flagged: an omitted price must not silently
// become a $0 rule.
func (r pricingRequest) toInput() pricingapp.Input {
	return pricingapp.Input{
		Provider: r.Provider, Model: r.Model,
		Input: *r.InputPrice1M, Output: *r.OutputPrice1M, Currency: r.Currency,
	}
}

type pricingDTO struct {
	ID            string `json:"id"`
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	InputPrice1M  string `json:"input_price_per_1m"`
	OutputPrice1M string `json:"output_price_per_1m"`
	Currency      string `json:"currency"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

func toPricingDTO(p *pricingdomain.PricingRule) pricingDTO {
	return pricingDTO{
		ID:            p.ID,
		Provider:      p.Provider,
		Model:         p.Model,
		InputPrice1M:  p.InputPricePer1M.String(),
		OutputPrice1M: p.OutputPricePer1M.String(),
		Currency:      p.Currency,
		CreatedAt:     p.CreatedAt.UTC().Format(timeLayout),
		UpdatedAt:     p.UpdatedAt.UTC().Format(timeLayout),
	}
}

func (h *PricingHandler) list(c echo.Context) error {
	rules, err := h.svc.List(c.Request().Context())
	if err != nil {
		return err
	}
	dtos := make([]pricingDTO, 0, len(rules))
	for _, p := range rules {
		dtos = append(dtos, toPricingDTO(p))
	}
	return response.Data(c, http.StatusOK, dtos)
}

func (h *PricingHandler) upsert(c echo.Context) error {
	var req pricingRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	p, err := h.svc.Upsert(c.Request().Context(), req.toInput())
	if err != nil {
		return err
	}
	return response.Created(c, toPricingDTO(p))
}

func (h *PricingHandler) update(c echo.Context) error {
	var req pricingRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	if err := h.svc.Update(c.Request().Context(), c.Param("id"), req.toInput()); err != nil {
		return err
	}
	return response.NoContent(c)
}

func (h *PricingHandler) delete(c echo.Context) error {
	if err := h.svc.Delete(c.Request().Context(), c.Param("id")); err != nil {
		return err
	}
	return response.NoContent(c)
}
```

- [ ] **Step 12: Wire Pricing in Fx**

In `apps/backend/cmd/server/main.go` `fx.Provide(...)` add:
```go
		fx.Annotate(postgres.NewPricingRepository, fx.As(new(pricing.PricingRepository))),
		pricingapp.NewService,      // (pricing.PricingRepository) -> *pricingapp.Service
		handler.NewPricingHandler,  // (*pricingapp.Service, *validator.Validator) -> *PricingHandler
```
imports:
```go
	pricing "router-lens/internal/domain/pricing"
	pricingapp "router-lens/internal/application/pricing"
```
add registrar + invoke:
```go
func registerPricingRoutes(e *echo.Echo, h *handler.PricingHandler, session echo.MiddlewareFunc) {
	h.Register(e.Group("/api/v1"), session)
}
```
```go
		fx.Invoke(registerPricingRoutes),
```

- [ ] **Step 13: Full verification**

Run:
```
cd apps/backend && gofmt -l . && go vet ./... && go build ./... && go test ./internal/... -count=1
```
Expected: gofmt prints nothing; vet/build clean; all non-integration tests PASS. If a Postgres is available, run the three repo integration tests against an isolated container on `:55432`.

- [ ] **Step 14: Smoke-test the wired app (optional, if Docker is up)**

If `docker compose up` is running, confirm the Fx graph builds and routes respond:
```
curl -s -X POST localhost:8080/api/v1/pricing -H 'Content-Type: application/json' \
  -d '{"provider":"anthropic","model":"claude-sonnet-4-5","input_price_per_1m":"3","output_price_per_1m":"15"}'
```
Expected: `401 unauthorized` without a session cookie (proves the route is mounted behind session auth), or `201` when called with a valid cookie. A `404 route not found` means the route was not wired — fix before commit.

- [ ] **Step 15: Commit**

```bash
git add apps/backend
git commit -m "feat: pricing CRUD + decimal codec (Plan 04 task 3)"
```

---

## Plan-level Definition of Done

- Projects: create (auto-slug, owner-stamped), list (offset-paginated envelope), get, update (name/description; slug immutable), delete — all behind session auth.
- API Keys: create under a project (plaintext returned exactly once, only hash stored), list (prefix only, no plaintext), revoke (soft delete). Unknown project → 404.
- Pricing: list, upsert by `(provider, model)`, update by id (409 on pair collision), delete. Negative price → 400 validation. Currency defaults to USD. `numeric` ↔ `decimal.Decimal` round-trips via the registered codec.
- One session middleware provided via Fx and shared by all registrars (no duplicated construction). One `isUniqueViolation` helper, one `response.Paginated` helper — no duplicated logic.
- All new i18n codes have EN + ID catalog entries. `gofmt`/`go vet`/`go build` clean; unit suites green; repo integration tests green against a real Postgres.
- Three commits (one per task) on `feat/foundation`.
