# RouterLens Plan 06 — Analytics Endpoints

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the seven read-only, session-authenticated analytics endpoints (`overview`, `tokens`, `cost`, `latency`, `errors`, `providers`, `models`) that aggregate `llm_events` over a mandatory bounded date range, reusing the ingestion pipeline and shared kit built in Plans 01–05.

**Architecture:** Extends the existing `event` bounded context (domain/usecase/adapter — same packages as Plan 05, new files) rather than a new bounded context, matching the design spec's own grouping ("EventRepository incl. analytics aggregate queries"). Persistence-first, mirroring Plan 05's decomposition: **Task 1** delivers the complete `AnalyticsRepository` port + Postgres implementation (raw SQL, no ORM, no rollups); **Task 2** adds the `AnalyticsService` use case that orchestrates repository calls into the seven response shapes and computes derived fields (error rate, most-used provider/model) in Go, not SQL; **Task 3** adds the HTTP handler, DTOs, and Fx wiring. A key design decision (stated once, applied throughout): the seven endpoints are served by **five repository queries, not seven** — `tokens`, `cost`, `latency`, and `errors` all bucket the same underlying per-interval aggregate (`Series`), so one SQL statement backs four endpoints instead of four near-identical ones (anti-duplication, and one less query to keep correct under future schema changes).

**Tech Stack:** Go 1.26, Echo v4, Uber Fx, pgx/v5 (pgxpool), shopspring/decimal. Reuses `internal/shared/datetime.ParseRange` (Plan 02, unmodified), `internal/shared/response` envelope, `internal/domain/event.EventRepository`'s existing `llm_events` schema knowledge (Plan 05, unmodified — this plan does not touch `event_repository.go`), and the session middleware (Plan 03/04, unmodified).

## Global Constraints

- **Layering (HARD):** `internal/domain/event/` (this plan adds one file to the existing package) imports only stdlib + `shopspring/decimal`. Never echo, pgx, or adapter packages. The `AnalyticsRepository` port lives in `domain/event/analytics.go`; the pgx implementation lives in `internal/adapter/postgres/analytics_repository.go`.
- **Mandatory bounded date range (decision 6):** every analytics endpoint requires `from`/`to` (or a `preset`), enforced by reusing `datetime.ParseRange` exactly as `EventLogHandler` already does — default last 24h, hard max 90 days. No analytics query ever runs unbounded.
- **`date_trunc` interval is an allow-list, never string-concatenated from raw input (gosec, HARD):** the bucket interval (`hour`/`day`/`week`) is validated **twice**, independently, at two different layers — the HTTP handler validates it for a clean 400 response, and the Postgres adapter validates it again (defense in depth) before interpolating the literal into the `date_trunc(...)` SQL fragment. Neither layer trusts the other. This is intentional double-validation, not duplication-for-duplication's-sake.
- **`cost_usd` stays nullable end-to-end (decision 1/10/11):** `SUM(cost_usd)` over a range where every event is unpriced returns SQL `NULL` — surfaced as `nil *decimal.Decimal`, rendered as JSON `null`, never `0`. The `Overview` response additionally carries `unpriced_count` so the UI can show "N of M events unpriced" instead of silently under-reporting cost.
- **No pre-aggregation, no rollups (decision 6, ponytail):** every endpoint is a raw `SELECT ... GROUP BY ...` over `llm_events`, riding the existing `(project_id, request_started_at DESC, id DESC)` index from Plan 01's migration. Do not add a rollup table, a materialized view, or a caching layer in this plan — only add one if a real query proves too slow against real data (§8 — never preemptively).
- **Optional project scope:** every endpoint accepts an optional `project_id` query param. Omitted = aggregate across all projects (the single-admin model sees everything by design, decision 4). `Overview`'s `top_projects` field is only informative when `project_id` is omitted; when a specific project is requested it trivially returns that one project — this is fine, not special-cased.
- **Anti-duplication:** reuse `datetime.ParseRange`, `response.Data`, the existing `llm_events` schema (no new migration in this plan — it already has every column needed). The bucket-interval allow-list check is written once in the Postgres adapter and reused by every series-shaped endpoint.

### Sonar guardrails — write compliant from the first commit

```
Go:
- go:S107 — <=7 params (<=5 preferred). AnalyticsFilter is a struct, not a long param list.
- go:S3776 — cognitive complexity <=15 -> extract helpers (e.g. bucketExpr, the per-endpoint DTO projectors).
- go:S1192 — const for any string literal duplicated 3+ times (SQL fragments, error codes, interval values).
- errcheck — handle every returned error; never `_ = fallible()`. Wrap with %w; sentinel + errors.Is/As.
- gosec — the ONLY non-parameterized SQL fragment in this plan is the allow-listed date_trunc interval;
  every other value is a `$n` placeholder. Never interpolate project_id, from, to, or any user string directly.

When fixing one instance, check sibling files for the same anti-pattern and fix-forward.
Review the diff against this list BEFORE marking compliant.
```

### Skill brief for implementer subagents (every task)

> Invoke `golang-expert` first — it is a hub skill and auto-chains the full Go discipline family (go-patterns / go-review / go-test / go-error-handling) + `senior-backend` + `senior-security` + `algorithmic-complexity`; follow its Auto-chain section. Apply `ponytail` (YAGNI): five repository queries serve seven endpoints by design — do not add a sixth or seventh query "for symmetry". Honor the Global Constraints + Sonar block.

### Algorithmic complexity (§8, Bahasa Indonesia)

- **Semua query analytics:** satu `SELECT ... WHERE request_started_at BETWEEN $1 AND $2 [AND project_id = $3] GROUP BY ...`. Kalau `project_id` diisi, query naik index komposit `(project_id, request_started_at DESC, id DESC)` langsung → seek + range scan, **O(log n + k)** dengan `k` = baris dalam rentang tanggal untuk project itu. Kalau `project_id` kosong (lintas-project), Postgres tetap bisa pakai index untuk kondisi `request_started_at` saja (index masih berguna sebagai range scan meski `project_id` bukan leading match tunggal — Postgres index scan tanpa prefix match jadi kurang optimal tapi tetap jauh lebih murah dari seq scan penuh tabel), **O(k log n)** kasus terburuk, `k` tetap dibatasi rentang tanggal (maks 90 hari, di-enforce `datetime.ParseRange`). Tidak ada N+1 — setiap endpoint memanggil tepat satu atau dua query independen (`Overview` memanggil 4 query kecil: totals + providers + models + top-projects, semuanya `GROUP BY` dengan cardinality kecil, bukan loop per baris).
- **Agregasi (`GROUP BY provider`, `GROUP BY model`, `GROUP BY bucket`):** hash aggregate Postgres, **O(k)** waktu (satu pass atas `k` baris dalam rentang), **O(distinct groups)** memori hasil — jumlah provider/model/bucket jauh lebih kecil dari `k` (puluhan, bukan jutaan). `TopProjects` pakai `LIMIT 5` di sisi SQL, bukan di-slice di Go — DB yang buang baris berlebih, bukan Go yang fetch semua lalu potong.
- **`Series` melayani 4 endpoint dari 1 query:** menghindari 4x scan tabel yang sama untuk data yang secara struktural sama (per-bucket aggregate) — ini optimisasi nyata, bukan preemptive, karena keempat endpoint (`tokens`/`cost`/`latency`/`errors`) butuh `GROUP BY bucket` yang identik.
- Tidak ada query-in-loop, tidak ada `.find()`-in-loop di plan ini — semua turunan (most-used provider, error rate) dihitung di Go dari hasil query yang sudah kecil (≤ jumlah provider/model berbeda, ≤ 2160 bucket untuk rentang 90 hari granularitas jam), bukan dengan query tambahan per item.

