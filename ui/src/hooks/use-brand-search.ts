import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { useApi } from "@/api/apiProvider";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { brandListOptions } from "@/lib/queries";

export function useBrandSearch(organizationId: string, open: boolean) {
  const api = useApi();
  const [input, setInput] = useState("");
  const [page, setPage] = useState(1);
  const trimmed = input.trim();
  const debounced = useDebouncedValue(trimmed);
  const search = trimmed ? debounced : "";
  const waiting = trimmed !== search;

  const result = useQuery({
    ...brandListOptions(api, organizationId, { q: search || undefined, page, pageSize: 20 }),
    enabled: open && !waiting,
  });

  function changeSearch(value: string) {
    setInput(value);
    setPage(1);
  }

  return {
    result,
    input,
    page,
    setPage,
    changeSearch,
    searching: waiting || result.isLoading,
    items: waiting || result.isError ? [] : (result.data?.items ?? []),
  };
}
