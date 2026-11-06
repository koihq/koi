import type {
  BrandLookup,
  BrandPatchRequest,
  BrandRequest,
  BrandRes,
  BrandsApi,
  CountryOption,
  OnboardingEventStepEnum,
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
    lookup: (
      identifiers: { domain: string; githubRepo: string },
      init?: RequestInit,
    ): Promise<BrandLookup> =>
      api.lookupBrand(
        identifiers.githubRepo
          ? { githubRepo: identifiers.githubRepo }
          : { domain: identifiers.domain },
        init,
      ),
    recordStep: (step: OnboardingEventStepEnum): Promise<void> =>
      api.recordOnboardingStep({ onboardingEvent: { step } }),
    listCountries: (query?: string, initOverrides?: RequestInit): Promise<BrandCountry[]> =>
      api.listBrandCountries({ q: query }, initOverrides),
  };
}
