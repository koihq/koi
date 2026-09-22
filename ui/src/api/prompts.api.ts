import type { PromptPatchRequest, PromptRequest, PromptRes, PromptsApi } from "@/api/generated";
import { PromptResFromJSON } from "@/api/generated";
import { mapPage } from "@/api/shared/mappers";
import type { ApiListResponse, BaseListParams } from "@/api/shared/types";

export type Prompt = PromptRes;
export type CreatePrompt = PromptRequest;
export type UpdatePrompt = PromptPatchRequest;

export interface PromptListParams extends BaseListParams {
  confirmed?: boolean;
}

export function createPromptsApi(client: PromptsApi) {
  return {
    list: async (
      brandID: string,
      params: PromptListParams = {},
      initOverrides?: RequestInit,
    ): Promise<ApiListResponse<Prompt>> =>
      mapPage(
        await client.listPrompts(
          brandID,
          params.page,
          params.pageSize,
          params.confirmed,
          initOverrides,
        ),
        PromptResFromJSON,
      ),
    getById: (brandID: string, id: string) => client.getPrompt(brandID, id),
    create: (brandID: string, request: CreatePrompt) => client.createPrompt(brandID, request),
    update: (brandID: string, id: string, request: UpdatePrompt) =>
      client.updatePrompt(brandID, id, request),
    delete: (brandID: string, id: string) => client.deletePrompt(brandID, id),
    confirm: (brandID: string, id: string) => client.confirmPrompt(brandID, id),
    confirmAll: (brandID: string) => client.confirmAllPrompts(brandID),
  };
}
