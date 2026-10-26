import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useApi } from "@/api/apiProvider";
import type { CreateCompetitor, UpdateCompetitor } from "@/api/competitors.api";
import { invalidateProduct } from "@/lib/queries";

export function useCompetitorMutations(brandID: string) {
  const api = useApi();
  const queryClient = useQueryClient();
  const invalidate = () => invalidateProduct(queryClient, brandID, "competitors");
  const create = useMutation({
    mutationFn: (request: CreateCompetitor) =>
      api.competitors.create({ brandId: brandID, ...request }),
    onSuccess: invalidate,
  });

  const update = useMutation({
    mutationFn: ({ id, request }: { id: string; request: UpdateCompetitor }) =>
      api.competitors.update(id, request),
    onSuccess: invalidate,
  });

  const remove = useMutation({
    mutationFn: (id: string) => api.competitors.delete(id),
    onSuccess: invalidate,
  });

  const confirm = useMutation({
    mutationFn: (id: string) => api.competitors.confirm(id),
    onSuccess: invalidate,
  });

  return { create, update, remove, confirm };
}
