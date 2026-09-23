import { apiErrorFromResponse } from "@/api/errors";
import {
  AuthApi,
  BrandsApi,
  CompetitorsApi,
  Configuration,
  HealthApi,
  PromptsApi,
} from "@/api/generated";

export function createApiClient() {
  const configuration = new Configuration({
    basePath: "",
    credentials: "include",
  });

  return {
    auth: new AuthApi(configuration),
    brands: new BrandsApi(configuration),
    competitors: new CompetitorsApi(configuration),
    health: new HealthApi(configuration),
    prompts: new PromptsApi(configuration),
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
