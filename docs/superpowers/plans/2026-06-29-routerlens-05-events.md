# RouterLens Plan 05 — Event Ingestion + Request Logs + CSV Export

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the observability write + read path: ingest LLM events through API-key auth (validated, priced, idempotent), list them with keyset pagination + filters, fetch one, and stream a range-bounded, injection-safe CSV export.

**Architecture:** Hexagonal + DDD, mirroring Plans 03/04. New `event` bounded context. Decomposed **persistence-first** so every task is independently buildable: Task 1 delivers the complete `event` domain + the full `EventRepository` adapter (all methods + tests); Task 2 adds the ingestion write path (API-key middleware + ingest use case + ingest handler); Task 3 adds the session-authenticated read path (logs list/get + CSV export). Cost is computed **at ingest** with a price snapshot (or NULL when unpriced); `is_error` is derived once and stored.

**Tech Stack:** Go 1.26, Echo v4, Uber Fx, pgx/v5 (pgxpool), shopspring/decimal (codec registered in Plan 04 `db.go`), goose (migration 006 `llm_events` already applied). Reuses existing shared helpers: `pricing.CalculateCost` + `pricing.TokenUsage` (Plan 02), `pagination` keyset `Cursor`/`EncodeCursor`/`DecodeCursor` + `ParseOffset` (Plan 02), `datetime.ParseRange` (Plan 02), `csv.NewWriter` (Plan 02), `response`, `errors`, `i18n`, `validator`, and the `setupTestDB(t)` integration-test helper (Plan 04).

> **DEPENDS ON PLAN 04.** Do not execute Plan 05 until Plan 04 (Projects + API Keys + Pricing CRUD) is merged. Plan 05 extends the `apikey` and `pricing` ports Plan 04 creates: `apikey.APIKeyRepository.FindByHash` + `TouchLastUsed` (+ `APIKey.IsRevoked()` rule), and `pricing.PricingRepository.FindByProviderModel`. It relies on the `decimal` codec Plan 04 registers and the `setupTestDB` helper Plan 04 defines.

## Global Constraints

- **Layering (HARD):** `domain/` imports only stdlib + `shopspring/decimal`. Never `echo`, `i18n`, `pgx`, or `infrastructure/`. Repository interfaces in `domain/<ctx>/repository.go`, implementations in `infrastructure/postgres/`. Use cases never import `echo`. Handlers contain no business logic and run no SQL. Middleware (infrastructure) may depend on domain ports (mirrors `session_middleware.go`).
- **Naming clash:** application package and domain package share the name `event`. In the application layer alias the domain import: `eventdomain "router-lens/internal/domain/event"` (and `pricingdomain`, `apikeydomain` where used). Keep the application package named `event`.
- **Two auth boundaries stay separate:** dashboard/read routes use the session middleware (Plan 03/04); `POST /events` uses the API-key middleware ONLY. Never mix them on one route. The ingest and read surfaces are TWO handlers (`IngestHandler`, `EventLogHandler`) precisely so the auth boundary is structural.
- **Authorization footgun (decision 5):** the ingest request body must not contain `project_id`. `project_id` comes solely from the resolved API key, via the request context.
- **Cost at ingest (decisions 1/10/11):** look up the rule for `(provider, model)`; found → store `cost_usd` + `input_price_1m` + `output_price_1m` (snapshot); absent (unpriced) → store all three as **NULL** (never `0`). `cost_usd = NULL` is the unpriced signal.
- **`is_error` computed once at ingest and stored (decision 8):** `status_code >= 400 || error_message != ""`. Never re-derived at query time.
- **Idempotency (decision 7):** `INSERT ... ON CONFLICT (project_id, event_id) WHERE event_id IS NOT NULL DO NOTHING`. New → `202 {deduplicated:false}`; duplicate → `202 {deduplicated:true}` (never an error). A NULL `event_id` never conflicts (always inserts).
- **Keyset only for events (decision 9):** logs list uses `(request_started_at, id)` cursor — never offset. Cursor + index agree on column order including `id`.
- **Anti-duplication:** reuse `pricing.CalculateCost`, the `pagination` keyset helpers, `datetime.ParseRange`, `csv.Writer`, `setupTestDB` — do not reimplement. The event row column list is one `const`.
- **Money:** `decimal.Decimal` end to end; nullable money is `*decimal.Decimal` (nil ⇒ NULL ⇒ unpriced).

### Sonar guardrails — write compliant from the first commit

```
Go:
- go:S107 — ≤7 params (≤5 preferred). Ingest input is a struct (IngestInput), not a long param list.
- go:S3776 — cognitive complexity ≤15 → extract helpers (validation, query building); tests use t.Run subtests.
- go:S1192 — const for any string literal duplicated 3+ times (column lists, error codes, header names).
- errcheck — handle every returned error; never `_ = fallible()` except a documented best-effort (TouchLastUsed). Wrap with %w; sentinel + errors.Is/As. Tests use t.Fatalf on every fallible call.
- gosec — parameterized SQL only ($n); never concat user values into SQL. Identifiers in the query are fixed literals.

When fixing one instance, check sibling files for the same anti-pattern and fix-forward.
Review the diff against this list BEFORE marking compliant.
```

### Skill brief for implementer subagents (every task)

> Invoke `golang-expert` first — it is a hub skill and auto-chains the full Go discipline family (go-patterns / go-review / go-test / go-error-handling) + `senior-backend` + `senior-security` + `algorithmic-complexity`; follow its Auto-chain section. The ingest path is a security boundary (Bearer auth + untrusted input) — apply `senior-security` deliberately. Apply `ponytail` (YAGNI): reuse existing shared helpers, no speculative repository methods (events are immutable — no Update/Delete). Honor the Global Constraints + Sonar block.

### Algorithmic complexity (§8, Bahasa Indonesia)

- **Ingest (`Insert`):** satu `INSERT ... ON CONFLICT DO NOTHING`. Idempotency lewat partial unique index `idx_llm_events_idempotency (project_id, event_id)` → **O(log n)** insert, bukan SELECT-dulu-lalu-INSERT (hindari race + query ganda). Pricing lookup `(provider, model)` ke unique index → **O(log m)**, `m` = jumlah pricing rule (kecil). `CalculateCost` **O(1)**. Total per event: **O(log n)** — tanpa loop.
- **Logs list (keyset, project-filtered):** `WHERE project_id=$1 AND (request_started_at, id) < ($2,$3) ORDER BY request_started_at DESC, id DESC LIMIT $n` naik ke composite index `(project_id, request_started_at DESC, id DESC)` → **O(log n + limit)**, jauh lebih murah dari offset yang **O(n)** skip. Kita fetch `limit+1` baris untuk deteksi halaman terakhir — tetap O(log n + limit).
- **Logs list tanpa filter project (cross-project):** index dipimpin `project_id`, jadi keyset lintas-project `(request_started_at, id)` **tidak** pakai index itu optimal → Postgres sort, **O(n log n)**. Aman untuk MVP; tambah index khusus `(request_started_at DESC, id DESC)` HANYA bila terbukti lambat (§8 — jangan preemptif).
- **CSV export:** stream baris langsung ke response, dibatasi date range, **memori O(1)** (tidak buffer semua baris di RAM) — `pgx.Rows` di-iterasi, tiap baris langsung ditulis ke `csv.Writer`. `k` = baris dalam range.
- Tidak ada N+1, tidak ada query-in-loop, tidak ada `.find()`-in-loop di Plan ini.

### TDD verdicts (per §16)

- **Task 1 (persistence):** boundary `Validate` + `DeriveIsError` `TDD: yes` (pure, clear input→output; §8 edge cases ARE the tests). `buildEventWhere` filter builder `TDD: yes` (pure string+args, no DB). `EventRepository` impl (Insert/List/FindByID/Export) `TDD: no` (integration against real Postgres after; keyset cursor already unit-tested in Plan 02).
- **Task 2 (ingestion):** ingest use case `TDD: yes` (fake repos: priced→cost+snapshot, unpriced→NULL, duplicate→deduplicated). apikey/pricing repo method extensions + api-key middleware + ingest handler `TDD: no` (integration/smoke after).
- **Task 3 (read path):** query use case `Get` not-found mapping `TDD: yes` (fake repo). `eventToCSVRow` mapper `TDD: yes` (pure; unpriced→empty cell). Log + CSV handlers `TDD: no` (integration/manual; `csv.Writer` escaping already unit-tested in Plan 02).

---

## Task 1: Event persistence (domain + complete repository)

Delivers the full `event` bounded context at the persistence layer: the entity, boundary validation, the filter, the repository port, and a complete pgx adapter implementing **every** port method — so `var _ event.EventRepository = (*EventRepository)(nil)` compiles now and later tasks only consume it. No HTTP, no use cases, no Fx wiring in this task.

**Files:**
- Create: `apps/backend/internal/domain/event/entity.go`
- Create: `apps/backend/internal/domain/event/validation.go`
- Create: `apps/backend/internal/domain/event/validation_test.go`
- Create: `apps/backend/internal/domain/event/filter.go`
- Create: `apps/backend/internal/domain/event/repository.go`
- Create: `apps/backend/internal/infrastructure/postgres/event_filter.go`
- Create: `apps/backend/internal/infrastructure/postgres/event_filter_test.go`
- Create: `apps/backend/internal/infrastructure/postgres/event_repository.go`
- Create: `apps/backend/internal/infrastructure/postgres/event_repository_test.go`

