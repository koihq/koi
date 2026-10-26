import type { PromptPatchRequest, PromptRequest, PromptRes, PromptsApi } from "@/api/generated";
import type { ApiListResponse, BaseListParams } from "@/api/shared/types";
import type { VisibilityScope } from "@/lib/visibility";

export type Prompt = PromptRes;

export type CreatePrompt = Omit<PromptRequest, "brandId">;

export type UpdatePrompt = PromptPatchRequest;
export type PromptSort = "original" | "prompt" | "rate" | "change";

export interface PromptListParams extends BaseListParams {
  pageSize?: 10 | 20 | 50;
  q?: string;
  confirmed?: boolean;
  sort?: PromptSort;
  descending?: boolean;
  scope?: VisibilityScope;
}

export function createPromptsApi(api: PromptsApi) {
  return {
    list: (
      { brandId, scope, ...params }: PromptListParams & { brandId: string },
      init?: RequestInit,
    ): Promise<ApiListResponse<Prompt>> =>
      api.listPrompts(
        { ...params, brandId, days: scope?.days, engines: scope?.engines.join(",") || undefined },
        init,
      ),
    getById: (id: string, init?: RequestInit): Promise<Prompt> =>
      api.getPrompt({ promptID: id }, init),
    create: (request: CreatePrompt & { brandId: string }): Promise<Prompt> =>
      api.createPrompt({ promptRequest: request }),
    update: (id: string, request: UpdatePrompt): Promise<Prompt> =>
      api.updatePrompt({ promptID: id, promptPatchRequest: request }),
    delete: (id: string): Promise<void> => api.deletePrompt({ promptID: id }),
    confirm: (id: string): Promise<Prompt> => api.confirmPrompt({ promptID: id }),
    confirmAll: (brandId: string): Promise<void> =>
      api.confirmAllPrompts({ brandRequestScope: { brandId } }),
  };
}
