import { apiErrorFromResponse } from "@/api/errors";
import {
  AuthApi,
  BillingApi,
  BrandsApi,
  CapabilitiesApi,
  CompetitorsApi,
  Configuration,
  CredentialsApi,
  GenerationsApi,
  HealthApi,
  PromptsApi,
  VisibilityApi,
} from "@/api/generated";

export function createApiClient() {
  const configuration = new Configuration({
    basePath: "",
    credentials: "include",
    middleware: [
      {
        post: async ({ response }) => {
          if (!response.ok) throw await apiErrorFromResponse(response);
          return response;
        },
      },
    ],
  });

  return {
    auth: new AuthApi(configuration),
    billing: new BillingApi(configuration),
    brands: new BrandsApi(configuration),
    capabilities: new CapabilitiesApi(configuration),
    competitors: new CompetitorsApi(configuration),
    credentials: new CredentialsApi(configuration),
    generations: new GenerationsApi(configuration),
    health: new HealthApi(configuration),
    prompts: new PromptsApi(configuration),
    runs: new VisibilityApi(configuration),
  };
}
