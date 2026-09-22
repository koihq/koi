import { createContext, type ReactNode, useContext } from "react";
import { createBrandsApi } from "@/api/brands.api";
import { createCompetitorsApi } from "@/api/competitors.api";
import { envelopeMiddleware } from "@/api/errors";
import {
  AuthApi,
  BrandsApi,
  CompetitorsApi,
  Configuration,
  HealthApi,
  PromptsApi,
} from "@/api/generated";
import { createPromptsApi } from "@/api/prompts.api";

export function createApi() {
  const config = new Configuration({
    basePath: "",
    credentials: "include",
    middleware: [envelopeMiddleware],
  });

  const brands = new BrandsApi(config);
  const prompts = new PromptsApi(config);
  const competitors = new CompetitorsApi(config);
  return {
    auth: new AuthApi(config),
    brands: createBrandsApi(brands),
    prompts: createPromptsApi(prompts),
    competitors: createCompetitorsApi(competitors),
    health: new HealthApi(config),
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
