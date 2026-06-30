import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { Placeholder } from "@/components/Placeholder";

export const Route = createFileRoute("/_app/logs")({
  component: LogsRoute,
});

function LogsRoute() {
  const { t } = useTranslation();
  return <Placeholder title={t("nav.logs")} />;
}
