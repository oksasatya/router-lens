// Provider brand icons for the suggestion picker. Sourced from `simple-icons`
// (CC0-licensed, exact brand SVG data) rather than hand-copied paths — see
// docs/adr/0001-pricing-suggestions-openrouter.md.
//
// ponytail: importing from the `simple-icons` package root (or `simple-icons/icons`)
// pulls in its ~5MB single-file bundle of *every* brand icon — Vite's dependency
// optimizer has to parse the whole thing and crashes with an out-of-memory error
// on this machine. Each icon is imported individually as a raw SVG file instead
// (`?raw`, a Vite feature) — a few hundred bytes each, no barrel.
//
// Provider → icon mapping notes:
// - openai has NO icon in simple-icons at all (only "openaigym" exists, a
//   different product) — falls back to the generic Bot icon like x-ai/cohere.
// - meta-llama → the "meta" (company) icon file.
// - x-ai, cohere → no icon in simple-icons — generic Bot fallback.
import { Bot } from "lucide-react";
import type { ReactNode } from "react";
import anthropicSvg from "simple-icons/icons/anthropic.svg?raw";
import deepseekSvg from "simple-icons/icons/deepseek.svg?raw";
import googleSvg from "simple-icons/icons/google.svg?raw";
import metaSvg from "simple-icons/icons/meta.svg?raw";
import mistralaiSvg from "simple-icons/icons/mistralai.svg?raw";
import qwenSvg from "simple-icons/icons/qwen.svg?raw";

/** simple-icons SVGs are always `<svg ...><title>...</title><path d="..."/></svg>`. */
function extractPath(raw: string): string {
  return /<path d="([^"]+)"/.exec(raw)?.[1] ?? "";
}

const PROVIDER_ICON_PATHS: Record<string, string> = {
  anthropic: extractPath(anthropicSvg),
  google: extractPath(googleSvg),
  "meta-llama": extractPath(metaSvg),
  mistralai: extractPath(mistralaiSvg),
  deepseek: extractPath(deepseekSvg),
  qwen: extractPath(qwenSvg),
};

/** Renders a provider's brand icon (in the current text color), or a generic fallback for unlisted providers. */
export function ProviderLogo({ provider, className }: { readonly provider: string; readonly className?: string }): ReactNode {
  const path = PROVIDER_ICON_PATHS[provider.toLowerCase()];
  if (!path) return <Bot className={className} aria-hidden />;
  return (
    <svg role="img" viewBox="0 0 24 24" className={className} fill="currentColor" aria-hidden>
      <path d={path} />
    </svg>
  );
}
