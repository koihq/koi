import type { ApiListResponse } from "@/api/shared/types";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import C from "../-locales/mvp.en.json";

export function copy(text: string, values: Record<string, string | number>) {
  return text.replace(/\{(\w+)\}/g, (match, key: string) => String(values[key] ?? match));
}

export function LocalChoice({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: string;
  options: { value: string; label: string }[];
  onChange: (value: string) => void;
}) {
  return (
    <Select
      value={value}
      items={options}
      onValueChange={(next) => {
        if (next !== null) onChange(next);
      }}
    >
      <SelectTrigger aria-label={label}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent className="hc-brand-menu">
        {options.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

export function ListPagination({
  page,
  pageSize,
  onPage,
  onPageSize,
}: {
  page: Pick<ApiListResponse<unknown>, "currentPage" | "totalPages" | "hasPrev" | "hasNext">;
  pageSize: number;
  onPage: (value: number) => void;
  onPageSize?: (value: number) => void;
}) {
  if (page.totalPages <= 1 && !onPageSize) return null;

  return (
    <div className="hc-list-pagination">
      <span>{copy(C.page, { page: page.currentPage, total: Math.max(1, page.totalPages) })}</span>
      {onPageSize && (
        <LocalChoice
          label={C.perPage}
          value={String(pageSize)}
          options={[10, 20, 50].map((value) => ({ value: String(value), label: String(value) }))}
          onChange={(value) => onPageSize(Number(value))}
        />
      )}
      <div className="hc-actions">
        <Button
          variant="ghost"
          size="sm"
          disabled={!page.hasPrev}
          onClick={() => onPage(page.currentPage - 1)}
        >
          {C.previous}
        </Button>
        <Button
          variant="ghost"
          size="sm"
          disabled={!page.hasNext}
          onClick={() => onPage(page.currentPage + 1)}
        >
          {C.next}
        </Button>
      </div>
    </div>
  );
}