### TDD verdicts (per §16)

- **Task 1 (repository):** `bucketExpr` (interval allow-list → SQL literal) `TDD: yes` (pure, security-relevant, easy to get wrong — Step 1 is the failing test). The five `AnalyticsRepository` methods (`Overview`/`Series`/`Providers`/`Models`/`TopProjects`) `TDD: no` — integration tests against a real Postgres after implementation (same convention as Plan 05 Task 1's `EventRepository`).
- **Task 2 (usecase):** `AnalyticsService.Overview`'s derived-field assembly (most-used provider/model, most-expensive model, error rate, division-by-zero guard when `TotalRequests == 0`) `TDD: yes` (fake repo, clear input→output, the exact kind of arithmetic that silently breaks). The four series-projector methods (`TokensSeries`/`CostSeries`/`LatencySeries`/`ErrorSeries`, each just re-shaping `Series`'s already-tested output) `TDD: yes` as well — they are pure, one-line-per-field projections and cheap to pin down with a table-driven test.
- **Task 3 (handler + wiring):** the seven handler methods + `parseAnalyticsFilter` + Fx wiring `TDD: no` — thin HTTP glue over the already-tested use case (same convention as Plan 05 Task 3). Verify by build + the optional docker smoke test.

---

## Task 1: Analytics domain types + complete repository (Postgres)

Delivers the `AnalyticsFilter`/result types and the full `AnalyticsRepository` port + a pgx implementation covering all five queries — so `var _ event.AnalyticsRepository = (*AnalyticsRepository)(nil)` compiles now and Task 2 only consumes it. No HTTP, no use case, no Fx wiring in this task.

**Files:**
- Create: `apps/backend/internal/domain/event/analytics.go`
- Create: `apps/backend/internal/adapter/postgres/analytics_repository.go`
- Create: `apps/backend/internal/adapter/postgres/analytics_bucket_test.go`
- Create: `apps/backend/internal/adapter/postgres/analytics_repository_test.go`

**Interfaces:**
- Consumes: `github.com/shopspring/decimal`, `github.com/jackc/pgx/v5` / `pgxpool`, the existing `llm_events` table (Plan 01 migration 006, unmodified) and `projects` table (Plan 01 migration 002).
- Produces (Task 2 relies on these): `event.{AnalyticsFilter, OverviewTotals, SeriesPoint, ProviderStat, ModelStat, ProjectStat, AnalyticsRepository}`, `postgres.NewAnalyticsRepository`.

- [ ] **Step 1: Write the failing test for the interval allow-list**

`apps/backend/internal/adapter/postgres/analytics_bucket_test.go`:
```go
package postgres

import "testing"

func TestBucketExpr(t *testing.T) {
	t.Run("accepts hour, day, week", func(t *testing.T) {
		for _, interval := range []string{"hour", "day", "week"} {
			expr, err := bucketExpr(interval)
			if err != nil {
				t.Fatalf("interval %q: unexpected error %v", interval, err)
			}
			want := "date_trunc('" + interval + "', request_started_at)"
			if expr != want {
				t.Fatalf("interval %q: expr = %q, want %q", interval, expr, want)
			}
		}
	})
	t.Run("rejects anything not on the allow-list", func(t *testing.T) {
		for _, bad := range []string{"", "month", "year", "day'; DROP TABLE llm_events; --"} {
			if _, err := bucketExpr(bad); err == nil {
				t.Fatalf("interval %q: want error, got nil", bad)
			}
		}
	})
}
```

Run: `cd apps/backend && go test ./internal/adapter/postgres/ -run TestBucketExpr`
Expected: FAIL — `undefined: bucketExpr`.

- [ ] **Step 2: Write the domain types + repository port**

`apps/backend/internal/domain/event/analytics.go`:
```go
// Package event (analytics.go): the read-side aggregate types and the
// AnalyticsRepository port. Same domain-purity rule as entity.go — stdlib +
// shopspring/decimal only, no echo/pgx/adapter imports.
package event

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// AnalyticsFilter bounds every analytics query. Interval is only meaningful
// for Series ("" is fine for Overview/Providers/Models/TopProjects, which are
// not bucketed by time). ProjectID == "" means "all projects".
type AnalyticsFilter struct {
	ProjectID string
	From      time.Time
	To        time.Time
	Interval  string // "hour" | "day" | "week", validated by the caller
}

// OverviewTotals is the raw single-row aggregate for a date range. It carries
// no derived fields (error rate, most-used provider) — those are computed in
// the usecase layer from this plus Providers/Models/TopProjects, keeping SQL
// dumb and the business interpretation testable in Go.
type OverviewTotals struct {
	TotalRequests     int64
	TotalInputTokens  int64
	TotalOutputTokens int64
	TotalCostUSD      *decimal.Decimal // nil when every event in range is unpriced (or zero rows)
	UnpricedCount     int64
	AvgLatencyMs      *float64 // nil when no event in range carries a latency_ms
	P95LatencyMs      *float64
	ErrorCount        int64
}

// SeriesPoint is one time-bucketed row. It backs FOUR endpoints (tokens,
// cost, latency, errors) from ONE query — each endpoint's handler projects
// out only the fields it needs (see Task 2/3).
type SeriesPoint struct {
	Bucket       time.Time
	InputTokens  int64
	OutputTokens int64
	CostUSD      *decimal.Decimal
	RequestCount int64
	ErrorCount   int64
	AvgLatencyMs *float64
	P95LatencyMs *float64
}

// ProviderStat is one provider's distribution over the range.
type ProviderStat struct {
	Provider     string
	RequestCount int64
	InputTokens  int64
	OutputTokens int64
	CostUSD      *decimal.Decimal
}

// ModelStat is one (provider, model) pair's distribution over the range.
type ModelStat struct {
	Provider     string
	Model        string
	RequestCount int64
	InputTokens  int64
	OutputTokens int64
	CostUSD      *decimal.Decimal
}

// ProjectStat is one project's request count over the range, used by
// Overview's "top projects by usage".
type ProjectStat struct {
	ProjectID    string
	ProjectName  string
	RequestCount int64
}

// AnalyticsRepository is the read-only port over llm_events aggregates.
// Every method is bounded by AnalyticsFilter.From/To (enforced by the caller
// via datetime.ParseRange) — none of these ever scans the whole table.
type AnalyticsRepository interface {
	Overview(ctx context.Context, f AnalyticsFilter) (*OverviewTotals, error)
	// Series buckets by f.Interval ("hour"/"day"/"week") — backs tokens/cost/latency/errors.
	Series(ctx context.Context, f AnalyticsFilter) ([]SeriesPoint, error)
	Providers(ctx context.Context, f AnalyticsFilter) ([]ProviderStat, error)
	Models(ctx context.Context, f AnalyticsFilter) ([]ModelStat, error)
	// TopProjects ignores f.ProjectID (it exists to compare projects) and returns
	// at most limit rows, ordered by request count descending.
	TopProjects(ctx context.Context, f AnalyticsFilter, limit int) ([]ProjectStat, error)
}
```

