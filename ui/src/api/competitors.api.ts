import type {
  CompetitorPatchRequest,
  CompetitorRequest,
  CompetitorRes,
  CompetitorsApi,
} from "@/api/generated";
import type { ApiListResponse, BaseListParams } from "@/api/shared/types";

export type Competitor = CompetitorRes;

export type CreateCompetitor = Omit<CompetitorRequest, "brandId">;

export type UpdateCompetitor = CompetitorPatchRequest;

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
      params: CompetitorListParams & { brandId: string },
      init?: RequestInit,
    ): Promise<ApiListResponse<Competitor>> => api.listCompetitors(params, init),
    getById: (id: string): Promise<Competitor> => api.getCompetitor({ competitorID: id }),
    search: (brandId: string, q: string, init?: RequestInit): Promise<CompetitorSearchResult[]> =>
      api.searchCompetitors({ brandId, q }, init),
    create: (request: CreateCompetitor & { brandId: string }): Promise<Competitor> =>
      api.createCompetitor({ competitorRequest: request }),
    update: (id: string, request: UpdateCompetitor): Promise<Competitor> =>
      api.patchCompetitor({ competitorID: id, competitorPatchRequest: request }),
    delete: (id: string): Promise<void> => api.deleteCompetitor({ competitorID: id }),
    confirm: (id: string): Promise<Competitor> => api.confirmCompetitor({ competitorID: id }),
    confirmAll: (brandId: string): Promise<void> =>
      api.confirmAllCompetitors({ brandRequestScope: { brandId } }),
  };
}
