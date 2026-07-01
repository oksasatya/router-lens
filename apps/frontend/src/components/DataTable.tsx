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