- [ ] **Step 3: Write the bucket-interval allow-list + repository skeleton**

`apps/backend/internal/adapter/postgres/analytics_repository.go` (part 1 of 2 — the rest is appended in Step 5):
```go
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"router-lens/internal/domain/event"
)

type AnalyticsRepository struct{ pool *pgxpool.Pool }

func NewAnalyticsRepository(pool *pgxpool.Pool) *AnalyticsRepository {
	return &AnalyticsRepository{pool: pool}
}

var _ event.AnalyticsRepository = (*AnalyticsRepository)(nil)

// bucketExpr validates interval against a fixed allow-list and returns the
// date_trunc SQL fragment. This is the ONLY place a non-parameterized value
// is interpolated into a query in this file — interval can only ever be one
// of the three literal strings below, never raw user input (gosec).
func bucketExpr(interval string) (string, error) {
	switch interval {
	case "hour", "day", "week":
		return fmt.Sprintf("date_trunc('%s', request_started_at)", interval), nil
	default:
		return "", fmt.Errorf("postgres: invalid analytics interval %q", interval)
	}
}

// analyticsWhere builds the mandatory bounded-range WHERE clause + optional
// project scope, shared by every method below. Returns the clause and its
// positional args; the caller appends any further $n placeholders after it.
func analyticsWhere(f event.AnalyticsFilter) (string, []any) {
	if f.ProjectID == "" {
		return "WHERE request_started_at BETWEEN $1 AND $2", []any{f.From, f.To}
	}
	return "WHERE request_started_at BETWEEN $1 AND $2 AND project_id = $3", []any{f.From, f.To, f.ProjectID}
}
```

- [ ] **Step 4: Run it to verify the allow-list test passes**

Run: `cd apps/backend && go test ./internal/adapter/postgres/ -run TestBucketExpr`
Expected: PASS.

- [ ] **Step 5: Append the five query methods**

Append to `apps/backend/internal/adapter/postgres/analytics_repository.go`:
```go
func (r *AnalyticsRepository) Overview(ctx context.Context, f event.AnalyticsFilter) (*event.OverviewTotals, error) {
	where, args := analyticsWhere(f)
	q := `SELECT
			COUNT(*),
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			SUM(cost_usd),
			COUNT(*) FILTER (WHERE cost_usd IS NULL),
			AVG(latency_ms)::float8,
			percentile_cont(0.95) WITHIN GROUP (ORDER BY latency_ms)::float8,
			COUNT(*) FILTER (WHERE is_error)
		FROM llm_events ` + where
	var t event.OverviewTotals
	err := r.pool.QueryRow(ctx, q, args...).Scan(
		&t.TotalRequests, &t.TotalInputTokens, &t.TotalOutputTokens, &t.TotalCostUSD,
		&t.UnpricedCount, &t.AvgLatencyMs, &t.P95LatencyMs, &t.ErrorCount,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *AnalyticsRepository) Series(ctx context.Context, f event.AnalyticsFilter) ([]event.SeriesPoint, error) {
	bucket, err := bucketExpr(f.Interval)
	if err != nil {
		return nil, err
	}
	where, args := analyticsWhere(f)
	q := `SELECT ` + bucket + ` AS bucket,
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			SUM(cost_usd),
			COUNT(*),
			COUNT(*) FILTER (WHERE is_error),
			AVG(latency_ms)::float8,
			percentile_cont(0.95) WITHIN GROUP (ORDER BY latency_ms)::float8
		FROM llm_events ` + where + `
		GROUP BY bucket ORDER BY bucket`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]event.SeriesPoint, 0)
	for rows.Next() {
		var p event.SeriesPoint
		if err := rows.Scan(&p.Bucket, &p.InputTokens, &p.OutputTokens, &p.CostUSD,
			&p.RequestCount, &p.ErrorCount, &p.AvgLatencyMs, &p.P95LatencyMs); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *AnalyticsRepository) Providers(ctx context.Context, f event.AnalyticsFilter) ([]event.ProviderStat, error) {
	where, args := analyticsWhere(f)
	q := `SELECT provider, COUNT(*), COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0), SUM(cost_usd)
		FROM llm_events ` + where + `
		GROUP BY provider ORDER BY 2 DESC`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]event.ProviderStat, 0)
	for rows.Next() {
		var s event.ProviderStat
		if err := rows.Scan(&s.Provider, &s.RequestCount, &s.InputTokens, &s.OutputTokens, &s.CostUSD); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *AnalyticsRepository) Models(ctx context.Context, f event.AnalyticsFilter) ([]event.ModelStat, error) {
	where, args := analyticsWhere(f)
	q := `SELECT provider, model, COUNT(*), COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0), SUM(cost_usd)
		FROM llm_events ` + where + `
		GROUP BY provider, model ORDER BY 3 DESC`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]event.ModelStat, 0)
	for rows.Next() {
		var s event.ModelStat
		if err := rows.Scan(&s.Provider, &s.Model, &s.RequestCount, &s.InputTokens, &s.OutputTokens, &s.CostUSD); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *AnalyticsRepository) TopProjects(ctx context.Context, f event.AnalyticsFilter, limit int) ([]event.ProjectStat, error) {
	q := `SELECT e.project_id, p.name, COUNT(*)
		FROM llm_events e
		JOIN projects p ON p.id = e.project_id
		WHERE e.request_started_at BETWEEN $1 AND $2
		GROUP BY e.project_id, p.name
		ORDER BY 3 DESC
		LIMIT $3`
	rows, err := r.pool.Query(ctx, q, f.From, f.To, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]event.ProjectStat, 0, limit)
	for rows.Next() {
		var s event.ProjectStat
		if err := rows.Scan(&s.ProjectID, &s.ProjectName, &s.RequestCount); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
```
NOTE: `TopProjects` deliberately ignores `f.ProjectID` (see the interface doc comment) — it always compares across all projects, which is the only way "top projects by usage" means anything.

