import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { MoreHorizontal, Plus } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { DataTable, type DataTableColumn } from "@/components/DataTable";
import { OffsetPagination } from "@/components/OffsetPagination";
import { ProjectFormDialog } from "@/components/projects/ProjectFormDialog";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { ApiError } from "@/lib/api";
import { formatTimestamp } from "@/lib/date";
import { projectsQueryOptions } from "@/lib/projects";
import type { Project } from "@/lib/schemas";
import { deleteProject } from "@/services/projectService";

const LIMIT = 20;

export const Route = createFileRoute("/_app/projects")({
  validateSearch: (search: Record<string, unknown>) => ({
    page: Number(search.page) > 0 ? Number(search.page) : 1,
  }),
  component: ProjectsRoute,
});

function ProjectsRoute() {
  const { t } = useTranslation();
  const navigate = Route.useNavigate();
  const { page } = Route.useSearch();
  const queryClient = useQueryClient();
  const query = useQuery({ ...projectsQueryOptions(page, LIMIT), placeholderData: keepPreviousData });

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Project | null>(null);
  const [deleting, setDeleting] = useState<Project | null>(null);

  const deleteMutation = useMutation({
    mutationFn: deleteProject,
    onSuccess: () => {
      toast.success(t("projects.deleted"));
      setDeleting(null);
      void queryClient.invalidateQueries({ queryKey: ["projects"] });
    },
    onError: (error) => {
      const message = error instanceof ApiError ? error.message : t("auth.errors.generic");
      toast.error(message);
    },
  });

  const columns: DataTableColumn<Project>[] = [
    {
      key: "name",
      header: t("projects.fields.name"),
      cell: (p) => (
        <Link to="/projects/$projectId" params={{ projectId: p.id }} search={{ page: 1 }} className="font-medium hover:underline">
          {p.name}
        </Link>
      ),
    },
    { key: "slug", header: t("projects.fields.slug"), cell: (p) => <span className="text-muted-foreground">{p.slug}</span> },
    {
      key: "description",
      header: t("projects.fields.description"),
      cell: (p) => <span className="line-clamp-1 text-muted-foreground">{p.description || t("projects.noDescription")}</span>,
    },
    { key: "createdAt", header: t("projects.fields.createdAt"), cell: (p) => formatTimestamp(p.created_at) },
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
        <h1 className="font-heading text-2xl font-semibold tracking-tight">{t("nav.projects")}</h1>
        <Button
          onClick={() => {
            setEditing(null);
            setFormOpen(true);
          }}
        >
          <Plus className="size-4" />
          {t("projects.new")}
        </Button>
      </div>

      <DataTable
        rows={query.data?.items ?? []}
        rowKey={(p) => p.id}
        isLoading={query.isLoading}
        emptyMessage={t("projects.empty")}
        columns={columns}
      />

      {query.data && (
        <OffsetPagination
          page={query.data.pagination.page}
          limit={query.data.pagination.limit}
          total={query.data.pagination.total}
          onPageChange={(p) => navigate({ search: { page: p } })}
        />
      )}

      <ProjectFormDialog open={formOpen} onOpenChange={setFormOpen} project={editing} />

      <ConfirmDialog
        open={!!deleting}
        onOpenChange={(o) => !o && setDeleting(null)}
        title={t("projects.deleteTitle")}
        description={t("projects.deleteDescription", { name: deleting?.name })}
        confirmLabel={t("common.delete")}
        cancelLabel={t("common.cancel")}
        destructive
        loading={deleteMutation.isPending}
        onConfirm={() => deleting && deleteMutation.mutate(deleting.id)}
      />
    </div>
  );
}
