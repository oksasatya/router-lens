# RouterLens — Plan 03: Auth & First-Run Setup

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A user can complete first-run setup (create the one admin, race-safe), log in over an httpOnly session cookie, read `GET /auth/me`, and log out — all localized (EN/ID) and validated.

**Architecture:** Builds on Plan 01 (server, response/i18n, config, migrations) + Plan 02 (security, validator). Adds the `user` domain (User + Session + repo interfaces), Postgres repositories, auth use cases, a session middleware, and the setup/auth HTTP handlers, wired via Uber Fx providers + a route-registration invoke in `cmd/server/main.go`.

**Tech Stack:** Go 1.26, Echo v4, pgx/v5, the Plan 02 `security`/`validator` packages, `shared/response`/`errors`/`i18n`.

## Global Constraints

Inherits Plan 01 + 02 Global Constraints (module `router-lens`, hexagonal layering, Sonar block, golang-expert hub, verification routine). Plan-specific:

- **Auth model (per CLAUDE.md decisions 2, 5, 13):** DB-backed session; the cookie holds an opaque random token, the server stores only `sha256(token)`. `HttpOnly` always, `Secure` when `cfg.IsProduction()`, `SameSite=Lax` (or `None` when `cfg.CookieCrossSite`). **No CSRF token** (SameSite=Lax + no state change on GET). Ingestion stays Bearer API key (Plan 05) — not touched here.
- **First-run setup is race-safe:** the admin is created via a conditional insert (`INSERT … SELECT … WHERE NOT EXISTS (SELECT 1 FROM users)`) + the `users.email` unique constraint. Locked (`403 auth.setup_locked`) once any user exists.
- **Login leaks nothing:** unknown email and wrong password both return the same `401 auth.invalid_credentials`.
- **i18n:** new namespaced error codes (`auth.invalid_credentials`, `auth.setup_locked`) + their `i18n` consts are added to the catalog (EN + ID), per the CLAUDE.md "Error codes" convention. Validation errors are already localized by the Plan 02 validator.
- **TDD verdicts:** domain `Session.IsExpired` + use cases (Setup/Login/Logout) + session middleware = **YES** (in-memory fake repos). Postgres repositories = **integration tests** gated on `TEST_DATABASE_URL` (skip when unset). Handlers/wiring = **NO** (verify via the Task 7 e2e curl flow).

---

### Task 1: `user` domain — entities, sentinel, repository interfaces

**TDD:** yes for `Session.IsExpired` (pure logic); the rest is structs/interfaces.

**Files:**
- Create: `internal/domain/user/entity.go`
- Create: `internal/domain/user/repository.go`
- Test: `internal/domain/user/entity_test.go`

**Interfaces:**
- Produces:
  ```go
  var ErrNotFound = errors.New("user: not found")

  type User struct { ID, Email, PasswordHash, Name string; CreatedAt, UpdatedAt time.Time }
  type Session struct { ID, UserID, TokenHash string; ExpiresAt, CreatedAt time.Time; UserAgent, IP string }
  func (s Session) IsExpired(now time.Time) bool

  type UserRepository interface {
      CreateInitialAdmin(ctx context.Context, u *User) (created bool, err error) // false if any user exists
      FindByEmail(ctx context.Context, email string) (*User, error)              // ErrNotFound if absent
      FindByID(ctx context.Context, id string) (*User, error)                    // ErrNotFound if absent
      AnyExists(ctx context.Context) (bool, error)
  }
  type SessionRepository interface {
      Create(ctx context.Context, s *Session) error
      FindByTokenHash(ctx context.Context, tokenHash string) (*Session, error)   // ErrNotFound if absent
      DeleteByTokenHash(ctx context.Context, tokenHash string) error
  }
  ```

- [ ] **Step 1: Write the failing test**

```go
package user

import (
	"testing"
	"time"
)

func TestSessionIsExpired(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	if (Session{ExpiresAt: now.Add(time.Hour)}).IsExpired(now) {
		t.Fatal("future expiry should not be expired")
	}
	if !(Session{ExpiresAt: now.Add(-time.Hour)}).IsExpired(now) {
		t.Fatal("past expiry should be expired")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/user/ -v`
Expected: FAIL — undefined.

- [ ] **Step 3: Write `entity.go`**

