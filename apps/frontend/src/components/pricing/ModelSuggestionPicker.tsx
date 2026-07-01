import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { formatUSD } from "@/lib/money";
import { pricingSuggestionsQueryOptions } from "@/lib/pricingSuggestions";
import { ProviderLogo } from "@/lib/providerLogos";
import type { PriceSuggestion } from "@/services/pricingSuggestionsService";

interface ModelSuggestionPickerProps {
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
  readonly onSelect: (suggestion: PriceSuggestion) => void;
}

export function ModelSuggestionPicker({ open, onOpenChange, onSelect }: ModelSuggestionPickerProps) {
  const { t } = useTranslation();
  const [search, setSearch] = useState("");
  const query = useQuery({ ...pricingSuggestionsQueryOptions, enabled: open });

  const groups = useMemo(() => {
    const byProvider = new Map<string, PriceSuggestion[]>();
    for (const s of query.data ?? []) {
      const list = byProvider.get(s.provider) ?? [];
      list.push(s);
      byProvider.set(s.provider, list);
    }
    return [...byProvider.entries()].sort(([a], [b]) => a.localeCompare(b));
  }, [query.data]);

  function handleOpenChange(next: boolean) {
    if (!next) setSearch("");
    onOpenChange(next);
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("pricing.suggestions.title")}</DialogTitle>
          <DialogDescription>{t("pricing.suggestions.caption")}</DialogDescription>
        </DialogHeader>
        <Command shouldFilter value={search} onValueChange={setSearch}>
          <CommandInput placeholder={t("pricing.suggestions.searchPlaceholder")} />
          <CommandList>
            <CommandEmpty>
              {query.isError ? t("pricing.suggestions.unavailable") : t("pricing.suggestions.empty")}
            </CommandEmpty>
            {groups.map(([provider, items]) => (
              <CommandGroup
                key={provider}
                heading={
                  <span className="flex items-center gap-1.5">
                    <ProviderLogo provider={provider} className="size-3.5" />
                    {provider}
                  </span>
                }
              >
                {items.map((s) => (
                  <CommandItem
                    key={`${s.provider}/${s.model}`}
                    value={`${s.provider} ${s.model}`}
                    onSelect={() => onSelect(s)}
                    className="justify-between"
                  >
                    <span className="font-mono text-xs">{s.model}</span>
                    <span className="text-xs text-muted-foreground">
                      {formatUSD(s.input_price_per_1m)} / {formatUSD(s.output_price_per_1m)}
                    </span>
                  </CommandItem>
                ))}
              </CommandGroup>
            ))}
          </CommandList>
        </Command>
      </DialogContent>
    </Dialog>
  );
}
