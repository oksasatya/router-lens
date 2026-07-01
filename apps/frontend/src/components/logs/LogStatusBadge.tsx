import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";
import type { Event } from "@/lib/schemas";

/** Renders is_error + status_code as a single semantic badge — color is the
 * existing destructive/secondary tokens, never a new ad-hoc hex. */
export function LogStatusBadge({ event }: { readonly event: Event }) {
  const { t } = useTranslation();
  if (event.is_error) {
    return <Badge variant="destructive">{event.status_code ?? t("logs.error")}</Badge>;
  }
  return <Badge variant="secondary">{event.status_code ?? t("logs.ok")}</Badge>;
}
