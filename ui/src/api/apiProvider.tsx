import { createContext, type ReactNode, useContext } from "react";
import { createAuthApi } from "@/api/auth.api";
import { createBillingApi } from "@/api/billing.api";
import { createBrandsApi } from "@/api/brands.api";
import { createCapabilitiesApi } from "@/api/capabilities.api";
import { createCompetitorsApi } from "@/api/competitors.api";
import { createCredentialsApi } from "@/api/credentials.api";
import { createGenerationsApi } from "@/api/generations.api";
import { createHealthApi } from "@/api/health.api";
import { createOAuthApi } from "@/api/oauth.api";
import { createPromptsApi } from "@/api/prompts.api";
import { createRunsApi } from "@/api/runs.api";
import { createApiClient } from "@/api/shared/apiClient";

export function createApi() {
  const client = createApiClient();

  return {
    auth: createAuthApi(client.auth),
    brands: createBrandsApi(client.brands),
    billing: createBillingApi(client.billing),
    capabilities: createCapabilitiesApi(client.capabilities),
    prompts: createPromptsApi(client.prompts),
    runs: createRunsApi(client.runs),
    competitors: createCompetitorsApi(client.competitors),
    credentials: createCredentialsApi(client.credentials),
    generations: createGenerationsApi(client.generations),
    health: createHealthApi(client.health),
    oauth: createOAuthApi(),
  };
}

export type Api = ReturnType<typeof createApi>;

const ApiContext = createContext<Api | null>(null);

export function ApiProvider({ api, children }: { api: Api; children: ReactNode }) {
  return <ApiContext.Provider value={api}>{children}</ApiContext.Provider>;
}

export function useApi(): Api {
  const api = useContext(ApiContext);
  if (!api) {
    throw new Error("useApi must be used within ApiProvider");
  }

  return api;
}
