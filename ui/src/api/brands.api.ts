import type {
  BrandPatchRequest,
  BrandRequest,
  BrandRes,
  BrandsApi,
  CountryOption,
} from "@/api/generated";
import { BrandResFromJSON } from "@/api/generated";
import { mapPage } from "@/api/shared/mappers";
import type { ApiListResponse, BaseListParams } from "@/api/shared/types";

export type Brand = BrandRes;
export type CreateBrand = BrandRequest;
export type UpdateBrand = BrandPatchRequest;
export type BrandCountry = CountryOption;

export function createBrandsApi(client: BrandsApi) {
  return {
    list: async (params: BaseListParams = {}): Promise<ApiListResponse<Brand>> =>
      mapPage(await client.listBrands(params.page, params.pageSize), BrandResFromJSON),
    getById: (id: string) => client.getBrand(id),
    create: (request: CreateBrand) => client.createBrand(request),
    update: (id: string, request: UpdateBrand) => client.updateBrand(id, request),
    listCountries: (query?: string, initOverrides?: RequestInit) =>
      client.listBrandCountries(query, initOverrides),
  };
}
