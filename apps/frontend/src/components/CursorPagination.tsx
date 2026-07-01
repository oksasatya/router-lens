import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";

interface CursorPaginationProps {
  /** True when the current page's response returned a non-empty next_cursor. */
  readonly hasMore: boolean;
  /** True once at least one page back exists (cursor stack is non-empty). */
  readonly hasPrevious: boolean;
  readonly onNext: () => void;
  readonly onPrevious: () => void;
  readonly isLoading?: boolean;
}

/**
 * Forward-cursor pager: the backend only ever gives a next_cursor, never a
 * previous one. "Previous" is implemented by the CALLER maintaining a stack
 * of visited cursors (see the logs route, Task 2) — this component is
 * presentation-only, it does not own the stack.
 */
export function CursorPagination({ hasMore, hasPrevious, onNext, onPrevious, isLoading }: CursorPaginationProps) {
  const { t } = useTranslation();
  return (
    <div className="flex items-center justify-end gap-2">
      <Button variant="outline" size="sm" disabled={!hasPrevious || isLoading} onClick={onPrevious}>
        {t("common.previous")}
      </Button>
      <Button variant="outline" size="sm" disabled={!hasMore || isLoading} onClick={onNext}>
        {t("common.next")}
      </Button>
    </div>
  );
}
