import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useEffect } from "react";

import { useApi } from "@/api/apiProvider";
import { useTableState } from "@/hooks/use-table-state";
import { promptKey } from "@/lib/query-keys";


export function usePromptList(brandID: string) {
  const api = useApi();
  const table = useTableState();
  const params = {
    page: table.page,
    pageSize: table.pageSize,
    q: table.debouncedQuery || undefined,
    confirmed: table.confirmed,
  };
  const prompts = useQuery({
    queryKey: [...promptKey(brandID), params],
    queryFn: ({ signal }) => api.prompts.list(brandID, params, { signal }),
    placeholderData: keepPreviousData,
    enabled: table.q === table.debouncedQuery,
  });
  const unconfirmed = useQuery({
    queryKey: [...promptKey(brandID), { page: 1, pageSize: 10, confirmed: false }],
    queryFn: ({ signal }) =>
      api.prompts.list(brandID, { page: 1, pageSize: 10, confirmed: false }, { signal }),
  });
  const { page, clampPage, q, debouncedQuery } = table;
  const { data, isPlaceholderData, isFetching, isSuccess } = prompts;
  useEffect(() => {
    if (
      isSuccess &&
      !isPlaceholderData &&
      !isFetching &&
      q === debouncedQuery &&
      data?.currentPage === page &&
      page > Math.max(1, data.totalPages)
    ) {
      void clampPage(data.totalPages);
    }
  }, [clampPage, page, q, debouncedQuery, data, isPlaceholderData, isFetching, isSuccess]);

  return { table, prompts, unconfirmed };
}
