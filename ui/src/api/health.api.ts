import type { HealthApi, HealthResponse } from "@/api/generated";

export type Health = HealthResponse;

export function createHealthApi(api: HealthApi) {
  return {
    get: (): Promise<Health> => api.get(),
  };
}
