import type { HealthApi, HealthResponse } from "@/api/generated";
import { callApi } from "@/api/shared/apiClient";

export type Health = HealthResponse;

export function createHealthApi(api: HealthApi) {
  return {
    get: (): Promise<Health> => callApi(() => api.get()),
  };
}
