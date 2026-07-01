# Dashboard + Settings Design

> Fills the two remaining placeholder pages: the home Dashboard (`/`) and Settings (`/settings`).
> Both were `<Placeholder>` components with no design behind them.

## Goal

Give the single admin user a fast-orientation home page distinct from the existing deep-dive
Analytics page, and let them manage their own account (name + password) without touching the
database directly.

## Non-goals (explicit)

- Email change (email is the login identity — out of scope for v0.1)
- Avatar / profile picture
- Multi-session management UI (list/revoke individual sessions)
- 2FA
- Account deletion
- Retention-policy UI (retention stays env-var-only per CLAUDE.md decision 17)
- System-info panel (version, DB stats) — considered and dropped during design
- Any new CSRF middleware or per-endpoint rate-limiting (see Global Constraints)

## Dashboard (`/`)

Pure frontend change. **Zero new backend endpoints** — composes three endpoints that already
exist.

| Widget | Source | Notes |
|---|---|---|
| 3 StatCards (requests, cost, error rate) | `GET /analytics/overview?preset=24h` | Fixed 24h window, no date picker — this is what differentiates it from Analytics (which has full historical filtering). Cost rendering must distinguish "no requests yet" from "requests exist but unpriced" (see below). |
| Recent events (5 rows) | `GET /events?preset=24h&limit=5` | Reuses the Logs page's query shape; pass `limit=5` directly (the backend already parses and clamps it in `event_log_handler.go`) rather than fetching a full page and slicing client-side. Columns: time, provider, model, status. Link to `/logs`. |
| Recent projects (5 rows) | `GET /projects?page=1&limit=5` | Existing offset-paginated CRUD endpoint. Name + link to project detail. Empty state: "Create your first project" CTA linking to `/projects`. Link to `/projects` ("view all"). |

**Cost null-semantics fix (applies to both Dashboard and the pre-existing Analytics page):**
`total_cost_usd` is `NULL` both when there are zero requests in range AND when requests exist
but are all unpriced. The existing `costOrUnpriced()` helper on the Analytics page conflates
these. Both pages must gate on `total_requests`:
- `total_requests === 0` → render `"—"` (no data)
- `total_requests > 0 && total_cost_usd === null` → render `UNPRICED` ("Unpriced")
- otherwise → `formatUSD(total_cost_usd)`

This is a one-line fix-forward to `apps/frontend/src/routes/_app.analytics.tsx`'s
`costOrUnpriced` call sites alongside the new Dashboard code, not a scope expansion — same
helper, used correctly in both places.

## Settings (`/settings`) — Account only

Scope decided with the user: account management only. Theme and language toggles already live
globally in the Topbar and are not duplicated here.

### Backend additions

**`domain/user/repository.go`** — extend `UserRepository`:
```go
UpdateName(ctx context.Context, id, name string) error
UpdatePasswordHash(ctx context.Context, id, hash string) error
```

**`domain/user/repository.go`** — extend `SessionRepository`:
```go
DeleteByUserIDExceptTokenHash(ctx context.Context, userID, keepTokenHash string) error
```

**`adapter/postgres`** — implement all three as parameterized `UPDATE`/`DELETE` queries.

**`usecase/auth/auth.go`** — add to `Service`:
- `UpdateProfile(ctx, userID, name string) (*user.User, error)` — `repo.UpdateName` then
  `FindByID` to return the fresh row.
- `ChangePassword(ctx, userID, currentTokenHash, currentPassword, newPassword string) error`:
  1. `FindByID(userID)`
  2. `security.VerifyPassword(currentPassword, u.PasswordHash)` — on mismatch, return an
     `apperrors.KindValidation` error with i18n code `auth.invalid_current_password`
  3. `security.HashPassword(newPassword)`
  4. `repo.UpdatePasswordHash(userID, newHash)`
  5. `sessions.DeleteByUserIDExceptTokenHash(userID, currentTokenHash)` — revokes every other
     active session (defense in depth: a leaked session cookie stops working the moment the
     admin changes their password). The session making this request survives.

**`shared/i18n`** — new code `CodeAuthInvalidCurrentPassword` = `"auth.invalid_current_password"`,
EN: "Current password is incorrect.", ID: "Kata sandi saat ini salah."

