import { ChevronLeft, ChevronRight } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";

interface OffsetPaginationProps {
  readonly page: number;
  readonly limit: number;
  readonly total: number;
  readonly onPageChange: (page: number) => void;
}

export function OffsetPagination({ page, limit, total, onPageChange }: OffsetPaginationProps) {
  const { t } = useTranslation();
  const totalPages = Math.max(1, Math.ceil(total / limit));
  return (
    <div className="flex items-center justify-between text-sm text-muted-foreground">
      <span>{t("common.pageOf", { page, totalPages })}</span>
      <div className="flex gap-2">
        <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => onPageChange(page - 1)}>
          <ChevronLeft className="size-4" />
          {t("common.previous")}
        </Button>
        <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => onPageChange(page + 1)}>
          {t("common.next")}
          <ChevronRight className="size-4" />
        </Button>
      </div>
    </div>
  );
}
