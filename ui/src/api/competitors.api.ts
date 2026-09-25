import type { CompetitorRequest, CompetitorRes, CompetitorsApi } from "@/api/generated";
import { CompetitorResFromJSON } from "@/api/generated";
import { callApi } from "@/api/shared/apiClient";
import { mapPage } from "@/api/shared/mappers";
import type { ApiListResponse, BaseListParams } from "@/api/shared/types";

export type Competitor = CompetitorRes;
export type CreateCompetitor = CompetitorRequest;
export interface CompetitorSearchResult {
  name: string;
  domain: string;
}

export interface CompetitorListParams extends BaseListParams {
  pageSize?: 10 | 20 | 50;
  q?: string;
  confirmed?: boolean;
}

export function createCompetitorsApi(api: CompetitorsApi) {
  return {
    list: async (
      brandID: string,
      params: CompetitorListParams = {},
      initOverrides?: RequestInit,
    ): Promise<ApiListResponse<Competitor>> => {
      const response = await callApi(() =>
        api.listCompetitors(
          brandID,
          params.page,
          params.pageSize,
          params.q,
          params.confirmed,
          initOverrides,
        ),
      );
      return mapPage(response, CompetitorResFromJSON);
    },
    getById: (brandID: string, id: string): Promise<Competitor> =>
      callApi(() => api.getCompetitor(brandID, id)),
    search: (
      brandID: string,
      query: string,
      initOverrides?: RequestInit,
    ): Promise<CompetitorSearchResult[]> =>
      callApi(() => api.searchCompetitors(brandID, query, initOverrides)),
    create: (brandID: string, request: CreateCompetitor): Promise<Competitor> =>
      callApi(() => api.createCompetitor(brandID, request)),
    delete: (brandID: string, id: string): Promise<void> =>
      callApi(() => api.deleteCompetitor(brandID, id)),
    confirm: (brandID: string, id: string): Promise<Competitor> =>
      callApi(() => api.confirmCompetitor(brandID, id)),
    confirmAll: (brandID: string): Promise<void> =>
      callApi(() => api.confirmAllCompetitors(brandID)),
  };
}
