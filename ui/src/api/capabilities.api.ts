import type { CapabilitiesApi } from "@/api/generated";
import { callApi } from "@/api/shared/apiClient";

export function createCapabilitiesApi(api: CapabilitiesApi) {
  return {
    get: () => callApi(() => api.getCapabilities()),
  };
}