```go
// Package user is the auth bounded context: the User identity, the Session that
// authenticates a dashboard request, and their repository ports. Imports only
// stdlib (domain purity).
package user

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned by repositories when a row is absent.
var ErrNotFound = errors.New("user: not found")

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Session struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
	UserAgent string
	IP        string
}

// IsExpired reports whether the session is no longer valid at now.
func (s Session) IsExpired(now time.Time) bool { return !now.Before(s.ExpiresAt) }

// (interfaces below live in repository.go)
var _ = context.Background
```

- [ ] **Step 4: Write `repository.go`**

```go
package user

import "context"

type UserRepository interface {
	CreateInitialAdmin(ctx context.Context, u *User) (created bool, err error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	AnyExists(ctx context.Context) (bool, error)
}

type SessionRepository interface {
	Create(ctx context.Context, s *Session) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
}
```

(Then remove the temporary `var _ = context.Background` line from `entity.go` — it was only a placeholder to keep the file compiling before `repository.go` existed. Re-run `go build ./internal/domain/user/`.)

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/domain/user/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/user/
git commit -m "feat: user domain (User, Session, repository ports)"
```

---

### Task 2: Postgres user + session repositories

**TDD:** no — integration. Tests connect to `TEST_DATABASE_URL` and skip when unset.

**Files:**
- Create: `internal/infrastructure/postgres/user_repository.go`
- Create: `internal/infrastructure/postgres/session_repository.go`
- Test: `internal/infrastructure/postgres/auth_repository_test.go`

**Interfaces:**
- Consumes: `domain/user` (interfaces, `ErrNotFound`), `*pgxpool.Pool`.
- Produces: `func NewUserRepository(pool *pgxpool.Pool) *UserRepository` and `func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository`, each implementing the matching `user.*Repository` interface.

- [ ] **Step 1: Write `user_repository.go`**

```go
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"router-lens/internal/domain/user"
)

type UserRepository struct{ pool *pgxpool.Pool }

func NewUserRepository(pool *pgxpool.Pool) *UserRepository { return &UserRepository{pool: pool} }

var _ user.UserRepository = (*UserRepository)(nil)

// CreateInitialAdmin inserts the admin only when no user exists yet (race-safe).
func (r *UserRepository) CreateInitialAdmin(ctx context.Context, u *user.User) (bool, error) {
	const q = `
		INSERT INTO users (email, password_hash, name)
		SELECT $1, $2, $3
		WHERE NOT EXISTS (SELECT 1 FROM users)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, q, u.Email, u.PasswordHash, u.Name).
		Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil // a user already exists
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	return r.scanOne(ctx, `SELECT id, email, password_hash, name, created_at, updated_at
		FROM users WHERE email = $1`, email)
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*user.User, error) {
	return r.scanOne(ctx, `SELECT id, email, password_hash, name, created_at, updated_at
		FROM users WHERE id = $1`, id)
}

func (r *UserRepository) AnyExists(ctx context.Context) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users)`).Scan(&exists)
	return exists, err
}

func (r *UserRepository) scanOne(ctx context.Context, q string, arg any) (*user.User, error) {
	var u user.User
	err := r.pool.QueryRow(ctx, q, arg).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
```

- [ ] **Step 2: Write `session_repository.go`**

```go
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"router-lens/internal/domain/user"
)

type SessionRepository struct{ pool *pgxpool.Pool }

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

var _ user.SessionRepository = (*SessionRepository)(nil)

func (r *SessionRepository) Create(ctx context.Context, s *user.Session) error {
	const q = `
		INSERT INTO sessions (user_id, token_hash, expires_at, user_agent, ip)
		VALUES ($1, $2, $3, $4, NULLIF($5, '')::inet)
		RETURNING id, created_at`
	return r.pool.QueryRow(ctx, q, s.UserID, s.TokenHash, s.ExpiresAt, s.UserAgent, s.IP).
		Scan(&s.ID, &s.CreatedAt)
}

func (r *SessionRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*user.Session, error) {
	var s user.Session
	const q = `SELECT id, user_id, token_hash, expires_at, created_at
		FROM sessions WHERE token_hash = $1`
	err := r.pool.QueryRow(ctx, q, tokenHash).
		Scan(&s.ID, &s.UserID, &s.TokenHash, &s.ExpiresAt, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}
```

- [ ] **Step 3: Write the integration test**

```go
package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"router-lens/internal/domain/user"
)

