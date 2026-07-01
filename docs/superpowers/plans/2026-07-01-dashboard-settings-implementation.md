# Dashboard + Settings Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the two placeholder pages (`/` Dashboard, `/settings` Settings) with real functionality: a fast-orientation home snapshot, and account management (name + password change with other-session revocation).

**Architecture:** Dashboard is pure frontend composition of three already-existing endpoints (analytics overview, events, projects) — zero new backend surface. Settings needs two new backend endpoints (`PUT /auth/me`, `POST /auth/change-password`) added to the existing hexagonal `domain/user` → `usecase/auth` → `adapter/postgres` + `adapter/http` chain, then a frontend form pair.

**Tech Stack:** Go 1.26 + Echo + Uber Fx + pgx/v5 (backend); Vite + React + TanStack Router/Query + react-hook-form + zod + shadcn/ui (frontend). No new dependencies.

**Design doc:** `docs/superpowers/specs/2026-07-01-dashboard-settings-design.md` — read it first for the full rationale (including the cross-model review that added other-session revocation).

## Global Constraints

- Hexagonal layering: `domain/user` stays framework-free; SQL only in `adapter/postgres`; HTTP only in `adapter/http`. No business logic in handlers.
- CSRF: no new middleware — CLAUDE.md decision 13 already covers this (same-origin, `SameSite=Lax`, all mutations via POST/PUT/DELETE). Both new routes comply by construction.
- Rate limiting: none added for `change-password` — deferred per CLAUDE.md decision 14, mark the deferral with a `ponytail:` comment at the handler.
- i18n: every new error code gets an EN + ID catalog entry in `shared/i18n/i18n.go`, following the existing `auth.*` namespace pattern.
- No new npm/Go dependencies. No database migration (no new columns/tables — only new queries against existing `users`/`sessions` columns).
- Bun is the frontend package manager (`bun.lock`) — never introduce npm/yarn/pnpm lockfiles or commands.

### Sonar guardrails (Go + TypeScript/React) — write compliant from the start

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
- typescript:S3358 — no nested ternaries (extract to function/if-else).
- typescript:S3923 — no identical-branch ternaries (delete the ternary).
- typescript:S4624 — no nested template literals.
- typescript:S6582 — prefer optional chaining `?.`.
- typescript:S7755 — `arr.at(-1)` over `arr[arr.length - 1]`.
- typescript:S6819 — real elements over ARIA roles.
- typescript:S1874 — no deprecated APIs (zod v4: `z.email()` not `z.string().email()`).
- typescript:S6606 — prefer `??=`.
- typescript:S6479 — stable keys for lists (never array index).

When fixing one instance, check sibling files for the same anti-pattern and fix-forward.
```

---

### Task 1: Dashboard (`/`) — frontend only

**Files:**
- Modify: `apps/frontend/src/services/eventService.ts`
- Modify: `apps/frontend/src/routes/_app.index.tsx`
- Modify: `apps/frontend/src/routes/_app.analytics.tsx`
- Modify: `apps/frontend/src/i18n/en.json`
- Modify: `apps/frontend/src/i18n/id.json`

**Interfaces:**
- Consumes: `overviewQueryOptions(filters)` from `@/lib/analytics` (existing), `eventsQueryOptions(filters, cursor)` from `@/lib/events` (existing, extended here), `projectsQueryOptions(page, limit)` from `@/lib/projects` (existing), `StatCard` from `@/components/analytics/StatCard` (existing), `DataTable` from `@/components/DataTable` (existing), `formatUSD`/`UNPRICED` from `@/lib/money` (existing), `formatTokens` from `@/lib/token` (existing), `formatTimestamp` from `@/lib/date` (existing), `LogStatusBadge` from `@/components/logs/LogStatusBadge` (existing).
- Produces: `costOrUnpriced(usd: string | null, totalRequests: number): string` — a corrected version of the existing helper, exported from `_app.index.tsx` is not needed since each route keeps its own copy (matches the existing pattern where `_app.analytics.tsx` already has a private module-level `costOrUnpriced`); this task changes its signature in BOTH files identically.

This task has no backend dependency and does not depend on Task 2/3.

- [ ] **Step 1: Add `limit` to `EventFilters` so the dashboard widget can ask for 5 rows directly**

Edit `apps/frontend/src/services/eventService.ts`. Add `limit` to the interface and to `filterParams`:

```ts
export interface EventFilters {
  readonly projectId?: string;
  readonly preset?: "24h" | "7d" | "30d";
  readonly provider?: string;
  readonly model?: string;
  readonly isError?: boolean;
  readonly limit?: number;
}

