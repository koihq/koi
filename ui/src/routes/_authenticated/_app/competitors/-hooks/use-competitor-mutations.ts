import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useApi } from "@/api/apiProvider";
import type { CreateCompetitor } from "@/api/competitors.api";
import { competitorKey } from "@/routes/_authenticated/_app/competitors/-hooks/use-competitor-list";

export function useCompetitorMutations(brandID: string) {
  const api = useApi();
  const queryClient = useQueryClient();
  const invalidate = () => queryClient.invalidateQueries({ queryKey: competitorKey(brandID) });
  const create = useMutation({
    mutationFn: (request: CreateCompetitor) => api.competitors.create(brandID, request),
    onSuccess: invalidate,
  });
  const update = useMutation({
    mutationFn: ({ id, request }: { id: string; request: CreateCompetitor }) =>
      api.competitors.update(brandID, id, request),
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
  return { create, update, remove, confirm, confirmAll };
}
