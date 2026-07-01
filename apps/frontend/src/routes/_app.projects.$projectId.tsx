import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { ArrowLeft, KeyRound, MoreHorizontal, Plus } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { ApiKeyCreateDialog } from "@/components/projects/ApiKeyCreateDialog";
import { ProjectFormDialog } from "@/components/projects/ProjectFormDialog";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { DataTable, type DataTableColumn } from "@/components/DataTable";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { apiKeysQueryOptions } from "@/lib/apikeys";
import { formatRelative, formatTimestamp } from "@/lib/date";
import { projectQueryOptions } from "@/lib/projects";
import type { ApiKey } from "@/lib/schemas";
import { revokeApiKey } from "@/services/apiKeyService";
import { deleteProject } from "@/services/projectService";

export const Route = createFileRoute("/_app/projects/$projectId")({
  component: ProjectDetailRoute,
});

function ProjectDetailRoute() {
  const { t } = useTranslation();
  const { projectId } = Route.useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data: project } = useQuery(projectQueryOptions(projectId));
  const { data: keyRows, isLoading: keysLoading } = useQuery(apiKeysQueryOptions(projectId));

  const [editOpen, setEditOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [keyDialogOpen, setKeyDialogOpen] = useState(false);
  const [revoking, setRevoking] = useState<ApiKey | null>(null);

  const deleteMutation = useMutation({
    mutationFn: () => deleteProject(projectId),
    onSuccess: async () => {
      toast.success(t("projects.deleted"));
      await queryClient.invalidateQueries({ queryKey: ["projects"] });
      await navigate({ to: "/projects", search: { page: 1 } });
    },
  });

  const revokeMutation = useMutation({
    mutationFn: revokeApiKey,
    onSuccess: () => {
      toast.success(t("apiKeys.revoked"));
      setRevoking(null);
      void queryClient.invalidateQueries({ queryKey: ["projects", projectId, "api-keys"] });
    },
  });

  const keyColumns: DataTableColumn<ApiKey>[] = [
    { key: "name", header: t("apiKeys.fields.name"), cell: (k) => k.name },
    {
      key: "prefix",
      header: t("apiKeys.fields.prefix"),
      cell: (k) => <span className="font-mono text-xs">{k.key_prefix}…</span>,
    },
    {
      key: "lastUsed",
      header: t("apiKeys.fields.lastUsed"),
      cell: (k) => (k.last_used_at ? formatRelative(k.last_used_at) : t("apiKeys.neverUsed")),
    },
    { key: "createdAt", header: t("apiKeys.fields.createdAt"), cell: (k) => formatTimestamp(k.created_at) },
    {
      key: "status",
      header: t("apiKeys.fields.status"),
      cell: (k) =>
        k.revoked_at ? (
          <span className="text-destructive">{t("apiKeys.revokedStatus")}</span>
        ) : (
          <span className="text-emerald-600 dark:text-emerald-400">{t("apiKeys.activeStatus")}</span>
        ),
    },
    {
      key: "actions",
      header: "",
      className: "w-10",
      cell: (k) =>
        k.revoked_at ? null : (
          <DropdownMenu>
            <DropdownMenuTrigger render={<Button variant="ghost" size="icon" aria-label={t("common.menu")} />}>
              <MoreHorizontal className="size-4" />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem variant="destructive" onClick={() => setRevoking(k)}>
                {t("apiKeys.revoke")}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ),
    },
  ];

  // ponytail: no skeleton for the pre-load gap and the back-link below always
  // returns to page 1 (not the page the user came from) — both noted in FE-03
  // task reviews as optional polish; deferred, add if it becomes a real
  // complaint rather than build a skeleton/origin-page-passthrough speculatively.
  if (!project) return null;

  return (
    <div className="space-y-6">
      <Link to="/projects" search={{ page: 1 }} className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
        <ArrowLeft className="size-4" />
        {t("nav.projects")}
      </Link>

      <Card className="flex items-start justify-between p-6">
        <div>
          <h1 className="font-heading text-2xl font-semibold tracking-tight">{project.name}</h1>
          <p className="mt-1 text-sm text-muted-foreground">{project.slug}</p>
          <p className="mt-2 max-w-prose text-sm text-muted-foreground">
            {project.description || t("projects.noDescription")}
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
            {t("common.edit")}
          </Button>
          <Button variant="destructive" size="sm" onClick={() => setDeleteOpen(true)}>
            {t("common.delete")}
          </Button>
        </div>
      </Card>

      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="flex items-center gap-2 font-heading text-lg font-medium">
            <KeyRound className="size-4" />
            {t("nav.apiKeys")}
          </h2>
          <Button size="sm" onClick={() => setKeyDialogOpen(true)}>
            <Plus className="size-4" />
            {t("apiKeys.new")}
          </Button>
        </div>

        <DataTable
          rows={keyRows ?? []}
          rowKey={(k) => k.id}
          isLoading={keysLoading}
          emptyMessage={t("apiKeys.empty")}
          columns={keyColumns}
        />
      </div>

      <ProjectFormDialog open={editOpen} onOpenChange={setEditOpen} project={project} />
      <ApiKeyCreateDialog projectId={projectId} open={keyDialogOpen} onOpenChange={setKeyDialogOpen} />

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t("projects.deleteTitle")}
        description={t("projects.deleteDescription", { name: project.name })}
        confirmLabel={t("common.delete")}
        cancelLabel={t("common.cancel")}
        destructive
        loading={deleteMutation.isPending}
        onConfirm={() => deleteMutation.mutate()}
      />
      <ConfirmDialog
        open={!!revoking}
        onOpenChange={(o) => !o && setRevoking(null)}
        title={t("apiKeys.revokeTitle")}
        description={t("apiKeys.revokeDescription", { name: revoking?.name })}
        confirmLabel={t("apiKeys.revoke")}
        cancelLabel={t("common.cancel")}
        destructive
        loading={revokeMutation.isPending}
        onConfirm={() => revoking && revokeMutation.mutate(revoking.id)}
      />
    </div>
  );
}
