import {
  createStandardSchemaV1,
  parseAsInteger,
  parseAsNumberLiteral,
  parseAsString,
  parseAsStringLiteral,
  useQueryStates,
} from "nuqs";
import { useCallback, useEffect } from "react";

import { useDebouncedValue } from "@/hooks/use-debounced-value";

export const tableSearchParams = {
  page: parseAsInteger.withDefault(1).withOptions({ clearOnDefault: true }),
  pageSize: parseAsNumberLiteral([10, 20, 50] as const)
    .withDefault(10)
    .withOptions({ clearOnDefault: true }),
  q: parseAsString.withDefault("").withOptions({ clearOnDefault: true }),
  status: parseAsStringLiteral(["all", "confirmed", "unconfirmed"] as const)
    .withDefault("all")
    .withOptions({ clearOnDefault: true }),
};

export const validateTableSearch = createStandardSchemaV1(tableSearchParams, {
  partialOutput: true,
});

export type TableStatus = "all" | "confirmed" | "unconfirmed";

export function confirmedFromStatus(status: TableStatus): boolean | undefined {
  if (status === "confirmed") return true;
  if (status === "unconfirmed") return false;
  return undefined;
}

export function useTableState() {
  const [raw, setState] = useQueryStates(tableSearchParams);
  const page = Math.max(1, raw.page);
  const debouncedQuery = useDebouncedValue(raw.q);
  useEffect(() => {
    if (raw.page < 1) void setState({ page: 1 }, { history: "replace" });
  }, [raw.page, setState]);
  const clampPage = useCallback(
    (totalPages: number) =>
      setState({ page: Math.min(page, Math.max(1, totalPages)) }, { history: "replace" }),
    [page, setState],
  );
  const writeState = (updates: Partial<typeof raw>, history: "push" | "replace") => {
    const next = { ...raw, ...updates };
    return setState(
      {
        page: next.page === 1 ? null : next.page,
        pageSize: next.pageSize === 10 ? null : next.pageSize,
        q: next.q === "" ? null : next.q,
        status: next.status === "all" ? null : next.status,
      },
      { history },
    );
  };

  return {
    page,
    pageSize: raw.pageSize,
    q: raw.q,
    debouncedQuery,
    status: raw.status,
    confirmed: confirmedFromStatus(raw.status),
    setSearch: (q: string) => writeState({ q, page: 1 }, "replace"),
    setStatus: (status: TableStatus) => writeState({ status, page: 1 }, "replace"),
    setPageSize: (pageSize: 10 | 20 | 50) => writeState({ pageSize, page: 1 }, "replace"),
    setPage: (nextPage: number) => writeState({ page: Math.max(1, nextPage) }, "push"),
    clampPage,
    clearFilters: () => writeState({ q: "", status: "all", page: 1 }, "replace"),
  };
}
