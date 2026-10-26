import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useApi } from "@/api/apiProvider";
import type { CreatePrompt, UpdatePrompt } from "@/api/prompts.api";
import { invalidateProduct } from "@/lib/queries";

export function usePromptMutations(brandID: string) {
  const api = useApi();
  const queryClient = useQueryClient();
  const invalidate = () => invalidateProduct(queryClient, brandID, "prompts");
  const create = useMutation({
    mutationFn: (request: CreatePrompt) => api.prompts.create({ brandId: brandID, ...request }),
    onSuccess: invalidate,
  });

  const update = useMutation({
    mutationFn: ({ id, request }: { id: string; request: UpdatePrompt }) =>
      api.prompts.update(id, request),
    onSuccess: invalidate,
  });

  const remove = useMutation({
    mutationFn: (id: string) => api.prompts.delete(id),
    onSuccess: invalidate,
  });

  const confirm = useMutation({
    mutationFn: (id: string) => api.prompts.confirm(id),
    onSuccess: invalidate,
  });

  return { create, update, remove, confirm };
}
