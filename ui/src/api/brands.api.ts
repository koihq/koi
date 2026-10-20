import type {
  BrandPatchRequest,
  BrandRequest,
  BrandRes,
  BrandsApi,
  CountryOption,
} from "@/api/generated";
import { callApi } from "@/api/shared/apiClient";
import type { ApiListResponse, BaseListParams } from "@/api/shared/types";

export type Brand = BrandRes;

export type CreateBrand = BrandRequest;

export type UpdateBrand = BrandPatchRequest;

export type BrandCountry = CountryOption;

export function createBrandsApi(api: BrandsApi) {
  return {
    list: (params: BaseListParams = {}): Promise<ApiListResponse<Brand>> =>
      callApi(() => api.listBrands(params.page, params.pageSize)),
    getById: (id: string): Promise<Brand> => callApi(() => api.getBrand(id)),
    activate: (id: string): Promise<Brand> => callApi(() => api.activateBrand(id)),
    create: (request: CreateBrand): Promise<Brand> => callApi(() => api.createBrand(request)),
    update: (id: string, request: UpdateBrand): Promise<Brand> =>
      callApi(() => api.updateBrand(id, request)),
    listCountries: (query?: string, initOverrides?: RequestInit): Promise<BrandCountry[]> =>
      callApi(() => api.listBrandCountries(query, initOverrides)),
  };
}
