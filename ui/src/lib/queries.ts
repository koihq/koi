import { type QueryClient, queryOptions } from "@tanstack/react-query";
import type { Api } from "@/api/apiProvider";
import type { BrandListParams } from "@/api/brands.api";
import type { PromptListParams } from "@/api/prompts.api";
import { isCollecting } from "@/lib/answer-state";
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
import type { VisibilityScope } from "@/lib/visibility";

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

export const latestRunOptions = (api: Api, brandId: string) =>
  queryOptions({
    queryKey: [...runKey(brandId), "latest"],
    queryFn: ({ signal }) => api.runs.list(brandId, 1, 1, { signal }),
    refetchInterval: (query) => (isCollecting(query.state.data?.items[0]) ? 5_000 : 60_000),
    refetchIntervalInBackground: false,
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

export const visibilitySummaryOptions = (api: Api, brandId: string, scope: VisibilityScope) =>
  queryOptions({
    queryKey: [...promptKey(brandId), "visibility-summary", scope],
    queryFn: ({ signal }) => api.runs.summary(brandId, scope, { signal }),
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

// Retain filtered results only while viewing the same brand/resource.
export function keepPreviousDataForScope(scope: readonly string[]) {
  return <T>(data: T | undefined, query: { queryKey: readonly unknown[] } | undefined) =>
    scope.every((part, index) => query?.queryKey[index] === part) ? data : undefined;
}