func testPool(t *testing.T) context.Context {
	t.Helper()
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	return context.Background()
}

func TestAuthRepositories(t *testing.T) {
	ctx := testPool(t)
	pool, err := NewPool(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	_, _ = pool.Exec(ctx, "TRUNCATE users, sessions CASCADE")

	users := NewUserRepository(pool)
	sessions := NewSessionRepository(pool)

	admin := &user.User{Email: "admin@example.com", PasswordHash: "hash", Name: "Admin"}
	created, err := users.CreateInitialAdmin(ctx, admin)
	if err != nil || !created || admin.ID == "" {
		t.Fatalf("first admin should be created: created=%v err=%v", created, err)
	}
	again, _ := users.CreateInitialAdmin(ctx, &user.User{Email: "x@y.com", PasswordHash: "h"})
	if again {
		t.Fatal("second CreateInitialAdmin must be locked (false)")
	}

	got, err := users.FindByEmail(ctx, "admin@example.com")
	if err != nil || got.ID != admin.ID {
		t.Fatalf("FindByEmail: %v %+v", err, got)
	}
	if _, err := users.FindByEmail(ctx, "nobody@x.com"); err != user.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	s := &user.Session{UserID: admin.ID, TokenHash: "abc", ExpiresAt: time.Now().Add(time.Hour)}
	if err := sessions.Create(ctx, s); err != nil {
		t.Fatalf("session create: %v", err)
	}
	if found, err := sessions.FindByTokenHash(ctx, "abc"); err != nil || found.UserID != admin.ID {
		t.Fatalf("session find: %v %+v", err, found)
	}
	if err := sessions.DeleteByTokenHash(ctx, "abc"); err != nil {
		t.Fatalf("session delete: %v", err)
	}
	if _, err := sessions.FindByTokenHash(ctx, "abc"); err != user.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}
```

- [ ] **Step 4: Run the integration test against the compose Postgres**

Run: `docker compose up -d postgres && TEST_DATABASE_URL="postgres://routerlens:routerlens@localhost:5432/routerlens?sslmode=disable" go test ./internal/infrastructure/postgres/ -run TestAuthRepositories -v`
Expected: PASS (or SKIP if you choose not to run a DB).

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/postgres/user_repository.go internal/infrastructure/postgres/session_repository.go internal/infrastructure/postgres/auth_repository_test.go
git commit -m "feat: postgres user and session repositories (race-safe admin insert)"
```

---

### Task 3: Auth use cases (Setup, Login, Logout) + i18n codes

**TDD:** yes — branching logic (setup-locked, invalid-credentials). In-memory fake repos.

**Files:**
- Create: `internal/application/auth/auth.go` (the three use cases + a small Deps)
- Test: `internal/application/auth/auth_test.go`
- Modify: `internal/shared/i18n/i18n.go` (add `invalid_credentials`, `setup_locked` to the catalog)

**Interfaces:**
- Consumes: `domain/user`, `shared/security`, `shared/errors`.
- Produces:
  ```go
  const SessionTTL = 7 * 24 * time.Hour
  type Service struct { /* users, sessions repos */ }
  func NewService(users user.UserRepository, sessions user.SessionRepository) *Service
  func (s *Service) Setup(ctx, email, password, name string) error                 // 403 setup_locked if a user exists
  func (s *Service) Login(ctx, email, password, userAgent, ip string) (plaintextToken string, err error) // 401 invalid_credentials
  func (s *Service) Logout(ctx, tokenHash string) error
  func (s *Service) NeedsSetup(ctx) (bool, error)
  ```

- [ ] **Step 1: Add the new i18n codes**

Edit `internal/shared/i18n/i18n.go` — add the namespaced `auth.*` code constants and a catalog section (per the CLAUDE.md "Error codes" convention: domain codes are namespaced + declared as consts):

```go
// add to the const block:
const (
	CodeAuthInvalidCredentials = "auth.invalid_credentials"
	CodeAuthSetupLocked        = "auth.setup_locked"
)

// add a section to the catalog map:
	// --- auth ---
	CodeAuthInvalidCredentials: {EN: "Invalid email or password", ID: "Email atau kata sandi salah"},
	CodeAuthSetupLocked:        {EN: "Setup is already completed", ID: "Setup sudah pernah dilakukan"},
```

- [ ] **Step 2: Write the failing test**

```go
package auth

import (
	"context"
	"testing"

	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
	"router-lens/internal/shared/security"
	"router-lens/internal/domain/user"
)

// fakeUsers / fakeSessions are minimal in-memory repos.
type fakeUsers struct {
	byEmail map[string]*user.User
	created bool
}

func (f *fakeUsers) CreateInitialAdmin(_ context.Context, u *user.User) (bool, error) {
	if f.created {
		return false, nil
	}
	f.created = true
	u.ID = "u1"
	if f.byEmail == nil {
		f.byEmail = map[string]*user.User{}
	}
	f.byEmail[u.Email] = u
	return true, nil
}
func (f *fakeUsers) FindByEmail(_ context.Context, e string) (*user.User, error) {
	if u, ok := f.byEmail[e]; ok {
		return u, nil
	}
	return nil, user.ErrNotFound
}
func (f *fakeUsers) FindByID(context.Context, string) (*user.User, error) { return nil, user.ErrNotFound }
func (f *fakeUsers) AnyExists(context.Context) (bool, error)              { return f.created, nil }

type fakeSessions struct{ saved *user.Session }

func (f *fakeSessions) Create(_ context.Context, s *user.Session) error { f.saved = s; return nil }
func (f *fakeSessions) FindByTokenHash(context.Context, string) (*user.Session, error) {
	return nil, user.ErrNotFound
}
func (f *fakeSessions) DeleteByTokenHash(context.Context, string) error { return nil }

func TestAuthService(t *testing.T) {
	ctx := context.Background()

	t.Run("setup creates then locks", func(t *testing.T) {
		svc := NewService(&fakeUsers{}, &fakeSessions{})
		if err := svc.Setup(ctx, "a@b.com", "password123", "Admin"); err != nil {
			t.Fatalf("first setup: %v", err)
		}
		err := svc.Setup(ctx, "c@d.com", "password123", "Two")
		ae, ok := apperrors.As(err)
		if !ok || ae.Code != i18n.CodeAuthSetupLocked {
			t.Fatalf("second setup should be locked, got %v", err)
		}
	})

	t.Run("login succeeds and stores session", func(t *testing.T) {
		hash, _ := security.HashPassword("password123")
		fu := &fakeUsers{created: true, byEmail: map[string]*user.User{
			"a@b.com": {ID: "u1", Email: "a@b.com", PasswordHash: hash},
		}}
		fs := &fakeSessions{}
		svc := NewService(fu, fs)
		tok, err := svc.Login(ctx, "a@b.com", "password123", "agent", "127.0.0.1")
		if err != nil || tok == "" {
			t.Fatalf("login: tok=%q err=%v", tok, err)
		}
		if fs.saved == nil || fs.saved.TokenHash != security.HashToken(tok) {
			t.Fatal("session must be stored with the token hash")
		}
	})

	t.Run("login rejects wrong password and unknown email identically", func(t *testing.T) {
		hash, _ := security.HashPassword("password123")
		fu := &fakeUsers{created: true, byEmail: map[string]*user.User{
			"a@b.com": {ID: "u1", Email: "a@b.com", PasswordHash: hash},
		}}
		svc := NewService(fu, &fakeSessions{})
		for _, c := range []struct{ email, pw string }{{"a@b.com", "wrong"}, {"nobody@x.com", "password123"}} {
			_, err := svc.Login(ctx, c.email, c.pw, "", "")
			ae, ok := apperrors.As(err)
			if !ok || ae.Code != i18n.CodeAuthInvalidCredentials {
				t.Fatalf("expected invalid_credentials for %v, got %v", c, err)
			}
		}
	})
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/application/auth/ -v`
Expected: FAIL — undefined.

- [ ] **Step 4: Write `auth.go`**

```go
// Package auth holds the authentication use cases. It depends only on domain
// ports + the security/errors shared packages (no HTTP, no SQL).
package auth

import (
	"context"
	"errors"
	"time"

	"router-lens/internal/domain/user"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
	"router-lens/internal/shared/security"
)

const SessionTTL = 7 * 24 * time.Hour

type Service struct {
	users    user.UserRepository
	sessions user.SessionRepository
}

func NewService(users user.UserRepository, sessions user.SessionRepository) *Service {
	return &Service{users: users, sessions: sessions}
}

func (s *Service) NeedsSetup(ctx context.Context) (bool, error) {
	exists, err := s.users.AnyExists(ctx)
	return !exists, err
}

// Setup creates the single admin. Returns 403 setup_locked once any user exists.
func (s *Service) Setup(ctx context.Context, email, password, name string) error {
	hash, err := security.HashPassword(password)
	if err != nil {
		return err
	}
	created, err := s.users.CreateInitialAdmin(ctx, &user.User{Email: email, PasswordHash: hash, Name: name})
	if err != nil {
		return err
	}
	if !created {
		return apperrors.New(apperrors.KindForbidden, i18n.CodeAuthSetupLocked, "setup already completed")
	}
	return nil
}

// Login verifies credentials and creates a session, returning the opaque cookie token.
func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (string, error) {
	u, err := s.users.FindByEmail(ctx, email)
	if errors.Is(err, user.ErrNotFound) {
		return "", s.invalidCredentials()
	}
	if err != nil {
		return "", err
	}
	ok, err := security.VerifyPassword(password, u.PasswordHash)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", s.invalidCredentials()
	}

	token, err := security.GenerateSessionToken()
	if err != nil {
		return "", err
	}
	sess := &user.Session{
		UserID:    u.ID,
		TokenHash: security.HashToken(token),
		ExpiresAt: time.Now().Add(SessionTTL),
		UserAgent: userAgent,
		IP:        ip,
	}
	if err := s.sessions.Create(ctx, sess); err != nil {
		return "", err
	}
	return token, nil
}

func (s *Service) Logout(ctx context.Context, tokenHash string) error {
	return s.sessions.DeleteByTokenHash(ctx, tokenHash)
}

func (s *Service) invalidCredentials() error {
	return apperrors.New(apperrors.KindUnauthorized, i18n.CodeAuthInvalidCredentials, "invalid email or password")
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/application/auth/ ./internal/shared/i18n/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/application/auth/ internal/shared/i18n/i18n.go
git commit -m "feat: auth use cases (setup race-safe, login, logout) + i18n codes"
```

---

### Task 4: Session middleware

**TDD:** yes — auth gate behavior (no cookie → 401, valid → passes, expired → 401).

**Files:**
- Create: `internal/infrastructure/http/middleware/session_middleware.go`
- Test: `internal/infrastructure/http/middleware/session_middleware_test.go`

**Interfaces:**
- Consumes: `domain/user`, `shared/security`, `shared/errors`.
- Produces:
  ```go
  const ContextUserKey = "auth_user"
  const ContextSessionKey = "auth_session"
  func Session(sessions user.SessionRepository, users user.UserRepository) echo.MiddlewareFunc
  func CurrentUser(c echo.Context) *user.User       // nil if unauthenticated
  func CurrentSession(c echo.Context) *user.Session
  ```

- [ ] **Step 1: Write the failing test**

```go
package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"router-lens/internal/domain/user"
	"router-lens/internal/shared/security"
)

type stubSessions struct{ s *user.Session }

func (x stubSessions) Create(context.Context, *user.Session) error { return nil }
func (x stubSessions) FindByTokenHash(_ context.Context, h string) (*user.Session, error) {
	if x.s != nil && x.s.TokenHash == h {
		return x.s, nil
	}
	return nil, user.ErrNotFound
}
func (x stubSessions) DeleteByTokenHash(context.Context, string) error { return nil }

type stubUsers struct{ u *user.User }

func (x stubUsers) CreateInitialAdmin(context.Context, *user.User) (bool, error) { return false, nil }
func (x stubUsers) FindByEmail(context.Context, string) (*user.User, error)      { return nil, user.ErrNotFound }
func (x stubUsers) FindByID(_ context.Context, id string) (*user.User, error) {
	if x.u != nil && x.u.ID == id {
		return x.u, nil
	}
	return nil, user.ErrNotFound
}
func (x stubUsers) AnyExists(context.Context) (bool, error) { return true, nil }

func TestSessionMiddleware(t *testing.T) {
	e := echo.New()
	ok := func(c echo.Context) error { return c.String(http.StatusOK, "ok") }
	token := "tok"
	hash := security.HashToken(token)

	run := func(cookie *http.Cookie, sessions user.SessionRepository, users user.UserRepository) int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if cookie != nil {
			req.AddCookie(cookie)
		}
		c := e.NewContext(req, rec)
		_ = Session(sessions, users)(ok)(c)
		return rec.Code
	}

	t.Run("no cookie -> handled as error (nil body, error returned)", func(t *testing.T) {
		// Session returns an AppError; without the central handler the recorder stays 200,
		// so assert the error directly instead.
		c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
		if err := Session(stubSessions{}, stubUsers{})(ok)(c); err == nil {
			t.Fatal("missing cookie must return an error")
		}
	})

	valid := &user.Session{ID: "s1", UserID: "u1", TokenHash: hash, ExpiresAt: time.Now().Add(time.Hour)}
	if code := run(&http.Cookie{Name: security.SessionCookieName, Value: token},
		stubSessions{s: valid}, stubUsers{u: &user.User{ID: "u1"}}); code != http.StatusOK {
		t.Fatalf("valid session should pass, got %d", code)
	}

	expired := &user.Session{TokenHash: hash, ExpiresAt: time.Now().Add(-time.Hour)}
	c := e.NewContext(func() *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: security.SessionCookieName, Value: token})
		return r
	}(), httptest.NewRecorder())
	if err := Session(stubSessions{s: expired}, stubUsers{})(ok)(c); err == nil {
		t.Fatal("expired session must return an error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/infrastructure/http/middleware/ -run TestSessionMiddleware -v`
Expected: FAIL — undefined.

- [ ] **Step 3: Write `session_middleware.go`**

```go
package middleware

import (
	"errors"
	"time"

	"github.com/labstack/echo/v4"

	"router-lens/internal/domain/user"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/security"
)

const (
	ContextUserKey    = "auth_user"
	ContextSessionKey = "auth_session"
	codeUnauthorized  = "unauthorized"
)

// Session authenticates a request from the session cookie, loading the user and
// session into the context, or returns 401.
func Session(sessions user.SessionRepository, users user.UserRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie(security.SessionCookieName)
			if err != nil || cookie.Value == "" {
				return unauthorized()
			}
			sess, err := sessions.FindByTokenHash(c.Request().Context(), security.HashToken(cookie.Value))
			if errors.Is(err, user.ErrNotFound) || (sess != nil && sess.IsExpired(time.Now())) {
				return unauthorized()
			}
			if err != nil {
				return err
			}
			u, err := users.FindByID(c.Request().Context(), sess.UserID)
			if err != nil {
				return unauthorized()
			}
			c.Set(ContextUserKey, u)
			c.Set(ContextSessionKey, sess)
			return next(c)
		}
	}
}

func unauthorized() error {
	return apperrors.New(apperrors.KindUnauthorized, codeUnauthorized, "authentication required")
}

func CurrentUser(c echo.Context) *user.User {
	if u, ok := c.Get(ContextUserKey).(*user.User); ok {
		return u
	}
	return nil
}

func CurrentSession(c echo.Context) *user.Session {
	if s, ok := c.Get(ContextSessionKey).(*user.Session); ok {
		return s
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/infrastructure/http/middleware/ -run TestSessionMiddleware -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/http/middleware/session_middleware.go internal/infrastructure/http/middleware/session_middleware_test.go
git commit -m "feat: session middleware (cookie -> user/session in context, 401 otherwise)"
```

---

### Task 5: Setup + auth handlers, DTOs, and wiring

**TDD:** no — thin HTTP + composition root. Verified by the Task 7 e2e flow.

**Files:**
- Create: `internal/infrastructure/http/handler/auth_handler.go`
- Modify: `cmd/server/main.go` (wire repos, validator, auth service, handler, session middleware, routes)

**Interfaces:**
- Consumes: `application/auth.Service`, `shared/validator`, `shared/response`, `app.Config`, the session middleware.
- Produces:
  ```go
  type AuthHandler struct { /* svc, validator, cfg */ }
  func NewAuthHandler(svc *auth.Service, v *validator.Validator, cfg app.Config) *AuthHandler
  func (h *AuthHandler) Register(api *echo.Group, session echo.MiddlewareFunc)
  ```
  Routes: `GET /setup/status`, `POST /setup`, `POST /auth/login`, `POST /auth/logout` (session), `GET /auth/me` (session).

- [ ] **Step 1: Write `auth_handler.go`**

```go
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"router-lens/internal/application/auth"
	"router-lens/internal/app"
	mw "router-lens/internal/infrastructure/http/middleware"
	"router-lens/internal/shared/response"
	"router-lens/internal/shared/security"
	"router-lens/internal/shared/validator"
)

type AuthHandler struct {
	svc *auth.Service
	v   *validator.Validator
	cfg app.Config
}

func NewAuthHandler(svc *auth.Service, v *validator.Validator, cfg app.Config) *AuthHandler {
	return &AuthHandler{svc: svc, v: v, cfg: cfg}
}

func (h *AuthHandler) Register(api *echo.Group, session echo.MiddlewareFunc) {
	api.GET("/setup/status", h.setupStatus)
	api.POST("/setup", h.setup)
	api.POST("/auth/login", h.login)
	api.POST("/auth/logout", h.logout, session)
	api.GET("/auth/me", h.me, session)
}

type setupRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=128"`
	Name     string `json:"name" validate:"max=100"`
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type userDTO struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (h *AuthHandler) setupStatus(c echo.Context) error {
	needs, err := h.svc.NeedsSetup(c.Request().Context())
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, map[string]bool{"needs_setup": needs})
}

