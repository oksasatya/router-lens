import { ApiError } from "@/lib/api";

/**
 * Form-level error banner for auth screens. Resolves the message from the
 * backend's localized ApiError; falls back to a generic string for transport
 * errors. role="alert" announces it assertively to screen readers.
 */
export function FormError({ error, fallback }: { readonly error: unknown; readonly fallback: string }) {
  if (!error) return null;
  const message = error instanceof ApiError ? error.message : fallback;
  return (
    <div
      role="alert"
      className="rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive ring-1 ring-destructive/20"
    >
      {message}
    </div>
  );
}
