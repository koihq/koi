import type { PromptPatchRequest, PromptRequest, PromptRes, PromptsApi } from "@/api/generated";
import { PromptResFromJSON } from "@/api/generated";
import { callApi } from "@/api/shared/apiClient";
import { mapPage } from "@/api/shared/mappers";
import type { ApiListResponse, BaseListParams } from "@/api/shared/types";

export type Prompt = PromptRes;

export type CreatePrompt = PromptRequest;

export type UpdatePrompt = PromptPatchRequest;

export interface PromptListParams extends BaseListParams {
  pageSize?: 10 | 20 | 50;
  q?: string;
  confirmed?: boolean;
}

export function createPromptsApi(api: PromptsApi) {
  return {
    list: async (
      brandID: string,
      params: PromptListParams = {},
      initOverrides?: RequestInit,
    ): Promise<ApiListResponse<Prompt>> => {
      const response = await callApi(() =>
        api.listPrompts(
          brandID,
          params.page,
          params.pageSize,
          params.q,
          params.confirmed,
          initOverrides,
        ),
      );

      return mapPage(response, PromptResFromJSON);
    },
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