function filterParams(filters: EventFilters, extra: Record<string, string> = {}): Record<string, string> {
  const params: Record<string, string> = { ...extra };
  if (filters.projectId) params.project_id = filters.projectId;
  if (filters.preset) params.preset = filters.preset;
  if (filters.provider) params.provider = filters.provider;
  if (filters.model) params.model = filters.model;
  if (filters.isError !== undefined) params.is_error = String(filters.isError);
  if (filters.limit) params.limit = String(filters.limit);
  return params;
}
```

(The Logs page never sets `limit`, so it keeps getting the server default — this is additive only.)

- [ ] **Step 2: Verify the frontend still builds**

Run: `cd apps/frontend && bun run build`
Expected: builds clean (no TS errors — `limit` is optional so `EventFilters` call sites without it are unaffected).

- [ ] **Step 3: Fix the cost null-semantics bug (fix-forward to Analytics, then reuse in Dashboard)**

The existing `costOrUnpriced` in `apps/frontend/src/routes/_app.analytics.tsx` renders `total_cost_usd === null` as "Unpriced" even when `total_requests === 0` (no data at all, not an unpriced-events case). Fix it to take `totalRequests` and gate on it.

Edit `apps/frontend/src/routes/_app.analytics.tsx` — replace:

```ts
function costOrUnpriced(usd: string | null): string {
  return usd ? formatUSD(usd) : UNPRICED;
}
```

with:

```ts
function costOrUnpriced(usd: string | null, totalRequests: number): string {
  if (totalRequests === 0) return "—";
  return usd ? formatUSD(usd) : UNPRICED;
}
```

Update its two call sites in the same file. The stat card call:

```tsx
<StatCard label={t("analytics.stats.cost")} value={o ? costOrUnpriced(o.total_cost_usd, o.total_requests) : "—"} hint={o && o.unpriced_count > 0 ? t("analytics.stats.unpricedHint", { count: o.unpriced_count }) : undefined} />
```

The two `DataTable` column cells (provider + model tables) currently call `costOrUnpriced(s.cost_usd)` — these operate on per-provider/per-model rows, not the overview, and a row only exists if it has `request_count > 0`, so change them to pass `s.request_count`:

```ts
{ key: "cost", header: t("analytics.columns.cost"), cell: (s) => costOrUnpriced(s.cost_usd, s.request_count) },
```

(Same replacement in both `providerColumns` and `modelColumns`.)

- [ ] **Step 4: Run the frontend build again to confirm the signature change compiles everywhere it's called**

Run: `cd apps/frontend && bun run build`
Expected: builds clean.

- [ ] **Step 5: Add the `dashboard.*` i18n keys**

Edit `apps/frontend/src/i18n/en.json`, add a new top-level `"dashboard"` key (alongside the existing `"nav"`, `"auth"`, etc. keys):

```json
"dashboard": {
  "statRequests": "Requests (24h)",
  "statCost": "Cost (24h)",
  "statErrorRate": "Error rate (24h)",
  "recentEvents": "Recent events",
  "recentProjects": "Projects",
  "viewAllLogs": "View all logs",
  "viewAllProjects": "View all",
  "noProjectsTitle": "No projects yet",
  "noProjectsCta": "Create your first project",
  "noEvents": "No events in the last 24 hours"
}
```

Edit `apps/frontend/src/i18n/id.json`, add the matching Indonesian block in the same position:

```json
"dashboard": {
  "statRequests": "Permintaan (24 jam)",
  "statCost": "Biaya (24 jam)",
  "statErrorRate": "Tingkat error (24 jam)",
  "recentEvents": "Event terbaru",
  "recentProjects": "Proyek",
  "viewAllLogs": "Lihat semua log",
  "viewAllProjects": "Lihat semua",
  "noProjectsTitle": "Belum ada proyek",
  "noProjectsCta": "Buat proyek pertama",
  "noEvents": "Tidak ada event dalam 24 jam terakhir"
}
```

- [ ] **Step 6: Write the Dashboard route**

Replace the full contents of `apps/frontend/src/routes/_app.index.tsx`:

```tsx
import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { DataTable, type DataTableColumn } from "@/components/DataTable";
import { LogStatusBadge } from "@/components/logs/LogStatusBadge";
import { StatCard } from "@/components/analytics/StatCard";
import { Button } from "@/components/ui/button";
import { overviewQueryOptions } from "@/lib/analytics";
import { eventsQueryOptions } from "@/lib/events";
import { projectsQueryOptions } from "@/lib/projects";
import { UNPRICED, formatUSD } from "@/lib/money";
import { formatTokens } from "@/lib/token";
import { formatTimestamp } from "@/lib/date";
import type { Event, Project } from "@/lib/schemas";

export const Route = createFileRoute("/_app/")({
  component: DashboardRoute,
});

/** Same null-semantics rule as Analytics: zero requests renders "—", not "Unpriced". */
function costOrUnpriced(usd: string | null, totalRequests: number): string {
  if (totalRequests === 0) return "—";
  return usd ? formatUSD(usd) : UNPRICED;
}

