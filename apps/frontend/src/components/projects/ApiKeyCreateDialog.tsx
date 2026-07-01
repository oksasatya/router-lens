import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Copy, LoaderCircle } from "lucide-react";
import { useState } from "react";
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
import { createApiKey } from "@/services/apiKeyService";

interface ApiKeyCreateDialogProps {
  readonly projectId: string;
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
}

/** Two-step dialog: name the key, then reveal the plaintext exactly once. */
export function ApiKeyCreateDialog({ projectId, open, onOpenChange }: ApiKeyCreateDialogProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [createdKey, setCreatedKey] = useState<string | null>(null);

  const schema = z.object({ name: z.string().min(1, t("apiKeys.errors.nameRequired")).max(120) });
  const form = useForm({ resolver: zodResolver(schema), defaultValues: { name: "" } });

  const mutation = useMutation({
    mutationFn: (values: { name: string }) => createApiKey(projectId, values.name),
    onSuccess: (key) => {
      setCreatedKey(key.key);
      void queryClient.invalidateQueries({ queryKey: ["projects", projectId, "api-keys"] });
    },
  });

  function handleOpenChange(next: boolean) {
    if (!next) {
      setCreatedKey(null);
      form.reset();
      mutation.reset();
    }
    onOpenChange(next);
  }

  function copyKey() {
    if (!createdKey) return;
    void navigator.clipboard.writeText(createdKey);
    toast.success(t("apiKeys.createdTitle"));
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        {createdKey ? (
          <>
            <DialogHeader>
              <DialogTitle>{t("apiKeys.createdTitle")}</DialogTitle>
              <DialogDescription>{t("apiKeys.createdWarning")}</DialogDescription>
            </DialogHeader>
            <div className="rounded-lg bg-muted px-3 py-2 font-mono text-xs break-all">{createdKey}</div>
            <DialogFooter>
              <Button variant="outline" onClick={copyKey}>
                <Copy className="size-4" />
                {t("common.copy")}
              </Button>
              <Button onClick={() => handleOpenChange(false)}>{t("common.done")}</Button>
            </DialogFooter>
          </>
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>{t("apiKeys.newTitle")}</DialogTitle>
              <DialogDescription>{t("apiKeys.newDescription")}</DialogDescription>
            </DialogHeader>
            <form onSubmit={form.handleSubmit((v) => mutation.mutate(v))} noValidate className="space-y-4">
              <FormError error={mutation.error} fallback={t("auth.errors.generic")} />
              <Field id="key-name" label={t("apiKeys.fields.name")} error={form.formState.errors.name?.message}>
                <Input
                  id="key-name"
                  autoFocus
                  aria-invalid={!!form.formState.errors.name}
                  {...form.register("name")}
                />
              </Field>
              <DialogFooter>
                <Button type="submit" disabled={mutation.isPending}>
                  {mutation.isPending ? (
                    <LoaderCircle className="size-4 animate-spin motion-reduce:animate-none" />
                  ) : null}
                  {t("apiKeys.create")}
                </Button>
              </DialogFooter>
            </form>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
