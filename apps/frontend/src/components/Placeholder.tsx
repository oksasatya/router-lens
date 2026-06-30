import { Crosshair } from "lucide-react";
import { useTranslation } from "react-i18next";

/**
 * Empty-state placeholder for routes whose screens land in later FE plans.
 * The reticle (Crosshair) echoes the logo's scope motif — the brand signature.
 */
export function Placeholder({ title }: { readonly title: string }) {
  const { t } = useTranslation();
  return (
    <div className="grid place-items-center rounded-xl border border-dashed border-border py-24 text-center">
      <Crosshair className="size-8 text-muted-foreground/40" strokeWidth={1.5} />
      <p className="mt-3 font-heading text-lg font-medium">{title}</p>
      <p className="text-sm text-muted-foreground">{t("common.comingSoon")}</p>
    </div>
  );
}
