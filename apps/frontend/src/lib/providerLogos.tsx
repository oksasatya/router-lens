// Provider brand icons for the suggestion picker. Sourced from `simple-icons`
// (CC0-licensed, exact brand SVG data) rather than hand-copied paths — see
// docs/adr/0001-pricing-suggestions-openrouter.md.
import { Bot } from "lucide-react";
import type { ReactNode } from "react";

// NOTE: Import from 'simple-icons' directly (not 'simple-icons/icons'),
// using the camelCase kebab-to-camel naming pattern that simple-icons exports.
// Some provider names map to different icon names:
// - google → siGoogle (no Gemini-specific icon at this version)
// - meta-llama → siMeta (Meta company icon)
// - x-ai → use Bot fallback (no X/Twitter-specific icon for X.AI)
// - mistralai → siMistralai (matches provider name)
// - cohere → use Bot fallback (no Cohere icon in simple-icons)
import {
  siAnthropic,
  siGoogle,
  siMeta,
  siMistralai,
  siDeepseek,
  siQwen,
} from "simple-icons";

interface BrandIcon {
  readonly path: string;
  readonly hex: string;
}

const PROVIDER_ICONS: Record<string, BrandIcon> = {
  anthropic: siAnthropic,
  google: siGoogle,
  "meta-llama": siMeta,
  mistralai: siMistralai,
  deepseek: siDeepseek,
  qwen: siQwen,
};

/** Renders a provider's brand icon, or a generic fallback for unlisted providers. */
export function ProviderLogo({ provider, className }: { readonly provider: string; readonly className?: string }): ReactNode {
  const icon = PROVIDER_ICONS[provider.toLowerCase()];
  if (!icon) return <Bot className={className} aria-hidden />;
  return (
    <svg role="img" viewBox="0 0 24 24" className={className} fill={`#${icon.hex}`} aria-hidden>
      <path d={icon.path} />
    </svg>
  );
}