**Interfaces:**
- Consumes: `pagination.Cursor` (Filter), `shopspring/decimal`, pgx.
- Produces (Tasks 2–3 + Plan 06 rely on these): `eventdomain.{Event, DeriveIsError, IngestInput, Validate, Filter, EventRepository, ErrNotFound}`, `postgres.{NewEventRepository, buildEventWhere}`.

- [ ] **Step 1: Write the event domain entity**

`apps/backend/internal/domain/event/entity.go`:
```go
// Package event is the LLM-event bounded context: the immutable observed call,
// its ingest input + boundary validation, the query filter, and the repository
// port. Imports only stdlib + shopspring/decimal (domain purity).
package event

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

// ErrNotFound is returned when an llm_events row is absent.
var ErrNotFound = errors.New("event: not found")

// Event is an immutable observed LLM call. Nil *decimal pointers mean unpriced;
// nil *int / *time mean the client omitted that optional field.
type Event struct {
	ID                string
	ProjectID         string
	EventID           string // optional client idempotency key; "" = none
	Provider          string
	Model             string
	RouteSource       string
	Agent             string
	InputTokens       int64
	OutputTokens      int64
	CostUSD           *decimal.Decimal
	InputPrice1M      *decimal.Decimal
	OutputPrice1M     *decimal.Decimal
	LatencyMs         *int
	StatusCode        *int
	IsError           bool
	ErrorMessage      string
	RequestStartedAt  time.Time
	RequestFinishedAt *time.Time
	ReceivedAt        time.Time
	Metadata          json.RawMessage // nil = none
	CreatedAt         time.Time
}

// DeriveIsError computes the stored is_error flag once at ingest: an HTTP status
// >= 400, or any non-empty error message, marks the call failed.
func DeriveIsError(statusCode *int, errorMessage string) bool {
	if errorMessage != "" {
		return true
	}
	return statusCode != nil && *statusCode >= 400
}
```

- [ ] **Step 2: Write the failing validation test**

`apps/backend/internal/domain/event/validation_test.go`:
```go
package event

import (
	"strings"
	"testing"
	"time"
)

func baseInput() IngestInput {
	return IngestInput{
		Provider:         "anthropic",
		Model:            "claude-sonnet-4-5",
		InputTokens:      12000,
		OutputTokens:     1800,
		RequestStartedAt: time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC),
	}
}

func TestValidate(t *testing.T) {
	now := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)
	maxBackdate := 7 * 24 * time.Hour

	t.Run("accepts a sane event", func(t *testing.T) {
		if err := Validate(baseInput(), now, maxBackdate); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})
	t.Run("rejects negative tokens", func(t *testing.T) {
		in := baseInput()
		in.InputTokens = -1
		if err := Validate(in, now, maxBackdate); err == nil {
			t.Fatal("want error for negative tokens")
		}
	})
	t.Run("rejects future timestamp", func(t *testing.T) {
		in := baseInput()
		in.RequestStartedAt = now.Add(time.Hour)
		if err := Validate(in, now, maxBackdate); err == nil {
			t.Fatal("want error for future timestamp")
		}
	})
	t.Run("rejects timestamp older than max backdate", func(t *testing.T) {
		in := baseInput()
		in.RequestStartedAt = now.Add(-8 * 24 * time.Hour)
		if err := Validate(in, now, maxBackdate); err == nil {
			t.Fatal("want error for stale timestamp")
		}
	})
	t.Run("rejects out-of-range status code", func(t *testing.T) {
		in := baseInput()
		bad := 700
		in.StatusCode = &bad
		if err := Validate(in, now, maxBackdate); err == nil {
			t.Fatal("want error for status 700")
		}
	})
	t.Run("rejects oversized metadata", func(t *testing.T) {
		in := baseInput()
		in.Metadata = []byte(`{"x":"` + strings.Repeat("a", maxMetadataBytes) + `"}`)
		if err := Validate(in, now, maxBackdate); err == nil {
			t.Fatal("want error for oversized metadata")
		}
	})
	t.Run("rejects negative latency", func(t *testing.T) {
		in := baseInput()
		neg := -5
		in.LatencyMs = &neg
		if err := Validate(in, now, maxBackdate); err == nil {
			t.Fatal("want error for negative latency")
		}
	})
	t.Run("validation error exposes an i18n code", func(t *testing.T) {
		in := baseInput()
		in.InputTokens = -1
		err := Validate(in, now, maxBackdate)
		var ve interface{ Code() string }
		if !errorsAs(err, &ve) || ve.Code() == "" {
			t.Fatalf("want a coded validation error, got %v", err)
		}
	})
}

// errorsAs is a tiny local shim so the test reads clearly; use errors.As in code.
func errorsAs(err error, target any) bool { return errorsAsImpl(err, target) }
```
NOTE: replace the `errorsAs`/`errorsAsImpl` shim with a direct `errors.As` call and `import "errors"` — it is shown abstractly only to express intent. The real test body should be: `var ve interface{ Code() string }; if !errors.As(err, &ve) || ve.Code() == "" { ... }`.

- [ ] **Step 3: Run it to verify it fails**

Run: `cd apps/backend && go test ./internal/domain/event/ -run TestValidate`
Expected: FAIL — `undefined: IngestInput` / `undefined: Validate`.

- [ ] **Step 4: Write the ingest input + boundary validation**

`apps/backend/internal/domain/event/validation.go`:
```go
package event

import (
	"encoding/json"
	"time"
)

// Sane upper bounds for a single observed call.
const (
	maxTokens        = 100_000_000 // 100M tokens per call is already absurd
	minStatusCode    = 100
	maxStatusCode    = 599
	maxMetadataBytes = 8 * 1024 // ~8 KB JSONB cap
	maxStringLen     = 256      // provider/model/agent/route_source
)

// Validation error codes. These string literals MUST match the i18n catalog
// keys added in Task 2 — the domain stays i18n-free (no import), so the strings
// live here and are mirrored, by value, in shared/i18n. This is the one accepted
// literal duplication (domain purity vs S1192).
const (
	codeInvalidTokens    = "event.invalid_tokens"
	codeInvalidLatency   = "event.invalid_latency"
	codeInvalidStatus    = "event.invalid_status"
	codeFutureTimestamp  = "event.future_timestamp"
	codeBackdateExceeded = "event.backdate_exceeded"
	codeStringTooLong    = "event.string_too_long"
	codeMetadataTooLarge = "event.metadata_too_large"
)

// IngestInput is the validated ingest command (a params object — keeps the use
// case under S107 and is the unit the validator operates on). It carries no
// project_id: that comes from the authenticated API key (decision 5).
type IngestInput struct {
	EventID           string
	Provider          string
	Model             string
	RouteSource       string
	Agent             string
	InputTokens       int64
	OutputTokens      int64
	LatencyMs         *int
	StatusCode        *int
	ErrorMessage      string
	RequestStartedAt  time.Time
	RequestFinishedAt *time.Time
	Metadata          json.RawMessage
}

// validationError is a typed boundary error carrying an i18n code. The
// application layer maps it to a KindValidation AppError (via errors.As on the
// Code() method), so the domain owns the rules without importing shared/errors.
// The field is unexported (`code`) to avoid a field/method name collision with
// the Code() accessor.
type validationError struct {
	code string
	msg  string
}

func (e validationError) Error() string { return e.msg }

// Code exposes the i18n code for the application-layer mapper.
func (e validationError) Code() string { return e.code }

// Validate enforces the ingest boundary rules. now + maxBackdate are injected so
// callers/tests control the reference point. Returns a validationError (use
// errors.As against `interface{ Code() string }`) on the first violated rule.
func Validate(in IngestInput, now time.Time, maxBackdate time.Duration) error {
	if err := validateTokens(in); err != nil {
		return err
	}
	if err := validateLatencyStatus(in); err != nil {
		return err
	}
	if err := validateTimestamps(in, now, maxBackdate); err != nil {
		return err
	}
	return validateSizes(in)
}

func validateTokens(in IngestInput) error {
	if in.InputTokens < 0 || in.OutputTokens < 0 || in.InputTokens > maxTokens || in.OutputTokens > maxTokens {
		return validationError{codeInvalidTokens, "token counts must be between 0 and the maximum"}
	}
	return nil
}

func validateLatencyStatus(in IngestInput) error {
	if in.LatencyMs != nil && *in.LatencyMs < 0 {
		return validationError{codeInvalidLatency, "latency_ms must not be negative"}
	}
	if in.StatusCode != nil && (*in.StatusCode < minStatusCode || *in.StatusCode > maxStatusCode) {
		return validationError{codeInvalidStatus, "status_code must be between 100 and 599"}
	}
	return nil
}

func validateTimestamps(in IngestInput, now time.Time, maxBackdate time.Duration) error {
	if in.RequestStartedAt.After(now) {
		return validationError{codeFutureTimestamp, "request_started_at must not be in the future"}
	}
	if now.Sub(in.RequestStartedAt) > maxBackdate {
		return validationError{codeBackdateExceeded, "request_started_at is older than the allowed backdate window"}
	}
	return nil
}

func validateSizes(in IngestInput) error {
	for _, s := range []string{in.Provider, in.Model, in.Agent, in.RouteSource} {
		if len(s) > maxStringLen {
			return validationError{codeStringTooLong, "a provider/model/agent/route_source field is too long"}
		}
	}
	if len(in.Metadata) > maxMetadataBytes {
		return validationError{codeMetadataTooLarge, "metadata exceeds the 8KB limit"}
	}
	return nil
}
```