function DashboardRoute() {
  const { t } = useTranslation();
  const overview = useQuery(overviewQueryOptions({ preset: "24h" }));
  const events = useQuery(eventsQueryOptions({ preset: "24h", limit: 5 }, ""));
  const projects = useQuery(projectsQueryOptions(1, 5));

  const o = overview.data;

  const eventColumns: DataTableColumn<Event>[] = [
    { key: "time", header: t("logs.columns.time"), cell: (e) => formatTimestamp(e.request_started_at) },
    { key: "provider", header: t("logs.columns.provider"), cell: (e) => e.provider },
    { key: "model", header: t("logs.columns.model"), cell: (e) => e.model },
    { key: "status", header: t("logs.columns.status"), cell: (e) => <LogStatusBadge event={e} /> },
  ];
  const projectColumns: DataTableColumn<Project>[] = [
    {
      key: "name",
      header: t("projects.columns.name"),
      cell: (p) => (
        <Link to="/projects/$projectId" params={{ projectId: p.id }} className="font-medium hover:underline">
          {p.name}
        </Link>
      ),
    },
  ];

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-heading font-semibold">{t("nav.dashboard")}</h1>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <StatCard label={t("dashboard.statRequests")} value={o ? formatTokens(o.total_requests) : "—"} />
        <StatCard label={t("dashboard.statCost")} value={o ? costOrUnpriced(o.total_cost_usd, o.total_requests) : "—"} />
        <StatCard
          label={t("dashboard.statErrorRate")}
          value={o ? `${(o.error_rate * 100).toFixed(1)}%` : "—"}
          tone={o && o.error_rate > 0 ? "danger" : "default"}
        />
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-heading font-semibold">{t("dashboard.recentEvents")}</h2>
            <Button render={<Link to="/logs" />} nativeButton={false} variant="link" size="sm">
              {t("dashboard.viewAllLogs")}
            </Button>
          </div>
          <DataTable
            columns={eventColumns}
            rows={events.data?.items ?? []}
            rowKey={(e) => e.id}
            isLoading={events.isLoading}
            emptyMessage={t("dashboard.noEvents")}
          />
        </div>

        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-heading font-semibold">{t("dashboard.recentProjects")}</h2>
            <Button render={<Link to="/projects" />} nativeButton={false} variant="link" size="sm">
              {t("dashboard.viewAllProjects")}
            </Button>
          </div>
          {!projects.isLoading && projects.data?.items.length === 0 ? (
            <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed border-border py-12 text-center">
              <p className="text-sm text-muted-foreground">{t("dashboard.noProjectsTitle")}</p>
              <Button render={<Link to="/projects" />} nativeButton={false} size="sm">
                {t("dashboard.noProjectsCta")}
              </Button>
            </div>
          ) : (
            <DataTable
              columns={projectColumns}
              rows={projects.data?.items ?? []}
              rowKey={(p) => p.id}
              isLoading={projects.isLoading}
              emptyMessage={t("dashboard.noProjectsTitle")}
            />
          )}
        </div>
      </div>
    </div>
  );
}
```

Event rows show status, not latency, to keep the mini-table narrow — do not import `formatDuration` in this file.

- [ ] **Step 7: Run lint and build**

Run: `cd apps/frontend && bun run lint && bun run build`
Expected: both clean. Fix any unused-import or type errors before proceeding.

- [ ] **Step 8: Live verification**

Run: `docker compose up -d` (or the existing dev stack), log in, navigate to `/`.
Verify: 3 stat cards render with real numbers (or "—" on a fresh empty instance), recent events table shows up to 5 rows linking correctly, recent projects list shows up to 5 projects with a working link to project detail, empty-state CTA appears when there are zero projects. Take a screenshot.

- [ ] **Step 9: Commit**

```bash
git add apps/frontend/src/services/eventService.ts apps/frontend/src/routes/_app.index.tsx apps/frontend/src/routes/_app.analytics.tsx apps/frontend/src/i18n/en.json apps/frontend/src/i18n/id.json
git commit -m "feat(dashboard): home snapshot (stats, recent events, recent projects)"
```

---

### Task 2: Auth backend — repository extensions + UpdateProfile/ChangePassword usecase

**Files:**
- Modify: `apps/backend/internal/domain/user/repository.go`
- Modify: `apps/backend/internal/adapter/postgres/user_repository.go`
- Modify: `apps/backend/internal/adapter/postgres/session_repository.go`
- Modify: `apps/backend/internal/adapter/http/middleware/session_middleware_test.go`
- Modify: `apps/backend/internal/shared/i18n/i18n.go`
- Modify: `apps/backend/internal/usecase/auth/auth.go`
- Modify: `apps/backend/internal/usecase/auth/auth_test.go`

**Interfaces:**
- Consumes: `security.VerifyPassword`, `security.HashPassword` (existing, unchanged), `apperrors.New` (existing).
- Produces: `Service.UpdateProfile(ctx, userID, name string) (*user.User, error)` and `Service.ChangePassword(ctx, userID, currentSessionTokenHash, currentPassword, newPassword string) error` — Task 3's HTTP handler calls both by these exact signatures.

TDD: **yes for `ChangePassword`** (security path with a clear input→output contract — wrong current password must be rejected with no mutation; correct password must rotate the hash and revoke every other session while leaving the calling session alone). `UpdateProfile` and the repository/postgres methods are thin CRUD — no TDD, implemented directly.

- [ ] **Step 1: Extend the repository interfaces**

Edit `apps/backend/internal/domain/user/repository.go`:

```go
package user