func (h *AuthHandler) setup(c echo.Context) error {
	var req setupRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	if err := h.svc.Setup(c.Request().Context(), req.Email, req.Password, req.Name); err != nil {
		return err
	}
	return response.Data(c, http.StatusCreated, map[string]bool{"created": true})
}

func (h *AuthHandler) login(c echo.Context) error {
	var req loginRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	token, err := h.svc.Login(c.Request().Context(), req.Email, req.Password,
		c.Request().UserAgent(), c.RealIP())
	if err != nil {
		return err
	}
	c.SetCookie(security.BuildSessionCookie(token, h.cookieOpts()))
	return response.Data(c, http.StatusOK, map[string]bool{"ok": true})
}

func (h *AuthHandler) logout(c echo.Context) error {
	if s := mw.CurrentSession(c); s != nil {
		if err := h.svc.Logout(c.Request().Context(), s.TokenHash); err != nil {
			return err
		}
	}
	c.SetCookie(security.ClearSessionCookie(h.cookieOpts()))
	return response.NoContent(c)
}

func (h *AuthHandler) me(c echo.Context) error {
	u := mw.CurrentUser(c)
	return response.Data(c, http.StatusOK, userDTO{ID: u.ID, Email: u.Email, Name: u.Name})
}

func (h *AuthHandler) cookieOpts() security.CookieOpts {
	return security.CookieOpts{
		Secure:    h.cfg.IsProduction(),
		CrossSite: h.cfg.CookieCrossSite,
		MaxAge:    auth.SessionTTL,
	}
}

