---
status: accepted
---

# Pricing Suggestions call OpenRouter's public API server-side

RouterLens's backend has made zero outbound network calls until now — it is a self-hosted
observability tool that only listens (dashboard + ingestion). The Pricing Suggestions feature
(pre-filling a Pricing Rule form from OpenRouter's public model/price list) is the first
exception: the **backend** fetches `https://openrouter.ai/api/v1/models` server-side, caches
the result in memory for 1 hour, and exposes it to the dashboard via a same-origin endpoint.

We considered fetching directly from the browser instead, which would keep the backend
fully offline. Rejected: it would break the project's same-origin/no-CORS invariant (the
browser would depend on a third party's CORS policy, unverified and outside our control), and
it would require every admin's *browser* to reach the internet rather than just the server —
a meaningfully worse fit for self-hosted deployments behind restrictive network policies.

**Consequence:** a RouterLens instance with no outbound internet access still works fully —
Pricing Suggestions degrades to "unavailable, enter manually" (a normal, expected failure
mode, not a crash) — but an admin who wants the convenience feature needs the *server* (not
necessarily the browser) to reach `openrouter.ai`. Nothing else in the product depends on
outbound internet access.

## Additional decisions (added after cross-model review)

- **Config gate, default on.** `PRICING_SUGGESTIONS_ENABLED` (default `true`) lets an operator
  turn this off entirely — self-hosted deployments with strict egress policies get a documented,
  one-line way to guarantee zero outbound calls, without losing the feature's default value for
  everyone else. When off, the endpoint returns 404 and the frontend hides the entry point.
- **Layering.** The suggestion-source port lives in `internal/usecase/pricing` (application
  layer), not `internal/domain/pricing` — it's a UI convenience backed by a third-party
  integration, not a domain rule. `internal/domain/pricing` stays exactly as it is today.
- **Cache hardening.** The in-memory cache has a request timeout (short — a few seconds), a
  response size cap, and single-flight de-duplication so concurrent requests during a cache
  miss don't fan out into multiple simultaneous OpenRouter calls. A stale cache is served (with
  a logged warning) if OpenRouter is unreachable but a prior successful fetch exists.
- **Response filtering.** OpenRouter's model list includes entries RouterLens must not turn into
  suggestions verbatim: prices marked `-1` (unknown/unavailable), non-`provider/model`-shaped
  IDs (aliases prefixed `~`), and anything that fails RouterLens's own field-length validation.
  These are filtered out before the list ever reaches the frontend.
- **Provider logos are sourced from Simple Icons** (CC0-licensed, purpose-built for exactly this
  — third-party brand-icon display) rather than ad hoc assets, to avoid the trademark/licensing
  ambiguity a hand-collected logo set would carry.
