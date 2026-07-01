import { zodResolver } from "@hookform/resolvers/zod";
import { createFileRoute } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { LoaderCircle } from "lucide-react";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { z } from "zod";
import { Field } from "@/components/form/Field";
import { FormError } from "@/components/form/FormError";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { meQueryOptions } from "@/lib/auth";
import { changePassword, updateProfile } from "@/services/authService";

export const Route = createFileRoute("/_app/settings")({
  component: SettingsRoute,
});

function SettingsRoute() {
  const { t } = useTranslation();
  return (
    <div className="flex max-w-xl flex-col gap-6">
      <h1 className="text-2xl font-heading font-semibold">{t("nav.settings")}</h1>
      <ProfileCard />
      <PasswordCard />
    </div>
  );
}

function ProfileCard() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const me = useQuery(meQueryOptions);

  const schema = z.object({ name: z.string().min(1, t("settings.profile.errors.nameRequired")).max(100) });
  const form = useForm({ resolver: zodResolver(schema), defaultValues: { name: "" } });

  useEffect(() => {
    if (me.data) form.reset({ name: me.data.name });
  }, [me.data, form]);

  const mutation = useMutation({
    mutationFn: (values: { name: string }) => updateProfile(values),
    onSuccess: (updated) => {
      queryClient.setQueryData(meQueryOptions.queryKey, updated);
      toast.success(t("settings.profile.updated"));
    },
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("settings.profile.title")}</CardTitle>
        <CardDescription>{t("settings.profile.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={form.handleSubmit((values) => mutation.mutate(values))} noValidate className="space-y-4">
          <FormError error={mutation.error} fallback={t("auth.errors.generic")} />
          <Field id="settings-name" label={t("settings.profile.nameLabel")} error={form.formState.errors.name?.message}>
            <Input id="settings-name" aria-invalid={!!form.formState.errors.name} {...form.register("name")} />
          </Field>
          <Field id="settings-email" label={t("settings.profile.emailLabel")}>
            <Input id="settings-email" value={me.data?.email ?? ""} disabled readOnly />
          </Field>
          <Button type="submit" disabled={mutation.isPending}>
            {mutation.isPending ? <LoaderCircle className="size-4 animate-spin motion-reduce:animate-none" /> : null}
            {t("settings.profile.save")}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}

function PasswordCard() {
  const { t } = useTranslation();

  const schema = z
    .object({
      currentPassword: z.string().min(1, t("settings.password.errors.required")),
      newPassword: z.string().min(8, t("settings.password.errors.tooShort")).max(128),
      confirmPassword: z.string().min(1, t("settings.password.errors.required")),
    })
    .refine((v) => v.newPassword === v.confirmPassword, {
      message: t("settings.password.errors.mismatch"),
      path: ["confirmPassword"],
    });

  const form = useForm({
    resolver: zodResolver(schema),
    defaultValues: { currentPassword: "", newPassword: "", confirmPassword: "" },
  });

  const mutation = useMutation({
    mutationFn: (values: { currentPassword: string; newPassword: string }) =>
      changePassword({ current_password: values.currentPassword, new_password: values.newPassword }),
    onSuccess: () => {
      toast.success(t("settings.password.updated"));
      form.reset({ currentPassword: "", newPassword: "", confirmPassword: "" });
    },
  });

  const onSubmit = form.handleSubmit((values) => mutation.mutate(values));

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("settings.password.title")}</CardTitle>
        <CardDescription>{t("settings.password.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={onSubmit} noValidate className="space-y-4">
          <FormError error={mutation.error} fallback={t("auth.errors.generic")} />
          <Field
            id="settings-current-password"
            label={t("settings.password.currentLabel")}
            error={form.formState.errors.currentPassword?.message}
          >
            <Input
              id="settings-current-password"
              type="password"
              aria-invalid={!!form.formState.errors.currentPassword}
              {...form.register("currentPassword")}
            />
          </Field>
          <Field
            id="settings-new-password"
            label={t("settings.password.newLabel")}
            error={form.formState.errors.newPassword?.message}
          >
            <Input
              id="settings-new-password"
              type="password"
              aria-invalid={!!form.formState.errors.newPassword}
              {...form.register("newPassword")}
            />
          </Field>
          <Field
            id="settings-confirm-password"
            label={t("settings.password.confirmLabel")}
            error={form.formState.errors.confirmPassword?.message}
          >
            <Input
              id="settings-confirm-password"
              type="password"
              aria-invalid={!!form.formState.errors.confirmPassword}
              {...form.register("confirmPassword")}
            />
          </Field>
          <Button type="submit" disabled={mutation.isPending}>
            {mutation.isPending ? <LoaderCircle className="size-4 animate-spin motion-reduce:animate-none" /> : null}
            {t("settings.password.save")}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
