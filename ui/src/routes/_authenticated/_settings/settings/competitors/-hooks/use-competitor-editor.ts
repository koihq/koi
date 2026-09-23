import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { useApi } from "@/api/apiProvider";
import type { CreateCompetitor } from "@/api/competitors.api";
import { lastPage } from "@/utils/pagination";

const pageSize = 10;
const competitorKey = (brandID: string) => ["brands", brandID, "competitors"] as const;
const competitorListKey = (brandID: string, page: number, confirmed?: boolean) =>
  [
    ...competitorKey(brandID),
    { page, pageSize: confirmed === undefined ? pageSize : 1, confirmed },
  ] as const;

export function useCompetitorEditor(brandID: string) {
  const api = useApi();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const competitors = useQuery({
    queryKey: competitorListKey(brandID, page),
    queryFn: ({ signal }) => api.competitors.list(brandID, { page, pageSize }, { signal }),
    placeholderData: keepPreviousData,
  });
  const unconfirmed = useQuery({
    queryKey: competitorListKey(brandID, 1, false),
    queryFn: ({ signal }) =>
      api.competitors.list(brandID, { page: 1, pageSize: 1, confirmed: false }, { signal }),
  });
  const invalidate = () => queryClient.invalidateQueries({ queryKey: competitorKey(brandID) });
  const create = useMutation({
    mutationFn: (request: CreateCompetitor) => api.competitors.create(brandID, request),
    onSuccess: async () => {
      setPage(lastPage((competitors.data?.totalItems ?? 0) + 1, pageSize));
      await invalidate();
    },
  });
  const update = useMutation({
    mutationFn: ({ id, request }: { id: string; request: CreateCompetitor }) =>
      api.competitors.update(brandID, id, request),
    onSuccess: invalidate,
  });
  const remove = useMutation({
    mutationFn: (id: string) => api.competitors.delete(brandID, id),
    onSuccess: async () => {
      setPage(
        Math.min(
          competitors.data?.currentPage ?? 1,
          lastPage((competitors.data?.totalItems ?? 1) - 1, pageSize),
        ),
      );
      await invalidate();
    },
  });
  const confirm = useMutation({
    mutationFn: (id: string) => api.competitors.confirm(brandID, id),
    onSuccess: invalidate,
  });
  const confirmAll = useMutation({
    mutationFn: () => api.competitors.confirmAll(brandID),
    onSuccess: invalidate,
  });

  return { competitors, unconfirmed, setPage, create, update, remove, confirm, confirmAll };
}