- [ ] **Step 5: Run it to verify it passes**

Run: `cd apps/backend && go test ./internal/domain/event/`
Expected: PASS (after replacing the test's `errorsAs` shim with `errors.As` per Step 2's note).

- [ ] **Step 6: Write the Filter + the repository port**

`apps/backend/internal/domain/event/filter.go`:
```go
package event

import (
	"time"

	"router-lens/internal/shared/pagination"
)

// Filter selects events for List/Export. All fields are optional except Limit
// (List). A zero ProjectID means "all projects" (cross-project list).
type Filter struct {
	ProjectID string
	From      time.Time // zero = unbounded
	To        time.Time // zero = unbounded
	Provider  string
	Model     string
	IsError   *bool
	Cursor    pagination.Cursor // zero = first page
	Limit     int
}
```

`apps/backend/internal/domain/event/repository.go`:
```go
package event

import "context"

// EventRepository persists and queries observed events. Events are immutable —
// no Update/Delete.
type EventRepository interface {
	// Insert stores e idempotently. inserted=false means a row with the same
	// (project_id, event_id) already existed (event_id non-empty).
	Insert(ctx context.Context, e *Event) (inserted bool, err error)
	// List returns events matching f, newest first, using keyset pagination.
	List(ctx context.Context, f Filter) ([]*Event, error)
	FindByID(ctx context.Context, id string) (*Event, error)
	// Export streams events matching f (range-bounded) to fn, newest first.
	Export(ctx context.Context, f Filter, fn func(*Event) error) error
}
```

- [ ] **Step 7: Write the failing filter-builder test**

`apps/backend/internal/infrastructure/postgres/event_filter_test.go`:
```go
package postgres

import (
	"testing"
	"time"

	"router-lens/internal/domain/event"
	"router-lens/internal/shared/pagination"
)

func TestBuildEventWhere(t *testing.T) {
	t.Run("project + range + cursor produce ordered placeholders", func(t *testing.T) {
		f := event.Filter{
			ProjectID: "p1",
			From:      time.Unix(1000, 0).UTC(),
			To:        time.Unix(2000, 0).UTC(),
			Cursor:    pagination.Cursor{Time: time.Unix(1500, 0).UTC(), ID: "c1"},
			Limit:     20,
		}
		where, args := buildEventWhere(f)
		if where == "" || len(args) == 0 {
			t.Fatalf("expected conditions, got where=%q args=%v", where, args)
		}
		if args[0] != "p1" {
			t.Fatalf("first arg should be project id, got %v", args[0])
		}
		// last two args are the cursor tuple (time, id)
		if args[len(args)-1] != "c1" {
			t.Fatalf("last arg should be cursor id, got %v", args[len(args)-1])
		}
	})
	t.Run("no filters yields empty where and no args", func(t *testing.T) {
		where, args := buildEventWhere(event.Filter{})
		if where != "" || len(args) != 0 {
			t.Fatalf("expected empty, got where=%q args=%v", where, args)
		}
	})
}
```

- [ ] **Step 8: Run it to verify it fails**

Run: `cd apps/backend && go test ./internal/infrastructure/postgres/ -run TestBuildEventWhere`
Expected: FAIL — `undefined: buildEventWhere`.

- [ ] **Step 9: Write the filter→WHERE builder**

`apps/backend/internal/infrastructure/postgres/event_filter.go`:
```go
package postgres

import (
	"fmt"
	"strings"

	"router-lens/internal/domain/event"
)

// buildEventWhere assembles a parameterized WHERE clause + ordered args from a
// Filter. Column names are fixed literals; only values are parameterized ($n) —
// no SQL injection surface. Returns ("", nil) when no condition applies. The
// returned args are positional: the caller appends LIMIT as the next $n.
func buildEventWhere(f event.Filter) (string, []any) {
	var conds []string
	var args []any
	add := func(tmpl string, val any) {
		args = append(args, val)
		conds = append(conds, fmt.Sprintf(tmpl, len(args)))
	}
	if f.ProjectID != "" {
		add("project_id = $%d", f.ProjectID)
	}
	if !f.From.IsZero() {
		add("request_started_at >= $%d", f.From)
	}
	if !f.To.IsZero() {
		add("request_started_at <= $%d", f.To)
	}
	if f.Provider != "" {
		add("provider = $%d", f.Provider)
	}
	if f.Model != "" {
		add("model = $%d", f.Model)
	}
	if f.IsError != nil {
		add("is_error = $%d", *f.IsError)
	}
	if !f.Cursor.Time.IsZero() || f.Cursor.ID != "" {
		// keyset: (request_started_at, id) strictly before the cursor, DESC order.
		args = append(args, f.Cursor.Time, f.Cursor.ID)
		conds = append(conds, fmt.Sprintf("(request_started_at, id) < ($%d, $%d)", len(args)-1, len(args)))
	}
	if len(conds) == 0 {
		return "", nil
	}
	return "WHERE " + strings.Join(conds, " AND "), args
}
```

- [ ] **Step 10: Run it to verify it passes**

Run: `cd apps/backend && go test ./internal/infrastructure/postgres/ -run TestBuildEventWhere`
Expected: PASS.

- [ ] **Step 11: Write the complete event repository**

`apps/backend/internal/infrastructure/postgres/event_repository.go`:
```go
package postgres

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"router-lens/internal/domain/event"
)

type EventRepository struct{ pool *pgxpool.Pool }

func NewEventRepository(pool *pgxpool.Pool) *EventRepository { return &EventRepository{pool: pool} }

var _ event.EventRepository = (*EventRepository)(nil)

// eventColumns is the full select/return column list (kept once — S1192).
const eventColumns = `id, project_id, event_id, provider, model, route_source, agent,
	input_tokens, output_tokens, cost_usd, input_price_1m, output_price_1m,
	latency_ms, status_code, is_error, error_message,
	request_started_at, request_finished_at, received_at, metadata, created_at`

// nullEventID maps "" to a NULL event_id so the partial idempotency index does
// not treat blank keys as collidable.
func nullEventID(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// metadataArg passes JSON to the jsonb column as TEXT. pgx encodes a Go string
// into jsonb correctly; a raw []byte would be sent as bytea (wrong type). NULL
// when empty.
func metadataArg(m []byte) any {
	if len(m) == 0 {
		return nil
	}
	return string(m)
}

func (r *EventRepository) Insert(ctx context.Context, e *event.Event) (bool, error) {
	const q = `INSERT INTO llm_events (
			project_id, event_id, provider, model, route_source, agent,
			input_tokens, output_tokens, cost_usd, input_price_1m, output_price_1m,
			latency_ms, status_code, is_error, error_message,
			request_started_at, request_finished_at, received_at, metadata)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
		ON CONFLICT (project_id, event_id) WHERE event_id IS NOT NULL DO NOTHING
		RETURNING id`
	err := r.pool.QueryRow(ctx, q,
		e.ProjectID, nullEventID(e.EventID), e.Provider, e.Model, e.RouteSource, e.Agent,
		e.InputTokens, e.OutputTokens, e.CostUSD, e.InputPrice1M, e.OutputPrice1M,
		e.LatencyMs, e.StatusCode, e.IsError, e.ErrorMessage,
		e.RequestStartedAt, e.RequestFinishedAt, e.ReceivedAt, metadataArg(e.Metadata),
	).Scan(&e.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil // duplicate: ON CONFLICT DO NOTHING returned no row
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// scanEvent reads one row in eventColumns order. Shared by List/FindByID/Export.
func scanEvent(row pgx.Row, e *event.Event) error {
	return row.Scan(
		&e.ID, &e.ProjectID, &e.EventID, &e.Provider, &e.Model, &e.RouteSource, &e.Agent,
		&e.InputTokens, &e.OutputTokens, &e.CostUSD, &e.InputPrice1M, &e.OutputPrice1M,
		&e.LatencyMs, &e.StatusCode, &e.IsError, &e.ErrorMessage,
		&e.RequestStartedAt, &e.RequestFinishedAt, &e.ReceivedAt, &e.Metadata, &e.CreatedAt,
	)
}

func (r *EventRepository) List(ctx context.Context, f event.Filter) ([]*event.Event, error) {
	where, args := buildEventWhere(f)
	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	args = append(args, limit)
	q := `SELECT ` + eventColumns + ` FROM llm_events ` + where +
		` ORDER BY request_started_at DESC, id DESC LIMIT $` + strconv.Itoa(len(args))
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*event.Event, 0, limit)
	for rows.Next() {
		var e event.Event
		if err := scanEvent(rows, &e); err != nil {
			return nil, err
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

func (r *EventRepository) FindByID(ctx context.Context, id string) (*event.Event, error) {
	q := `SELECT ` + eventColumns + ` FROM llm_events WHERE id = $1`
	var e event.Event
	err := scanEvent(r.pool.QueryRow(ctx, q, id), &e)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, event.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *EventRepository) Export(ctx context.Context, f event.Filter, fn func(*event.Event) error) error {
	where, args := buildEventWhere(f)
	q := `SELECT ` + eventColumns + ` FROM llm_events ` + where +
		` ORDER BY request_started_at DESC, id DESC`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var e event.Event
		if err := scanEvent(rows, &e); err != nil {
			return err
		}
		if err := fn(&e); err != nil {
			return err
		}
	}
	return rows.Err()
}
```

