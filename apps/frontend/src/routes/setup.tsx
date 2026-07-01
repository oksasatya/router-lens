import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router";
import { LoaderCircle } from "lucide-react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { z } from "zod";
import { AuthLayout } from "@/components/auth/AuthLayout";
import { Field } from "@/components/form/Field";
import { FormError } from "@/components/form/FormError";
import { PasswordInput } from "@/components/auth/PasswordInput";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { meQueryOptions, setupStatusQueryOptions } from "@/lib/auth";
import { login, setup, type SetupInput } from "@/services/authService";

export const Route = createFileRoute("/setup")({
  // First-run only: once an admin exists, setup is locked — send them to login.
  beforeLoad: async ({ context }) => {
    const status = await context.queryClient.ensureQueryData(setupStatusQueryOptions);
    if (!status.needs_setup) throw redirect({ to: "/login" });
  },
  component: SetupRoute,
});

function SetupRoute() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const schema = z.object({
    name: z.string().max(100),
    email: z.email(t("auth.errors.email")),
    password: z.string().min(8, t("auth.errors.passwordMin")).max(128),
  });

  const form = useForm({
    resolver: zodResolver(schema),
    defaultValues: { name: "", email: "", password: "" },
  });

  const mutation = useMutation({
    // Create the admin, then sign in with the same credentials so first-run
    // lands straight in the dashboard instead of bouncing through /login.
    mutationFn: async (input: SetupInput) => {
      await setup(input);
      await login({ email: input.email, password: input.password });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: setupStatusQueryOptions.queryKey });
      await queryClient.invalidateQueries({ queryKey: meQueryOptions.queryKey });
      toast.success(t("auth.setup.success"));
      await navigate({ to: "/" });
    },
  });

  const onSubmit = form.handleSubmit((values) => mutation.mutate(values));

  return (
    <AuthLayout title={t("auth.setup.title")} subtitle={t("auth.setup.subtitle")}>
      <form onSubmit={onSubmit} noValidate className="space-y-4">
        <FormError error={mutation.error} fallback={t("auth.errors.generic")} />
        <Field id="name" label={t("auth.fields.name")} error={form.formState.errors.name?.message}>
          <Input
            id="name"
            autoComplete="name"
            autoFocus
            placeholder={t("auth.fields.namePlaceholder")}
            className="h-11"
            {...form.register("name")}
          />
        </Field>
        <Field id="email" label={t("auth.fields.email")} error={form.formState.errors.email?.message}>
          <Input
            id="email"
            type="email"
            autoComplete="email"
            className="h-11"
            aria-invalid={!!form.formState.errors.email}
            {...form.register("email")}
          />
        </Field>
        <Field
          id="password"
          label={t("auth.fields.password")}
          error={form.formState.errors.password?.message}
        >
          <PasswordInput
            id="password"
            autoComplete="new-password"
            className="h-11"
            aria-invalid={!!form.formState.errors.password}
            aria-describedby="password-hint"
            {...form.register("password")}
          />
          <p id="password-hint" className="text-xs text-muted-foreground">
            {t("auth.fields.passwordHint")}
          </p>
        </Field>
        <Button type="submit" className="h-11 w-full" disabled={mutation.isPending}>
          {mutation.isPending ? (
            <LoaderCircle className="size-4 animate-spin motion-reduce:animate-none" />
          ) : null}
          {t("auth.setup.submit")}
        </Button>
      </form>
    </AuthLayout>
  );
}