import "context"

// UserRepository is the port for persisting and querying User aggregates.
type UserRepository interface {
	CreateInitialAdmin(ctx context.Context, u *User) (created bool, err error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	AnyExists(ctx context.Context) (bool, error)
	UpdateName(ctx context.Context, id, name string) error
	UpdatePasswordHash(ctx context.Context, id, hash string) error
}

// SessionRepository is the port for persisting and querying Session aggregates.
type SessionRepository interface {
	Create(ctx context.Context, s *Session) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	// DeleteByUserIDExceptTokenHash revokes every session belonging to userID except the
	// one identified by keepTokenHash (the session making the current request). Used after
	// a password change so a leaked session cookie stops working immediately.
	DeleteByUserIDExceptTokenHash(ctx context.Context, userID, keepTokenHash string) error
}
```

- [ ] **Step 2: Implement the two new `UserRepository` methods in postgres**

Edit `apps/backend/internal/adapter/postgres/user_repository.go`, add after `FindByID`:

```go
func (r *UserRepository) UpdateName(ctx context.Context, id, name string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET name = $1, updated_at = now() WHERE id = $2`, name, id)
	return err
}

func (r *UserRepository) UpdatePasswordHash(ctx context.Context, id, hash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET password_hash = $1, updated_at = now() WHERE id = $2`, hash, id)
	return err
}
```

- [ ] **Step 3: Implement the new `SessionRepository` method in postgres**

Edit `apps/backend/internal/adapter/postgres/session_repository.go`, add after `DeleteByTokenHash`:

```go
func (r *SessionRepository) DeleteByUserIDExceptTokenHash(ctx context.Context, userID, keepTokenHash string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM sessions WHERE user_id = $1 AND token_hash != $2`, userID, keepTokenHash)
	return err
}
```

- [ ] **Step 4: Fix the now-broken `SessionRepository` stub in the middleware test**

`apps/backend/internal/adapter/http/middleware/session_middleware_test.go`'s `stubSessions` implements `user.SessionRepository` and will no longer compile once Step 1 lands. Add the missing method (no behavior needed — this test never exercises revocation):

```go
func (x stubSessions) DeleteByUserIDExceptTokenHash(context.Context, string, string) error { return nil }
```

Add this line directly below the existing `func (x stubSessions) DeleteByTokenHash(...)` method in that file.

- [ ] **Step 5: Verify the backend builds (interfaces satisfied everywhere except `usecase/auth`, which we fix next)**

Run: `cd apps/backend && go build ./... 2>&1 | grep -v "usecase/auth"`
Expected: no output (clean) outside the `usecase/auth` package, which we're about to fix in the next steps.

- [ ] **Step 6: Add the new i18n error code**

Edit `apps/backend/internal/shared/i18n/i18n.go`. In the `--- auth ---` const block:

```go
	// --- auth ---
	CodeAuthInvalidCredentials    = "auth.invalid_credentials"
	CodeAuthSetupLocked           = "auth.setup_locked"
	CodeAuthInvalidCurrentPassword = "auth.invalid_current_password"
```

In the `--- auth ---` catalog block:

```go
	// --- auth ---
	CodeAuthInvalidCredentials:    {EN: "Invalid email or password", ID: "Email atau kata sandi salah"},
	CodeAuthSetupLocked:           {EN: "Setup is already completed", ID: "Setup sudah pernah dilakukan"},
	CodeAuthInvalidCurrentPassword: {EN: "Current password is incorrect", ID: "Kata sandi saat ini salah"},
```

- [ ] **Step 7: Write the failing tests for `ChangePassword` and `UpdateProfile`**

Edit `apps/backend/internal/usecase/auth/auth_test.go`. First, extend the fakes to satisfy the grown interfaces and to record what happened (replace the existing `fakeUsers`/`fakeSessions` type+method blocks with these):

```go
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
func (f *fakeUsers) FindByID(_ context.Context, id string) (*user.User, error) {
	for _, u := range f.byEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, user.ErrNotFound
}
func (f *fakeUsers) AnyExists(context.Context) (bool, error) { return f.created, nil }
func (f *fakeUsers) UpdateName(ctx context.Context, id, name string) error {
	u, err := f.FindByID(ctx, id)
	if err != nil {
		return err
	}
	u.Name = name
	return nil
}
func (f *fakeUsers) UpdatePasswordHash(ctx context.Context, id, hash string) error {
	u, err := f.FindByID(ctx, id)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	return nil
}

type fakeSessions struct {
	saved         *user.Session
	deletedUserID string
	keptTokenHash string
}

func (f *fakeSessions) Create(_ context.Context, s *user.Session) error { f.saved = s; return nil }
func (f *fakeSessions) FindByTokenHash(context.Context, string) (*user.Session, error) {
	return nil, user.ErrNotFound
}
func (f *fakeSessions) DeleteByTokenHash(context.Context, string) error { return nil }
func (f *fakeSessions) DeleteByUserIDExceptTokenHash(_ context.Context, userID, keepTokenHash string) error {
	f.deletedUserID = userID
	f.keptTokenHash = keepTokenHash
	return nil
}
```

Then add two new `t.Run` blocks inside `TestAuthService`, after the existing `"login rejects wrong password..."` block:

```go
	t.Run("update profile updates the name", func(t *testing.T) {
		fu := &fakeUsers{created: true, byEmail: map[string]*user.User{
			"a@b.com": {ID: "u1", Email: "a@b.com", Name: "Old Name"},
		}}
		svc := NewService(fu, &fakeSessions{})
		u, err := svc.UpdateProfile(ctx, "u1", "New Name")
		if err != nil {
			t.Fatalf("update profile: %v", err)
		}
		if u.Name != "New Name" {
			t.Fatalf("expected name updated, got %q", u.Name)
		}
	})

	t.Run("change password succeeds, rotates hash, revokes other sessions", func(t *testing.T) {
		hash, err := security.HashPassword(testPassword)
		if err != nil {
			t.Fatalf("HashPassword: %v", err)
		}
		fu := &fakeUsers{created: true, byEmail: map[string]*user.User{
			"a@b.com": {ID: "u1", Email: "a@b.com", PasswordHash: hash},
		}}
		fs := &fakeSessions{}
		svc := NewService(fu, fs)
		if err := svc.ChangePassword(ctx, "u1", "keep-hash", testPassword, "newpassword123"); err != nil {
			t.Fatalf("change password: %v", err)
		}
		ok, _ := security.VerifyPassword("newpassword123", fu.byEmail["a@b.com"].PasswordHash)
		if !ok {
			t.Fatal("password hash was not updated")
		}
		if fs.deletedUserID != "u1" || fs.keptTokenHash != "keep-hash" {
			t.Fatalf("expected other sessions revoked for u1 keeping keep-hash, got userID=%q keep=%q",
				fs.deletedUserID, fs.keptTokenHash)
		}
	})

	t.Run("change password rejects wrong current password without mutating state", func(t *testing.T) {
		hash, err := security.HashPassword(testPassword)
		if err != nil {
			t.Fatalf("HashPassword: %v", err)
		}
		fu := &fakeUsers{created: true, byEmail: map[string]*user.User{
			"a@b.com": {ID: "u1", Email: "a@b.com", PasswordHash: hash},
		}}
		fs := &fakeSessions{}
		svc := NewService(fu, fs)
		err = svc.ChangePassword(ctx, "u1", "keep-hash", "wrong-password", "newpassword123")
		ae, ok := apperrors.As(err)
		if !ok || ae.Code != i18n.CodeAuthInvalidCurrentPassword {
			t.Fatalf("expected invalid_current_password, got %v", err)
		}
		if fu.byEmail["a@b.com"].PasswordHash != hash {
			t.Fatal("password hash must not change on wrong current password")
		}
		if fs.deletedUserID != "" {
			t.Fatal("sessions must not be revoked when the password change fails")
		}
	})
```

- [ ] **Step 8: Run the tests to verify they fail (Service methods don't exist yet)**

Run: `cd apps/backend && go test ./internal/usecase/auth/... -run TestAuthService -v`
Expected: compile error — `svc.UpdateProfile` and `svc.ChangePassword` undefined.

- [ ] **Step 9: Implement `UpdateProfile` and `ChangePassword`**

Edit `apps/backend/internal/usecase/auth/auth.go`. Add after the existing `Logout` method:

```go
// UpdateProfile changes the admin's display name.
func (s *Service) UpdateProfile(ctx context.Context, userID, name string) (*user.User, error) {
	if err := s.users.UpdateName(ctx, userID, name); err != nil {
		return nil, err
	}
	return s.users.FindByID(ctx, userID)
}

// ChangePassword verifies currentPassword, rotates the password hash, and revokes every
// other active session — the session identified by currentSessionTokenHash (the one
// making this request) is left alone. This closes the "leaked session cookie" case: the
// moment the admin changes their password, any other session stops working.
func (s *Service) ChangePassword(ctx context.Context, userID, currentSessionTokenHash, currentPassword, newPassword string) error {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	ok, err := security.VerifyPassword(currentPassword, u.PasswordHash)
	if err != nil {
		return err
	}
	if !ok {
		return apperrors.New(apperrors.KindValidation, i18n.CodeAuthInvalidCurrentPassword, "current password is incorrect")
	}
	hash, err := security.HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.users.UpdatePasswordHash(ctx, userID, hash); err != nil {
		return err
	}
	return s.sessions.DeleteByUserIDExceptTokenHash(ctx, userID, currentSessionTokenHash)
}
```

- [ ] **Step 10: Run the tests to verify they pass**

Run: `cd apps/backend && go test ./internal/usecase/auth/... -run TestAuthService -v`
Expected: PASS, all subtests including the two new ones and `TestSessionMiddleware` in the middleware package.

Also run: `cd apps/backend && gofmt -w ./internal/... && go build ./... && go vet ./...`
Expected: clean (`gofmt -w` fixes the const-block alignment from Step 6 automatically).

- [ ] **Step 11: Commit**

```bash
git add apps/backend/internal/domain/user/repository.go apps/backend/internal/adapter/postgres/user_repository.go apps/backend/internal/adapter/postgres/session_repository.go apps/backend/internal/adapter/http/middleware/session_middleware_test.go apps/backend/internal/shared/i18n/i18n.go apps/backend/internal/usecase/auth/auth.go apps/backend/internal/usecase/auth/auth_test.go
git commit -m "feat(auth): UpdateProfile + ChangePassword usecases with other-session revocation"
```

---

### Task 3: Auth backend — HTTP layer (DTOs, handler, routes)

**Files:**
- Modify: `apps/backend/internal/adapter/http/dto/auth.go`
- Modify: `apps/backend/internal/adapter/http/handler/auth_handler.go`

**Interfaces:**
- Consumes: `Service.UpdateProfile`, `Service.ChangePassword` (from Task 2), `mw.CurrentUser(c)`, `mw.CurrentSession(c)` (existing middleware accessors), `bindAndValidate` (existing shared handler helper), `response.Data`, `response.NoContent` (existing).
- Produces: `PUT /auth/me`, `POST /auth/change-password` — consumed by Task 4's frontend.

TDD: no (thin HTTP pass-through with no branching logic beyond what `usecase/auth` already covers — this project does not unit-test the Echo handler layer for any existing endpoint either). Verified via live check in Step 5.

- [ ] **Step 1: Add the request DTOs**

Edit `apps/backend/internal/adapter/http/dto/auth.go`, add after `LoginRequest`:

```go
// UpdateProfileRequest is the PUT /auth/me payload.
type UpdateProfileRequest struct {
	Name string `json:"name" validate:"required,max=100"`
}

// ChangePasswordRequest is the POST /auth/change-password payload.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=128"`
}
```

- [ ] **Step 2: Add the handler methods**

Edit `apps/backend/internal/adapter/http/handler/auth_handler.go`. Add to `Register`:

```go
func (h *AuthHandler) Register(api *echo.Group, session echo.MiddlewareFunc) {
	api.GET("/setup/status", h.setupStatus)
	api.POST("/setup", h.setup)
	api.POST("/auth/login", h.login)
	api.POST("/auth/logout", h.logout, session)
	api.GET("/auth/me", h.me, session)
	api.PUT("/auth/me", h.updateProfile, session)
	api.POST("/auth/change-password", h.changePassword, session)
}
```

Add the two handler methods after `me`:

```go
func (h *AuthHandler) updateProfile(c echo.Context) error {
	var req dto.UpdateProfileRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	u := mw.CurrentUser(c)
	updated, err := h.svc.UpdateProfile(c.Request().Context(), u.ID, req.Name)
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, dto.FromUser(updated))
}