- [ ] **Step 12: Write the repository integration test (idempotency + keyset + metadata round-trip)**

`apps/backend/internal/infrastructure/postgres/event_repository_test.go`. Use the shared `setupTestDB(t) (context.Context, *pgxpool.Pool)` helper from Plan 04 (`project_repository_test.go`) — do NOT redefine it. Extract a `seedProject(t, ctx, pool) *project.Project` helper (insert user + project, `t.Fatalf` on error) shared by the subtests.
```go
package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"router-lens/internal/domain/event"
	"router-lens/internal/domain/project"
)

func seedProject(t *testing.T, ctx context.Context, pool *pgxpool.Pool) *project.Project {
	t.Helper()
	var ownerID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name) VALUES ($1,$2,$3) RETURNING id`,
		"event-owner@test.local", "x", "Owner").Scan(&ownerID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	p := &project.Project{OwnerUserID: ownerID, Name: "Ev", Slug: "ev", Description: ""}
	if err := NewProjectRepository(pool).Create(ctx, p); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	return p
}

func TestEventRepositoryInsert(t *testing.T) {
	ctx, pool := setupTestDB(t)
	proj := seedProject(t, ctx, pool)
	repo := NewEventRepository(pool)

	mk := func() *event.Event {
		return &event.Event{
			ProjectID: proj.ID, EventID: "evt-1", Provider: "anthropic", Model: "claude",
			InputTokens: 100, OutputTokens: 50, IsError: false,
			Metadata:         json.RawMessage(`{"workspace":"nuvora"}`),
			RequestStartedAt: time.Now().UTC().Add(-time.Minute), ReceivedAt: time.Now().UTC(),
		}
	}

	t.Run("first insert stores + metadata round-trips, duplicate deduplicates", func(t *testing.T) {
		e := mk()
		inserted, err := repo.Insert(ctx, e)
		if err != nil || !inserted {
			t.Fatalf("first insert: inserted=%v err=%v", inserted, err)
		}
		got, err := repo.FindByID(ctx, e.ID)
		if err != nil {
			t.Fatalf("find: %v", err)
		}
		if string(got.Metadata) == "" || got.Provider != "anthropic" {
			t.Fatalf("round-trip mismatch: %+v", got)
		}
		inserted2, err := repo.Insert(ctx, mk())
		if err != nil {
			t.Fatalf("dup insert: %v", err)
		}
		if inserted2 {
			t.Fatal("duplicate event_id should not insert again")
		}
	})

	t.Run("empty event_id always inserts", func(t *testing.T) {
		e1, e2 := mk(), mk()
		e1.EventID, e2.EventID = "", ""
		i1, err := repo.Insert(ctx, e1)
		if err != nil {
			t.Fatalf("insert e1: %v", err)
		}
		i2, err := repo.Insert(ctx, e2)
		if err != nil {
			t.Fatalf("insert e2: %v", err)
		}
		if !i1 || !i2 {
			t.Fatalf("null event_id must always insert: %v %v", i1, i2)
		}
	})
}

