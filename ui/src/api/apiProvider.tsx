import { createContext, type ReactNode, useContext } from "react";
import { envelopeMiddleware } from "@/api/errors";
import { AuthApi, Configuration, HealthApi } from "@/api/generated";

export function createApi() {
  const config = new Configuration({
    basePath: "",
    credentials: "include",
    middleware: [envelopeMiddleware],
  });
  return {
    auth: new AuthApi(config),
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