// bindAndValidate binds the JSON body and validates it in the request language.
func bindAndValidate(c echo.Context, v *validator.Validator, dst any) error {
	if err := c.Bind(dst); err != nil {
		return err
	}
	return v.Struct(dst, response.LangOf(c))
}
```

- [ ] **Step 2: Wire auth into the Fx app (`cmd/server/main.go`)**

Plan 01 built the Fx app. Extend its `fx.New(...)` with the auth providers and a route-registration invoke — the existing `app.Load` / `providePool` / `infrahttp.NewServer` / `runMigrations` / `startServer` stay. The `fx.New(...)` in `main()` becomes:

```go
	fx.New(
		fx.Provide(
			app.Load,
			providePool,
			infrahttp.NewServer,
			validator.New, // () -> (*validator.Validator, error)
			fx.Annotate(postgres.NewUserRepository, fx.As(new(user.UserRepository))),
			fx.Annotate(postgres.NewSessionRepository, fx.As(new(user.SessionRepository))),
			auth.NewService,         // (user.UserRepository, user.SessionRepository) -> *auth.Service
			handler.NewAuthHandler,  // (*auth.Service, *validator.Validator, app.Config) -> *AuthHandler
		),
		fx.Invoke(runMigrations),
		fx.Invoke(registerAuthRoutes),
		fx.Invoke(startServer),
	).Run()
