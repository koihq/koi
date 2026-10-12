import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { useApi } from "@/api/apiProvider";
import { usePageClamp } from "@/hooks/use-page-clamp";
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

  usePageClamp(table, prompts);

  return { table, prompts };
}
