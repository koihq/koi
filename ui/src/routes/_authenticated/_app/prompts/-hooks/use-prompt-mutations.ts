import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useApi } from "@/api/apiProvider";
import type { CreatePrompt } from "@/api/prompts.api";
import { promptKey } from "@/routes/_authenticated/_app/prompts/-hooks/use-prompt-list";

export function usePromptMutations(brandID: string) {
  const api = useApi();
  const queryClient = useQueryClient();
  const invalidate = () => queryClient.invalidateQueries({ queryKey: promptKey(brandID) });
  const create = useMutation({
    mutationFn: (request: CreatePrompt) => api.prompts.create(brandID, request),
    onSuccess: invalidate,
  });
  const update = useMutation({
    mutationFn: ({ id, request }: { id: string; request: CreatePrompt }) =>
      api.prompts.update(brandID, id, request),
    onSuccess: invalidate,
  });
  const remove = useMutation({
    mutationFn: (id: string) => api.prompts.delete(brandID, id),
    onSuccess: invalidate,
  });
  const confirm = useMutation({
    mutationFn: (id: string) => api.prompts.confirm(brandID, id),
    onSuccess: invalidate,
  });
  const confirmAll = useMutation({
    mutationFn: () => api.prompts.confirmAll(brandID),
    onSuccess: invalidate,
  });

  return { create, update, remove, confirm, confirmAll };
}