- [ ] **Step 6: Write the repository integration test**

`apps/backend/internal/adapter/postgres/analytics_repository_test.go`. Reuse `setupTestDB` (from `project_repository_test.go`) and `seedProject`/`NewEventRepository` (from `event_repository_test.go`, same package) — do not redefine either.
```go
package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"router-lens/internal/domain/event"
	"router-lens/internal/domain/project"
)

// seedEvents inserts n events for proj, evenly spaced one hour apart starting
// at base, alternating priced/unpriced and one error, so every aggregate
// (cost NULL-handling, error count, latency) has non-trivial data to check.
func seedEvents(t *testing.T, ctx context.Context, pool *pgxpool.Pool, proj *project.Project, base time.Time, n int) {
	t.Helper()
	repo := NewEventRepository(pool)
	for i := 0; i < n; i++ {
		lat := 100 + i*10
		status := 200
		e := &event.Event{
			ProjectID: proj.ID, Provider: "anthropic", Model: "claude",
			InputTokens: 1000, OutputTokens: 500, LatencyMs: &lat, StatusCode: &status,
			RequestStartedAt: base.Add(time.Duration(i) * time.Hour), ReceivedAt: time.Now().UTC(),
		}
		if i%2 == 0 { // half the events are priced
			cost := decimal.NewFromFloat(0.01)
			e.CostUSD, e.InputPrice1M, e.OutputPrice1M = &cost, &cost, &cost
		}
		if i == 0 { // exactly one error
			bad := 500
			e.StatusCode = &bad
			e.IsError = true
		}
		if _, err := repo.Insert(ctx, e); err != nil {
			t.Fatalf("seed event %d: %v", i, err)
		}
	}
}

func TestAnalyticsRepository(t *testing.T) {
	ctx, pool := setupTestDB(t)
	proj := seedProject(t, ctx, pool)
	base := time.Now().UTC().Add(-24 * time.Hour)
	seedEvents(t, ctx, pool, proj, base, 4)
	repo := NewAnalyticsRepository(pool)
	rng := event.AnalyticsFilter{ProjectID: proj.ID, From: base.Add(-time.Minute), To: base.Add(24 * time.Hour)}

	t.Run("Overview totals + unpriced count + one error", func(t *testing.T) {
		got, err := repo.Overview(ctx, rng)
		if err != nil {
			t.Fatalf("overview: %v", err)
		}
		if got.TotalRequests != 4 || got.ErrorCount != 1 || got.UnpricedCount != 2 {
			t.Fatalf("got %+v", got)
		}
		if got.TotalCostUSD == nil {
			t.Fatal("expected a non-nil total cost (2 of 4 events priced)")
		}
	})

	t.Run("Series buckets by hour and sums each bucket independently", func(t *testing.T) {
		f := rng
		f.Interval = "hour"
		points, err := repo.Series(ctx, f)
		if err != nil {
			t.Fatalf("series: %v", err)
		}
		if len(points) != 4 {
			t.Fatalf("expected 4 hourly buckets, got %d", len(points))
		}
	})

	t.Run("Providers groups by provider", func(t *testing.T) {
		stats, err := repo.Providers(ctx, rng)
		if err != nil || len(stats) != 1 || stats[0].Provider != "anthropic" || stats[0].RequestCount != 4 {
			t.Fatalf("stats=%+v err=%v", stats, err)
		}
	})

	t.Run("Models groups by provider+model", func(t *testing.T) {
		stats, err := repo.Models(ctx, rng)
		if err != nil || len(stats) != 1 || stats[0].Model != "claude" {
			t.Fatalf("stats=%+v err=%v", stats, err)
		}
	})

	t.Run("TopProjects ranks by request count across all projects", func(t *testing.T) {
		stats, err := repo.TopProjects(ctx, event.AnalyticsFilter{From: rng.From, To: rng.To}, 5)
		if err != nil || len(stats) == 0 || stats[0].ProjectID != proj.ID {
			t.Fatalf("stats=%+v err=%v", stats, err)
		}
	})
}
```

- [ ] **Step 7: Verify + report**

