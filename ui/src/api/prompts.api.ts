import type { PromptPatchRequest, PromptRequest, PromptRes, PromptsApi } from "@/api/generated";
import { callApi } from "@/api/shared/apiClient";
import type { ApiListResponse, BaseListParams } from "@/api/shared/types";
import type { VisibilityScope } from "@/lib/visibility";

export type Prompt = PromptRes;

export type CreatePrompt = PromptRequest;

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
      brandID: string,
      params: PromptListParams = {},
      initOverrides?: RequestInit,
    ): Promise<ApiListResponse<Prompt>> =>
      callApi(() =>
        api.listPrompts(
          brandID,
          params.page,
          params.pageSize,
          params.q,
          params.confirmed,
          params.sort,
          params.descending,
          params.scope?.days,
          params.scope?.engines.join(",") || undefined,
          initOverrides,
        ),
      ),
    getById: (brandID: string, id: string): Promise<Prompt> =>
      callApi(() => api.getPrompt(brandID, id)),
    create: (brandID: string, request: CreatePrompt): Promise<Prompt> =>
      callApi(() => api.createPrompt(brandID, request)),
    update: (brandID: string, id: string, request: UpdatePrompt): Promise<Prompt> =>
      callApi(() => api.updatePrompt(brandID, id, request)),
    delete: (brandID: string, id: string): Promise<void> =>
      callApi(() => api.deletePrompt(brandID, id)),
    confirm: (brandID: string, id: string): Promise<Prompt> =>
      callApi(() => api.confirmPrompt(brandID, id)),
    confirmAll: (brandID: string): Promise<void> => callApi(() => api.confirmAllPrompts(brandID)),
  };
}
