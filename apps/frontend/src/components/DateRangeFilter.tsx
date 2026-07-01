import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export type DateRangePreset = "24h" | "7d" | "30d";

interface DateRangeFilterProps {
  readonly value: DateRangePreset;
  readonly onChange: (preset: DateRangePreset) => void;
}

const PRESETS: readonly DateRangePreset[] = ["24h", "7d", "30d"];

/**
 * A 3-button segmented preset toggle — the backend's datetime.ParseRange only
 * accepts these three presets (or explicit from/to, unused here). Not a
 * calendar picker; see the plan's Global Constraints for why.
 */
export function DateRangeFilter({ value, onChange }: DateRangeFilterProps) {
  const { t } = useTranslation();
  return (
    <div role="group" aria-label={t("common.dateRange")} className="inline-flex rounded-md border border-border">
      {PRESETS.map((preset) => (
        <Button
          key={preset}
          type="button"
          variant="ghost"
          size="sm"
          aria-pressed={value === preset}
          onClick={() => onChange(preset)}
          className={cn(
            "rounded-none first:rounded-l-md last:rounded-r-md border-r border-border last:border-r-0",
            value === preset && "bg-accent text-accent-foreground",
          )}
        >
          {t(`common.preset.${preset}`)}
        </Button>
      ))}
    </div>
  );
}