Run: `cd apps/backend && gofmt -l . && go vet ./... && go build ./... && go test ./internal/... -count=1`
Expected: gofmt prints nothing; vet/build clean; `TestBucketExpr` passes; `TestAnalyticsRepository` skips cleanly without `TEST_DATABASE_URL` (run it against an isolated Postgres if available — see Plan 05's task 1 report for exactly how to do this locally: `docker compose up -d postgres` may collide with a native Postgres already on host port 5432, in which case run a one-off container on a different host port, e.g. `docker run -d --rm -e POSTGRES_USER=routerlens -e POSTGRES_PASSWORD=routerlens -e POSTGRES_DB=routerlens -p 55432:5432 postgres:17` and set `TEST_DATABASE_URL=postgres://routerlens:routerlens@localhost:55432/routerlens?sslmode=disable`).

Do NOT git commit — this project commits once per plan, at the end, after the controller shows the full diff to the user (see `.superpowers/sdd/progress.md`).

---

## Task 2: Analytics use case (orchestration + derived fields)

Adds the `AnalyticsService` on top of Task 1's repository: assembles `Overview`'s derived fields (most-used provider/model, most-expensive model, error rate) from the five raw queries, and projects `Series` into the four shapes `tokens`/`cost`/`latency`/`errors` need. No HTTP, no SQL in this task.

**Files:**
- Create: `apps/backend/internal/usecase/event/analytics.go`
- Create: `apps/backend/internal/usecase/event/analytics_test.go`

**Interfaces:**
- Consumes: `eventdomain.{AnalyticsFilter, OverviewTotals, SeriesPoint, ProviderStat, ModelStat, ProjectStat, AnalyticsRepository}` (Task 1).
- Produces (Task 3 relies on these): `event.NewAnalyticsService`, `(*AnalyticsService).{Overview, TokensSeries, CostSeries, LatencySeries, ErrorSeries, Providers, Models}`, and the result types `event.{OverviewResult, TokenPoint, CostPoint, LatencyPoint, ErrorPoint}`.

- [ ] **Step 1: Write the failing test for Overview's derived fields**

`apps/backend/internal/usecase/event/analytics_test.go`:
```go
package event

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	eventdomain "router-lens/internal/domain/event"
)

type fakeAnalyticsRepo struct {
	overview    *eventdomain.OverviewTotals
	series      []eventdomain.SeriesPoint
	providers   []eventdomain.ProviderStat
	models      []eventdomain.ModelStat
	topProjects []eventdomain.ProjectStat
}

func (f *fakeAnalyticsRepo) Overview(context.Context, eventdomain.AnalyticsFilter) (*eventdomain.OverviewTotals, error) {
	return f.overview, nil
}
func (f *fakeAnalyticsRepo) Series(context.Context, eventdomain.AnalyticsFilter) ([]eventdomain.SeriesPoint, error) {
	return f.series, nil
}
func (f *fakeAnalyticsRepo) Providers(context.Context, eventdomain.AnalyticsFilter) ([]eventdomain.ProviderStat, error) {
	return f.providers, nil
}
func (f *fakeAnalyticsRepo) Models(context.Context, eventdomain.AnalyticsFilter) ([]eventdomain.ModelStat, error) {
	return f.models, nil
}
func (f *fakeAnalyticsRepo) TopProjects(context.Context, eventdomain.AnalyticsFilter, int) ([]eventdomain.ProjectStat, error) {
	return f.topProjects, nil
}

func TestAnalyticsServiceOverview(t *testing.T) {
	t.Run("assembles most-used provider/model, most-expensive model, error rate", func(t *testing.T) {
		cheapCost := decimal.NewFromFloat(1)
		pricyCost := decimal.NewFromFloat(99)
		repo := &fakeAnalyticsRepo{
			overview: &eventdomain.OverviewTotals{TotalRequests: 10, ErrorCount: 2},
			providers: []eventdomain.ProviderStat{
				{Provider: "anthropic", RequestCount: 7},
				{Provider: "openai", RequestCount: 3},
			},
			models: []eventdomain.ModelStat{
				{Provider: "anthropic", Model: "claude", RequestCount: 7, CostUSD: &cheapCost},
				{Provider: "openai", Model: "gpt", RequestCount: 3, CostUSD: &pricyCost},
			},
		}
		got, err := NewAnalyticsService(repo).Overview(context.Background(), eventdomain.AnalyticsFilter{})
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if got.MostUsedProvider != "anthropic" || got.MostUsedModel != "claude" {
			t.Fatalf("most-used: provider=%q model=%q", got.MostUsedProvider, got.MostUsedModel)
		}
		if got.MostExpensiveModel != "gpt" {
			t.Fatalf("most-expensive model = %q, want gpt", got.MostExpensiveModel)
		}
		if got.ErrorRate != 0.2 {
			t.Fatalf("error rate = %v, want 0.2", got.ErrorRate)
		}
	})
	t.Run("zero requests -> error rate 0, no panic, empty most-used fields", func(t *testing.T) {
		repo := &fakeAnalyticsRepo{overview: &eventdomain.OverviewTotals{}}
		got, err := NewAnalyticsService(repo).Overview(context.Background(), eventdomain.AnalyticsFilter{})
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if got.ErrorRate != 0 || got.MostUsedProvider != "" || got.MostExpensiveModel != "" {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("no priced model -> most-expensive model stays empty, not the first row", func(t *testing.T) {
		repo := &fakeAnalyticsRepo{
			overview: &eventdomain.OverviewTotals{TotalRequests: 1},
			models:   []eventdomain.ModelStat{{Provider: "p", Model: "unpriced-model", RequestCount: 1, CostUSD: nil}},
		}
		got, err := NewAnalyticsService(repo).Overview(context.Background(), eventdomain.AnalyticsFilter{})
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if got.MostExpensiveModel != "" {
			t.Fatalf("most-expensive model = %q, want empty (nothing priced)", got.MostExpensiveModel)
		}
	})
}

func TestAnalyticsServiceSeriesProjections(t *testing.T) {
	lat := 42.0
	series := []eventdomain.SeriesPoint{
		{InputTokens: 10, OutputTokens: 20, RequestCount: 5, ErrorCount: 1, AvgLatencyMs: &lat, P95LatencyMs: &lat},
	}
	repo := &fakeAnalyticsRepo{series: series}
	svc := NewAnalyticsService(repo)

	t.Run("TokensSeries projects input/output tokens only", func(t *testing.T) {
		got, err := svc.TokensSeries(context.Background(), eventdomain.AnalyticsFilter{})
		if err != nil || len(got) != 1 || got[0].InputTokens != 10 || got[0].OutputTokens != 20 {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})
	t.Run("ErrorSeries computes rate per bucket, zero requests -> rate 0", func(t *testing.T) {
		zero := []eventdomain.SeriesPoint{{RequestCount: 0, ErrorCount: 0}}
		got, err := NewAnalyticsService(&fakeAnalyticsRepo{series: zero}).ErrorSeries(context.Background(), eventdomain.AnalyticsFilter{})
		if err != nil || len(got) != 1 || got[0].ErrorRate != 0 {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})
}
```

Run: `cd apps/backend && go test ./internal/usecase/event/ -run TestAnalyticsService`
Expected: FAIL — `undefined: NewAnalyticsService`.

- [ ] **Step 2: Write the analytics use case**

`apps/backend/internal/usecase/event/analytics.go`:
```go
// Package event (analytics.go): the AnalyticsService use case. Orchestrates
// AnalyticsRepository calls into the seven endpoint shapes and computes every
// derived number (rates, "most used") here in Go — the repository only ever
// returns raw grouped totals, never a business interpretation.
package event

import (
	"context"

	eventdomain "router-lens/internal/domain/event"
	"github.com/shopspring/decimal"
)

const topProjectsLimit = 5

type AnalyticsService struct{ repo eventdomain.AnalyticsRepository }

func NewAnalyticsService(repo eventdomain.AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

// OverviewResult is Overview's fully-assembled response shape.
type OverviewResult struct {
	Totals             *eventdomain.OverviewTotals
	ErrorRate          float64
	MostUsedProvider   string
	MostUsedModel      string
	MostExpensiveModel string
	TopProjects        []eventdomain.ProjectStat
}

func (s *AnalyticsService) Overview(ctx context.Context, f eventdomain.AnalyticsFilter) (*OverviewResult, error) {
	totals, err := s.repo.Overview(ctx, f)
	if err != nil {
		return nil, err
	}
	providers, err := s.repo.Providers(ctx, f)
	if err != nil {
		return nil, err
	}
	models, err := s.repo.Models(ctx, f)
	if err != nil {
		return nil, err
	}
	top, err := s.repo.TopProjects(ctx, f, topProjectsLimit)
	if err != nil {
		return nil, err
	}
	return &OverviewResult{
		Totals:             totals,
		ErrorRate:          safeRate(totals.ErrorCount, totals.TotalRequests),
		MostUsedProvider:   mostUsedProvider(providers),
		MostUsedModel:      mostUsedModel(models),
		MostExpensiveModel: mostExpensiveModel(models),
		TopProjects:        top,
	}, nil
}

// safeRate divides part/whole, returning 0 (not NaN/Inf) when whole is zero.
func safeRate(part, whole int64) float64 {
	if whole == 0 {
		return 0
	}
	return float64(part) / float64(whole)
}

// mostUsedProvider returns the provider with the highest request count, or ""
// when there are none. Providers is already the full grouped result — small
// (a handful of distinct providers), so a linear scan is the right tool.
func mostUsedProvider(stats []eventdomain.ProviderStat) string {
	var best eventdomain.ProviderStat
	for _, s := range stats {
		if s.RequestCount > best.RequestCount {
			best = s
		}
	}
	return best.Provider
}

func mostUsedModel(stats []eventdomain.ModelStat) string {
	var best eventdomain.ModelStat
	for _, s := range stats {
		if s.RequestCount > best.RequestCount {
			best = s
		}
	}
	return best.Model
}

// mostExpensiveModel returns the model with the highest total cost, skipping
// unpriced models entirely (nil CostUSD) so an unpriced model never wins by
// virtue of being compared against a zero value.
func mostExpensiveModel(stats []eventdomain.ModelStat) string {
	var best eventdomain.ModelStat
	var bestCost decimal.Decimal
	found := false
	for _, s := range stats {
		if s.CostUSD == nil {
			continue
		}
		if !found || s.CostUSD.GreaterThan(bestCost) {
			best, bestCost, found = s, *s.CostUSD, true
		}
	}
	return best.Model
}

// TokenPoint, CostPoint, LatencyPoint, and ErrorPoint are the four
// projections of eventdomain.SeriesPoint that the four series endpoints need.
type TokenPoint struct {
	Bucket       string
	InputTokens  int64
	OutputTokens int64
}
type CostPoint struct {
	Bucket  string
	CostUSD *decimal.Decimal
}
type LatencyPoint struct {
	Bucket       string
	AvgLatencyMs *float64
	P95LatencyMs *float64
}
type ErrorPoint struct {
	Bucket       string
	RequestCount int64
	ErrorCount   int64
	ErrorRate    float64
}

func (s *AnalyticsService) series(ctx context.Context, f eventdomain.AnalyticsFilter) ([]eventdomain.SeriesPoint, error) {
	return s.repo.Series(ctx, f)
}

func (s *AnalyticsService) TokensSeries(ctx context.Context, f eventdomain.AnalyticsFilter) ([]TokenPoint, error) {
	points, err := s.series(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]TokenPoint, 0, len(points))
	for _, p := range points {
		out = append(out, TokenPoint{Bucket: p.Bucket.UTC().Format(bucketLayout), InputTokens: p.InputTokens, OutputTokens: p.OutputTokens})
	}
	return out, nil
}

func (s *AnalyticsService) CostSeries(ctx context.Context, f eventdomain.AnalyticsFilter) ([]CostPoint, error) {
	points, err := s.series(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]CostPoint, 0, len(points))
	for _, p := range points {
		out = append(out, CostPoint{Bucket: p.Bucket.UTC().Format(bucketLayout), CostUSD: p.CostUSD})
	}
	return out, nil
}

func (s *AnalyticsService) LatencySeries(ctx context.Context, f eventdomain.AnalyticsFilter) ([]LatencyPoint, error) {
	points, err := s.series(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]LatencyPoint, 0, len(points))
	for _, p := range points {
		out = append(out, LatencyPoint{Bucket: p.Bucket.UTC().Format(bucketLayout), AvgLatencyMs: p.AvgLatencyMs, P95LatencyMs: p.P95LatencyMs})
	}
	return out, nil
}

func (s *AnalyticsService) ErrorSeries(ctx context.Context, f eventdomain.AnalyticsFilter) ([]ErrorPoint, error) {
	points, err := s.series(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]ErrorPoint, 0, len(points))
	for _, p := range points {
		out = append(out, ErrorPoint{
			Bucket: p.Bucket.UTC().Format(bucketLayout), RequestCount: p.RequestCount,
			ErrorCount: p.ErrorCount, ErrorRate: safeRate(p.ErrorCount, p.RequestCount),
		})
	}
	return out, nil
}

func (s *AnalyticsService) Providers(ctx context.Context, f eventdomain.AnalyticsFilter) ([]eventdomain.ProviderStat, error) {
	return s.repo.Providers(ctx, f)
}

func (s *AnalyticsService) Models(ctx context.Context, f eventdomain.AnalyticsFilter) ([]eventdomain.ModelStat, error) {
	return s.repo.Models(ctx, f)
}

// bucketLayout renders a series bucket timestamp on the wire. RFC3339 like
// every other timestamp in this API (dto/format.go's timeLayout, same value).
const bucketLayout = "2006-01-02T15:04:05Z07:00"
```
NOTE: `bucketLayout`'s literal value is `time.RFC3339`'s format string,
duplicated here (not imported from `dto`, since `usecase/event` must not
import the adapter-layer `dto` package — that would invert the dependency
direction). This is the one accepted literal duplication for the same reason
`domain/event/validation.go` mirrors its i18n codes by value.

- [ ] **Step 3: Run it to verify it passes**

Run: `cd apps/backend && go test ./internal/usecase/event/`
Expected: PASS (all `TestAnalyticsService*` subtests, plus everything from Plan 05's `TestIngest`/`TestGetNotFound`/`TestListProbesOneExtraForHasMore` in the same package, untouched).

- [ ] **Step 4: Verify + report**

Run: `cd apps/backend && gofmt -l . && go vet ./... && go build ./... && go test ./internal/... -count=1`
Expected: gofmt clean; vet/build clean; all non-integration tests PASS.

Do NOT git commit (see Task 1's note).

---

## Task 3: HTTP handler, DTOs, and Fx wiring

Adds the session-authenticated handler for all seven routes, the response DTOs, and wires everything into the existing `eventModule` in `bootstrap.go`.

**Files:**
- Create: `apps/backend/internal/adapter/http/dto/analytics.go`
- Create: `apps/backend/internal/adapter/http/handler/analytics_handler.go`
- Modify: `apps/backend/internal/platform/bootstrap/bootstrap.go` (extend the existing `eventModule`)
- Modify: `apps/backend/internal/shared/i18n/i18n.go` (add `analytics.invalid_interval`)

**Interfaces:**
- Consumes: `eventapp.{AnalyticsService, OverviewResult, TokenPoint, CostPoint, LatencyPoint, ErrorPoint}` (Task 2), `eventdomain.{AnalyticsFilter, ProviderStat, ModelStat, ProjectStat}` (Task 1), `datetime.ParseRange`, `response.Data`, the existing session middleware.
- Produces: `handler.NewAnalyticsHandler`.

- [ ] **Step 1: Register the `analytics.invalid_interval` i18n code**

In `apps/backend/internal/shared/i18n/i18n.go`, add a new `// --- analytics ---` section to both the const block (after `// --- event ---`) and the `catalog` map:
```go
	// --- analytics ---
	CodeAnalyticsInvalidInterval = "analytics.invalid_interval"
```
```go
	// --- analytics ---
	CodeAnalyticsInvalidInterval: {EN: "interval must be one of hour, day, or week", ID: "interval harus salah satu dari hour, day, atau week"},
```

- [ ] **Step 2: Write the response DTOs**

`apps/backend/internal/adapter/http/dto/analytics.go`:
```go
package dto

import (
	eventdomain "router-lens/internal/domain/event"
	eventapp "router-lens/internal/usecase/event"
)

// OverviewResponse is the wire shape of GET /analytics/overview.
type OverviewResponse struct {
	TotalRequests      int64          `json:"total_requests"`
	TotalInputTokens   int64          `json:"total_input_tokens"`
	TotalOutputTokens  int64          `json:"total_output_tokens"`
	TotalCostUSD       *string        `json:"total_cost_usd"`
	UnpricedCount      int64          `json:"unpriced_count"`
	AvgLatencyMs       *float64       `json:"avg_latency_ms"`
	P95LatencyMs       *float64       `json:"p95_latency_ms"`
	ErrorCount         int64          `json:"error_count"`
	ErrorRate          float64        `json:"error_rate"`
	MostUsedProvider   string         `json:"most_used_provider"`
	MostUsedModel      string         `json:"most_used_model"`
	MostExpensiveModel string         `json:"most_expensive_model"`
	TopProjects        []ProjectStat  `json:"top_projects"`
}

type ProjectStat struct {
	ProjectID    string `json:"project_id"`
	ProjectName  string `json:"project_name"`
	RequestCount int64  `json:"request_count"`
}

func FromOverviewResult(r *eventapp.OverviewResult) OverviewResponse {
	top := make([]ProjectStat, 0, len(r.TopProjects))
	for _, p := range r.TopProjects {
		top = append(top, ProjectStat{ProjectID: p.ProjectID, ProjectName: p.ProjectName, RequestCount: p.RequestCount})
	}
	return OverviewResponse{
		TotalRequests: r.Totals.TotalRequests, TotalInputTokens: r.Totals.TotalInputTokens,
		TotalOutputTokens: r.Totals.TotalOutputTokens, TotalCostUSD: decimalPtrString(r.Totals.TotalCostUSD),
		UnpricedCount: r.Totals.UnpricedCount, AvgLatencyMs: r.Totals.AvgLatencyMs, P95LatencyMs: r.Totals.P95LatencyMs,
		ErrorCount: r.Totals.ErrorCount, ErrorRate: r.ErrorRate,
		MostUsedProvider: r.MostUsedProvider, MostUsedModel: r.MostUsedModel, MostExpensiveModel: r.MostExpensiveModel,
		TopProjects: top,
	}
}

type TokenPointResponse struct {
	Bucket       string `json:"bucket"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
}

func FromTokenPoints(points []eventapp.TokenPoint) []TokenPointResponse {
	out := make([]TokenPointResponse, 0, len(points))
	for _, p := range points {
		out = append(out, TokenPointResponse{Bucket: p.Bucket, InputTokens: p.InputTokens, OutputTokens: p.OutputTokens})
	}
	return out
}

type CostPointResponse struct {
	Bucket  string  `json:"bucket"`
	CostUSD *string `json:"cost_usd"`
}

func FromCostPoints(points []eventapp.CostPoint) []CostPointResponse {
	out := make([]CostPointResponse, 0, len(points))
	for _, p := range points {
		out = append(out, CostPointResponse{Bucket: p.Bucket, CostUSD: decimalPtrString(p.CostUSD)})
	}
	return out
}

type LatencyPointResponse struct {
	Bucket       string   `json:"bucket"`
	AvgLatencyMs *float64 `json:"avg_latency_ms"`
	P95LatencyMs *float64 `json:"p95_latency_ms"`
}

func FromLatencyPoints(points []eventapp.LatencyPoint) []LatencyPointResponse {
	out := make([]LatencyPointResponse, 0, len(points))
	for _, p := range points {
		out = append(out, LatencyPointResponse{Bucket: p.Bucket, AvgLatencyMs: p.AvgLatencyMs, P95LatencyMs: p.P95LatencyMs})
	}
	return out
}

type ErrorPointResponse struct {
	Bucket       string  `json:"bucket"`
	RequestCount int64   `json:"request_count"`
	ErrorCount   int64   `json:"error_count"`
	ErrorRate    float64 `json:"error_rate"`
}

func FromErrorPoints(points []eventapp.ErrorPoint) []ErrorPointResponse {
	out := make([]ErrorPointResponse, 0, len(points))
	for _, p := range points {
		out = append(out, ErrorPointResponse{Bucket: p.Bucket, RequestCount: p.RequestCount, ErrorCount: p.ErrorCount, ErrorRate: p.ErrorRate})
	}
	return out
}

type ProviderStatResponse struct {
	Provider     string  `json:"provider"`
	RequestCount int64   `json:"request_count"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	CostUSD      *string `json:"cost_usd"`
}

func FromProviderStats(stats []eventdomain.ProviderStat) []ProviderStatResponse {
	out := make([]ProviderStatResponse, 0, len(stats))
	for _, s := range stats {
		out = append(out, ProviderStatResponse{
			Provider: s.Provider, RequestCount: s.RequestCount,
			InputTokens: s.InputTokens, OutputTokens: s.OutputTokens, CostUSD: decimalPtrString(s.CostUSD),
		})
	}
	return out
}

type ModelStatResponse struct {
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	RequestCount int64   `json:"request_count"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	CostUSD      *string `json:"cost_usd"`
}

func FromModelStats(stats []eventdomain.ModelStat) []ModelStatResponse {
	out := make([]ModelStatResponse, 0, len(stats))
	for _, s := range stats {
		out = append(out, ModelStatResponse{
			Provider: s.Provider, Model: s.Model, RequestCount: s.RequestCount,
			InputTokens: s.InputTokens, OutputTokens: s.OutputTokens, CostUSD: decimalPtrString(s.CostUSD),
		})
	}
	return out
}
```
NOTE: `decimalPtrString` already exists in `dto/event.go` (Task 3 of Plan 05) — same package, reused here, do not redefine.

- [ ] **Step 3: Write the handler**

`apps/backend/internal/adapter/http/handler/analytics_handler.go`:
```go
package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"router-lens/internal/adapter/http/dto"
	eventdomain "router-lens/internal/domain/event"
	eventapp "router-lens/internal/usecase/event"
	"router-lens/internal/shared/datetime"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
	"router-lens/internal/shared/response"
)

type AnalyticsHandler struct{ svc *eventapp.AnalyticsService }

func NewAnalyticsHandler(svc *eventapp.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc}
}

