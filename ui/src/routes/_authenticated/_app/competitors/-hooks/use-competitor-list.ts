import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { useApi } from "@/api/apiProvider";
import { usePageClamp } from "@/hooks/use-page-clamp";
import { useTableState } from "@/hooks/use-table-state";
import { competitorKey } from "@/lib/query-keys";

export function useCompetitorList(brandID: string) {
  const api = useApi();
  const table = useTableState();

  const params = {
    page: table.page,
    pageSize: table.pageSize,
    q: table.debouncedQuery || undefined,
    confirmed: table.confirmed,
  };

  const competitors = useQuery({
    queryKey: [...competitorKey(brandID), params],
    queryFn: ({ signal }) => api.competitors.list(brandID, params, { signal }),
    placeholderData: keepPreviousData,
    enabled: table.q === table.debouncedQuery,
  });

  usePageClamp(table, competitors);

  return { table, competitors };
}
