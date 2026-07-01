import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { MoreHorizontal, Plus, Sparkles } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { DataTable, type DataTableColumn } from "@/components/DataTable";
import { ModelSuggestionPicker } from "@/components/pricing/ModelSuggestionPicker";
import { PricingFormDialog } from "@/components/pricing/PricingFormDialog";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { formatTimestamp } from "@/lib/date";
import { formatUSD } from "@/lib/money";
import { pricingQueryOptions } from "@/lib/pricing";
import type { PricingRule } from "@/lib/schemas";
import { deletePricing } from "@/services/pricingService";

export const Route = createFileRoute("/_app/pricing")({
  component: PricingRoute,
});

function PricingRoute() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { data, isLoading } = useQuery(pricingQueryOptions);

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<PricingRule | null>(null);
  const [deleting, setDeleting] = useState<PricingRule | null>(null);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [suggestionDefaults, setSuggestionDefaults] = useState<{
    provider: string;
    model: string;
    input_price_per_1m: string;
    output_price_per_1m: string;
  } | null>(null);

  const deleteMutation = useMutation({
    mutationFn: deletePricing,
    onSuccess: () => {
      toast.success(t("pricing.deleted"));
      setDeleting(null);
      void queryClient.invalidateQueries({ queryKey: ["pricing"] });
    },
  });

  const columns: DataTableColumn<PricingRule>[] = [
    { key: "provider", header: t("pricing.fields.provider"), cell: (p) => p.provider },
    { key: "model", header: t("pricing.fields.model"), cell: (p) => <span className="font-mono text-xs">{p.model}</span> },
    { key: "input", header: t("pricing.fields.inputPrice"), cell: (p) => formatUSD(p.input_price_per_1m) },
    { key: "output", header: t("pricing.fields.outputPrice"), cell: (p) => formatUSD(p.output_price_per_1m) },
    { key: "updatedAt", header: t("pricing.fields.updatedAt"), cell: (p) => formatTimestamp(p.updated_at) },
    {
      key: "actions",
      header: "",
      className: "w-10",
      cell: (p) => (
        <DropdownMenu>
          <DropdownMenuTrigger render={<Button variant="ghost" size="icon" aria-label={t("common.menu")} />}>
            <MoreHorizontal className="size-4" />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              onClick={() => {
                setEditing(p);
                setFormOpen(true);
              }}
            >
              {t("common.edit")}
            </DropdownMenuItem>
            <DropdownMenuItem variant="destructive" onClick={() => setDeleting(p)}>
              {t("common.delete")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
    },
  ];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="font-heading text-2xl font-semibold tracking-tight">{t("nav.pricing")}</h1>
        <div className="flex gap-2">
          <Button
            variant="outline"
            onClick={() => setPickerOpen(true)}
          >
            <Sparkles className="size-4" />
            {t("pricing.suggestions.browse")}
          </Button>
          <Button
            onClick={() => {
              setEditing(null);
              setFormOpen(true);
            }}
          >
            <Plus className="size-4" />
            {t("pricing.new")}
          </Button>
        </div>
      </div>

      <DataTable
        rows={data ?? []}
        rowKey={(p) => p.id}
        isLoading={isLoading}
        emptyMessage={t("pricing.empty")}
        columns={columns}
      />

      <ModelSuggestionPicker
        open={pickerOpen}
        onOpenChange={setPickerOpen}
        onSelect={(s) => {
          setPickerOpen(false);
          setEditing(null);
          setSuggestionDefaults(s);
          setFormOpen(true);
        }}
      />

      <PricingFormDialog
        open={formOpen}
        onOpenChange={(o) => {
          setFormOpen(o);
          if (!o) setSuggestionDefaults(null);
        }}
        rule={editing}
        defaultValues={suggestionDefaults}
      />

      <ConfirmDialog
        open={!!deleting}
        onOpenChange={(o) => !o && setDeleting(null)}
        title={t("pricing.deleteTitle")}
        description={t("pricing.deleteDescription", { provider: deleting?.provider, model: deleting?.model })}
        confirmLabel={t("common.delete")}
        cancelLabel={t("common.cancel")}
        destructive
        loading={deleteMutation.isPending}
        onConfirm={() => deleting && deleteMutation.mutate(deleting.id)}
      />
    </div>
  );
}