// Register mounts the seven session-authenticated analytics routes.
func (h *AnalyticsHandler) Register(api *echo.Group, session echo.MiddlewareFunc) {
	api.GET("/analytics/overview", h.overview, session)
	api.GET("/analytics/tokens", h.tokens, session)
	api.GET("/analytics/cost", h.cost, session)
	api.GET("/analytics/latency", h.latency, session)
	api.GET("/analytics/errors", h.errorsSeries, session)
	api.GET("/analytics/providers", h.providers, session)
	api.GET("/analytics/models", h.models, session)
}

// parseFilter reads the mandatory bounded date range + optional project scope
// shared by every endpoint. requireInterval is true for the four series
// endpoints, which additionally validate `interval` against the allow-list
// (defense-in-depth: the Postgres adapter validates it again independently).
func (h *AnalyticsHandler) parseFilter(c echo.Context, requireInterval bool) (eventdomain.AnalyticsFilter, error) {
	rng, err := datetime.ParseRange(c.QueryParam("from"), c.QueryParam("to"), c.QueryParam("preset"), time.Now().UTC())
	if err != nil {
		return eventdomain.AnalyticsFilter{}, err
	}
	f := eventdomain.AnalyticsFilter{ProjectID: c.QueryParam("project_id"), From: rng.From, To: rng.To}
	if !requireInterval {
		return f, nil
	}
	interval := c.QueryParam("interval")
	if interval == "" {
		interval = "day"
	}
	switch interval {
	case "hour", "day", "week":
		f.Interval = interval
		return f, nil
	default:
		return eventdomain.AnalyticsFilter{}, apperrors.New(apperrors.KindValidation, i18n.CodeAnalyticsInvalidInterval, "invalid interval")
	}
}