**`adapter/http/dto/auth.go`** — new request DTOs:
```go
type UpdateProfileRequest struct {
    Name string `json:"name" validate:"required,max=100"`
}
type ChangePasswordRequest struct {
    CurrentPassword string `json:"current_password" validate:"required"`
    NewPassword     string `json:"new_password" validate:"required,min=8,max=128"`
}
```

**`adapter/http/handler/auth_handler.go`** — new routes, both behind the existing `session`
middleware:
- `PUT /auth/me` → `updateProfile` → calls `svc.UpdateProfile`, returns `dto.FromUser`
- `POST /auth/change-password` → `changePassword` → reads `mw.CurrentSession(c).TokenHash` to
  pass as `currentTokenHash`, calls `svc.ChangePassword`, returns `204 No Content`

### Frontend additions

**`services/authService.ts`** — add:
```ts
export async function updateProfile(input: { name: string }): Promise<User>
export async function changePassword(input: { currentPassword: string; newPassword: string }): Promise<void>
```

**`_app.settings.tsx`** — replace `<Placeholder>` with two card sections, following the existing
`PricingFormDialog`/`ProjectFormDialog` pattern (react-hook-form + zod + shadcn Form/Input/Button):

1. **Profile card** — name field (editable), email (read-only display). On submit success:
   update the `["auth","me"]` TanStack Query cache via `setQueryData`, toast success.
2. **Password card** — current password, new password, confirm new password. Zod `.refine()`
   checks `confirm === newPassword`. On submit success: reset the form, toast success. On
   `auth.invalid_current_password` error: surface inline on the current-password field via the
   backend's localized message (same error-mapping pattern already used elsewhere).

## Global Constraints

- Go 1.26 + Echo + Uber Fx hexagonal layering: domain (`user` package) stays framework-free;
  SQL lives only in `adapter/postgres`; HTTP concerns only in `adapter/http`.
- **CSRF:** CLAUDE.md decision 13 already governs this — same-origin deployment, `SameSite=Lax`,
  all state changes go through POST/PUT/DELETE (never GET). `PUT /auth/me` and
  `POST /auth/change-password` both comply as designed; **no new CSRF middleware is added**, per
  the existing decision.
- **Rate limiting:** CLAUDE.md decision 14 already defers full rate-limiting to v0.2. The
  change-password endpoint requires an authenticated session (not an anonymous brute-force
  surface like `/auth/login`), and the surviving risk (a leaked session guessing the password) is
  substantially closed by the other-session revocation above. Mark the deferral with a
  `ponytail:` comment at the handler; do not add new rate-limit infrastructure for this one route.
- i18n: EN (default) + ID, resolved via `Accept-Language`, following the existing
  `shared/i18n` catalog pattern.
- Sonar guardrails (Go + TS, per CLAUDE.md §6) apply to all new code.

## TDD fit

- **`ChangePassword` — TDD: yes.** Clear input→output security contract (wrong current password
  must be rejected; correct password must rotate the hash and revoke other sessions). Table-driven
  test in `auth_test.go` covering: success (hash rotated + other sessions deleted, current session
  untouched), wrong current password (rejected with `auth.invalid_current_password`, no mutation),
  repository error propagation.
- **`UpdateProfile`, repository methods, HTTP handlers — TDD: no.** Thin CRUD pass-through with
  no branching logic; covered by integration/regression tests after implementation.
- **Dashboard, Settings forms (frontend) — TDD: no.** Visual/layout composition; verified by
  running the app + `react-doctor`.

## Cross-model review (Codex, GPT-5.5)

An adversarial pass confirmed the Dashboard design as-is (one tweak: pass `limit=5` directly
instead of fetching-then-slicing) and flagged two real gaps in the initial Settings draft:

1. Password change should revoke other active sessions (adopted — see `ChangePassword` above).
2. Whether CSRF is addressed for the new mutating routes — resolved by confirming they comply
   with the project's existing decision (CLAUDE.md decision 13), not a new gap; no new CSRF
   middleware was added as a result.

It also flagged the cost-null-semantics ambiguity, adopted above.
