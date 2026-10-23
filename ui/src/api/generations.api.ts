import type {
  GenerationAccepted,
  GenerationPage,
  GenerationRequest,
  GenerationStatus,
  GenerationsApi,
} from "@/api/generated";

export type GenerationMode = GenerationRequest["mode"];

export type Generation = GenerationStatus;

export function createGenerationsApi(api: GenerationsApi) {
  return {
    create: (request: GenerationRequest, idempotencyKey?: string): Promise<GenerationAccepted> =>
      api.createGeneration({ generationRequest: request, idempotencyKey }),
    list: (brandId: string, init?: RequestInit): Promise<GenerationPage> =>
      api.listGenerations({ brandId, page: 1, pageSize: 1 }, init),
    getById: (generationID: string, init?: RequestInit): Promise<Generation> =>
      api.getGeneration({ generationID }, init),
  };
}
