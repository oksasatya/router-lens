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
import { Textarea } from "@/components/ui/textarea";
import type { Project } from "@/lib/schemas";
import { createProject, updateProject } from "@/services/projectService";

interface ProjectFormDialogProps {
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
  readonly project: Project | null;
}

/** Create-or-edit dialog for a project, shared by the list and detail routes. */
export function ProjectFormDialog({ open, onOpenChange, project }: ProjectFormDialogProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const isEdit = !!project;

  const schema = z.object({
    name: z.string().min(1, t("projects.errors.nameRequired")).max(120),
    description: z.string().max(500),
  });

  const form = useForm({
    resolver: zodResolver(schema),
    defaultValues: { name: "", description: "" },
  });

  // ponytail: react-doctor flags this as "event logic in an effect" — it's not;
  // this resyncs form state to the `open`/`project` props each time the dialog
  // opens (React's own documented use of useEffect), not a faked event handler.
  useEffect(() => {
    if (open) form.reset({ name: project?.name ?? "", description: project?.description ?? "" });
  }, [open, project, form]);

  const mutation = useMutation({
    mutationFn: (values: { name: string; description: string }) =>
      isEdit ? updateProject(project.id, values) : createProject(values),
    onSuccess: () => {
      toast.success(t(isEdit ? "projects.updated" : "projects.created"));
      void queryClient.invalidateQueries({ queryKey: ["projects"] });
      onOpenChange(false);
    },
  });

  const onSubmit = form.handleSubmit((values) => mutation.mutate(values));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t(isEdit ? "projects.editTitle" : "projects.newTitle")}</DialogTitle>
          <DialogDescription>{t("projects.formDescription")}</DialogDescription>
        </DialogHeader>
        <form onSubmit={onSubmit} noValidate className="space-y-4">
          <FormError error={mutation.error} fallback={t("auth.errors.generic")} />
          <Field id="project-name" label={t("projects.fields.name")} error={form.formState.errors.name?.message}>
            <Input id="project-name" autoFocus aria-invalid={!!form.formState.errors.name} {...form.register("name")} />
          </Field>
          <Field
            id="project-description"
            label={t("projects.fields.description")}
            error={form.formState.errors.description?.message}
          >
            <Textarea id="project-description" rows={3} {...form.register("description")} />
          </Field>
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
