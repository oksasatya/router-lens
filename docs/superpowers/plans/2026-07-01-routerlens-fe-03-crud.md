# RouterLens FE Plan 03 — Projects, API Keys, Pricing CRUD

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the `/projects` and `/pricing` placeholder routes with real CRUD screens, add a new `/projects/$projectId` detail route that hosts per-project API key management, and retire the flat `/api-keys` placeholder (the backend has no endpoint to list keys across projects — keys are only ever listed per project).

**Architecture:** Each domain (projects, api-keys, pricing) gets a zod schema in `lib/schemas.ts`, a thin service module (`services/*Service.ts`, axios + schema.parse, no branching logic), a TanStack Query `queryOptions` factory in `lib/<domain>.ts` (mirrors the existing `lib/auth.ts` pattern — no custom `use*` hook wrapper; components call `useQuery(xQueryOptions(...))` directly). Screens are plain function components using `react-hook-form` + `zod` + the existing `Field`/`FormError` pattern from `login.tsx`/`setup.tsx` (relocated to a neutral `components/form/` since they're no longer auth-only). Lists render through one reusable `<DataTable>` + `<OffsetPagination>`; destructive actions go through one reusable `<ConfirmDialog>`.

**Tech Stack:** Existing FE stack only — no new runtime dependency. Adds two shadcn primitives (`alert-dialog`, `textarea`) via the shadcn CLI.

## Global Constraints

- **Tailwind v4 ONLY (HARD).** No v3-era utilities (`bg-gradient-*`, bare `ring`, legacy `shadow`/`rounded` scale, `flex-shrink-*`, `bg-opacity-*`).
- **shadcn/ui, Base UI flavor (not Radix).** This project's `components.json` style is `base-nova` on `@base-ui/react` — primitives use the **`render={<Component />}`** prop to swap the rendered element (see `dialog.tsx`'s `DialogClose`), **not** Radix's `asChild`. Every new composition in this plan (`DropdownMenuTrigger`, `AlertDialogAction`, `AlertDialogCancel`) uses `render`, never `asChild`.
- **Anti-duplication (project rule):** one axios instance (already `lib/api.ts`); per-domain service + `queryOptions` factory; components never call axios directly. Reuse `<DataTable>`/`<OffsetPagination>`/`<ConfirmDialog>` across all three screens — no per-screen table/pagination/confirm-dialog reimplementation.
- **ponytail (YAGNI):** `<DataTable>` is a plain mapped `<table>` using the existing shadcn `Table` primitives — **not** `@tanstack/react-table**. These are small, offset-paginated CRUD lists with no sort/filter requirement; pulling in a table engine now is unjustified weight. `// ponytail:` comment marks the upgrade path for when the logs page (later FE plan) needs sort/column-filter at volume.
- **API surface is locked (CLAUDE.md §5) — do not invent endpoints.** Api-keys are nested under a project (`POST/GET /projects/:projectId/api-keys`, `DELETE /api-keys/:id`) — there is no flat "list all keys" endpoint, so there is no flat `/api-keys` screen either (decision made with the user this session).
- **i18n:** every user-facing string via `t("...")`, added to both `en.json` and `id.json` in the same step.
- **Sonar-TS/React (write compliant from the first commit):** props `readonly` (S6759) · `globalThis` not `window` (S7764) · no nested/identical ternaries (S3358/S3923) · optional chaining + `??=` (S6582/S6606) · `arr.at(-1)` (S7755) · real elements over ARIA roles (S6819) · **stable, explicit keys, never derived/positional** (S6479) — `DataTableColumn` carries its own `key: string` field for exactly this reason.
- **Quality floor (frontend-design):** responsive (dialogs scroll on small screens, table wrapped in horizontal-scroll container — already built into the shared `Table` primitive), visible keyboard focus, `prefers-reduced-motion` respected on the spinner (`motion-reduce:animate-none`, matching `login.tsx`). Behind-auth → performance gate applies, SEO gate does not.

### Skill chain

> Invoke `frontend-design` (auto-chains `ui-ux-pro-max` + `frontend-senior` + `tailwind-4` + `tailwind-design-system` + `tailwind-responsive-design` + `color-expert`) **plus `vercel:shadcn`**. The design system is already locked (Slate Aurora, FE Plan 01) — this plan is a **refine/implement pass within it**, not a greenfield craft pass; no `impeccable` needed. Apply `ponytail` (YAGNI, see Global Constraints). Finish each task's manual verification, and the whole plan with `react-doctor` (Task 5).

### TDD verdicts (§16)

- `DataTable` / `OffsetPagination` — **TDD: yes-ish.** Each has real branch logic (loading vs empty vs rows; boundary-disabled buttons) worth a small RTL regression test, written in Task 1. Not full TDD ceremony (no pre-existing red/green cycle demanded) but the test ships in the same task, not "later."
- `ConfirmDialog`, `ProjectFormDialog`, `PricingFormDialog`, `ApiKeyCreateDialog`, the three routes — **TDD: no** (visual composition + wiring, mirrors `login.tsx`/`setup.tsx`, which ship untested). Verified by running against the live backend (`bun run dev` + `docker compose up`), per step.
- Services / schemas / `queryOptions` factories — **TDD: no** (thin API wrappers, no branching — matches `authService.ts`/`lib/auth.ts`, which are also untested). Verified transitively by the route's manual test.

---

## Task 1: Shared list/dialog primitives

