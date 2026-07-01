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
