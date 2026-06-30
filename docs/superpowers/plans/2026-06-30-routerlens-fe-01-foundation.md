# RouterLens FE Plan 01 — Scaffold + Design System + App Shell

> **Execution model (differs from the backend plans):** the scaffold + data layer + formatters are mechanical (run the commands, transcribe the code). The **design system (Task 2) and shell visual (Task 4) are controller-led and interactive** — invoke `impeccable` (`shape` → `craft`) and `frontend-design`, render in the browser, screenshot, and iterate with the user before locking. This plan is the structured checklist + the deterministic code; the *look* is crafted live, not pre-frozen. Build on branch `dev`.

**Goal:** Stand up the Vite + React SPA at `apps/frontend/`, establish the brand design system (logo-derived palette → theme tokens + `DESIGN.md`, dark-default), and build the app shell (sidebar nav + topbar + theme toggle + EN/ID flag toggle) with the data-layer + i18n foundations — so later plans drop screens into a working, on-brand shell wired to the Go backend.

**Stack (locked):** Vite + React 19 + TypeScript (SPA) · TanStack Router (typed routes, `beforeLoad` guard, typed search params — replaces nuqs) · TanStack Query + axios (one instance + interceptors) · react-hook-form + zod · shadcn/ui + Tailwind v4 + Radix + lucide-react + sonner · TanStack Table · i18next + react-i18next + country-flag-icons (EN/ID) · date-fns + `Intl` · Vitest + RTL + MSW · ESLint + Prettier · **bun**. Output: Vite static build → embedded in the Go binary; dev: Vite proxy `/api` → `:8080`.

## Global Constraints