**Files:**
- Modify: `apps/frontend/package.json` (via shadcn CLI, no manual edit)
- Create: `apps/frontend/src/components/ui/alert-dialog.tsx` (shadcn-generated)
- Create: `apps/frontend/src/components/ui/textarea.tsx` (shadcn-generated)
- Move: `apps/frontend/src/components/auth/Field.tsx` → `apps/frontend/src/components/form/Field.tsx`
- Move: `apps/frontend/src/components/auth/FormError.tsx` → `apps/frontend/src/components/form/FormError.tsx`
- Modify: `apps/frontend/src/routes/login.tsx`, `apps/frontend/src/routes/setup.tsx` (import path update)
- Create: `apps/frontend/src/components/DataTable.tsx`
- Create: `apps/frontend/src/components/DataTable.test.tsx`
- Create: `apps/frontend/src/components/OffsetPagination.tsx`
- Create: `apps/frontend/src/components/OffsetPagination.test.tsx`
- Create: `apps/frontend/src/components/ConfirmDialog.tsx`
- Modify: `apps/frontend/src/i18n/en.json`, `apps/frontend/src/i18n/id.json`

**Interfaces:**
- Produces: `DataTable<T>({ columns, rows, rowKey, isLoading?, emptyMessage, skeletonRows? })`, `DataTableColumn<T> = { key: string; header: string; cell: (row: T) => ReactNode; className?: string }`; `OffsetPagination({ page, limit, total, onPageChange })`; `ConfirmDialog({ open, onOpenChange, title, description, confirmLabel, cancelLabel, destructive?, loading?, onConfirm })`; `Field`/`FormError` now live at `@/components/form/*` — every later task imports from there.

- [ ] **Step 1: Add the two missing shadcn primitives**

```bash
cd /Volumes/Project/router-lens/apps/frontend
bunx --bun shadcn@latest add alert-dialog textarea
```