```

Add the route-registration invoke (runs during startup, before the server's `OnStart` listens):

```go
// registerAuthRoutes mounts the setup/auth routes on the shared Echo, behind
// the session middleware where required.
func registerAuthRoutes(e *echo.Echo, h *handler.AuthHandler, sessions user.SessionRepository, users user.UserRepository) {
	api := e.Group("/api/v1")
	h.Register(api, middleware.Session(sessions, users))
}
```

Add the new imports to `cmd/server/main.go` (`go.uber.org/fx` is already imported from Plan 01):

```go
	"router-lens/internal/application/auth"
	"router-lens/internal/domain/user"
	"router-lens/internal/infrastructure/http/handler"
	"router-lens/internal/infrastructure/http/middleware"
	"router-lens/internal/shared/validator"
```

Notes:
- `fx.Annotate(postgres.NewUserRepository, fx.As(new(user.UserRepository)))` exposes the concrete `*postgres.UserRepository` as the `user.UserRepository` interface that `auth.NewService` and `registerAuthRoutes` consume — constructors keep returning concrete types (idiomatic), Fx adapts them. Same for the session repository.
- `validator.New` returns `(*validator.Validator, error)`; Fx surfaces the error as a startup failure.
- Routes register on the same Echo via a `/api/v1` group; the SPA `/*` fallback from `NewServer` still serves non-API paths (Echo matches the more specific `/api/v1/...` routes first). The invoke runs before `startServer`'s `OnStart`, so every route is live when the server listens.

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: succeeds.

- [ ] **Step 4: Commit**

```bash
git add internal/infrastructure/http/handler/auth_handler.go cmd/server/main.go
git commit -m "feat: setup/auth handlers and auth wiring"
```

---

### Task 6: End-to-end verification (docker)

**TDD:** no — manual e2e of the auth flow.

- [ ] **Step 1: Boot**

Run: `docker compose up --build -d && sleep 8`

- [ ] **Step 2: Setup status is true on a fresh DB**

Run: `curl -fsS http://localhost:8080/api/v1/setup/status`
Expected: `{"data":{"needs_setup":true},"meta":{...}}`

- [ ] **Step 3: Create the admin**

Run:
```bash
curl -fsS -X POST http://localhost:8080/api/v1/setup \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"password123","name":"Admin"}'
```
Expected: `201` with `{"data":{"created":true},...}`. A second identical call returns `403` with `error.code = "auth.setup_locked"`.

- [ ] **Step 4: Login sets the cookie; /me works; logout clears it**

Run:
```bash
curl -fsS -i -c cookies.txt -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"password123"}'
curl -fsS -b cookies.txt http://localhost:8080/api/v1/auth/me
curl -fsS -b cookies.txt -X POST http://localhost:8080/api/v1/auth/logout
```
Expected: login sets a `Set-Cookie: rl_session=…; HttpOnly`; `/me` returns the admin's `userDTO`; logout returns `204` and clears the cookie. A wrong password returns `401 auth.invalid_credentials`. Sending `Accept-Language: id` returns the Indonesian message.

- [ ] **Step 5: Tear down + commit any fixups**

```bash
docker compose down
git add -A && git commit -m "test: verify auth e2e flow" --allow-empty
```

---

## Self-Review

- **Spec coverage (Plan 03 scope):** User + Session domain ✓, race-safe `CreateInitialAdmin` ✓, postgres repos + integration test ✓, Setup (locked-after-first) ✓, Login (generic invalid_credentials) ✓, Logout ✓, NeedsSetup/`setup/status` ✓, session middleware (401 / loads user) ✓, httpOnly session cookie via `security.BuildSessionCookie` ✓, EN/ID error codes added to catalog ✓, no CSRF token (decision 13) ✓.
- **Placeholder scan:** none. (`entity.go`'s temporary `var _ = context.Background` line is explicitly removed in Task 1 Step 4.)
- **Type consistency:** uses `security.HashPassword/VerifyPassword/GenerateSessionToken/HashToken/BuildSessionCookie/ClearSessionCookie/CookieOpts/SessionCookieName` (Plan 02), `validator.New()/Struct` (Plan 02), `response.Data/NoContent/LangOf` + `apperrors` + `i18n` (Plan 01). `auth.Service` methods match the handler calls. `user.UserRepository`/`SessionRepository` implemented by the postgres repos and the test fakes alike.

## Next

Plan 04 — Projects + API Keys + Pricing CRUD (all behind the session middleware from this plan; the pricing repository here is what Plan 05 ingestion reads for cost calculation).
