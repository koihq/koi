import type {
  GenerationAccepted,
  GenerationPage,
  GenerationRequest,
  GenerationStatus,
  GenerationsApi,
} from "@/api/generated";
import { callApi } from "@/api/shared/apiClient";

export type GenerationMode = GenerationRequest["mode"];

export type Generation = GenerationStatus;

export function createGenerationsApi(api: GenerationsApi) {
  return {
    create: (
      brandID: string,
      mode: GenerationMode,
      idempotencyKey?: string,
    ): Promise<GenerationAccepted> =>
      callApi(() => api.createGeneration(brandID, { mode }, idempotencyKey)),
    list: (brandID: string, initOverrides?: RequestInit): Promise<GenerationPage> =>
      callApi(() => api.listGenerations(brandID, 1, 1, initOverrides)),
    getById: (
      brandID: string,
      generationID: string,
      initOverrides?: RequestInit,
    ): Promise<Generation> =>
      callApi(() => api.getGeneration(brandID, generationID, initOverrides)),
  };
}