Expected: `src/components/ui/alert-dialog.tsx` and `src/components/ui/textarea.tsx` are created, exporting (mirroring `dialog.tsx`'s shape) `AlertDialog`, `AlertDialogTrigger`, `AlertDialogContent`, `AlertDialogHeader`, `AlertDialogFooter`, `AlertDialogTitle`, `AlertDialogDescription`, `AlertDialogAction`, `AlertDialogCancel`, and `Textarea`. **If the generated export names differ from these** (CLI/registry drift), adjust `ConfirmDialog.tsx` in Step 5 to match what was actually generated — same escape hatch FE Plan 01 used for the router plugin export name.

- [ ] **Step 2: Relocate `Field` + `FormError` out of `components/auth/`**

They're about to be used by non-auth screens (projects, api-keys, pricing forms), so `components/auth/` is the wrong home — this project groups components by domain (`auth/`, `layout/`, `ui/`).

```bash
mkdir -p /Volumes/Project/router-lens/apps/frontend/src/components/form
git -C /Volumes/Project/router-lens mv apps/frontend/src/components/auth/Field.tsx apps/frontend/src/components/form/Field.tsx
git -C /Volumes/Project/router-lens mv apps/frontend/src/components/auth/FormError.tsx apps/frontend/src/components/form/FormError.tsx
```

In `apps/frontend/src/routes/login.tsx` and `apps/frontend/src/routes/setup.tsx`, update:

```ts
import { Field } from "@/components/auth/Field";
import { FormError } from "@/components/auth/FormError";
```

to:

```ts
import { Field } from "@/components/form/Field";
import { FormError } from "@/components/form/FormError";
```

- [ ] **Step 3: `DataTable.tsx`**

```tsx
import type { ReactNode } from "react";
import { Crosshair } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";

export interface DataTableColumn<T> {
  readonly key: string;
  readonly header: string;
  readonly cell: (row: T) => ReactNode;
  readonly className?: string;
}

interface DataTableProps<T> {
  readonly columns: ReadonlyArray<DataTableColumn<T>>;
  readonly rows: readonly T[];
  readonly rowKey: (row: T) => string;
  readonly isLoading?: boolean;
  readonly emptyMessage: string;
  readonly skeletonRows?: number;
}

// ponytail: plain mapped rows over the shadcn Table primitives, not
// @tanstack/react-table — these are small offset-paginated CRUD lists with no
// sort/filter requirement. Swap in react-table when the logs page needs
// column sort/filter at volume.
export function DataTable<T>({
  columns,
  rows,
  rowKey,
  isLoading = false,
  emptyMessage,
  skeletonRows = 5,
}: DataTableProps<T>) {
  return (
    <div className="rounded-xl border border-border">
      <Table>
        <TableHeader>
          <TableRow>
            {columns.map((col) => (
              <TableHead key={col.key} className={col.className}>
                {col.header}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading &&
            Array.from({ length: skeletonRows }, (_, i) => (
              <TableRow key={`skeleton-${i}`}>
                {columns.map((col) => (
                  <TableCell key={col.key}>
                    <Skeleton className="h-5 w-full max-w-40" />
                  </TableCell>
                ))}
              </TableRow>
            ))}
          {!isLoading && rows.length === 0 && (
            <TableRow>
              <TableCell colSpan={columns.length} className="py-16 text-center">
                <Crosshair className="mx-auto size-6 text-muted-foreground/40" strokeWidth={1.5} />
                <p className="mt-2 text-sm text-muted-foreground">{emptyMessage}</p>
              </TableCell>
            </TableRow>
          )}
          {!isLoading &&
            rows.map((row) => (
              <TableRow key={rowKey(row)}>
                {columns.map((col) => (
                  <TableCell key={col.key} className={col.className}>
                    {col.cell(row)}
                  </TableCell>
                ))}
              </TableRow>
            ))}
        </TableBody>
      </Table>
    </div>
  );
}
```

- [ ] **Step 4: `DataTable.test.tsx` (RTL smoke test — TDD: yes-ish)**

```tsx
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { DataTable, type DataTableColumn } from "./DataTable";

interface Row {
  id: string;
  name: string;
}

const columns: DataTableColumn<Row>[] = [{ key: "name", header: "Name", cell: (r) => r.name }];

describe("DataTable", () => {
  it("renders rows", () => {
    render(
      <DataTable columns={columns} rows={[{ id: "1", name: "Alpha" }]} rowKey={(r) => r.id} emptyMessage="Empty" />,
    );
    expect(screen.getByText("Alpha")).toBeInTheDocument();
  });

  it("renders the empty state when there are no rows", () => {
    render(<DataTable columns={columns} rows={[]} rowKey={(r) => r.id} emptyMessage="No projects yet" />);
    expect(screen.getByText("No projects yet")).toBeInTheDocument();
  });

  it("renders skeleton rows while loading, not the empty state", () => {
    render(<DataTable columns={columns} rows={[]} rowKey={(r) => r.id} emptyMessage="No projects yet" isLoading />);
    expect(screen.queryByText("No projects yet")).not.toBeInTheDocument();
  });
});
```

Run: `cd /Volumes/Project/router-lens/apps/frontend && bun run test src/components/DataTable.test.tsx`
Expected: 3 passed.

- [ ] **Step 5: `OffsetPagination.tsx`**

```tsx
import { ChevronLeft, ChevronRight } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";

interface OffsetPaginationProps {
  readonly page: number;
  readonly limit: number;
  readonly total: number;
  readonly onPageChange: (page: number) => void;
}

export function OffsetPagination({ page, limit, total, onPageChange }: OffsetPaginationProps) {
  const { t } = useTranslation();
  const totalPages = Math.max(1, Math.ceil(total / limit));
  return (
    <div className="flex items-center justify-between text-sm text-muted-foreground">
      <span>{t("common.pageOf", { page, totalPages })}</span>
      <div className="flex gap-2">
        <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => onPageChange(page - 1)}>
          <ChevronLeft className="size-4" />
          {t("common.previous")}
        </Button>
        <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => onPageChange(page + 1)}>
          {t("common.next")}
          <ChevronRight className="size-4" />
        </Button>
      </div>
    </div>
  );
}
```

- [ ] **Step 6: `OffsetPagination.test.tsx`**

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { OffsetPagination } from "./OffsetPagination";

describe("OffsetPagination", () => {
  it("disables previous on the first page and next on the last page", () => {
    render(<OffsetPagination page={1} limit={20} total={20} onPageChange={vi.fn()} />);
    expect(screen.getByRole("button", { name: /previous/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /next/i })).toBeDisabled();
  });

  it("calls onPageChange with the next page", async () => {
    const onPageChange = vi.fn();
    render(<OffsetPagination page={1} limit={20} total={100} onPageChange={onPageChange} />);
    await userEvent.click(screen.getByRole("button", { name: /next/i }));
    expect(onPageChange).toHaveBeenCalledWith(2);
  });
});
```

Run: `bun run test src/components/OffsetPagination.test.tsx`
Expected: 2 passed.

- [ ] **Step 7: `ConfirmDialog.tsx`**

```tsx
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";

interface ConfirmDialogProps {
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
  readonly title: string;
  readonly description: string;
  readonly confirmLabel: string;
  readonly cancelLabel: string;
  readonly destructive?: boolean;
  readonly loading?: boolean;
  readonly onConfirm: () => void;
}

/** Shared destructive-action confirmation — delete project, revoke key, delete pricing rule. */
export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  cancelLabel,
  destructive = false,
  loading = false,
  onConfirm,
}: ConfirmDialogProps) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel render={<Button variant="outline" />}>{cancelLabel}</AlertDialogCancel>
          <AlertDialogAction
            render={<Button variant={destructive ? "destructive" : "default"} disabled={loading} />}
            onClick={onConfirm}
          >
            {confirmLabel}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
```

- [ ] **Step 8: `common.*` i18n keys**

In `apps/frontend/src/i18n/en.json`, extend the `"common"` object:

```jsonc
"common": {
  "theme": "Theme",
  "language": "Language",
  "light": "Light",
  "dark": "Dark",
  "system": "System",
  "signIn": "Sign in",
  "comingSoon": "Coming soon",
  "menu": "Menu",
  "edit": "Edit",
  "delete": "Delete",
  "save": "Save",
  "create": "Create",
  "cancel": "Cancel",
  "copy": "Copy",
  "done": "Done",
  "previous": "Previous",
  "next": "Next",
  "pageOf": "Page {{page}} of {{totalPages}}"
}
```

In `apps/frontend/src/i18n/id.json`, extend the `"common"` object:

```jsonc
"common": {
  "theme": "Tema",
  "language": "Bahasa",
  "light": "Terang",
  "dark": "Gelap",
  "system": "Sistem",
  "signIn": "Masuk",
  "comingSoon": "Segera hadir",
  "menu": "Menu",
  "edit": "Ubah",
  "delete": "Hapus",
  "save": "Simpan",
  "create": "Buat",
  "cancel": "Batal",
  "copy": "Salin",
  "done": "Selesai",
  "previous": "Sebelumnya",
  "next": "Berikutnya",
  "pageOf": "Halaman {{page}} dari {{totalPages}}"
}
```

- [ ] **Step 9: Verify + commit**

```bash
cd /Volumes/Project/router-lens/apps/frontend
bun run test
bun run lint
```

Expected: all tests pass (formatters + api + the two new suites), lint clean.

```bash
git add apps/frontend/src/components/ui/alert-dialog.tsx apps/frontend/src/components/ui/textarea.tsx \
  apps/frontend/src/components/form apps/frontend/src/components/DataTable.tsx apps/frontend/src/components/DataTable.test.tsx \
  apps/frontend/src/components/OffsetPagination.tsx apps/frontend/src/components/OffsetPagination.test.tsx \
  apps/frontend/src/components/ConfirmDialog.tsx apps/frontend/src/routes/login.tsx apps/frontend/src/routes/setup.tsx \
  apps/frontend/src/i18n/en.json apps/frontend/src/i18n/id.json apps/frontend/package.json apps/frontend/bun.lock
