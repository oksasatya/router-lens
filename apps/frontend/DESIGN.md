# RouterLens — Design System

A living brand brief for the dashboard. The source of truth for tokens is
`src/index.css` (`:root` + `.dark`); this file explains the intent behind them.

## Subject

Self-hosted observability for LLM routers and AI coding agents, for developers.
Data-dense but calm. Dark-native (the product lives on engineers' dark screens).

## Direction — "Slate Aurora"

Indigo/violet on deep slate, with a magenta companion. Premium and a little
electric, more brand personality than a flat admin panel — but still quiet
enough to read dense metrics for hours.

## Palette (logo-derived family)

The logo's electric azure (`#1870F8`, ~257°) is kept as the brand mark; the UI
accent shifts to a cousin **indigo `#7B61FF`** (~280°) for a richer, premium feel.
Same blue-violet family, so the azure logo and indigo UI read as relatives.

- **Primary — indigo** `oklch(0.62 0.19 280)` dark / `oklch(0.55 0.20 280)` light (deepened for white-text contrast).
- **Companion — magenta** `oklch(0.66 0.21 350)` — chart series 2, occasional highlight only (10% of the 60-30-10).
- **Surface — deep slate** `oklch(0.18 0.025 280)` (dark hero) — cooler, slightly chromatic black.
- **Foreground** `oklch(0.96 0.01 280)` — near-white with a faint violet cast.
- **Charts** indigo → magenta → cyan → amber → green.
- **Destructive** warm red `oklch(0.63 0.22 18)`. **Unpriced** state renders as a dashed muted badge — never `$0`.

Contrast: foreground/background ≈ AAA in both modes; primary buttons use white text (primary deepened in light, brightened in dark). Focus ring = primary.

## Typography

- **Display / headings — Space Grotesk** (`--font-heading`): characterful grotesk, gives the wordmark + section titles personality without shouting.
- **Body / UI — Inter** (`--font-sans`): clean, legible at small sizes for dense tables and forms.
- **Data / metrics — JetBrains Mono** (`--font-mono`): tabular numerics for tokens, cost, latency, IDs.

All shipped locally via `@fontsource-variable/*` (no CDN; no layout shift).

## Layout principles

- 60-30-10: slate surfaces dominate, indigo structures, magenta accents sparingly.
- Generous spacing, hairline borders (`--border`), calm density. Numbers right-aligned and monospaced in tables.
- Mobile-first: sidebar → drawer + bottom tab bar on phones; touch targets ≥44px; inputs ≥16px.
- Dark mode is the default; light mode is fully supported.

## Signature

A subtle **indigo → magenta aurora** gradient on the hero and active states, and
the logo's **reticle / scope ring** motif echoed in the active-nav indicator and
empty states — tying the UI back to the "lens" in RouterLens.

> Logo note: the mark stays azure for now (a cooler cousin of the indigo UI). It
> can be recolored to indigo for an exact match later if desired.
