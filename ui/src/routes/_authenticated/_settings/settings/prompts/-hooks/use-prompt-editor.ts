import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { useApi } from "@/api/apiProvider";
import type { CreatePrompt } from "@/api/prompts.api";
import { lastPage } from "@/utils/pagination";

const pageSize = 10;
const promptKey = (brandID: string) => ["brands", brandID, "prompts"] as const;
const promptListKey = (brandID: string, page: number, confirmed?: boolean) =>
  [
    ...promptKey(brandID),
    { page, pageSize: confirmed === undefined ? pageSize : 1, confirmed },
  ] as const;

export function usePromptEditor(brandID: string) {
  const api = useApi();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const prompts = useQuery({
    queryKey: promptListKey(brandID, page),
    queryFn: ({ signal }) => api.prompts.list(brandID, { page, pageSize }, { signal }),
    placeholderData: keepPreviousData,
  });
  const unconfirmed = useQuery({
    queryKey: promptListKey(brandID, 1, false),
    queryFn: ({ signal }) =>
      api.prompts.list(brandID, { page: 1, pageSize: 1, confirmed: false }, { signal }),
  });
  const invalidate = () => queryClient.invalidateQueries({ queryKey: promptKey(brandID) });
  const create = useMutation({
    mutationFn: (request: CreatePrompt) => api.prompts.create(brandID, request),
    onSuccess: async () => {
      setPage(lastPage((prompts.data?.totalItems ?? 0) + 1, pageSize));
      await invalidate();
    },
  });
  const update = useMutation({
    mutationFn: ({ id, request }: { id: string; request: CreatePrompt }) =>
      api.prompts.update(brandID, id, request),
    onSuccess: invalidate,
  });
  const remove = useMutation({
    mutationFn: (id: string) => api.prompts.delete(brandID, id),
    onSuccess: async () => {
      setPage(
        Math.min(
          prompts.data?.currentPage ?? 1,
          lastPage((prompts.data?.totalItems ?? 1) - 1, pageSize),
        ),
      );
      await invalidate();
    },
  });
  const confirm = useMutation({
    mutationFn: (id: string) => api.prompts.confirm(brandID, id),
    onSuccess: invalidate,
  });
  const confirmAll = useMutation({
    mutationFn: () => api.prompts.confirmAll(brandID),
    onSuccess: invalidate,
  });

  return { prompts, unconfirmed, setPage, create, update, remove, confirm, confirmAll };
}