// ponytail: no rate-limiting on this route — full per-endpoint rate-limiting is deferred
// to v0.2 (CLAUDE.md decision 14); the surviving "leaked session guesses the password"
// risk is closed by ChangePassword revoking every other session on success.
func (h *AuthHandler) changePassword(c echo.Context) error {
	var req dto.ChangePasswordRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	u := mw.CurrentUser(c)
	sess := mw.CurrentSession(c)
	err := h.svc.ChangePassword(c.Request().Context(), u.ID, sess.TokenHash, req.CurrentPassword, req.NewPassword)
	if err != nil {
		return err
	}
	return response.NoContent(c)
}
```

- [ ] **Step 3: Verify the backend builds and existing tests pass**

Run: `cd apps/backend && go build ./... && go vet ./... && go test ./... -race -cover`
Expected: all clean, all existing tests still pass.

- [ ] **Step 4: Run `golangci-lint`**

Run: `cd apps/backend && golangci-lint run`
Expected: clean.

- [ ] **Step 5: Live verification**

Start the backend (`docker compose up -d` or `go run ./cmd/server`). With a valid session cookie (log in first):
- `curl -b cookies.txt -X PUT http://localhost:8080/api/v1/auth/me -H 'Content-Type: application/json' -d '{"name":"New Name"}'` → expect `200` with the updated user.
- `curl -b cookies.txt -X POST http://localhost:8080/api/v1/auth/change-password -H 'Content-Type: application/json' -d '{"current_password":"wrong","new_password":"newpassword123"}'` → expect `400` with `error.code == "auth.invalid_current_password"`.
- Repeat with the correct current password → expect `204`.
- Confirm any OTHER session for the same user (e.g. a second browser login) now gets `401` on its next request.

