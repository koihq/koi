import type {
  CompetitorPatchRequest,
  CompetitorRequest,
  CompetitorRes,
  CompetitorsApi,
} from "@/api/generated";
import { CompetitorResFromJSON } from "@/api/generated";
import { mapPage } from "@/api/shared/mappers";
import type { ApiListResponse, BaseListParams } from "@/api/shared/types";

export type Competitor = CompetitorRes;
export type CreateCompetitor = CompetitorRequest;
export type UpdateCompetitor = CompetitorPatchRequest;

export interface CompetitorListParams extends BaseListParams {
  confirmed?: boolean;
}

export function createCompetitorsApi(client: CompetitorsApi) {
  return {
    list: async (
      brandID: string,
      params: CompetitorListParams = {},
      initOverrides?: RequestInit,
    ): Promise<ApiListResponse<Competitor>> =>
      mapPage(
        await client.listCompetitors(
          brandID,
          params.page,
          params.pageSize,
          params.confirmed,
          initOverrides,
        ),
        CompetitorResFromJSON,
      ),
    getById: (brandID: string, id: string) => client.getCompetitor(brandID, id),
    create: (brandID: string, request: CreateCompetitor) =>
      client.createCompetitor(brandID, request),
    update: (brandID: string, id: string, request: UpdateCompetitor) =>
      client.updateCompetitor(brandID, id, request),
    delete: (brandID: string, id: string) => client.deleteCompetitor(brandID, id),
    confirm: (brandID: string, id: string) => client.confirmCompetitor(brandID, id),
    confirmAll: (brandID: string) => client.confirmAllCompetitors(brandID),
  };
}