func (h *AnalyticsHandler) overview(c echo.Context) error {
	f, err := h.parseFilter(c, false)
	if err != nil {
		return err
	}
	result, err := h.svc.Overview(c.Request().Context(), f)
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, dto.FromOverviewResult(result))
}

func (h *AnalyticsHandler) tokens(c echo.Context) error {
	f, err := h.parseFilter(c, true)
	if err != nil {
		return err
	}
	points, err := h.svc.TokensSeries(c.Request().Context(), f)
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, dto.FromTokenPoints(points))
}

func (h *AnalyticsHandler) cost(c echo.Context) error {
	f, err := h.parseFilter(c, true)
	if err != nil {
		return err
	}
	points, err := h.svc.CostSeries(c.Request().Context(), f)
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, dto.FromCostPoints(points))
}

func (h *AnalyticsHandler) latency(c echo.Context) error {
	f, err := h.parseFilter(c, true)
	if err != nil {
		return err
	}
	points, err := h.svc.LatencySeries(c.Request().Context(), f)
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, dto.FromLatencyPoints(points))
}

func (h *AnalyticsHandler) errorsSeries(c echo.Context) error {
	f, err := h.parseFilter(c, true)
	if err != nil {
		return err
	}
	points, err := h.svc.ErrorSeries(c.Request().Context(), f)
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, dto.FromErrorPoints(points))
}

