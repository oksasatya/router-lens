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
import { login } from "@/services/authService";

export const Route = createFileRoute("/login")({
  // Route to first-run setup if no admin exists; bounce to the dashboard if
  // already signed in. The /auth/me probe is cookie-driven, so .catch swallows
  // the expected 401 (the interceptor leaves auth-entry 401s alone).
  beforeLoad: async ({ context }) => {
    const status = await context.queryClient.ensureQueryData(setupStatusQueryOptions);
    if (status.needs_setup) throw redirect({ to: "/setup" });
    const me = await context.queryClient.ensureQueryData(meQueryOptions).catch(() => null);
    if (me) throw redirect({ to: "/" });
  },
  component: LoginRoute,
});

function LoginRoute() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const schema = z.object({
    email: z.email(t("auth.errors.email")),
    password: z.string().min(1, t("auth.errors.passwordRequired")),
  });

  const form = useForm({
    resolver: zodResolver(schema),
    defaultValues: { email: "", password: "" },
  });

  const mutation = useMutation({
    mutationFn: login,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: meQueryOptions.queryKey });
      toast.success(t("auth.login.success"));
      await navigate({ to: "/" });
    },
  });

  const onSubmit = form.handleSubmit((values) => mutation.mutate(values));

  return (
    <AuthLayout title={t("auth.login.title")} subtitle={t("auth.login.subtitle")}>
      <form onSubmit={onSubmit} noValidate className="space-y-4">
        <FormError error={mutation.error} fallback={t("auth.errors.generic")} />
        <Field id="email" label={t("auth.fields.email")} error={form.formState.errors.email?.message}>
          <Input
            id="email"
            type="email"
            autoComplete="email"
            autoFocus
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
            autoComplete="current-password"
            className="h-11"
            aria-invalid={!!form.formState.errors.password}
            {...form.register("password")}
          />
        </Field>
        <Button type="submit" className="h-11 w-full" disabled={mutation.isPending}>
          {mutation.isPending ? (
            <LoaderCircle className="size-4 animate-spin motion-reduce:animate-none" />
          ) : null}
          {t("auth.login.submit")}
        </Button>
      </form>
    </AuthLayout>
  );
}
