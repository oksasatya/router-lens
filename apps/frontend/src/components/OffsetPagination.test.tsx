import "@/i18n";
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
