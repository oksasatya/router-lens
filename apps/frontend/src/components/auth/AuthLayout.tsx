import type { ReactNode } from "react";
import { Card } from "@/components/ui/card";

interface AuthLayoutProps {
  readonly title: string;
  readonly subtitle: string;
  readonly children: ReactNode;
}

/**
 * Shared chrome for the unauthenticated entry screens (login + first-run setup).
 * Slate Aurora signature: a quiet indigo→magenta glow behind the card and the
 * logo framed in the reticle/scope ring — tying back to the "lens" in RouterLens.
 */
export function AuthLayout({ title, subtitle, children }: AuthLayoutProps) {
  return (
    <main className="relative grid min-h-svh place-items-center overflow-hidden bg-background p-6 text-foreground">
      <div
        aria-hidden
        className="pointer-events-none absolute -top-1/4 left-1/2 size-[36rem] -translate-x-1/2 rounded-full opacity-70 blur-3xl"
        style={{
          background:
            "radial-gradient(circle at 50% 50%, color-mix(in oklch, var(--primary) 32%, transparent), color-mix(in oklch, var(--chart-2) 16%, transparent) 45%, transparent 70%)",
        }}
      />
      <Card className="relative w-full max-w-sm p-8">
        <div className="flex flex-col items-center text-center">
          <span className="relative grid size-14 place-items-center rounded-full ring-1 ring-border">
            <span aria-hidden className="absolute inset-0 rounded-full ring-1 ring-primary/40" />
            <img src="/logo.webp" alt="" className="size-9 rounded-lg" />
          </span>
          <h1 className="mt-4 font-heading text-2xl font-semibold tracking-tight">{title}</h1>
          <p className="mt-1 text-balance text-sm text-muted-foreground">{subtitle}</p>
        </div>
        <div className="mt-6">{children}</div>
      </Card>
    </main>
  );
}
