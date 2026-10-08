import { apiErrorFromResponse } from "@/api/errors";
import {
  AuthApi,
  BrandsApi,
  CapabilitiesApi,
  CompetitorsApi,
  Configuration,
  GenerationsApi,
  HealthApi,
  PromptsApi,
  VisibilityApi,
} from "@/api/generated";

export function createApiClient() {
  const configuration = new Configuration({
    basePath: "",
    credentials: "include",
  });

  return {
    auth: new AuthApi(configuration),
    brands: new BrandsApi(configuration),
    capabilities: new CapabilitiesApi(configuration),
    competitors: new CompetitorsApi(configuration),
    generations: new GenerationsApi(configuration),
    health: new HealthApi(configuration),
    prompts: new PromptsApi(configuration),
    runs: new VisibilityApi(configuration),
  };
}

export async function callApi<T>(request: () => Promise<T>): Promise<T> {
  try {
    return await request();
  } catch (error) {
    if (error && typeof error === "object" && "response" in error) {
      const response = (error as { response?: unknown }).response;
      if (response instanceof Response) throw await apiErrorFromResponse(response);
    }

    throw error;
  }
}