- **Tailwind v4 ONLY (HARD).** v4 syntax: `@import "tailwindcss"`, `@theme`, CSS-first config, `@tailwindcss/vite` plugin. No `tailwind.config.js` theme block, no v3 utilities (`bg-gradient-*`, bare `ring`, `shadow`/`rounded` legacy scale, `flex-shrink-*`, `bg-opacity-*`). Any v3-era utility is a blocker.
- **shadcn/ui** for every primitive — Radix + `cn()` + `cva`, CSS-variable theming (`:root` light + `.dark`), `components.json` at `apps/frontend/`. Real elements over ARIA roles (S6819): `<button>` not `<div role="button">`, `<dialog>`/shadcn Dialog not `role="dialog"`.
- **Anti-duplication (project rule):** one axios instance (`lib/api.ts`); per-domain service + TanStack Query hook; formatters in `lib/{money,token,date,format}.ts`; reusable `<DataTable>`/`<StatCard>`/`<DateRangePicker>` (later plans). Components never call axios directly — only through a service/hook. No hardcoded endpoints scattered.
- **Auth state** from `GET /api/v1/auth/me` (TanStack Query) — never localStorage/sessionStorage (the cookie is httpOnly; JS can't read it).
- **i18n:** all user-facing UI strings go through i18next (`t("...")`) — no hardcoded literals in components. EN + ID resource files. Locale persisted to localStorage and sent as `Accept-Language` so the backend localizes error messages to match.
- **Sonar-TS/React (write compliant from the first commit):** props `readonly` (S6759) · `globalThis` not `window` w/ `?.` (S7764) · no nested/identical ternaries (S3358/S3923) · optional chaining + `??=` (S6582/S6606) · `arr.at(-1)` (S7755) · real elements over ARIA roles (S6819) · stable list keys, never array index (S6479) · zod v4 `z.email()` not `z.string().email()` (S1874).
- **Quality floor (frontend-design):** responsive to mobile (sidebar → drawer + touch targets ≥44px, inputs ≥16px), visible keyboard focus (ring token), `prefers-reduced-motion` respected. Behind-auth → performance gate applies, SEO gate does not.

### Skill chain (controller, for the design + shell tasks)

> Invoke `frontend-design` (auto-chains `ui-ux-pro-max` + `frontend-senior` + `tailwind-4` + `tailwind-design-system` + `tailwind-responsive-design` + `color-expert`) **plus `vercel:shadcn`** (project uses shadcn) for the design + shell work; the palette is already locked (see Task 2). Greenfield design → **`impeccable` LEADS**: `shape` to plan the visual identity (within the locked palette), then `craft` to build, then `critique`/`polish` to finish. Apply `ponytail` (YAGNI): no speculative components, reuse shadcn primitives, the shell ships with only the chrome the app needs now. Finish with `react-doctor` (lint/a11y/bundle) + the `claude-seo:seo-performance` perf gate before marking the plan done.

### TDD verdicts (§16)

- Formatters (`lib/money.ts`, `lib/token.ts`, `lib/date.ts`) — **TDD: yes** (pure input→output; Vitest first).
- axios interceptors / zod envelope schemas — **TDD: yes-ish** (a focused Vitest test: envelope unwrap + 401-redirect behavior via MSW).
- Scaffold / Tailwind / shadcn / theme / i18n wiring / shell layout — **TDD: no** (config + visual; verify by running + screenshot + `react-doctor`). A smoke render test (RTL: shell mounts, nav links present, theme/lang toggle present) is the regression net.

---

## Task 1: Scaffold the Vite + React SPA

Mechanical. Run the official CLIs (version-robust) + apply the config below. If a command/flag changed, the CLI will say so — adjust and note it.

**Files (created by CLIs + edits):** `apps/frontend/` tree, `vite.config.ts`, `tsconfig*.json`, `components.json`, `package.json`.

- [ ] **Step 1: Create the Vite React-TS app with bun**
```bash
cd /Volumes/Project/router-lens/apps
bun create vite frontend --template react-ts
cd frontend
bun install
```

- [ ] **Step 2: Tailwind v4 (Vite plugin) + path alias**
```bash
bun add tailwindcss @tailwindcss/vite
bun add -d @types/node
```
Replace `apps/frontend/src/index.css` first line block with v4 entry (full theme added in Task 2):
```css
@import "tailwindcss";
```
`apps/frontend/vite.config.ts` — Tailwind plugin, `@` alias, and the dev proxy to the Go backend:
```ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: { alias: { "@": path.resolve(__dirname, "./src") } },
  server: {
    port: 5173,
    proxy: { "/api": { target: "http://localhost:8080", changeOrigin: true } },
  },
});
```
Add the alias to `tsconfig.json` (and `tsconfig.app.json` if present):
```jsonc
{
  "compilerOptions": {
    "baseUrl": ".",
    "paths": { "@/*": ["./src/*"] }
  }
}
```

- [ ] **Step 3: shadcn/ui init (Tailwind v4)**
```bash
bunx --bun shadcn@latest init
# choose: TypeScript, base color "neutral" (we override with the brand palette in Task 2),
# CSS variables = yes. This writes components.json + the cn() util + base CSS variables.
```
Then install the primitives this plan + the next ones need (no speculative extras — ponytail):
```bash
bunx --bun shadcn@latest add button card input label badge dropdown-menu dialog sheet table sonner skeleton tooltip separator avatar form select
```

- [ ] **Step 4: TanStack Router (file-based routes + plugin) + TanStack Query + axios + the rest**
```bash
bun add @tanstack/react-router @tanstack/react-query
bun add -d @tanstack/router-plugin @tanstack/react-router-devtools @tanstack/react-query-devtools
bun add axios zod react-hook-form @hookform/resolvers
bun add i18next react-i18next country-flag-icons date-fns lucide-react
```
(`@tanstack/react-router-devtools` — NOT the old `@tanstack/router-devtools`. TanStack Table is NOT installed here; it lands in FE Plan 03 with the first `<DataTable>` — ponytail, not used in FE-01.)

Add the router plugin to `vite.config.ts` plugins **before `react()`**, file-based routing under `src/routes/`:
```ts
import { tanstackRouter } from "@tanstack/router-plugin/vite";
// plugins: [tanstackRouter({ target: "react", autoCodeSplitting: true }), react(), tailwindcss()]
```
**Verify the export name against the installed `@tanstack/router-plugin`:** current is `tanstackRouter`; older versions export `TanStackRouterVite` — use whichever the installed version exposes. The plugin generates `src/routeTree.gen.ts` from `src/routes/*`; `main.tsx` imports that generated tree into `createRouter({ routeTree })` (file-based throughout — no hand-written route tree).

- [ ] **Step 4b: Testing setup (Vitest + RTL + MSW)**
The Vite react-ts template ships NO test runner. Install + configure:
```bash
bun add -d vitest @vitest/ui jsdom @testing-library/react @testing-library/jest-dom @testing-library/user-event msw
```
Add a `test` block to `vite.config.ts` (or a `vitest.config.ts`): `environment: "jsdom"`, `globals: true`, `setupFiles: "./src/test/setup.ts"`. Create `src/test/setup.ts` importing `@testing-library/jest-dom/vitest`. Add scripts to `package.json`: `"test": "vitest run"`, `"test:watch": "vitest"`. Confirm `bun run test` runs (0 tests is fine until Task 3 adds the formatter tests).

- [ ] **Step 5: Verify the shell runs + proxy works**
```bash
cd /Volumes/Project/router-lens/apps/frontend && bun run dev
```
Expected: Vite serves on `:5173`; the default app renders. With the Go backend running (`docker compose up` or `go run`), `curl -s localhost:5173/api/v1/healthz` proxies to `:8080` and returns `{"status":"ok"}`. Stop the dev server.

- [ ] **Step 6: ESLint + Prettier baseline**
Vite's react-ts template ships ESLint. Add Prettier + the React/query plugins:
```bash
bun add -d prettier eslint-config-prettier eslint-plugin-jsx-a11y @tanstack/eslint-plugin-query
```
Add `.prettierrc` (`{ "semi": true, "singleQuote": false, "printWidth": 100 }`) and extend the ESLint config with `jsx-a11y` (recommended) + `@tanstack/eslint-plugin-query` (recommended) + `eslint-config-prettier` last. Confirm `bun run lint` is clean on the scaffold.

---

## Task 2: Design system (controller-led, impeccable)

**Invoke `impeccable` (`shape` then `craft`) + `frontend-design`.** The palette is LOCKED (logo-derived, approved dark-default — see the artifact preview). This task encodes it as theme tokens, writes `DESIGN.md`, and verifies the look in the browser with the user before locking. No subagent — interactive craft.

**Files:** `apps/frontend/src/index.css` (theme tokens), `apps/frontend/DESIGN.md`, `apps/frontend/src/lib/theme.ts` (theme provider hook).

- [ ] **Step 1: Write the brand theme tokens (the approved palette)**
Append to `apps/frontend/src/index.css` (after `@import "tailwindcss";`). These are the exact approved OKLCH values; shadcn consumes them via `@theme inline` mapping its variables to these.
```css
:root {
  --radius: 0.625rem;
  --background: oklch(0.99 0.003 257);
  --foreground: oklch(0.21 0.02 260);
  --card: oklch(1 0 0);
  --card-foreground: oklch(0.21 0.02 260);
  --popover: oklch(1 0 0);
  --popover-foreground: oklch(0.21 0.02 260);
  --primary: oklch(0.55 0.21 257);
  --primary-foreground: oklch(0.98 0.01 257);
  --secondary: oklch(0.96 0.008 260);
  --secondary-foreground: oklch(0.25 0.02 260);
  --muted: oklch(0.96 0.006 260);
  --muted-foreground: oklch(0.52 0.02 260);
  --accent: oklch(0.95 0.03 257);
  --accent-foreground: oklch(0.30 0.06 257);
  --destructive: oklch(0.58 0.22 27);
  --destructive-foreground: oklch(0.98 0 0);
  --border: oklch(0.92 0.006 260);
  --input: oklch(0.92 0.006 260);
  --ring: oklch(0.55 0.21 257);
  --chart-1: oklch(0.55 0.21 257);
  --chart-2: oklch(0.70 0.13 195);
  --chart-3: oklch(0.60 0.20 295);
  --chart-4: oklch(0.75 0.15 75);
  --chart-5: oklch(0.66 0.16 150);
}
.dark {
  --background: oklch(0.165 0.013 262);
  --foreground: oklch(0.96 0.005 262);
  --card: oklch(0.195 0.015 262);
  --card-foreground: oklch(0.96 0.005 262);
  --popover: oklch(0.195 0.015 262);
  --popover-foreground: oklch(0.96 0.005 262);
  --primary: oklch(0.62 0.20 257);
  --primary-foreground: oklch(0.98 0.01 257);
  --secondary: oklch(0.26 0.016 262);
  --secondary-foreground: oklch(0.96 0.005 262);
  --muted: oklch(0.24 0.013 262);
  --muted-foreground: oklch(0.70 0.015 262);
  --accent: oklch(0.30 0.05 257);
  --accent-foreground: oklch(0.96 0.005 262);
  --destructive: oklch(0.62 0.21 27);
  --destructive-foreground: oklch(0.98 0 0);
  --border: oklch(0.28 0.014 262);
  --input: oklch(0.28 0.014 262);
  --ring: oklch(0.62 0.20 257);
  --chart-1: oklch(0.62 0.20 257);
  --chart-2: oklch(0.72 0.13 195);
  --chart-3: oklch(0.64 0.20 295);
  --chart-4: oklch(0.78 0.15 75);
  --chart-5: oklch(0.70 0.16 150);
}
@theme inline {
  --color-background: var(--background); --color-foreground: var(--foreground);
  --color-card: var(--card); --color-card-foreground: var(--card-foreground);
  --color-popover: var(--popover); --color-popover-foreground: var(--popover-foreground);
  --color-primary: var(--primary); --color-primary-foreground: var(--primary-foreground);
  --color-secondary: var(--secondary); --color-secondary-foreground: var(--secondary-foreground);
  --color-muted: var(--muted); --color-muted-foreground: var(--muted-foreground);
  --color-accent: var(--accent); --color-accent-foreground: var(--accent-foreground);
  --color-destructive: var(--destructive); --color-destructive-foreground: var(--destructive-foreground);
  --color-border: var(--border); --color-input: var(--input); --color-ring: var(--ring);
  --color-chart-1: var(--chart-1); --color-chart-2: var(--chart-2); --color-chart-3: var(--chart-3);
  --color-chart-4: var(--chart-4); --color-chart-5: var(--chart-5);
  --radius-lg: var(--radius); --radius-md: calc(var(--radius) - 2px); --radius-sm: calc(var(--radius) - 4px);
}
```
Reconcile with whatever shadcn-init wrote — replace its default neutral tokens with these; keep its `@theme inline` mapping shape if it differs, but the values above win.

- [ ] **Step 2: Theme provider (dark default), no-FOUC**
Two parts. (1) **Initial class — inline blocking script in `index.html` `<head>`** (runs before the bundle, so no flash): read `localStorage("theme")` (default `"dark"`), resolve `"system"` via `matchMedia`, and add `.dark` to `document.documentElement` synchronously. A module-side-effect is NOT equivalent (it runs after the bundle loads → still flashes). (2) `apps/frontend/src/lib/theme.tsx` — a tiny zero-dep provider for *subsequent* changes: `useTheme()` → `{ theme, setTheme }` with `"light" | "dark" | "system"`, persists to localStorage + toggles the class. (No next-themes — ponytail.)

- [ ] **Step 3: Logo into public/**
Use absolute paths (cwd-independent). `logo.webp` currently sits untracked at the repo root, so plain `mv` (not `git mv` — the file isn't tracked yet; it gets added when `apps/frontend` is committed). The Vite template already created `apps/frontend/public/`.
```bash
mv /Volumes/Project/router-lens/logo.webp /Volumes/Project/router-lens/apps/frontend/public/logo.webp
```
(The PNG was already converted + deleted; this relocates the webp to the FE public dir as the user requested.)

- [ ] **Step 4: Write `DESIGN.md` (impeccable `shape`)**
`apps/frontend/DESIGN.md` — the brand brief: subject (LLM-router observability for developers), the palette (electric azure `#1870F8` on near-black, dark-native), type pairing (a characterful but legible pairing — propose + verify, e.g. a geometric/grotesk display + a clean mono for data/metrics; not the generic Inter-everywhere default), layout principles (data-dense but calm 60-30-10, generous spacing, hairline borders), and the **signature** element (e.g. the reticle/scope motif from the logo echoed in empty states / loading / the active-nav indicator). Keep it a living brief, not a spec.

- [ ] **Step 5: Verify the look (browser + screenshot) with the user**
Render a scratch page using the shadcn primitives + tokens (card, buttons, input, badge, the chart swatches). Screenshot light + dark. Confirm with the user it matches the approved palette + reads on-brand BEFORE building the shell. Iterate via impeccable `craft`/`polish` if needed.

---

## Task 3: Data layer + i18n

Mostly deterministic — full code below. Formatters are TDD=YES.

**Files:** `src/lib/api.ts`, `src/lib/query.ts`, `src/lib/schemas.ts`, `src/lib/{money,token,date,format}.ts` (+ `*.test.ts`), `src/i18n/{index.ts,en.json,id.json}`, `src/components/LanguageToggle.tsx`, `src/components/ThemeToggle.tsx`.

- [ ] **Step 1: axios instance + interceptors + ApiError**
`src/lib/api.ts`. No import cycle: `i18n/index.ts` must NOT import `api` (it only inits i18next).
```ts
import axios from "axios";
import i18n from "@/i18n";

export const api = axios.create({ baseURL: "/api/v1", withCredentials: true });

// ApiError keeps real Error semantics (stack, instanceof) while carrying the
// backend's localized { error } envelope + HTTP status, so callers / TanStack
// Query / error boundaries get a typed Error, never a bare object.
export class ApiError extends Error {
  constructor(
    readonly code: string,
    message: string,
    readonly status: number,
    readonly details?: unknown,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

// Attach the active locale so the backend localizes error messages.
api.interceptors.request.use((config) => {
  config.headers.set("Accept-Language", i18n.language || "en");
  return config;
});

// Success: unwrap the { data } envelope so services/hooks receive the payload
// directly (paginated payloads arrive as { items, pagination }). Error: 401
// (except the /auth/me probe) -> /login; otherwise reject a typed ApiError.
api.interceptors.response.use(
  (res) => {
    if (res.data && typeof res.data === "object" && "data" in res.data) {
      res.data = (res.data as { data: unknown }).data;
    }
    return res;
  },
  (err) => {
    const status: number = err.response?.status ?? 0;
    const url: string = err.config?.url ?? "";
    if (status === 401 && !url.includes("/auth/me")) {
      globalThis.location?.assign("/login");
    }
    const envelope = err.response?.data?.error;
    if (envelope) {
      return Promise.reject(new ApiError(envelope.code, envelope.message, status, envelope.details));
    }
    return Promise.reject(err instanceof Error ? err : new Error(String(err)));
  },
);
```

- [ ] **Step 2: TanStack Query client + zod payload schemas**
`src/lib/query.ts`: export a `QueryClient` (sane defaults: `staleTime: 30_000`, `retry: 1`). `src/lib/schemas.ts`: because the interceptor already strips the envelope, service/hook schemas validate the **unwrapped payload** — e.g. `userSchema`, `projectSchema`, and a generic `paginated(itemSchema)` helper for the unwrapped `{ items, pagination: { page, limit, total } }`. (The `{ data, meta }` envelope + `meta` block are the interceptor's concern, not the services'; if a raw-client test ever needs the full envelope, add an `envelopeSchema` there only.) Use zod v4 (`z.email()` not `z.string().email()` — S1874).

- [ ] **Step 3: Formatters (TDD=YES — write the tests first)**
`src/lib/money.test.ts` then `money.ts`: `formatUSD(value: string | number): string` via `Intl.NumberFormat("en-US", { style: "currency", currency: "USD", minimumFractionDigits: 2, maximumFractionDigits: 6 })`; `formatUnpriced()` → `"—"` (never `$0`). `token.test.ts` + `token.ts`: `formatTokens(n)` → compact (`12,000` / `1.2M`) via `Intl.NumberFormat` with `notation: "compact"` above a threshold. `date.test.ts` + `date.ts`: `formatTimestamp(iso)` + `formatRelative(iso)` via date-fns. Red → green for each.

- [ ] **Step 4: i18n setup (EN/ID) + toggles**
`src/i18n/index.ts`: init i18next + react-i18next, resources `en`/`id`, `lng` from `localStorage("lang")` default `"en"`, `fallbackLng: "en"`. `en.json`/`id.json`: the UI strings (nav labels, common actions, auth/CRUD labels added per later plan). `LanguageToggle.tsx`: a `DropdownMenu` showing the current locale as a `country-flag-icons` SVG (GB flag → EN, ID flag → Indonesian); selecting one calls `i18n.changeLanguage(code)` + persists to localStorage (the axios interceptor then sends it as `Accept-Language`). `ThemeToggle.tsx`: light/dark/system via `useTheme()` (lucide `Sun`/`Moon`/`Monitor` icons). All toggle labels via `t(...)`.

- [ ] **Step 5: Verify**
`bun run test` (formatters green), `bun run lint` clean, `bun run dev` (toggles flip theme + language live; flags render as SVG, not letters).

---

## Task 4: App shell (controller-led, frontend-design + impeccable `craft`)

**Files:** `src/main.tsx` (providers), `src/routes/__root.tsx`, `src/routes/_app.tsx` (protected layout), `src/routes/_app.index.tsx` + placeholder routes, `src/components/layout/{AppShell,Sidebar,Topbar}.tsx`, `src/components/layout/AppShell.test.tsx`.

- [ ] **Step 1: Providers + router**
`src/main.tsx`: wrap the app in `QueryClientProvider` (the client from `lib/query.ts`) + `RouterProvider` + `<Toaster />` (sonner) + import `@/i18n` + `@/lib/theme` (apply theme before paint). Router devtools + query devtools in dev only.

- [ ] **Step 2: Root + protected layout routes + login placeholder**
`__root.tsx`: the document shell + `<Outlet/>`. `_app.tsx`: the **protected** layout — `beforeLoad` reads the `/auth/me` query (via the query client); on 401/no-user it `throw redirect({ to: "/login" })`. Renders `<AppShell><Outlet/></AppShell>`.
**`routes/login.tsx` — minimal placeholder MUST exist in this plan** so `/login` is a real node in the TanStack typed route tree (otherwise the guard's `redirect({ to: "/login" })` fails typecheck/build and 404s at runtime). It renders a centered "Sign in" card stub (the real login form + setup wizard arrive in FE Plan 02). Likewise the guard only redirects to `/login`; `/setup` routing is FE-02's concern.

- [ ] **Step 3: AppShell (sidebar + topbar) — impeccable craft**
`AppShell` = responsive grid: a `Sidebar` (logo from `/logo.webp` + brand wordmark + nav links with lucide icons: Dashboard, Logs, Projects, API Keys, Pricing, Settings — active state echoes the logo's reticle motif per `DESIGN.md`) and a `Topbar` (page title slot, `ThemeToggle`, `LanguageToggle`, a user-menu placeholder). **Mobile-first:** sidebar `hidden md:flex`; on phone a shadcn `Sheet` drawer triggered from the topbar + a bottom tab bar for the primary destinations; touch targets ≥44px; content padding `p-4 md:p-6`. Nav labels via `t(...)`. Placeholder routes render an empty-state card (signature reticle motif) so the shell is demoable before real screens land.

- [ ] **Step 4: Smoke render test (regression net)**
`AppShell.test.tsx` (RTL + the router test util): mounts the shell, asserts the nav links + ThemeToggle + LanguageToggle are present, and that toggling theme adds `.dark`. TDD=NO for layout, but this guards the normal path.

- [ ] **Step 5: Finishing — react-doctor + perf gate**
Run `react-doctor` (lint/a11y/bundle/architecture) on the FE; fix findings. Run `claude-seo:seo-performance` (CWV/bundle budget) on `bun run build` output — the shell + deps should stay within budget (lazy-load nothing heavy yet; charts/virtual come later). Screenshot light + dark, desktop + mobile; confirm with the user.

---

## Definition of Done (FE Plan 01)

`apps/frontend/` is a Vite + React SPA: `bun run dev` serves an on-brand **dark-default** shell (sidebar + topbar), `/api` proxies to the Go backend, the theme toggle (light/dark/system) and EN/ID flag toggle work live, i18n drives all UI strings, the axios instance + interceptors + TanStack Query + zod envelope schemas are wired, formatters are tested green, and `DESIGN.md` + the logo (`public/logo.webp`) are in place. `bun run lint` + `bun run test` pass; `react-doctor` + the perf gate are clean. The protected layout redirects to `/login` when unauthenticated (login/setup screens arrive in FE Plan 02). Committed on `dev` (one commit for the plan after review).

> **Next:** FE Plan 02 — Auth (setup wizard, login, `/auth/me` + guard, logout/user menu), then FE Plan 03 — Projects + API Keys + Pricing CRUD screens.