func (h *AnalyticsHandler) providers(c echo.Context) error {
	f, err := h.parseFilter(c, false)
	if err != nil {
		return err
	}
	stats, err := h.svc.Providers(c.Request().Context(), f)
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, dto.FromProviderStats(stats))
}

func (h *AnalyticsHandler) models(c echo.Context) error {
	f, err := h.parseFilter(c, false)
	if err != nil {
		return err
	}
	stats, err := h.svc.Models(c.Request().Context(), f)
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, dto.FromModelStats(stats))
}
```
NOTE: the handler method is named `errorsSeries` (not `errors`) to avoid shadowing the stdlib `errors` package name within the file's method set — purely a naming choice, no behavior implication.

- [ ] **Step 4: Wire the analytics repository/service/handler into the existing `eventModule`**

In `apps/backend/internal/platform/bootstrap/bootstrap.go`:

(a) add one line to the existing `eventModule`'s `fx.Provide(...)` list (alongside the Plan 05 entries):
```go
		fx.Annotate(postgres.NewAnalyticsRepository, fx.As(new(event.AnalyticsRepository))),
		eventapp.NewAnalyticsService,
		handler.NewAnalyticsHandler,
```

(b) add a registrar function next to `registerEventLogRoutes`:
```go
// registerAnalyticsRoutes mounts the analytics routes behind the session middleware.
func registerAnalyticsRoutes(e *echo.Echo, h *handler.AnalyticsHandler, session echo.MiddlewareFunc) {
	h.Register(e.Group(apiBasePath), session)
}
```

(c) add it to the module's existing `fx.Invoke(...)` line:
```go
	fx.Invoke(registerEventIngestRoutes, registerEventLogRoutes, registerAnalyticsRoutes),
```

No new imports needed — `event`, `eventapp`, `postgres`, `handler` are already imported in `bootstrap.go` from Plan 05.

- [ ] **Step 5: Full verification**

Run: `cd apps/backend && gofmt -l . && go vet ./... && go build ./... && go test ./internal/... -count=1`
Expected: gofmt prints nothing; vet/build clean; all non-integration tests PASS.

Optional docker smoke test — only if a Postgres is up and the server can start; skip cleanly and note it if not:
```
curl -s 'localhost:8080/api/v1/analytics/overview?preset=7d' -b cookies.txt
curl -s 'localhost:8080/api/v1/analytics/tokens?preset=7d&interval=day' -b cookies.txt
curl -s 'localhost:8080/api/v1/analytics/latency?preset=7d&interval=hour' -b cookies.txt
curl -s 'localhost:8080/api/v1/analytics/providers?preset=7d' -b cookies.txt
curl -s 'localhost:8080/api/v1/analytics/tokens?preset=7d&interval=century' -b cookies.txt   # expect 400
```
Expected: the first four return `{"data": ...}` with the shapes above; the last returns a 400 with code `analytics.invalid_interval`.

- [ ] **Step 6: Commit**

Do NOT git commit — this project commits once per plan, at the end, after the controller shows the full diff to the user (see `.superpowers/sdd/progress.md`, the same one-commit-per-plan cadence used for Plan 05).

---

## Plan-level Definition of Done

- `GET /api/v1/analytics/overview` returns total requests/tokens/cost (nullable when unpriced, with an `unpriced_count` signal), avg + P95 latency, error rate, most-used provider/model, most-expensive priced model, and top-5 projects by usage — bounded by a mandatory date range (default 24h, max 90d, reusing `datetime.ParseRange`).
- `GET /api/v1/analytics/{tokens,cost,latency,errors}` each return a time series bucketed by `interval` (`hour`/`day`/`week`, default `day`), all four backed by one shared `Series` repository query.
- `GET /api/v1/analytics/{providers,models}` return distributions (count, tokens, cost) grouped by dimension.
- Every endpoint accepts an optional `project_id` filter; omitted means cross-project (single-admin model, decision 4).
- The `date_trunc` bucket interval is validated against a fixed allow-list independently at the handler AND the Postgres adapter — never string-concatenated from raw input.
- `cost_usd` aggregates stay nullable — never silently rendered as `0` when a range has no priced events.
- No new migration, no rollup table, no materialized view — five raw SQL queries over the existing `llm_events` schema.
- `gofmt`/`go vet`/`go build` clean; unit suites green; the analytics repository integration test green against real Postgres.
- Three commits' worth of work landed as ONE commit on `dev`, after a Codex review pass per task + a final whole-plan review (mirroring Plan 05's cadence exactly).
