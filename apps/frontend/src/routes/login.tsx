import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { Card } from "@/components/ui/card";

export const Route = createFileRoute("/login")({
  component: LoginRoute,
});

// Placeholder — the real login form + first-run setup wizard land in FE Plan 02.
// This route must exist so the protected-layout guard can redirect here.
function LoginRoute() {
  const { t } = useTranslation();
  return (
    <div className="grid min-h-svh place-items-center bg-background p-6 text-foreground">
      <Card className="w-full max-w-sm p-8 text-center">
        <img src="/logo.webp" alt="" className="mx-auto size-10 rounded-lg" />
        <h1 className="mt-4 font-heading text-2xl font-semibold tracking-tight">{t("app.name")}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t("app.tagline")}</p>
        <p className="mt-6 text-sm text-muted-foreground">
          {t("common.signIn")} — {t("common.comingSoon")}
        </p>
      </Card>
    </div>
  );
}
