<div align="center">

# RouterLens

### Self-hosted observability for LLM routers and AI coding agents.

Every model call your router makes — tokens, cost, latency, errors — in one dashboard you actually own.
No SaaS. No Redis. One Go binary and a Postgres.

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![React](https://img.shields.io/badge/React-TanStack_Start-0EA5E9?logo=react&logoColor=white)](https://tanstack.com/start)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-only_dependency-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org)
[![License](https://img.shields.io/badge/license-MIT-000000)](#license)
[![Status](https://img.shields.io/badge/status-MVP_v0.1-FACC15)](#roadmap)
[![PRs](https://img.shields.io/badge/PRs-welcome-22C55E)](#contributing)

[Quickstart](#quickstart) · [Why](#why) · [How it works](#how-it-works) · [Ingest](#ingest-an-event) · [Roadmap](#roadmap)

</div>

---

## TL;DR

You're running an LLM router or proxy — 9router, LiteLLM, an OpenRouter proxy, a Claude Code
proxy, Agent Zero, something homegrown. It's quietly burning tokens and money across a dozen
models and you have no idea where. RouterLens is the dashboard that tells you.

Point your router at one endpoint. Get request logs, token and cost analytics, latency (avg and
P95), error tracking, and provider/model breakdowns. Self-hosted, so your traffic never leaves
your box.

> **RouterLens is not a router.** It does no routing and makes zero model decisions. It only
> watches. That's the whole point.

---

## Why

Routers and gateways are great at *moving* traffic and terrible at *explaining* it. Once you have
fallbacks, multiple providers, and a handful of agents hitting the same gateway, three questions
get hard fast:

- **Where is the money going?** Which model, which project, which agent.
- **What's slow?** Average latency lies; P95 is where the pain lives.
- **What's failing?** Error rates per provider, with the actual messages.

RouterLens answers all three from a single append-only event stream. No agents to install, no
sampling, no per-seat pricing — just an HTTP endpoint and a dashboard.

### Works with anything that can POST JSON

`9router` · `LiteLLM` · OpenRouter proxies · Claude Code proxy · `Agent Zero` · your custom
gateway. If it can fire one HTTP request per LLM call, RouterLens can chart it.

---

## Features

| Capability | What you get |
|------------|--------------|
| **Request logs** | Every call, searchable, keyset-paginated, CSV-exportable |
| **Cost analytics** | Estimated cost per call, auto-computed from your pricing, with a price snapshot |
| **Token analytics** | Input/output tokens over time, per provider and model |
| **Latency** | Average and P95 — the percentile that actually matters |
| **Error tracking** | Error rate and timeline, with real messages |
| **Breakdowns** | By provider, model, and project |
| **Pricing** | Set a price per `(provider, model)`; unpriced models are flagged, never silently `$0` |
| **Projects + API keys** | Scope ingestion per project; keys are hashed and shown once |
| **Auth done right** | httpOnly cookie sessions, server-side revocable. No tokens in localStorage |

---

## How it works

RouterLens ships as a **single deployable monolith**. One Go binary serves the JSON API *and* the
frontend — the React/TanStack Start app is built to static assets and embedded straight into the
binary. One process, one port. Postgres is the only thing you have to run alongside it.

```
        your router / gateway / agent
                     │
                     │  POST /api/v1/events   (one call = one event)
                     ▼
        ┌─────────────────────────────────┐
        │  routerlens  (single Go binary)  │
        │    /api/v1/*  ->  Echo API        │
        │    /*         ->  embedded UI     │
        └────────────────┬─────────────────┘
                         ▼
                  ┌─────────────┐
                  │ PostgreSQL  │   no Redis, no queue, no extras
                  └─────────────┘
```

Backend is hexagonal + DDD + Clean Architecture (`domain ← application ← infrastructure ← cmd`).
The deep dives live in [`CLAUDE.md`](./CLAUDE.md) (project rules), [`CONTEXT.md`](./CONTEXT.md)
(domain glossary), and [the design spec](./docs/superpowers/specs/2026-06-29-routerlens-design.md).

---

## Quickstart

```bash
git clone https://github.com/oksasatya/router-lens.git
cd router-lens
cp .env.example .env        # edit the secrets
docker compose up --build
```

Open <http://localhost:8080>.

**First run** — there are no users yet, so the app drops you on `/setup` to create the admin
account. That endpoint locks itself the moment the first user exists.

Then: log in, create a **Project**, create an **API key** (copy it — it's shown once), and point
your router at the ingestion endpoint below.

---

## Ingest an event

```bash
curl -X POST http://localhost:8080/api/v1/events \
  -H "Authorization: Bearer rl_live_your_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "optional-idempotency-key",
    "provider": "anthropic",
    "model": "claude-sonnet-4-5",
    "route_source": "9router",
    "agent": "claude-code",
    "input_tokens": 12000,
    "output_tokens": 1800,
    "latency_ms": 8420,
    "status_code": 200,
    "request_started_at": "2026-06-29T10:00:00Z",
    "request_finished_at": "2026-06-29T10:00:08Z",
    "metadata": { "workspace": "nuvora", "session_id": "abc", "fallback_from": null }
  }'
```

Notes:
- The project is **derived from the API key** — never send `project_id` in the body.
- Pass a stable `event_id` and retries are deduplicated automatically (`202` either way).
- Don't put secrets in `metadata`.

---

## Stack

| Layer | Choice |
|-------|--------|
| Backend | Go 1.26 + Echo v4 |
| Frontend | React + TanStack Start (TypeScript), shadcn/ui, TanStack Query, Recharts |
| Database | PostgreSQL — the only dependency |
| Auth | DB-backed session in an httpOnly cookie |
| Deploy | One Go binary (frontend embedded) + Docker Compose |

---

## Project structure

```
cmd/server/         main.go — manual DI wiring; serves API + embedded UI
internal/
  app/              bootstrap, config
  domain/           entities, value objects, repository interfaces, cost calculator
  application/      use cases (zero HTTP knowledge)
  infrastructure/   postgres repositories; echo http (handlers, middleware, router)
  shared/           response, errors, pagination, validator, security, datetime, csv
  web/              embed of the built frontend + SPA fallback
migrations/         NNN_*.up.sql / NNN_*.down.sql
apps/web/           TanStack Start frontend (built into the binary)
```

---

## Development

Production is one binary. In dev you can split the pieces for hot reload:

```bash
docker compose up postgres                 # database only
make migrate                               # apply migrations
make dev                                   # Go API on :8080
cd apps/web && pnpm install && pnpm dev     # Vite dev server, proxies /api to the Go API
```

Common commands:

```bash
make migrate            # apply SQL migrations
make create-admin       # CLI fallback for the first admin
go test -race -cover ./...
golangci-lint run
cd apps/web && pnpm build   # produce the static frontend the binary embeds
```

---

## Configuration

Copy `.env.example` to `.env`:

| Variable | Purpose |
|----------|---------|
| `DATABASE_URL` | PostgreSQL connection string |
| `APP_ENV` | `development` / `production` (toggles `Secure` cookies) |
| `APP_PORT` | HTTP port (default `8080`) |
| `SESSION_SECRET` | secret used for session handling |
| `COOKIE_CROSS_SITE` | `true` only when UI and API live on different origins (reverse-proxied) |
| `MAX_BACKDATE_DAYS` | reject events older than this (default `7`) |
| `RETENTION_DAYS` | optional event pruning (off by default) |

---

## Roadmap

**v0.1 (now)** — observability core: auth, projects, API keys, ingestion, request logs,
token/cost/latency/error analytics, provider/model breakdowns, pricing, CSV export.

**Later** — multi-user and team access, alerting, pre-aggregated rollups for high-volume
deployments, more export formats, per-key rate limiting. Deliberately *not* planned: turning
RouterLens into a router.

---

## Contributing

Issues and PRs are welcome. The repo is opinionated about architecture and code quality — read
[`CLAUDE.md`](./CLAUDE.md) before a substantial PR so your change lands clean the first time.

---

## License

MIT. Add a `LICENSE` file before the first public release.
