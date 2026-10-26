import type { CapabilitiesApi } from "@/api/generated";

export function createCapabilitiesApi(api: CapabilitiesApi) {
  return {
    get: (init?: RequestInit) => api.getCapabilities(init),
  };
}
