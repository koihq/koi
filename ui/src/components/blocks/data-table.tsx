import { type ColumnDef, flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table";
import { cn } from "cn";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export type DataTableColumn<T> = {
  id: string;
  header: ReactNode;
  cell: (row: T) => ReactNode;
  headClassName?: string;
  cellClassName?: string;
  /** Current sort of a sortable column, exposed as `aria-sort` on its header. */
  sort?: "ascending" | "descending" | "none";
};

export function DataTable<T>({
  data,
  columns,
  getRowId,
  ariaLabel,
  className,
  headerRowClassName,
  rowClassName,
  isRowSelected,
  interactive = false,
}: {
  data: T[];
  columns: DataTableColumn<T>[];
  getRowId: (row: T) => string;
  ariaLabel?: string;
  className?: string;
  headerRowClassName?: string;
  rowClassName?: (row: T) => string;
  isRowSelected?: (row: T) => boolean;
  /** Borderless rows with one rounded highlight on hover, focus and selection. */
  interactive?: boolean;
}) {
  const { t } = useTranslation("common");
  const table = useReactTable({
    data,
    columns: columns.map(
      (column): ColumnDef<T> => ({
        id: column.id,
        header: () => column.header,
        cell: ({ row }) => column.cell(row.original),
      }),
    ),
    getRowId,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
  });

  return (
    <Table
      scrollHint={t("tableScrollHint")}
      aria-label={ariaLabel}
      className={cn(interactive && "list-table", className)}
    >
      <TableHeader>
        {table.getHeaderGroups().map((group) => (
          <TableRow
            key={group.id}
            className={cn(interactive && "hover:bg-transparent", headerRowClassName)}
          >
            {group.headers.map((header) => (
              <TableHead
                key={header.id}
                className={columns.find((column) => column.id === header.column.id)?.headClassName}
                aria-sort={columns.find((column) => column.id === header.column.id)?.sort}
              >
                {header.isPlaceholder
                  ? null
                  : flexRender(header.column.columnDef.header, header.getContext())}
              </TableHead>
            ))}
          </TableRow>
        ))}
      </TableHeader>
      <TableBody>
        {table.getRowModel().rows.map((row) => (
          <TableRow
            key={row.id}
            className={cn(
              interactive && "group hover:bg-transparent",
              rowClassName?.(row.original),
            )}
            data-selected={isRowSelected?.(row.original) || undefined}
          >
            {row.getVisibleCells().map((cell) => (
              <TableCell
                key={cell.id}
                className={columns.find((column) => column.id === cell.column.id)?.cellClassName}
              >
                {flexRender(cell.column.columnDef.cell, cell.getContext())}
              </TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
