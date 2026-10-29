import { type QueryClient, queryOptions } from "@tanstack/react-query";
import type { Api } from "@/api/apiProvider";
import type { BrandListParams } from "@/api/brands.api";
import type { PromptListParams } from "@/api/prompts.api";
import {
  accountKey,
  billingSubscriptionKey,
  brandsKey,
  capabilitiesKey,
  competitorKey,
  generationKey,
  promptKey,
  runKey,
  trackingKey,
} from "@/lib/query-keys";

export const accountOptions = (api: Api) =>
  queryOptions({
    queryKey: accountKey,
    queryFn: ({ signal }) => api.auth.me({ signal }),
    retry: false,
  });
export const capabilitiesOptions = (api: Api) =>
  queryOptions({
    queryKey: capabilitiesKey,
    queryFn: ({ signal }) => api.capabilities.get({ signal }),
    staleTime: Number.POSITIVE_INFINITY,
    retry: false,
  });
export const brandListOptions = (api: Api, organizationId: string, params: BrandListParams) =>
  queryOptions({
    queryKey: [...brandsKey(organizationId), "list", params],
    queryFn: ({ signal }) => api.brands.list(params, { signal }),
  });
export const billingOptions = (api: Api, organizationId: string) =>
  queryOptions({
    queryKey: billingSubscriptionKey(organizationId),
    queryFn: ({ signal }) => api.billing.subscription({ signal }),
    staleTime: 10_000,
  });
export const generationOptions = (api: Api, brandId: string) =>
  queryOptions({
    queryKey: generationKey(brandId),
    queryFn: ({ signal }) => api.generations.list(brandId, { signal }),
    retry: false,
  });

export async function ensureInitialGeneration(api: Api, queryClient: QueryClient, brandId: string) {
  const runs = await queryClient.fetchQuery(generationOptions(api, brandId));
  if (runs.items.length > 0) return;

  await api.generations.create({ brandId, mode: "initial" }, `setup-${brandId}`);
  await queryClient.invalidateQueries({ queryKey: generationKey(brandId), exact: true });
}
export const promptListOptions = (api: Api, brandId: string, params: PromptListParams) =>
  queryOptions({
    queryKey: [...promptKey(brandId), "list", params],
    queryFn: ({ signal }) => api.prompts.list({ brandId, ...params }, { signal }),
  });

export function invalidateProduct(
  queryClient: QueryClient,
  brandId: string,
  kind: "prompts" | "competitors",
) {
  const keys = [promptKey(brandId), runKey(brandId), trackingKey(brandId)];
  return Promise.all([
    ...keys.map((queryKey) => queryClient.invalidateQueries({ queryKey })),
    ...(kind === "competitors"
      ? [queryClient.invalidateQueries({ queryKey: competitorKey(brandId) })]
      : []),
  ]);
}

export async function clearAccountCache(queryClient: QueryClient) {
  await queryClient.cancelQueries();
  queryClient.clear();
}
