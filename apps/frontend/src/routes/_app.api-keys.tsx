import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { Placeholder } from "@/components/Placeholder";

export const Route = createFileRoute("/_app/api-keys")({
  component: ApiKeysRoute,
});

function ApiKeysRoute() {
  const { t } = useTranslation();
  return <Placeholder title={t("nav.apiKeys")} />;
}