git status apps/frontend/src/components/auth
git commit -m "feat(frontend): shared DataTable, OffsetPagination, ConfirmDialog; relocate Field/FormError"
```

(`git status apps/frontend/src/components/auth` should show the two files as deleted — confirming the `git mv` staged correctly; add that deletion too if it isn't already staged by the `mv`.)

---

## Task 2: Projects — data layer + list screen

**Files:**
- Modify: `apps/frontend/src/lib/schemas.ts`
- Create: `apps/frontend/src/services/projectService.ts`
- Create: `apps/frontend/src/lib/projects.ts`
- Create: `apps/frontend/src/components/projects/ProjectFormDialog.tsx`
- Modify: `apps/frontend/src/i18n/en.json`, `apps/frontend/src/i18n/id.json`
- Modify: `apps/frontend/src/routes/_app.projects.tsx`

**Interfaces:**
- Consumes: `DataTable`, `OffsetPagination`, `ConfirmDialog` (Task 1); `Field`, `FormError` from `@/components/form/*`; `paginated()` from `lib/schemas.ts`.
- Produces: `projectSchema`/`type Project` (`lib/schemas.ts`); `listProjects(page, limit)`, `getProject(id)`, `createProject(input)`, `updateProject(id, input)`, `deleteProject(id)` (`services/projectService.ts`); `projectsQueryOptions(page, limit)`, `projectQueryOptions(id)` (`lib/projects.ts`) — Task 3's detail route consumes `projectQueryOptions` and `deleteProject`.

- [ ] **Step 1: `projectSchema` in `lib/schemas.ts`**

Append to `apps/frontend/src/lib/schemas.ts`:

```ts
export const projectSchema = z.object({
  id: z.string(),
  name: z.string(),
  slug: z.string(),
  description: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});
export type Project = z.infer<typeof projectSchema>;
```

- [ ] **Step 2: `services/projectService.ts`**

```ts
import { api } from "@/lib/api";
import { paginated, projectSchema, type Project } from "@/lib/schemas";

export interface ProjectInput {
  name: string;
  description: string;
}

const listSchema = paginated(projectSchema);

/** GET /projects — offset-paginated. */
export async function listProjects(page: number, limit: number) {
  const res = await api.get("/projects", { params: { page, limit } });
  return listSchema.parse(res.data);
}

/** GET /projects/:id */
export async function getProject(id: string): Promise<Project> {
  const res = await api.get(`/projects/${id}`);
  return projectSchema.parse(res.data);
}

/** POST /projects */
export async function createProject(input: ProjectInput): Promise<Project> {
  const res = await api.post("/projects", input);
  return projectSchema.parse(res.data);
}

/** PUT /projects/:id */
export async function updateProject(id: string, input: ProjectInput): Promise<Project> {
  const res = await api.put(`/projects/${id}`, input);
  return projectSchema.parse(res.data);
}

/** DELETE /projects/:id */
export async function deleteProject(id: string): Promise<void> {
  await api.delete(`/projects/${id}`);
}
```

- [ ] **Step 3: `lib/projects.ts`**

```ts
import { queryOptions } from "@tanstack/react-query";
import { getProject, listProjects } from "@/services/projectService";

export function projectsQueryOptions(page: number, limit: number) {
  return queryOptions({
    queryKey: ["projects", { page, limit }],
    queryFn: () => listProjects(page, limit),
  });
}

