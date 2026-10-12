import type { UseQueryResult } from "@tanstack/react-query";
import { useEffect } from "react";

import type { ApiListResponse } from "@/api/shared/types";
import type { useTableState } from "@/hooks/use-table-state";

export function usePageClamp<T>(
  table: ReturnType<typeof useTableState>,
  query: UseQueryResult<ApiListResponse<T>>,
) {
  const { page, clampPage, q, debouncedQuery } = table;
  const { data, isPlaceholderData, isFetching, isSuccess } = query;

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
}