- [ ] **Step 6: Commit**

```bash
git add apps/backend/internal/adapter/http/dto/auth.go apps/backend/internal/adapter/http/handler/auth_handler.go
git commit -m "feat(auth): PUT /auth/me and POST /auth/change-password endpoints"
```

---

### Task 4: Settings (`/settings`) — frontend

**Files:**
- Modify: `apps/frontend/src/services/authService.ts`
- Modify: `apps/frontend/src/routes/_app.settings.tsx`
- Modify: `apps/frontend/src/i18n/en.json`
- Modify: `apps/frontend/src/i18n/id.json`

**Interfaces:**
- Consumes: `PUT /auth/me`, `POST /auth/change-password` (from Task 3), `meQueryOptions` from `@/lib/auth` (existing), `Field`/`FormError` from `@/components/form/*` (existing), `Card`/`Input`/`Button` shadcn primitives (existing).

Depends on Task 3 being complete (calls the new endpoints).

TDD: no (form composition, visual/layout work) — verified by running the app.

- [ ] **Step 1: Add the two service functions**

Edit `apps/frontend/src/services/authService.ts`, add at the end:

```ts
export interface UpdateProfileInput {
  name: string;
}

/** PUT /auth/me — updates the admin's display name. */
export async function updateProfile(input: UpdateProfileInput): Promise<User> {
  const res = await api.put("/auth/me", input);
  return userSchema.parse(res.data);
}

export interface ChangePasswordInput {
  current_password: string;
  new_password: string;
}

/** POST /auth/change-password — rotates the password hash and revokes every other session. */
export async function changePassword(input: ChangePasswordInput): Promise<void> {
  await api.post("/auth/change-password", input);
}
```