export function projectQueryOptions(id: string) {
  return queryOptions({
    queryKey: ["projects", id],
    queryFn: () => getProject(id),
  });
}
```

- [ ] **Step 4: i18n — `projects.*` keys**

In `en.json`, add a top-level `"projects"` object (sibling of `"auth"`):

```jsonc
"projects": {
  "new": "New project",
  "empty": "No projects yet.",
  "fields": {
    "name": "Name",
    "slug": "Slug",
    "description": "Description",
    "createdAt": "Created"
  },
  "newTitle": "New project",
  "editTitle": "Edit project",
  "formDescription": "Projects group API keys and the events ingested with them.",
  "created": "Project created",
  "updated": "Project updated",
  "deleted": "Project deleted",
  "deleteTitle": "Delete project?",
  "deleteDescription": "Delete \"{{name}}\"? Its API keys will stop working. This cannot be undone.",
  "noDescription": "No description.",
  "errors": {
    "nameRequired": "Project name is required."
  }
}
```

In `id.json`, add the matching object:

```jsonc
"projects": {
  "new": "Proyek baru",
  "empty": "Belum ada proyek.",
  "fields": {
    "name": "Nama",
    "slug": "Slug",
    "description": "Deskripsi",
    "createdAt": "Dibuat"
  },
  "newTitle": "Proyek baru",
  "editTitle": "Ubah proyek",
  "formDescription": "Proyek mengelompokkan kunci API dan event yang masuk lewat kunci itu.",
  "created": "Proyek dibuat",
  "updated": "Proyek diperbarui",
  "deleted": "Proyek dihapus",
  "deleteTitle": "Hapus proyek?",
  "deleteDescription": "Hapus \"{{name}}\"? Kunci API-nya akan berhenti berfungsi. Tindakan ini tidak bisa dibatalkan.",
  "noDescription": "Tidak ada deskripsi.",
  "errors": {
    "nameRequired": "Nama proyek wajib diisi."
  }
}
```

- [ ] **Step 5: `components/projects/ProjectFormDialog.tsx`**

```tsx
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
```

- [ ] **Step 6: Rewrite `routes/_app.projects.tsx`**

```tsx
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
  });

  const columns: DataTableColumn<Project>[] = [
    {
      key: "name",
      header: t("projects.fields.name"),
      cell: (p) => (
        <Link to="/projects/$projectId" params={{ projectId: p.id }} className="font-medium hover:underline">
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
```

- [ ] **Step 7: Verify manually + automated checks**

```bash
cd /Volumes/Project/router-lens
docker compose up -d postgres
cd apps/backend && go run ./cmd/server &
cd ../frontend && bun run dev
```

In the browser (signed in): navigate to `/projects`. Create a project (name + description) → appears in the table. Edit it → name/description update. Delete it → row disappears, list re-fetches. Confirm pagination controls appear once there are >20 projects (or verify the disabled-boundary state with 1).

```bash
cd /Volumes/Project/router-lens/apps/frontend
bun run test
bun run lint
```

Expected: all green.

- [ ] **Step 8: Commit**

```bash
git add apps/frontend/src/lib/schemas.ts apps/frontend/src/services/projectService.ts apps/frontend/src/lib/projects.ts \
  apps/frontend/src/components/projects/ProjectFormDialog.tsx apps/frontend/src/routes/_app.projects.tsx \
  apps/frontend/src/i18n/en.json apps/frontend/src/i18n/id.json
git commit -m "feat(frontend): projects list screen — create, edit, delete, paginate"
```

---

## Task 3: Project detail route + API Keys

**Files:**
- Modify: `apps/frontend/src/lib/schemas.ts`
- Create: `apps/frontend/src/services/apiKeyService.ts`
- Create: `apps/frontend/src/lib/apikeys.ts`
- Create: `apps/frontend/src/components/projects/ApiKeyCreateDialog.tsx`
- Modify: `apps/frontend/src/i18n/en.json`, `apps/frontend/src/i18n/id.json`
- Create: `apps/frontend/src/routes/_app.projects.$projectId.tsx`
- Delete: `apps/frontend/src/routes/_app.api-keys.tsx`
- Modify: `apps/frontend/src/components/layout/nav.ts`

**Interfaces:**
- Consumes: `projectQueryOptions`, `deleteProject` (Task 2); `DataTable`, `ConfirmDialog` (Task 1); `ProjectFormDialog` (Task 2, reused for the edit button here).
- Produces: `apiKeySchema`/`type ApiKey`, `apiKeyCreatedSchema`/`type ApiKeyCreated` (`lib/schemas.ts`); `listApiKeys(projectId)`, `createApiKey(projectId, name)`, `revokeApiKey(id)` (`services/apiKeyService.ts`); `apiKeysQueryOptions(projectId)` (`lib/apikeys.ts`).

- [ ] **Step 1: `apiKeySchema` + `apiKeyCreatedSchema` in `lib/schemas.ts`**

Append to `apps/frontend/src/lib/schemas.ts`:

```ts
export const apiKeySchema = z.object({
  id: z.string(),
  name: z.string(),
  key_prefix: z.string(),
  last_used_at: z.string().nullable(),
  revoked_at: z.string().nullable(),
  created_at: z.string(),
});
export type ApiKey = z.infer<typeof apiKeySchema>;

/** POST /projects/:id/api-keys response — carries the plaintext key exactly once. */
export const apiKeyCreatedSchema = apiKeySchema.extend({ key: z.string() });
export type ApiKeyCreated = z.infer<typeof apiKeyCreatedSchema>;
```

- [ ] **Step 2: `services/apiKeyService.ts`**

```ts
import { z } from "zod";
import { api } from "@/lib/api";
import { apiKeyCreatedSchema, apiKeySchema, type ApiKey, type ApiKeyCreated } from "@/lib/schemas";

/** GET /projects/:projectId/api-keys — plain array, no pagination (small, per-project). */
export async function listApiKeys(projectId: string): Promise<ApiKey[]> {
  const res = await api.get(`/projects/${projectId}/api-keys`);
  return z.array(apiKeySchema).parse(res.data);
}

/** POST /projects/:projectId/api-keys — the response's `key` field is shown exactly once. */
export async function createApiKey(projectId: string, name: string): Promise<ApiKeyCreated> {
  const res = await api.post(`/projects/${projectId}/api-keys`, { name });
  return apiKeyCreatedSchema.parse(res.data);
}

/** DELETE /api-keys/:id */
export async function revokeApiKey(id: string): Promise<void> {
  await api.delete(`/api-keys/${id}`);
}
```

- [ ] **Step 3: `lib/apikeys.ts`**

```ts
import { queryOptions } from "@tanstack/react-query";
import { listApiKeys } from "@/services/apiKeyService";

export function apiKeysQueryOptions(projectId: string) {
  return queryOptions({
    queryKey: ["projects", projectId, "api-keys"],
    queryFn: () => listApiKeys(projectId),
  });
}
```

- [ ] **Step 4: i18n — `apiKeys.*` keys**

`nav.apiKeys` ("API Keys" / "Kunci API") already exists and is reused as the section heading — no new nav key. Add a top-level `"apiKeys"` object in `en.json`:

```jsonc
"apiKeys": {
  "new": "New key",
  "empty": "No API keys yet.",
  "fields": {
    "name": "Name",
    "prefix": "Prefix",
    "lastUsed": "Last used",
    "createdAt": "Created",
    "status": "Status"
  },
  "neverUsed": "Never used",
  "activeStatus": "Active",
  "revokedStatus": "Revoked",
  "revoke": "Revoke",
  "revoked": "API key revoked",
  "revokeTitle": "Revoke API key?",
  "revokeDescription": "Revoke \"{{name}}\"? Any client using it will stop being able to send events. This cannot be undone.",
  "newTitle": "New API key",
  "newDescription": "Name it after where it will be used, e.g. \"prod-gateway\".",
  "create": "Create key",
  "createdTitle": "API key created",
  "createdWarning": "Copy this key now — it will not be shown again.",
  "errors": {
    "nameRequired": "Key name is required."
  }
}
```

And in `id.json`:

```jsonc
"apiKeys": {
  "new": "Kunci baru",
  "empty": "Belum ada kunci API.",
  "fields": {
    "name": "Nama",
    "prefix": "Prefiks",
    "lastUsed": "Terakhir dipakai",
    "createdAt": "Dibuat",
    "status": "Status"
  },
  "neverUsed": "Belum pernah dipakai",
  "activeStatus": "Aktif",
  "revokedStatus": "Dicabut",
  "revoke": "Cabut",
  "revoked": "Kunci API dicabut",
  "revokeTitle": "Cabut kunci API?",
  "revokeDescription": "Cabut \"{{name}}\"? Klien yang memakainya akan berhenti bisa kirim event. Tindakan ini tidak bisa dibatalkan.",
  "newTitle": "Kunci API baru",
  "newDescription": "Beri nama sesuai tempat pemakaiannya, mis. \"prod-gateway\".",
  "create": "Buat kunci",
  "createdTitle": "Kunci API dibuat",
  "createdWarning": "Salin kunci ini sekarang — tidak akan ditampilkan lagi.",
  "errors": {
    "nameRequired": "Nama kunci wajib diisi."
  }
}
```

- [ ] **Step 5: `components/projects/ApiKeyCreateDialog.tsx`**

```tsx
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
```

- [ ] **Step 6: `routes/_app.projects.$projectId.tsx`**

```tsx
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

  const project = useQuery(projectQueryOptions(projectId));
  const keys = useQuery(apiKeysQueryOptions(projectId));

  const [editOpen, setEditOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [keyDialogOpen, setKeyDialogOpen] = useState(false);
  const [revoking, setRevoking] = useState<ApiKey | null>(null);

  const deleteMutation = useMutation({
    mutationFn: () => deleteProject(projectId),
    onSuccess: async () => {
      toast.success(t("projects.deleted"));
      await queryClient.invalidateQueries({ queryKey: ["projects"] });
      await navigate({ to: "/projects" });
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

  if (!project.data) return null;

  return (
    <div className="space-y-6">
      <Link to="/projects" className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
        <ArrowLeft className="size-4" />
        {t("nav.projects")}
      </Link>

      <Card className="flex items-start justify-between p-6">
        <div>
          <h1 className="font-heading text-2xl font-semibold tracking-tight">{project.data.name}</h1>
          <p className="mt-1 text-sm text-muted-foreground">{project.data.slug}</p>
          <p className="mt-2 max-w-prose text-sm text-muted-foreground">
            {project.data.description || t("projects.noDescription")}
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
          rows={keys.data ?? []}
          rowKey={(k) => k.id}
          isLoading={keys.isLoading}
          emptyMessage={t("apiKeys.empty")}
          columns={keyColumns}
        />
      </div>

      <ProjectFormDialog open={editOpen} onOpenChange={setEditOpen} project={project.data} />
      <ApiKeyCreateDialog projectId={projectId} open={keyDialogOpen} onOpenChange={setKeyDialogOpen} />

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t("projects.deleteTitle")}
        description={t("projects.deleteDescription", { name: project.data.name })}
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
```

- [ ] **Step 7: Retire the flat `/api-keys` route + nav entry**

```bash
git -C /Volumes/Project/router-lens rm apps/frontend/src/routes/_app.api-keys.tsx
```

In `apps/frontend/src/components/layout/nav.ts`, remove the `apiKeys` entry (keep the rest):

```ts
export const NAV: readonly NavItem[] = [
  { to: "/", key: "nav.dashboard", icon: LayoutDashboard },
  { to: "/logs", key: "nav.logs", icon: ScrollText },
  { to: "/projects", key: "nav.projects", icon: FolderKanban },
  { to: "/pricing", key: "nav.pricing", icon: Tag },
  { to: "/settings", key: "nav.settings", icon: Settings },
];
```

Remove the now-unused `KeyRound` import from `nav.ts` if it's no longer referenced there (it's still used directly in the new detail route, imported separately — check with a grep before deleting the import to be safe):

```bash
grep -n "KeyRound" /Volumes/Project/router-lens/apps/frontend/src/components/layout/nav.ts
```

If it's unused in that file after the edit, remove it from the `lucide-react` import list.

- [ ] **Step 8: Verify manually + automated checks**

With the dev server + backend running: from `/projects`, click into a project's name → lands on `/projects/$projectId`. Edit the project inline (Edit button) → header updates. Click "New key", name it → dialog shows the plaintext key exactly once with a working copy button; closing and reopening "New key" never shows that plaintext again (only `key_prefix` in the table). Revoke the key → row shows "Revoked", the row's action menu disappears. Delete the project from the detail page → redirects to `/projects` and it's gone. Confirm the sidebar/mobile-drawer nav no longer shows a standalone "API Keys" entry.

```bash
cd /Volumes/Project/router-lens/apps/frontend
bun run test
bun run lint
```

Expected: all green (route tree regenerates automatically via the TanStack Router Vite plugin on `bun run dev`/`bun run build` — no manual `routeTree.gen.ts` edit).

- [ ] **Step 9: Commit**

```bash
git add apps/frontend/src/lib/schemas.ts apps/frontend/src/services/apiKeyService.ts apps/frontend/src/lib/apikeys.ts \
  apps/frontend/src/components/projects/ApiKeyCreateDialog.tsx apps/frontend/src/routes/_app.projects.\$projectId.tsx \
  apps/frontend/src/components/layout/nav.ts apps/frontend/src/i18n/en.json apps/frontend/src/i18n/id.json \
  apps/frontend/src/routeTree.gen.ts
git commit -m "feat(frontend): project detail route with per-project API key management"
```

---

## Task 4: Pricing — data layer + screen

**Files:**
- Modify: `apps/frontend/src/lib/schemas.ts`
- Create: `apps/frontend/src/services/pricingService.ts`
- Create: `apps/frontend/src/lib/pricing.ts`
- Create: `apps/frontend/src/components/pricing/PricingFormDialog.tsx`
- Modify: `apps/frontend/src/i18n/en.json`, `apps/frontend/src/i18n/id.json`
- Modify: `apps/frontend/src/routes/_app.pricing.tsx`

**Interfaces:**
- Consumes: `DataTable`, `ConfirmDialog` (Task 1); `Field`, `FormError` (Task 1); `formatUSD` (`lib/money.ts`, existing).
- Produces: `pricingRuleSchema`/`type PricingRule` (`lib/schemas.ts`); `listPricing()`, `createPricing(input)`, `updatePricing(id, input)`, `deletePricing(id)` (`services/pricingService.ts`); `pricingQueryOptions` (`lib/pricing.ts`).

- [ ] **Step 1: `pricingRuleSchema` in `lib/schemas.ts`**

```ts
export const pricingRuleSchema = z.object({
  id: z.string(),
  provider: z.string(),
  model: z.string(),
  input_price_per_1m: z.string(),
  output_price_per_1m: z.string(),
  currency: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});
export type PricingRule = z.infer<typeof pricingRuleSchema>;
```

- [ ] **Step 2: `services/pricingService.ts`**

Currency is fixed to `"USD"` in v0.1 (CLAUDE.md decision) and defaults server-side when omitted — the form never collects it.

```ts
import { z } from "zod";
import { api } from "@/lib/api";
import { pricingRuleSchema, type PricingRule } from "@/lib/schemas";

export interface PricingInput {
  provider: string;
  model: string;
  input_price_per_1m: string;
  output_price_per_1m: string;
}

/** GET /pricing — plain array, no pagination (small, admin-curated list). */
export async function listPricing(): Promise<PricingRule[]> {
  const res = await api.get("/pricing");
  return z.array(pricingRuleSchema).parse(res.data);
}

/** POST /pricing — upserts on (provider, model). */
export async function createPricing(input: PricingInput): Promise<PricingRule> {
  const res = await api.post("/pricing", input);
  return pricingRuleSchema.parse(res.data);
}

/** PUT /pricing/:id — 204 No Content, nothing to parse. */
export async function updatePricing(id: string, input: PricingInput): Promise<void> {
  await api.put(`/pricing/${id}`, input);
}

/** DELETE /pricing/:id */
export async function deletePricing(id: string): Promise<void> {
  await api.delete(`/pricing/${id}`);
}
```

- [ ] **Step 3: `lib/pricing.ts`**

```ts
import { queryOptions } from "@tanstack/react-query";
import { listPricing } from "@/services/pricingService";

export const pricingQueryOptions = queryOptions({
  queryKey: ["pricing"],
  queryFn: listPricing,
});
```

- [ ] **Step 4: i18n — `pricing.*` keys**

`en.json`:

```jsonc
"pricing": {
  "new": "New rule",
  "empty": "No pricing rules yet — events for unpriced models show cost as unpriced.",
  "fields": {
    "provider": "Provider",
    "model": "Model",
    "inputPrice": "Input $/1M",
    "outputPrice": "Output $/1M",
    "updatedAt": "Updated"
  },
  "newTitle": "New pricing rule",
  "editTitle": "Edit pricing rule",
  "formDescription": "Prices are per 1M tokens in USD, applied to events at ingest time.",
  "created": "Pricing rule created",
  "updated": "Pricing rule updated",
  "deleted": "Pricing rule deleted",
  "deleteTitle": "Delete pricing rule?",
  "deleteDescription": "Delete the rule for \"{{provider}} / {{model}}\"? Future events for this model become unpriced.",
  "errors": {
    "providerRequired": "Provider is required.",
    "modelRequired": "Model is required.",
    "priceRequired": "Price is required.",
    "priceInvalid": "Enter a non-negative number."
  }
}
```

`id.json`:

```jsonc
"pricing": {
  "new": "Aturan baru",
  "empty": "Belum ada aturan harga — event untuk model tanpa harga tampil sebagai unpriced.",
  "fields": {
    "provider": "Provider",
    "model": "Model",
    "inputPrice": "Input $/1M",
    "outputPrice": "Output $/1M",
    "updatedAt": "Diperbarui"
  },
  "newTitle": "Aturan harga baru",
  "editTitle": "Ubah aturan harga",
  "formDescription": "Harga per 1 juta token dalam USD, diterapkan ke event saat masuk.",
  "created": "Aturan harga dibuat",
  "updated": "Aturan harga diperbarui",
  "deleted": "Aturan harga dihapus",
  "deleteTitle": "Hapus aturan harga?",
  "deleteDescription": "Hapus aturan untuk \"{{provider}} / {{model}}\"? Event mendatang untuk model ini jadi tanpa harga.",
  "errors": {
    "providerRequired": "Provider wajib diisi.",
    "modelRequired": "Model wajib diisi.",
    "priceRequired": "Harga wajib diisi.",
    "priceInvalid": "Masukkan angka non-negatif."
  }
}
```

- [ ] **Step 5: `components/pricing/PricingFormDialog.tsx`**

```tsx
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
}

function isNonNegativeNumber(value: string): boolean {
  const n = Number(value);
  return !Number.isNaN(n) && n >= 0;
}

export function PricingFormDialog({ open, onOpenChange, rule }: PricingFormDialogProps) {
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

  useEffect(() => {
    if (open) {
      form.reset({
        provider: rule?.provider ?? "",
        model: rule?.model ?? "",
        input_price_per_1m: rule?.input_price_per_1m ?? "",
        output_price_per_1m: rule?.output_price_per_1m ?? "",
      });
    }
  }, [open, rule, form]);

  const mutation = useMutation({
    mutationFn: (values: {
      provider: string;
      model: string;
      input_price_per_1m: string;
      output_price_per_1m: string;
    }) => (isEdit ? updatePricing(rule.id, values) : createPricing(values)),
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
```

- [ ] **Step 6: Rewrite `routes/_app.pricing.tsx`**

```tsx
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { MoreHorizontal, Plus } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { DataTable, type DataTableColumn } from "@/components/DataTable";
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
  const query = useQuery(pricingQueryOptions);

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<PricingRule | null>(null);
  const [deleting, setDeleting] = useState<PricingRule | null>(null);

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

      <DataTable
        rows={query.data ?? []}
        rowKey={(p) => p.id}
        isLoading={query.isLoading}
        emptyMessage={t("pricing.empty")}
        columns={columns}
      />

      <PricingFormDialog open={formOpen} onOpenChange={setFormOpen} rule={editing} />

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
```

- [ ] **Step 7: Verify manually + automated checks**

With the dev server + backend running: navigate to `/pricing`. Create a rule (e.g. provider `openai`, model `gpt-4o`, input `2.50`, output `10.00`) → appears formatted as currency. Edit it → values persist. Delete it → row disappears. Try submitting an empty/negative price → inline validation error, no request sent.

```bash
cd /Volumes/Project/router-lens/apps/frontend
bun run test
bun run lint
```

- [ ] **Step 8: Commit**

```bash
git add apps/frontend/src/lib/schemas.ts apps/frontend/src/services/pricingService.ts apps/frontend/src/lib/pricing.ts \
  apps/frontend/src/components/pricing/PricingFormDialog.tsx apps/frontend/src/routes/_app.pricing.tsx \
  apps/frontend/src/i18n/en.json apps/frontend/src/i18n/id.json
git commit -m "feat(frontend): pricing rules screen — create, edit, delete"
```

---

## Task 5: Finishing — react-doctor, perf gate, full E2E, docs

**Files:**
- Modify: `docs/progress.md`

- [ ] **Step 1: `react-doctor`**

Run `react-doctor` (Skill tool) over the frontend. Fix lint/a11y/bundle/architecture findings it surfaces; if something is consciously deferred, note it inline as a `// ponytail:` comment rather than silently skipping.

- [ ] **Step 2: Performance gate (behind-auth — no SEO gate)**

```bash
cd /Volumes/Project/router-lens/apps/frontend
bun run build
```

Run `claude-seo:seo-performance` (Skill tool) against the build output / a locally served build. Confirm the bundle stays within the budget `frontend-senior` set in FE Plan 01 — this plan added no new runtime dependency, so no material bundle growth is expected. Fix any regression.

- [ ] **Step 3: Full manual end-to-end walkthrough**

With `docker compose up` (or `go run ./cmd/server` + `bun run dev`) live: create a project → edit its name/description → create an API key on it (confirm the plaintext shows exactly once) → revoke the key → create a pricing rule → edit it → delete the pricing rule → delete the project. Confirm every toast, every empty state (fresh project's key list, zero pricing rules), and both light/dark + EN/ID render correctly.

- [ ] **Step 4: Update `docs/progress.md`**

Mark FE-03 (this plan) as executed in the plan roadmap table and the "Next actions" list — mirror the update style already used for Plans 01–04 (✅ Executed).

- [ ] **Step 5: Commit**

```bash
git add docs/progress.md
git commit -m "docs: mark FE Plan 03 (projects/api-keys/pricing CRUD) as executed"
```

---

## Definition of Done (FE Plan 03)

`/projects` lists, creates, edits, and deletes projects with offset pagination. `/projects/$projectId` shows project details (editable) and manages that project's API keys (create with a one-time plaintext reveal, revoke). The flat `/api-keys` placeholder and its nav entry are removed — api-keys are only ever managed from their owning project, matching the locked API surface. `/pricing` lists, creates, edits, and deletes pricing rules. All three screens share one `<DataTable>`, one `<OffsetPagination>`, and one `<ConfirmDialog>` — no duplicated table/pagination/confirm logic. `bun run test` and `bun run lint` are green; `react-doctor` and the performance gate are clean. Committed on `dev`, one commit per task.
