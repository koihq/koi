import type { CompetitorRequest, CompetitorRes, CompetitorsApi } from "@/api/generated";
import { callApi } from "@/api/shared/apiClient";
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
    list: (
      brandID: string,
      params: CompetitorListParams = {},
      initOverrides?: RequestInit,
    ): Promise<ApiListResponse<Competitor>> =>
      callApi(() =>
        api.listCompetitors(
          brandID,
          params.page,
          params.pageSize,
          params.q,
          params.confirmed,
          initOverrides,
        ),
      ),
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