func TestEventRepositoryList(t *testing.T) {
	ctx, pool := setupTestDB(t)
	proj := seedProject(t, ctx, pool)
	repo := NewEventRepository(pool)

	base := time.Now().UTC().Add(-time.Hour)
	for i := 0; i < 3; i++ {
		e := &event.Event{
			ProjectID: proj.ID, Provider: "p", Model: "m", InputTokens: 1, OutputTokens: 1,
			RequestStartedAt: base.Add(time.Duration(i) * time.Minute), ReceivedAt: time.Now().UTC(),
		}
		if _, err := repo.Insert(ctx, e); err != nil {
			t.Fatalf("seed event %d: %v", i, err)
		}
	}

	t.Run("newest first, project-scoped", func(t *testing.T) {
		list, err := repo.List(ctx, event.Filter{ProjectID: proj.ID, Limit: 10})
		if err != nil || len(list) != 3 {
			t.Fatalf("list: len=%d err=%v", len(list), err)
		}
		if !list[0].RequestStartedAt.After(list[1].RequestStartedAt) {
			t.Fatal("expected newest first")
		}
	})

	t.Run("findbyid missing -> ErrNotFound", func(t *testing.T) {
		if _, err := repo.FindByID(ctx, "00000000-0000-0000-0000-000000000000"); err != event.ErrNotFound {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})
}
```

- [ ] **Step 13: Verify + Commit**

Run: `cd apps/backend && gofmt -l . && go vet ./... && go build ./... && go test ./internal/... -count=1`
Expected: gofmt prints nothing; vet/build clean; unit suites PASS; repo integration tests skip without `TEST_DATABASE_URL` (run them against an isolated Postgres if available).
```bash
git add apps/backend
git commit -m "feat: event persistence — domain + complete repository (Plan 05 task 1)"
```

---

## Task 2: Event ingestion (API-key boundary, validation, cost, idempotent insert)

Adds the write path on top of Task 1's repository: extend the apikey + pricing ports with the ingest-time lookups, the ingest use case, the Bearer API-key middleware, the ingest handler, and the Fx wiring.

**Files:**
- Modify: `apps/backend/internal/domain/apikey/entity.go` (add `IsRevoked`)
- Modify: `apps/backend/internal/domain/apikey/repository.go` (add `FindByHash`, `TouchLastUsed`)
- Modify: `apps/backend/internal/infrastructure/postgres/apikey_repository.go` (impl both)
- Modify: `apps/backend/internal/domain/pricing/repository.go` (add `FindByProviderModel`)
- Modify: `apps/backend/internal/infrastructure/postgres/pricing_repository.go` (impl it)
- Create: `apps/backend/internal/application/event/ingest.go`
- Create: `apps/backend/internal/application/event/ingest_test.go`
- Create: `apps/backend/internal/infrastructure/http/middleware/apikey_middleware.go`
- Create: `apps/backend/internal/infrastructure/http/handler/ingest_handler.go`
- Modify: `apps/backend/cmd/server/main.go` (wire event repo + ingest service + ingest handler + api-key route)
- Modify: `apps/backend/internal/shared/i18n/i18n.go` (add `event.*` validation codes)

**Interfaces:**
- Consumes: `eventdomain.{Event, IngestInput, Validate, EventRepository}`, `pricingdomain.PricingRepository.FindByProviderModel`, `pricing.CalculateCost` + `pricing.TokenUsage`, `apikeydomain.APIKeyRepository.{FindByHash,TouchLastUsed}` + `APIKey.IsRevoked`, `app.Config.MaxBackdateDays`, `security.HashAPIKey`, `apperrors`, `i18n`, `response`, the `handler` package's `bindAndValidate`.
- Produces: `event.NewIngestService`, `mw.APIKey` + `mw.CurrentProjectID`, `handler.NewIngestHandler`.

- [ ] **Step 1: Extend the apikey domain port + revoke rule**

In `apps/backend/internal/domain/apikey/entity.go` add:
```go
// IsRevoked reports whether the key has been revoked and may no longer ingest.
func (k APIKey) IsRevoked() bool { return k.RevokedAt != nil }
```
In `apps/backend/internal/domain/apikey/repository.go` add to the interface:
```go
	// FindByHash resolves a key by its sha256 hash. Returns ErrNotFound when absent.
	FindByHash(ctx context.Context, keyHash string) (*APIKey, error)
	// TouchLastUsed sets last_used_at = now() for the key. Best-effort; a missing
	// row is not an error here (the caller already authenticated the key).
	TouchLastUsed(ctx context.Context, id string) error
```

- [ ] **Step 2: Implement the new apikey repository methods**

Append to `apps/backend/internal/infrastructure/postgres/apikey_repository.go`:
```go
func (r *APIKeyRepository) FindByHash(ctx context.Context, keyHash string) (*apikey.APIKey, error) {
	const q = `SELECT id, project_id, name, key_prefix, last_used_at, revoked_at, created_at
		FROM api_keys WHERE key_hash = $1`
	var k apikey.APIKey
	err := r.pool.QueryRow(ctx, q, keyHash).
		Scan(&k.ID, &k.ProjectID, &k.Name, &k.KeyPrefix, &k.LastUsedAt, &k.RevokedAt, &k.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apikey.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *APIKeyRepository) TouchLastUsed(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE api_keys SET last_used_at = now() WHERE id = $1`, id)
	return err
}
```

- [ ] **Step 3: Extend the pricing port with a (provider, model) lookup**

In `apps/backend/internal/domain/pricing/repository.go` add to the interface:
```go
	// FindByProviderModel resolves a rule by its unique (provider, model) pair.
	// Returns ErrNotFound when the pair is unpriced.
	FindByProviderModel(ctx context.Context, provider, model string) (*PricingRule, error)
```
Append to `apps/backend/internal/infrastructure/postgres/pricing_repository.go`:
```go
func (r *PricingRepository) FindByProviderModel(ctx context.Context, provider, model string) (*pricing.PricingRule, error) {
	q := `SELECT ` + pricingColumns + ` FROM pricing_rules WHERE provider = $1 AND model = $2`
	var rule pricing.PricingRule
	err := scanRule(r.pool.QueryRow(ctx, q, provider, model), &rule)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, pricing.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rule, nil
}
```

- [ ] **Step 4: Write the failing ingest use-case test**

`apps/backend/internal/application/event/ingest_test.go`:
```go
package event

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	eventdomain "router-lens/internal/domain/event"
	pricingdomain "router-lens/internal/domain/pricing"
	apperrors "router-lens/internal/shared/errors"
)

type fakeEventRepo struct {
	inserted bool
	got      *eventdomain.Event
}

func (f *fakeEventRepo) Insert(_ context.Context, e *eventdomain.Event) (bool, error) {
	f.got = e
	return f.inserted, nil
}
func (f *fakeEventRepo) List(context.Context, eventdomain.Filter) ([]*eventdomain.Event, error) {
	return nil, nil
}
func (f *fakeEventRepo) FindByID(context.Context, string) (*eventdomain.Event, error) { return nil, nil }
func (f *fakeEventRepo) Export(context.Context, eventdomain.Filter, func(*eventdomain.Event) error) error {
	return nil
}

type fakePricingRepo struct{ rule *pricingdomain.PricingRule }

func (f *fakePricingRepo) FindByProviderModel(context.Context, string, string) (*pricingdomain.PricingRule, error) {
	if f.rule == nil {
		return nil, pricingdomain.ErrNotFound
	}
	return f.rule, nil
}

func validInput() eventdomain.IngestInput {
	return eventdomain.IngestInput{
		Provider: "anthropic", Model: "claude", InputTokens: 1_000_000, OutputTokens: 1_000_000,
		RequestStartedAt: time.Now().Add(-time.Minute),
	}
}

func TestIngest(t *testing.T) {
	t.Run("prices a known model and stores the snapshot", func(t *testing.T) {
		fr := &fakeEventRepo{inserted: true}
		pr := &fakePricingRepo{rule: &pricingdomain.PricingRule{
			InputPricePer1M: decimal.NewFromInt(3), OutputPricePer1M: decimal.NewFromInt(15),
		}}
		res, err := NewIngestService(fr, pr, 7*24*time.Hour).Ingest(context.Background(), "p1", validInput())
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if res.Deduplicated {
			t.Fatal("new event should not be deduplicated")
		}
		if fr.got.CostUSD == nil || !fr.got.CostUSD.Equal(decimal.NewFromInt(18)) {
			t.Fatalf("cost = %v, want 18", fr.got.CostUSD)
		}
		if fr.got.InputPrice1M == nil || fr.got.ProjectID != "p1" {
			t.Fatal("snapshot/project not stored")
		}
	})
	t.Run("leaves cost NULL for an unpriced model", func(t *testing.T) {
		fr := &fakeEventRepo{inserted: true}
		_, err := NewIngestService(fr, &fakePricingRepo{rule: nil}, 7*24*time.Hour).
			Ingest(context.Background(), "p1", validInput())
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if fr.got.CostUSD != nil || fr.got.InputPrice1M != nil {
			t.Fatal("unpriced event must have NULL cost + snapshot")
		}
	})
	t.Run("duplicate -> deduplicated true", func(t *testing.T) {
		fr := &fakeEventRepo{inserted: false}
		res, _ := NewIngestService(fr, &fakePricingRepo{}, 7*24*time.Hour).
			Ingest(context.Background(), "p1", validInput())
		if !res.Deduplicated {
			t.Fatal("want deduplicated true")
		}
	})
	t.Run("maps validation error to KindValidation AppError", func(t *testing.T) {
		in := validInput()
		in.InputTokens = -1
		_, err := NewIngestService(&fakeEventRepo{}, &fakePricingRepo{}, 7*24*time.Hour).
			Ingest(context.Background(), "p1", in)
		ae, ok := apperrors.As(err)
		if !ok || ae.Kind != apperrors.KindValidation {
			t.Fatalf("want validation AppError, got %v", err)
		}
	})
}
```

- [ ] **Step 5: Run it to verify it fails**

Run: `cd apps/backend && go test ./internal/application/event/`
Expected: FAIL — `undefined: NewIngestService`.

- [ ] **Step 6: Write the ingest use case**

`apps/backend/internal/application/event/ingest.go`:
```go
// Package event holds the event use cases: ingest (write path) and list/get/
// export (read path, Task 3). Depends only on domain ports + the cost calculator
// + shared errors (no HTTP, no SQL).
package event

import (
	"context"
	"errors"
	"time"

	eventdomain "router-lens/internal/domain/event"
	pricingdomain "router-lens/internal/domain/pricing"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
)

// pricingLookup is the slice of the pricing port the ingest path needs.
type pricingLookup interface {
	FindByProviderModel(ctx context.Context, provider, model string) (*pricingdomain.PricingRule, error)
}

// IngestResult reports whether the event was newly stored or a duplicate.
type IngestResult struct {
	ID           string
	Deduplicated bool
}

type IngestService struct {
	events      eventdomain.EventRepository
	pricing     pricingLookup
	maxBackdate time.Duration
}

func NewIngestService(events eventdomain.EventRepository, pricing pricingLookup, maxBackdate time.Duration) *IngestService {
	return &IngestService{events: events, pricing: pricing, maxBackdate: maxBackdate}
}

// Ingest validates, prices, and idempotently stores an event for projectID.
func (s *IngestService) Ingest(ctx context.Context, projectID string, in eventdomain.IngestInput) (IngestResult, error) {
	now := time.Now().UTC()
	if err := eventdomain.Validate(in, now, s.maxBackdate); err != nil {
		return IngestResult{}, mapValidation(err)
	}
	e := buildEvent(projectID, in, now)
	if err := s.applyPricing(ctx, e); err != nil {
		return IngestResult{}, err
	}
	inserted, err := s.events.Insert(ctx, e)
	if err != nil {
		return IngestResult{}, err
	}
	return IngestResult{ID: e.ID, Deduplicated: !inserted}, nil
}

// applyPricing looks up the rule and snapshots cost; unpriced leaves NULLs.
func (s *IngestService) applyPricing(ctx context.Context, e *eventdomain.Event) error {
	rule, err := s.pricing.FindByProviderModel(ctx, e.Provider, e.Model)
	if errors.Is(err, pricingdomain.ErrNotFound) {
		return nil // unpriced: cost + snapshot stay nil
	}
	if err != nil {
		return err
	}
	r := rule.Rule()
	cost := pricingdomain.CalculateCost(
		pricingdomain.TokenUsage{InputTokens: e.InputTokens, OutputTokens: e.OutputTokens},
		&r,
	)
	if cost != nil {
		e.CostUSD = &cost.USD
		e.InputPrice1M = &cost.InputPrice1M
		e.OutputPrice1M = &cost.OutputPrice1M
	}
	return nil
}

func buildEvent(projectID string, in eventdomain.IngestInput, now time.Time) *eventdomain.Event {
	return &eventdomain.Event{
		ProjectID:         projectID,
		EventID:           in.EventID,
		Provider:          in.Provider,
		Model:             in.Model,
		RouteSource:       in.RouteSource,
		Agent:             in.Agent,
		InputTokens:       in.InputTokens,
		OutputTokens:      in.OutputTokens,
		LatencyMs:         in.LatencyMs,
		StatusCode:        in.StatusCode,
		IsError:           eventdomain.DeriveIsError(in.StatusCode, in.ErrorMessage),
		ErrorMessage:      in.ErrorMessage,
		RequestStartedAt:  in.RequestStartedAt,
		RequestFinishedAt: in.RequestFinishedAt,
		ReceivedAt:        now,
		Metadata:          in.Metadata,
	}
}

// mapValidation converts a domain validationError (carrying an i18n code via a
// Code() method) to a localized KindValidation AppError; any other error
// becomes a generic validation AppError.
func mapValidation(err error) error {
	var ve interface{ Code() string }
	if errors.As(err, &ve) {
		return apperrors.New(apperrors.KindValidation, ve.Code(), err.Error())
	}
	return apperrors.New(apperrors.KindValidation, i18n.CodeValidation, err.Error())
}
```
NOTE: `CalculateCost` takes `*Rule`; bind `rule.Rule()` to a local `r` then pass `&r` (cannot take the address of a method return directly).

- [ ] **Step 7: Register the `event.*` validation i18n codes**

In `apps/backend/internal/shared/i18n/i18n.go` add to the const block + catalog. The catalog keys MUST equal the literal strings in `domain/event/validation.go` (the domain stays i18n-free). The `i18n.CodeEvent*` consts document them + are used elsewhere:
```go
	// --- event ---
	CodeEventInvalidTokens    = "event.invalid_tokens"
	CodeEventInvalidLatency   = "event.invalid_latency"
	CodeEventInvalidStatus    = "event.invalid_status"
	CodeEventFutureTimestamp  = "event.future_timestamp"
	CodeEventBackdateExceeded = "event.backdate_exceeded"
	CodeEventStringTooLong    = "event.string_too_long"
	CodeEventMetadataTooLarge = "event.metadata_too_large"
```
```go
	// --- event ---
	CodeEventInvalidTokens:    {EN: "Token counts are out of range", ID: "Jumlah token di luar rentang"},
	CodeEventInvalidLatency:   {EN: "Latency must not be negative", ID: "Latensi tidak boleh negatif"},
	CodeEventInvalidStatus:    {EN: "Status code must be between 100 and 599", ID: "Kode status harus antara 100 dan 599"},
	CodeEventFutureTimestamp:  {EN: "request_started_at must not be in the future", ID: "request_started_at tidak boleh di masa depan"},
	CodeEventBackdateExceeded: {EN: "request_started_at is older than the allowed window", ID: "request_started_at lebih lama dari jendela yang diizinkan"},
	CodeEventStringTooLong:    {EN: "A field is too long", ID: "Salah satu field terlalu panjang"},
	CodeEventMetadataTooLarge: {EN: "metadata exceeds the 8KB limit", ID: "metadata melebihi batas 8KB"},
```

- [ ] **Step 8: Run it to verify it passes**

Run: `cd apps/backend && go test ./internal/application/event/`
Expected: PASS.

- [ ] **Step 9: Write the API-key middleware**

`apps/backend/internal/infrastructure/http/middleware/apikey_middleware.go`:
```go
package middleware

import (
	"errors"
	"strings"

	"github.com/labstack/echo/v4"

	"router-lens/internal/domain/apikey"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
	"router-lens/internal/shared/security"
)

// ContextProjectIDKey holds the project id resolved from the API key.
const ContextProjectIDKey = "ingest_project_id"

const bearerPrefix = "Bearer "

// APIKey authenticates an ingestion request from a Bearer API key: resolve by
// hash, reject if missing or revoked, best-effort touch last_used_at, inject the
// project id. SECOND auth boundary — never combined with the session middleware.
func APIKey(keys apikey.APIKeyRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			k, err := resolveKey(c, keys)
			if err != nil {
				return err
			}
			_ = keys.TouchLastUsed(c.Request().Context(), k.ID) // best-effort; never blocks ingestion
			c.Set(ContextProjectIDKey, k.ProjectID)
			return next(c)
		}
	}
}

// resolveKey extracts the Bearer key, hashes it, looks it up, and enforces the
// revoked rule. Extracted to keep the outer closure under S3776.
func resolveKey(c echo.Context, keys apikey.APIKeyRepository) (*apikey.APIKey, error) {
	header := c.Request().Header.Get(echo.HeaderAuthorization)
	if !strings.HasPrefix(header, bearerPrefix) {
		return nil, unauthorizedKey()
	}
	plaintext := strings.TrimPrefix(header, bearerPrefix)
	if plaintext == "" {
		return nil, unauthorizedKey()
	}
	k, err := keys.FindByHash(c.Request().Context(), security.HashAPIKey(plaintext))
	if errors.Is(err, apikey.ErrNotFound) {
		return nil, unauthorizedKey()
	}
	if err != nil {
		return nil, err
	}
	if k.IsRevoked() {
		return nil, unauthorizedKey()
	}
	return k, nil
}

func unauthorizedKey() error {
	return apperrors.New(apperrors.KindUnauthorized, i18n.CodeUnauthorized, "invalid or revoked API key")
}

// CurrentProjectID returns the project id injected by the APIKey middleware.
func CurrentProjectID(c echo.Context) string {
	if id, ok := c.Get(ContextProjectIDKey).(string); ok {
		return id
	}
	return ""
}
```

- [ ] **Step 10: Write the ingest handler**

`apps/backend/internal/infrastructure/http/handler/ingest_handler.go`:
```go
package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	eventapp "router-lens/internal/application/event"
	eventdomain "router-lens/internal/domain/event"
	mw "router-lens/internal/infrastructure/http/middleware"
	"router-lens/internal/shared/response"
	"router-lens/internal/shared/validator"
)

type IngestHandler struct {
	svc *eventapp.IngestService
	v   *validator.Validator
}

func NewIngestHandler(svc *eventapp.IngestService, v *validator.Validator) *IngestHandler {
	return &IngestHandler{svc: svc, v: v}
}

// Register mounts POST /events behind the API-key middleware ONLY.
func (h *IngestHandler) Register(api *echo.Group, apiKey echo.MiddlewareFunc) {
	api.POST("/events", h.create, apiKey)
}

// ingestRequest is the wire payload. It deliberately has NO project_id field
// (decision 5 — project comes from the API key).
type ingestRequest struct {
	EventID           string          `json:"event_id" validate:"max=200"`
	Provider          string          `json:"provider" validate:"required,max=256"`
	Model             string          `json:"model" validate:"required,max=256"`
	RouteSource       string          `json:"route_source" validate:"max=256"`
	Agent             string          `json:"agent" validate:"max=256"`
	InputTokens       int64           `json:"input_tokens" validate:"gte=0"`
	OutputTokens      int64           `json:"output_tokens" validate:"gte=0"`
	LatencyMs         *int            `json:"latency_ms" validate:"omitempty,gte=0"`
	StatusCode        *int            `json:"status_code" validate:"omitempty,gte=100,lte=599"`
	ErrorMessage      string          `json:"error_message"`
	RequestStartedAt  time.Time       `json:"request_started_at" validate:"required"`
	RequestFinishedAt *time.Time      `json:"request_finished_at"`
	Metadata          json.RawMessage `json:"metadata"`
}

func (r ingestRequest) toInput() eventdomain.IngestInput {
	return eventdomain.IngestInput{
		EventID: r.EventID, Provider: r.Provider, Model: r.Model,
		RouteSource: r.RouteSource, Agent: r.Agent,
		InputTokens: r.InputTokens, OutputTokens: r.OutputTokens,
		LatencyMs: r.LatencyMs, StatusCode: r.StatusCode, ErrorMessage: r.ErrorMessage,
		RequestStartedAt: r.RequestStartedAt.UTC(), RequestFinishedAt: r.RequestFinishedAt,
		Metadata: r.Metadata,
	}
}

func (h *IngestHandler) create(c echo.Context) error {
	var req ingestRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	res, err := h.svc.Ingest(c.Request().Context(), mw.CurrentProjectID(c), req.toInput())
	if err != nil {
		return err
	}
	body := map[string]any{"deduplicated": res.Deduplicated}
	if !res.Deduplicated {
		body["id"] = res.ID
	}
	return response.Data(c, http.StatusAccepted, body)
}
```

- [ ] **Step 11: Wire ingestion in Fx**

In `apps/backend/cmd/server/main.go`:

(a) `fx.Provide(...)` add:
```go
		fx.Annotate(postgres.NewEventRepository, fx.As(new(event.EventRepository))),
		provideIngestService,    // (event.EventRepository, pricing.PricingRepository, app.Config) -> *eventapp.IngestService
		handler.NewIngestHandler, // (*eventapp.IngestService, *validator.Validator) -> *IngestHandler
```
imports:
```go
	event "router-lens/internal/domain/event"
	eventapp "router-lens/internal/application/event"
```
(b) provider that derives the backdate window from config (keeps `NewIngestService` free of `app.Config` — hexagonal):
```go
// provideIngestService builds the ingest use case, deriving the backdate window
// from config (MaxBackdateDays). pricing.PricingRepository satisfies the use
// case's narrow pricingLookup interface (structural typing).
func provideIngestService(events event.EventRepository, prices pricing.PricingRepository, cfg app.Config) *eventapp.IngestService {
	return eventapp.NewIngestService(events, prices, time.Duration(cfg.MaxBackdateDays)*24*time.Hour)
}
```
(c) registrar + invoke (API-key middleware built inline — it is the only consumer):
```go
func registerEventIngestRoutes(e *echo.Echo, h *handler.IngestHandler, keys apikey.APIKeyRepository) {
	h.Register(e.Group("/api/v1"), middleware.APIKey(keys))
}
```
```go
		fx.Invoke(registerEventIngestRoutes),
```
`apikey`, `pricing`, `time` are already imported from Plan 04 wiring — confirm and add any missing.

- [ ] **Step 12: Verify + Commit**

Run: `cd apps/backend && gofmt -l . && go vet ./... && go build ./... && go test ./internal/... -count=1`
Expected: clean; all non-integration tests PASS. Optional docker smoke (after `docker compose up`, a project + key created via the dashboard):
```
curl -s -X POST localhost:8080/api/v1/events -H 'Authorization: Bearer <plaintext-key>' \
  -H 'Content-Type: application/json' \
  -d '{"provider":"anthropic","model":"claude-sonnet-4-5","input_tokens":12000,"output_tokens":1800,"latency_ms":8420,"status_code":200,"request_started_at":"2026-06-29T10:00:00Z"}'
```
Expected: `202 {"data":{"id":"…","deduplicated":false},"meta":{…}}`; repeat with the same `event_id` → `202 {"data":{"deduplicated":true}}`; bad/missing key → `401`.
```bash
git add apps/backend
git commit -m "feat: event ingestion — api-key auth, validation, cost, idempotent insert (Plan 05 task 2)"
```

---

## Task 3: Request logs + CSV export (session-authenticated read path)

Adds the read surface on top of Task 1's repository: the query use case, a session-authenticated log handler (list with keyset + filters, get), and a streamed CSV export.

**Files:**
- Create: `apps/backend/internal/application/event/query.go`
- Create: `apps/backend/internal/application/event/query_test.go`
- Create: `apps/backend/internal/infrastructure/http/handler/event_log_handler.go`
- Create: `apps/backend/internal/infrastructure/http/handler/event_csv_test.go`
- Modify: `apps/backend/cmd/server/main.go` (wire query service + session log/export routes)
- Modify: `apps/backend/internal/shared/i18n/i18n.go` (add `event.not_found`)

**Interfaces:**
- Consumes: `eventdomain.{Event, Filter, EventRepository, ErrNotFound}`, `pagination.{Cursor, EncodeCursor, DecodeCursor, ParseOffset}`, `datetime.ParseRange`, `csv.NewWriter`, the session middleware (`provideSessionMiddleware`, Plan 04), `response.Data`, the `handler` helpers `timeLayout`/`formatNullableTime` (Plan 04).
- Produces: `event.NewQueryService` (List/Get/Export), `handler.NewEventLogHandler`.

- [ ] **Step 1: Write the failing query use-case test (Get not-found mapping)**

`apps/backend/internal/application/event/query_test.go`:
```go
package event

import (
	"context"
	"testing"

	eventdomain "router-lens/internal/domain/event"
	apperrors "router-lens/internal/shared/errors"
)

type notFoundRepo struct{ fakeEventRepo }

func (notFoundRepo) FindByID(context.Context, string) (*eventdomain.Event, error) {
	return nil, eventdomain.ErrNotFound
}

func TestGetNotFound(t *testing.T) {
	_, err := NewQueryService(notFoundRepo{}).Get(context.Background(), "x")
	ae, ok := apperrors.As(err)
	if !ok || ae.Kind != apperrors.KindNotFound {
		t.Fatalf("want not_found AppError, got %v", err)
	}
}

func TestListProbesOneExtraForHasMore(t *testing.T) {
	// repo returns 3 rows for a requested limit of 2 -> hasMore true, 2 returned.
	repo := &limitProbeRepo{rows: 3}
	events, hasMore, err := NewQueryService(repo).List(context.Background(), eventdomain.Filter{Limit: 2})
	if err != nil || !hasMore || len(events) != 2 {
		t.Fatalf("hasMore=%v len=%d err=%v", hasMore, len(events), err)
	}
	if repo.askedLimit != 3 {
		t.Fatalf("expected probe limit 3, got %d", repo.askedLimit)
	}
}

type limitProbeRepo struct {
	fakeEventRepo
	rows       int
	askedLimit int
}

func (r *limitProbeRepo) List(_ context.Context, f eventdomain.Filter) ([]*eventdomain.Event, error) {
	r.askedLimit = f.Limit
	out := make([]*eventdomain.Event, 0, r.rows)
	for i := 0; i < r.rows; i++ {
		out = append(out, &eventdomain.Event{ID: "e"})
	}
	return out, nil
}
```
NOTE: `fakeEventRepo` is defined in `ingest_test.go` (same package) — reuse it; `notFoundRepo`/`limitProbeRepo` embed it and override only the methods they exercise.

- [ ] **Step 2: Run it to verify it fails**

Run: `cd apps/backend && go test ./internal/application/event/ -run 'TestGetNotFound|TestListProbes'`
Expected: FAIL — `undefined: NewQueryService`.

- [ ] **Step 3: Write the query use case (List with hasMore, Get, Export)**

`apps/backend/internal/application/event/query.go`:
```go
package event

import (
	"context"
	"errors"

	eventdomain "router-lens/internal/domain/event"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
)

type QueryService struct{ events eventdomain.EventRepository }

func NewQueryService(events eventdomain.EventRepository) *QueryService {
	return &QueryService{events: events}
}

// List fetches one extra row beyond f.Limit to detect whether another page
// exists, then returns exactly f.Limit rows + hasMore. This avoids the
// off-by-one where a final page of exactly Limit rows emits a dead cursor.
func (s *QueryService) List(ctx context.Context, f eventdomain.Filter) (events []*eventdomain.Event, hasMore bool, err error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	probe := f
	probe.Limit = limit + 1
	rows, err := s.events.List(ctx, probe)
	if err != nil {
		return nil, false, err
	}
	if len(rows) > limit {
		return rows[:limit], true, nil
	}
	return rows, false, nil
}

func (s *QueryService) Get(ctx context.Context, id string) (*eventdomain.Event, error) {
	e, err := s.events.FindByID(ctx, id)
	if errors.Is(err, eventdomain.ErrNotFound) {
		return nil, apperrors.New(apperrors.KindNotFound, i18n.CodeEventNotFound, "event not found")
	}
	return e, err
}

func (s *QueryService) Export(ctx context.Context, f eventdomain.Filter, fn func(*eventdomain.Event) error) error {
	return s.events.Export(ctx, f, fn)
}
```
Add to `apps/backend/internal/shared/i18n/i18n.go` (const + catalog): `CodeEventNotFound = "event.not_found"` / `{EN: "Event not found", ID: "Event tidak ditemukan"}`.

- [ ] **Step 4: Run it to verify it passes**

Run: `cd apps/backend && go test ./internal/application/event/`
Expected: PASS.

- [ ] **Step 5: Write the failing CSV row-mapper test**

`apps/backend/internal/infrastructure/http/handler/event_csv_test.go`:
```go
package handler

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	eventdomain "router-lens/internal/domain/event"
)

func TestEventToCSVRow(t *testing.T) {
	cost := decimal.RequireFromString("0.054")
	status := 200
	e := &eventdomain.Event{
		ID: "e1", ProjectID: "p1", Provider: "anthropic", Model: "claude",
		InputTokens: 12000, OutputTokens: 1800, CostUSD: &cost, StatusCode: &status,
		IsError: false, RequestStartedAt: time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC),
	}
	row := eventToCSVRow(e)
	if len(row) != len(csvHeader) {
		t.Fatalf("row width %d != header width %d", len(row), len(csvHeader))
	}

	unpriced := &eventdomain.Event{ID: "e2", Provider: "x", Model: "y", RequestStartedAt: time.Now().UTC()}
	urow := eventToCSVRow(unpriced)
	costIdx := indexOf(csvHeader, "cost_usd")
	if urow[costIdx] != "" {
		t.Fatalf("unpriced cost cell must be empty, got %q", urow[costIdx])
	}
}

func indexOf(ss []string, target string) int {
	for i, s := range ss {
		if s == target {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 6: Run it to verify it fails**

Run: `cd apps/backend && go test ./internal/infrastructure/http/handler/ -run TestEventToCSVRow`
Expected: FAIL — `undefined: eventToCSVRow` / `undefined: csvHeader`.

- [ ] **Step 7: Write the log + CSV handler**

`apps/backend/internal/infrastructure/http/handler/event_log_handler.go`:
```go
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"

	eventapp "router-lens/internal/application/event"
	eventdomain "router-lens/internal/domain/event"
	"router-lens/internal/shared/csv"
	"router-lens/internal/shared/datetime"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/pagination"
	"router-lens/internal/shared/response"
	"router-lens/internal/shared/validator"
)

type EventLogHandler struct {
	query *eventapp.QueryService
	v     *validator.Validator
}

func NewEventLogHandler(query *eventapp.QueryService, v *validator.Validator) *EventLogHandler {
	return &EventLogHandler{query: query, v: v}
}

// Register mounts the session-authenticated read routes.
func (h *EventLogHandler) Register(api *echo.Group, session echo.MiddlewareFunc) {
	api.GET("/events", h.list, session)
	api.GET("/events/export.csv", h.exportCSV, session)
	api.GET("/events/:id", h.get, session)
}

type eventDTO struct {
	ID                string          `json:"id"`
	ProjectID         string          `json:"project_id"`
	Provider          string          `json:"provider"`
	Model             string          `json:"model"`
	RouteSource       string          `json:"route_source"`
	Agent             string          `json:"agent"`
	InputTokens       int64           `json:"input_tokens"`
	OutputTokens      int64           `json:"output_tokens"`
	CostUSD           *string         `json:"cost_usd"` // null = unpriced
	InputPrice1M      *string         `json:"input_price_1m"`
	OutputPrice1M     *string         `json:"output_price_1m"`
	LatencyMs         *int            `json:"latency_ms"`
	StatusCode        *int            `json:"status_code"`
	IsError           bool            `json:"is_error"`
	ErrorMessage      string          `json:"error_message"`
	RequestStartedAt  string          `json:"request_started_at"`
	RequestFinishedAt *string         `json:"request_finished_at"`
	Metadata          json.RawMessage `json:"metadata,omitempty"`
}

func decimalPtrString(d *decimal.Decimal) *string {
	if d == nil {
		return nil
	}
	s := d.String()
	return &s
}

func toEventDTO(e *eventdomain.Event) eventDTO {
	return eventDTO{
		ID: e.ID, ProjectID: e.ProjectID, Provider: e.Provider, Model: e.Model,
		RouteSource: e.RouteSource, Agent: e.Agent,
		InputTokens: e.InputTokens, OutputTokens: e.OutputTokens,
		CostUSD: decimalPtrString(e.CostUSD), InputPrice1M: decimalPtrString(e.InputPrice1M),
		OutputPrice1M: decimalPtrString(e.OutputPrice1M),
		LatencyMs: e.LatencyMs, StatusCode: e.StatusCode, IsError: e.IsError, ErrorMessage: e.ErrorMessage,
		RequestStartedAt:  e.RequestStartedAt.UTC().Format(timeLayout),
		RequestFinishedAt: formatNullableTime(e.RequestFinishedAt),
		Metadata:          e.Metadata,
	}
}

func (h *EventLogHandler) list(c echo.Context) error {
	f, err := h.parseFilter(c)
	if err != nil {
		return err
	}
	events, hasMore, err := h.query.List(c.Request().Context(), f)
	if err != nil {
		return err
	}
	dtos := make([]eventDTO, 0, len(events))
	for _, e := range events {
		dtos = append(dtos, toEventDTO(e))
	}
	cursor := ""
	if hasMore && len(events) > 0 {
		last := events[len(events)-1]
		cursor = pagination.EncodeCursor(pagination.Cursor{Time: last.RequestStartedAt, ID: last.ID})
	}
	return response.Data(c, http.StatusOK, map[string]any{"items": dtos, "next_cursor": cursor})
}

func (h *EventLogHandler) get(c echo.Context) error {
	e, err := h.query.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, toEventDTO(e))
}

// parseFilter reads query params into a domain Filter: optional project/provider/
// model/is_error, a keyset cursor, and a bounded date range (default 24h, max 90d).
func (h *EventLogHandler) parseFilter(c echo.Context) (eventdomain.Filter, error) {
	cursor, err := pagination.DecodeCursor(c.QueryParam("cursor"))
	if err != nil {
		return eventdomain.Filter{}, apperrors.New(apperrors.KindValidation, "validation_failed", "invalid cursor")
	}
	rng, err := datetime.ParseRange(c.QueryParam("from"), c.QueryParam("to"), c.QueryParam("preset"), time.Now().UTC())
	if err != nil {
		return eventdomain.Filter{}, err
	}
	off := pagination.ParseOffset("1", c.QueryParam("limit")) // reuse the [1,100] limit clamp only
	return eventdomain.Filter{
		ProjectID: c.QueryParam("project_id"),
		From:      rng.From,
		To:        rng.To,
		Provider:  c.QueryParam("provider"),
		Model:     c.QueryParam("model"),
		IsError:   parseBoolPtr(c.QueryParam("is_error")),
		Cursor:    cursor,
		Limit:     off.Limit,
	}, nil
}

func parseBoolPtr(s string) *bool {
	switch s {
	case "true":
		v := true
		return &v
	case "false":
		v := false
		return &v
	default:
		return nil
	}
}

// csvHeader is the export column order (kept once — S1192).
var csvHeader = []string{
	"id", "project_id", "provider", "model", "route_source", "agent",
	"input_tokens", "output_tokens", "cost_usd", "latency_ms", "status_code",
	"is_error", "error_message", "request_started_at",
}

// eventToCSVRow maps an event to a CSV record in csvHeader order. Unpriced cost
// renders as an empty cell (never "0" — preserves the unpriced signal).
func eventToCSVRow(e *eventdomain.Event) []string {
	return []string{
		e.ID, e.ProjectID, e.Provider, e.Model, e.RouteSource, e.Agent,
		strconv.FormatInt(e.InputTokens, 10), strconv.FormatInt(e.OutputTokens, 10),
		decimalCell(e.CostUSD), intCell(e.LatencyMs), intCell(e.StatusCode),
		strconv.FormatBool(e.IsError), e.ErrorMessage,
		e.RequestStartedAt.UTC().Format(timeLayout),
	}
}

func decimalCell(d *decimal.Decimal) string {
	if d == nil {
		return ""
	}
	return d.String()
}

func intCell(i *int) string {
	if i == nil {
		return ""
	}
	return strconv.Itoa(*i)
}

// exportCSV streams a range-bounded, injection-safe CSV. The HTTP status + CSV
// header row are written LAZILY on the first row (or for zero rows after the
// stream opens cleanly) so a query-open error returns a proper error envelope
// instead of a partial 200.
func (h *EventLogHandler) exportCSV(c echo.Context) error {
	f, err := h.parseFilter(c)
	if err != nil {
		return err
	}
	f.Cursor = pagination.Cursor{} // export ignores the paging cursor
	w := csv.NewWriter(c.Response())
	headerWritten := false
	writeHeader := func() error {
		c.Response().Header().Set(echo.HeaderContentType, "text/csv; charset=utf-8")
		c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="events.csv"`)
		c.Response().WriteHeader(http.StatusOK)
		headerWritten = true
		return w.Write(csvHeader)
	}
	streamErr := h.query.Export(c.Request().Context(), f, func(e *eventdomain.Event) error {
		if !headerWritten {
			if err := writeHeader(); err != nil {
				return err
			}
		}
		return w.Write(eventToCSVRow(e))
	})
	if streamErr != nil && !headerWritten {
		return streamErr // query failed before any output -> proper error envelope
	}
	if streamErr != nil {
		return streamErr // partial stream after headers sent (rare); Echo logs it
	}
	if !headerWritten { // zero rows -> still return a valid header-only CSV
		if err := writeHeader(); err != nil {
			return err
		}
	}
	return w.Flush()
}
```

- [ ] **Step 8: Run it to verify it passes**

Run: `cd apps/backend && go test ./internal/infrastructure/http/handler/`
Expected: PASS.

- [ ] **Step 9: Wire the query service + session read routes in Fx**

In `apps/backend/cmd/server/main.go` `fx.Provide(...)` add:
```go
		eventapp.NewQueryService,    // (event.EventRepository) -> *eventapp.QueryService
		handler.NewEventLogHandler,  // (*eventapp.QueryService, *validator.Validator) -> *EventLogHandler
```
Add the registrar + invoke (session-authenticated):
```go
func registerEventLogRoutes(e *echo.Echo, h *handler.EventLogHandler, session echo.MiddlewareFunc) {
	h.Register(e.Group("/api/v1"), session)
}
```
```go
		fx.Invoke(registerEventLogRoutes),
```

- [ ] **Step 10: Add a logs integration test**

Append to `apps/backend/internal/infrastructure/postgres/event_repository_test.go` (or a new `_test.go`) a keyset paging assertion if not already covered by `TestEventRepositoryList`: insert > limit events, `List` with `Limit: 2` + a cursor from the 2nd row returns the next page. (The repo + cursor are unit-/integration-tested; this strengthens the keyset round-trip.) Reuse `setupTestDB` + `seedProject`.

- [ ] **Step 11: Full verification**

Run: `cd apps/backend && gofmt -l . && go vet ./... && go build ./... && go test ./internal/... -count=1`
Expected: gofmt prints nothing; vet/build clean; all non-integration tests PASS. Optional docker smoke (valid session cookie in `cookies.txt`):
```
curl -s 'localhost:8080/api/v1/events?preset=7d' -b cookies.txt
curl -s 'localhost:8080/api/v1/events/export.csv?preset=7d' -b cookies.txt
```
Expected: list returns `{data:{items:[…],next_cursor:"…"}}`; export streams a CSV (header + rows). Without a cookie → `401`.

- [ ] **Step 12: Commit**

```bash
git add apps/backend
git commit -m "feat: request logs + CSV export — keyset list, get, streamed export (Plan 05 task 3)"
```

---

## Plan-level Definition of Done

- `POST /api/v1/events` ingests through Bearer API-key auth: validates the payload (tokens/latency/status/timestamps/backdate/metadata-size), derives `is_error` once, looks up pricing for `(provider, model)`, stores `cost_usd` + price snapshot (or NULL when unpriced), and inserts idempotently — new → `202 {deduplicated:false}`, duplicate → `202 {deduplicated:true}`. Missing/revoked key → `401`. Body never carries `project_id`.
- `GET /api/v1/events` returns session-authenticated keyset-paginated logs (`{items, next_cursor}`, last page emits no dead cursor) with optional project/provider/model/is_error filters and a bounded date range; `GET /api/v1/events/:id` returns one (404 when absent).
- `GET /api/v1/events/export.csv` streams a range-bounded, formula-injection-safe CSV; unpriced cost renders as an empty cell; a query-open error returns a proper error envelope (no partial 200).
- Two auth boundaries stay separate (API-key for ingest, session for reads) — enforced by two distinct handlers. Cost/snapshot, keyset cursor, date-range, CSV writer, and `setupTestDB` all reuse existing shared helpers — no duplicated logic. All new i18n codes have EN + ID entries.
- `gofmt`/`go vet`/`go build` clean; unit suites green; event repo integration tests green against real Postgres.
- Three commits (one per task) on `feat/foundation`.

> **Build-order note:** Analytics endpoints (spec §16 step 9) are NOT in this plan — they are Plan 06 (overview/tokens/cost/latency/errors/providers/models, integration-tested raw SQL over `llm_events`). This plan delivers the ingest + raw-log + export surface that analytics will aggregate.
