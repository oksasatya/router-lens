import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { LoaderCircle } from "lucide-react";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { z } from "zod";
import { Field } from "@/components/form/Field";
import { FormError } from "@/components/form/FormError";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import type { PricingRule } from "@/lib/schemas";
import { createPricing, updatePricing } from "@/services/pricingService";

interface PricingFormDialogProps {
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
  readonly rule: PricingRule | null;
  readonly defaultValues?: {
    provider: string;
    model: string;
    input_price_per_1m: string;
    output_price_per_1m: string;
  } | null;
}

function isNonNegativeNumber(value: string): boolean {
  const n = Number(value);
  return !Number.isNaN(n) && n >= 0;
}

export function PricingFormDialog({ open, onOpenChange, rule, defaultValues = null }: PricingFormDialogProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const isEdit = !!rule;

  const schema = z.object({
    provider: z.string().min(1, t("pricing.errors.providerRequired")).max(100),
    model: z.string().min(1, t("pricing.errors.modelRequired")).max(200),
    input_price_per_1m: z
      .string()
      .min(1, t("pricing.errors.priceRequired"))
      .refine(isNonNegativeNumber, t("pricing.errors.priceInvalid")),
    output_price_per_1m: z
      .string()
      .min(1, t("pricing.errors.priceRequired"))
      .refine(isNonNegativeNumber, t("pricing.errors.priceInvalid")),
  });

  const form = useForm({
    resolver: zodResolver(schema),
    defaultValues: { provider: "", model: "", input_price_per_1m: "", output_price_per_1m: "" },
  });

  // ponytail: react-doctor flags this as "event logic in an effect" — it's not;
  // this resyncs form state to the `open`/`rule`/`defaultValues` props each time the dialog
  // opens (React's own documented use of useEffect), not a faked event handler.
  useEffect(() => {
    if (open) {
      form.reset({
        provider: rule?.provider ?? defaultValues?.provider ?? "",
        model: rule?.model ?? defaultValues?.model ?? "",
        input_price_per_1m: rule?.input_price_per_1m ?? defaultValues?.input_price_per_1m ?? "",
        output_price_per_1m: rule?.output_price_per_1m ?? defaultValues?.output_price_per_1m ?? "",
      });
    }
  }, [open, rule, defaultValues, form]);

  const mutation = useMutation({
    mutationFn: async (values: {
      provider: string;
      model: string;
      input_price_per_1m: string;
      output_price_per_1m: string;
    }) => {
      if (isEdit) {
        await updatePricing(rule.id, values);
      } else {
        await createPricing(values);
      }
    },
    onSuccess: () => {
      toast.success(t(isEdit ? "pricing.updated" : "pricing.created"));
      void queryClient.invalidateQueries({ queryKey: ["pricing"] });
      onOpenChange(false);
    },
  });

  const onSubmit = form.handleSubmit((values) => mutation.mutate(values));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t(isEdit ? "pricing.editTitle" : "pricing.newTitle")}</DialogTitle>
          <DialogDescription>{t("pricing.formDescription")}</DialogDescription>
        </DialogHeader>
        <form onSubmit={onSubmit} noValidate className="space-y-4">
          <FormError error={mutation.error} fallback={t("auth.errors.generic")} />
          <Field id="provider" label={t("pricing.fields.provider")} error={form.formState.errors.provider?.message}>
            <Input id="provider" autoFocus aria-invalid={!!form.formState.errors.provider} {...form.register("provider")} />
          </Field>
          <Field id="model" label={t("pricing.fields.model")} error={form.formState.errors.model?.message}>
            <Input id="model" aria-invalid={!!form.formState.errors.model} {...form.register("model")} />
          </Field>
          <div className="grid grid-cols-2 gap-4">
            <Field
              id="input-price"
              label={t("pricing.fields.inputPrice")}
              error={form.formState.errors.input_price_per_1m?.message}
            >
              <Input
                id="input-price"
                inputMode="decimal"
                aria-invalid={!!form.formState.errors.input_price_per_1m}
                {...form.register("input_price_per_1m")}
              />
            </Field>
            <Field
              id="output-price"
              label={t("pricing.fields.outputPrice")}
              error={form.formState.errors.output_price_per_1m?.message}
            >
              <Input
                id="output-price"
                inputMode="decimal"
                aria-invalid={!!form.formState.errors.output_price_per_1m}
                {...form.register("output_price_per_1m")}
              />
            </Field>
          </div>
          <DialogFooter>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? (
                <LoaderCircle className="size-4 animate-spin motion-reduce:animate-none" />
              ) : null}
              {t(isEdit ? "common.save" : "common.create")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
