import type {
  BrandPatchRequest,
  BrandRequest,
  BrandRes,
  BrandsApi,
  CountryOption,
} from "@/api/generated";
import type { ApiListResponse, BaseListParams } from "@/api/shared/types";

export type Brand = BrandRes;

export type CreateBrand = BrandRequest;

export type UpdateBrand = BrandPatchRequest;

export type BrandCountry = CountryOption;

export interface BrandListParams extends BaseListParams {
  q?: string;
}

export function createBrandsApi(api: BrandsApi) {
  return {
    list: (params: BrandListParams = {}, init?: RequestInit): Promise<ApiListResponse<Brand>> =>
      api.listBrands(params, init),
    getById: (id: string): Promise<Brand> => api.getBrand({ id }),
    activate: (id: string): Promise<Brand> => api.activateBrand({ id }),
    create: (request: CreateBrand): Promise<Brand> => api.createBrand({ brandRequest: request }),
    update: (id: string, request: UpdateBrand): Promise<Brand> =>
      api.updateBrand({ id, brandPatchRequest: request }),
    listCountries: (query?: string, initOverrides?: RequestInit): Promise<BrandCountry[]> =>
      api.listBrandCountries({ q: query }, initOverrides),
  };
}