- [ ] **Step 2: Add the `settings.*` i18n keys**

Edit `apps/frontend/src/i18n/en.json`, add a new top-level `"settings"` key:

```json
"settings": {
  "profile": {
    "title": "Profile",
    "description": "Update your display name.",
    "nameLabel": "Name",
    "emailLabel": "Email",
    "save": "Save changes",
    "updated": "Profile updated",
    "errors": {
      "nameRequired": "Name is required"
    }
  },
  "password": {
    "title": "Password",
    "description": "Change your account password. Other active sessions will be signed out.",
    "currentLabel": "Current password",
    "newLabel": "New password",
    "confirmLabel": "Confirm new password",
    "save": "Change password",
    "updated": "Password changed",
    "errors": {
      "required": "This field is required",
      "tooShort": "Must be at least 8 characters",
      "mismatch": "Passwords do not match"
    }
  }
}
```

Edit `apps/frontend/src/i18n/id.json`, add the matching block:

```json
"settings": {
  "profile": {
    "title": "Profil",
    "description": "Ubah nama tampilan Anda.",
    "nameLabel": "Nama",
    "emailLabel": "Email",
    "save": "Simpan perubahan",
    "updated": "Profil diperbarui",
    "errors": {
      "nameRequired": "Nama wajib diisi"
    }
  },
  "password": {
    "title": "Kata Sandi",
    "description": "Ubah kata sandi akun Anda. Sesi aktif lainnya akan keluar otomatis.",
    "currentLabel": "Kata sandi saat ini",
    "newLabel": "Kata sandi baru",
    "confirmLabel": "Konfirmasi kata sandi baru",
    "save": "Ubah kata sandi",
    "updated": "Kata sandi diubah",
    "errors": {
      "required": "Wajib diisi",
      "tooShort": "Minimal 8 karakter",
      "mismatch": "Kata sandi tidak cocok"
    }
  }
}
```

- [ ] **Step 3: Write the Settings route**

Replace the full contents of `apps/frontend/src/routes/_app.settings.tsx`:

