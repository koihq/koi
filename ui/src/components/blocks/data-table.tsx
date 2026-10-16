import { type ColumnDef, flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";

import type { ApiListResponse, BaseEntity } from "@/api/shared/types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { TableStatus } from "@/hooks/use-table-state";

const emptyRows: never[] = [];

export function DataTable<T extends BaseEntity>({
  response,
  page,
  loading,
  failed,
  onRetry,
  columns,
  tableClassName,
  q,
  status,
  pageSize,
  searchPlaceholder,
  addAction,
  emptyCollection,
  onSearchChange,
  onStatusChange,
  onPageSizeChange,
  onPageChange,
  onClearFilters,
}: {
  response?: ApiListResponse<T>;
  page: number;
  loading: boolean;
  failed: boolean;
  onRetry: () => void;
  columns: ColumnDef<T>[];
  tableClassName?: string;
  q: string;
  status: TableStatus;
  pageSize: 10 | 20 | 50;
  searchPlaceholder: string;
  addAction: ReactNode;
  emptyCollection: ReactNode;
  onSearchChange: (value: string) => void;
  onStatusChange: (value: TableStatus) => void;
  onPageSizeChange: (value: 10 | 20 | 50) => void;
  onPageChange: (page: number) => void;
  onClearFilters: () => void;
}) {
  const { t } = useTranslation("management");

  const table = useReactTable({
    data: response?.items ?? emptyRows,
    columns,
    getRowId: (row) => row.id,
    getCoreRowModel: getCoreRowModel(),
    manualFiltering: true,
    manualPagination: true,
    pageCount: response?.totalPages ?? 0,
    state: {
      globalFilter: q,
      columnFilters: [{ id: "confirmed", value: status }],
      pagination: { pageIndex: page - 1, pageSize },
    },
    onGlobalFilterChange: (updater) =>
      onSearchChange(typeof updater === "function" ? updater(q) : updater),
    onColumnFiltersChange: (updater) => {
      const filters =
        typeof updater === "function" ? updater([{ id: "confirmed", value: status }]) : updater;

      onStatusChange(
        (filters.find((filter) => filter.id === "confirmed")?.value as TableStatus) ?? "all",
      );
    },
    onPaginationChange: (updater) => {
      const next =
        typeof updater === "function" ? updater({ pageIndex: page - 1, pageSize }) : updater;
      if (next.pageSize !== pageSize) onPageSizeChange(next.pageSize as 10 | 20 | 50);
      else onPageChange(next.pageIndex + 1);
    },
  });

  const filtered = q !== "" || status !== "all";

  return (
    <div>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <Input
          value={q}
          onChange={(event) => onSearchChange(event.target.value)}
          placeholder={searchPlaceholder}
          aria-label={searchPlaceholder}
          className="sm:max-w-sm"
        />
        <div className="flex flex-wrap items-center gap-2 sm:ms-auto">
          <Select value={status} onValueChange={(value) => onStatusChange(value as TableStatus)}>
            <SelectTrigger aria-label={t("statusLabel")}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t("all")}</SelectItem>
              <SelectItem value="confirmed">{t("confirmed")}</SelectItem>
              <SelectItem value="unconfirmed">{t("unconfirmed")}</SelectItem>
            </SelectContent>
          </Select>
          <Select
            value={String(pageSize)}
            onValueChange={(value) => onPageSizeChange(Number(value) as 10 | 20 | 50)}
          >
            <SelectTrigger aria-label={t("pageSizeLabel")}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="10">{t("rows10")}</SelectItem>
              <SelectItem value="20">{t("rows20")}</SelectItem>
              <SelectItem value="50">{t("rows50")}</SelectItem>
            </SelectContent>
          </Select>
          {addAction}
        </div>
      </div>

      {failed ? (
        <div role="alert" className="space-y-3 py-12 text-sm text-destructive">
          <p>{t("loadError")}</p>
          <Button type="button" variant="outline" onClick={onRetry}>
            {t("retry")}
          </Button>
        </div>
      ) : loading ? (
        <output className="block py-12 text-sm text-muted-foreground">{t("loading")}</output>
      ) : !response || response.items.length === 0 ? (
        <div className="py-16 text-center text-sm text-muted-foreground">
          {filtered ? (
            <div className="space-y-3">
              <p>{q ? t("noResults", { q }) : t("noStatusResults")}</p>
              <Button type="button" variant="outline" size="sm" onClick={onClearFilters}>
                {t("clearFilters")}
              </Button>
            </div>
          ) : (
            emptyCollection
          )}
        </div>
      ) : (
        <div className="mt-5">
          <Table aria-label={t("tableLabel")} className={tableClassName}>
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <TableHead key={header.id}>
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
                <TableRow key={row.id}>
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <div className="mt-5 flex items-center justify-between gap-3 text-xs text-muted-foreground">
        <p className="tabular-nums">
          {t("page", { current: page, total: Math.max(1, response?.totalPages ?? 0) })}
        </p>
        <div className="flex gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={loading || failed || page <= 1}
            onClick={() => onPageChange(page - 1)}
          >
            {t("previous")}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={loading || failed || !response?.hasNext}
            onClick={() => onPageChange(page + 1)}
          >
            {t("next")}
          </Button>
        </div>
      </div>
    </div>
  );
}
