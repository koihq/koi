import type {
  BrandPatchRequest,
  BrandRequest,
  BrandRes,
  BrandsApi,
  CountryOption,
} from "@/api/generated";
import { BrandResFromJSON } from "@/api/generated";
import { callApi } from "@/api/shared/apiClient";
import { mapPage } from "@/api/shared/mappers";
import type { ApiListResponse, BaseListParams } from "@/api/shared/types";

export type Brand = BrandRes;
export type CreateBrand = BrandRequest;
export type UpdateBrand = BrandPatchRequest;
export type BrandCountry = CountryOption;

export function createBrandsApi(api: BrandsApi) {
  return {
    list: async (params: BaseListParams = {}): Promise<ApiListResponse<Brand>> => {
      const response = await callApi(() => api.listBrands(params.page, params.pageSize));
      return mapPage(response, BrandResFromJSON);
    },
    getById: (id: string): Promise<Brand> => callApi(() => api.getBrand(id)),
    create: (request: CreateBrand): Promise<Brand> => callApi(() => api.createBrand(request)),
    update: (id: string, request: UpdateBrand): Promise<Brand> =>
      callApi(() => api.updateBrand(id, request)),
    listCountries: (query?: string, initOverrides?: RequestInit): Promise<BrandCountry[]> =>
      callApi(() => api.listBrandCountries(query, initOverrides)),
  };
}