```tsx
import { zodResolver } from "@hookform/resolvers/zod";
import { createFileRoute } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { LoaderCircle } from "lucide-react";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { z } from "zod";
import { Field } from "@/components/form/Field";
import { FormError } from "@/components/form/FormError";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { meQueryOptions } from "@/lib/auth";
import { changePassword, updateProfile } from "@/services/authService";

export const Route = createFileRoute("/_app/settings")({
  component: SettingsRoute,
});

function SettingsRoute() {
  const { t } = useTranslation();
  return (
    <div className="flex max-w-xl flex-col gap-6">
      <h1 className="text-2xl font-heading font-semibold">{t("nav.settings")}</h1>
      <ProfileCard />
      <PasswordCard />
    </div>
  );
}

function ProfileCard() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const me = useQuery(meQueryOptions);

  const schema = z.object({ name: z.string().min(1, t("settings.profile.errors.nameRequired")).max(100) });
  const form = useForm({ resolver: zodResolver(schema), defaultValues: { name: "" } });

  useEffect(() => {
    if (me.data) form.reset({ name: me.data.name });
  }, [me.data, form]);

  const mutation = useMutation({
    mutationFn: (values: { name: string }) => updateProfile(values),
    onSuccess: (updated) => {
      queryClient.setQueryData(meQueryOptions.queryKey, updated);
      toast.success(t("settings.profile.updated"));
    },
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("settings.profile.title")}</CardTitle>
        <CardDescription>{t("settings.profile.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={form.handleSubmit((values) => mutation.mutate(values))} noValidate className="space-y-4">
          <FormError error={mutation.error} fallback={t("auth.errors.generic")} />
          <Field id="settings-name" label={t("settings.profile.nameLabel")} error={form.formState.errors.name?.message}>
            <Input id="settings-name" aria-invalid={!!form.formState.errors.name} {...form.register("name")} />
          </Field>
          <Field id="settings-email" label={t("settings.profile.emailLabel")}>
            <Input id="settings-email" value={me.data?.email ?? ""} disabled readOnly />
          </Field>
          <Button type="submit" disabled={mutation.isPending}>
            {mutation.isPending ? <LoaderCircle className="size-4 animate-spin motion-reduce:animate-none" /> : null}
            {t("settings.profile.save")}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}

function PasswordCard() {
  const { t } = useTranslation();

  const schema = z
    .object({
      currentPassword: z.string().min(1, t("settings.password.errors.required")),
      newPassword: z.string().min(8, t("settings.password.errors.tooShort")).max(128),
      confirmPassword: z.string().min(1, t("settings.password.errors.required")),
    })
    .refine((v) => v.newPassword === v.confirmPassword, {
      message: t("settings.password.errors.mismatch"),
      path: ["confirmPassword"],
    });

  const form = useForm({
    resolver: zodResolver(schema),
    defaultValues: { currentPassword: "", newPassword: "", confirmPassword: "" },
  });

  const mutation = useMutation({
    mutationFn: (values: { currentPassword: string; newPassword: string }) =>
      changePassword({ current_password: values.currentPassword, new_password: values.newPassword }),
    onSuccess: () => {
      toast.success(t("settings.password.updated"));
      form.reset({ currentPassword: "", newPassword: "", confirmPassword: "" });
    },
  });

  const onSubmit = form.handleSubmit((values) => mutation.mutate(values));

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("settings.password.title")}</CardTitle>
        <CardDescription>{t("settings.password.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={onSubmit} noValidate className="space-y-4">
          <FormError error={mutation.error} fallback={t("auth.errors.generic")} />
          <Field
            id="settings-current-password"
            label={t("settings.password.currentLabel")}
            error={form.formState.errors.currentPassword?.message}
          >
            <Input
              id="settings-current-password"
              type="password"
              aria-invalid={!!form.formState.errors.currentPassword}
              {...form.register("currentPassword")}
            />
          </Field>
          <Field
            id="settings-new-password"
            label={t("settings.password.newLabel")}
            error={form.formState.errors.newPassword?.message}
          >
            <Input
              id="settings-new-password"
              type="password"
              aria-invalid={!!form.formState.errors.newPassword}
              {...form.register("newPassword")}
            />
          </Field>
          <Field
            id="settings-confirm-password"
            label={t("settings.password.confirmLabel")}
            error={form.formState.errors.confirmPassword?.message}
          >
            <Input
              id="settings-confirm-password"
              type="password"
              aria-invalid={!!form.formState.errors.confirmPassword}
              {...form.register("confirmPassword")}
            />
          </Field>
          <Button type="submit" disabled={mutation.isPending}>
            {mutation.isPending ? <LoaderCircle className="size-4 animate-spin motion-reduce:animate-none" /> : null}
            {t("settings.password.save")}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
```

- [ ] **Step 4: Run lint and build**

Run: `cd apps/frontend && bun run lint && bun run build`
Expected: both clean.

- [ ] **Step 5: Live verification**

With the backend running (Task 3 complete), log in, navigate to `/settings`:
- Confirm the profile form is pre-filled with the current name and read-only email.
- Change the name, save, confirm the toast fires and the Topbar user-menu (which also reads `meQueryOptions`) reflects the new name without a page reload.
- Attempt a password change with the wrong current password — confirm the inline `FormError` banner shows the localized "Current password is incorrect" message and the form is NOT cleared.
- Attempt again with the correct current password and matching new/confirm — confirm success toast and the form resets.
- Log in from a second browser/incognito session as the same user, then change the password from the first — confirm the second session's next request redirects to `/login` (401).
- Confirm mismatched new/confirm passwords are caught client-side before submission.

- [ ] **Step 6: Commit**

```bash
git add apps/frontend/src/services/authService.ts apps/frontend/src/routes/_app.settings.tsx apps/frontend/src/i18n/en.json apps/frontend/src/i18n/id.json
git commit -m "feat(settings): profile + password change forms"
```

---

## Final verification (after all 4 tasks)

- [ ] Run `cd apps/backend && go build ./... && go vet ./... && golangci-lint run && go test -race -cover ./...` — all clean.
- [ ] Run `cd apps/frontend && bun run lint && bun run build` — clean.
- [ ] Run `make build-all` to produce the single embedded binary, run it, and re-verify both Dashboard and Settings end-to-end against the production artifact (matches the DoD verification approach already used for the rest of the app).
