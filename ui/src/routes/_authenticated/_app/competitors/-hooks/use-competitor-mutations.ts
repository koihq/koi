import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useApi } from "@/api/apiProvider";
import type { CreateCompetitor } from "@/api/competitors.api";
import { competitorKey, runKey, trackingKey } from "@/lib/query-keys";

export function useCompetitorMutations(brandID: string) {
  const api = useApi();
  const queryClient = useQueryClient();
  const invalidate = () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: competitorKey(brandID) }),
      queryClient.invalidateQueries({ queryKey: trackingKey(brandID) }),
      queryClient.invalidateQueries({ queryKey: runKey(brandID) }),
    ]);
  const create = useMutation({
    mutationFn: (request: CreateCompetitor) => api.competitors.create(brandID, request),
    onSuccess: invalidate,
  });

  const remove = useMutation({
    mutationFn: (id: string) => api.competitors.delete(brandID, id),
    onSuccess: invalidate,
  });

  const confirm = useMutation({
    mutationFn: (id: string) => api.competitors.confirm(brandID, id),
    onSuccess: invalidate,
  });

  const confirmAll = useMutation({
    mutationFn: () => api.competitors.confirmAll(brandID),
    onSuccess: invalidate,
  });

  return { create, remove, confirm, confirmAll };
}
