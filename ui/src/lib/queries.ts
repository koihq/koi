import { type QueryClient, queryOptions } from "@tanstack/react-query";
import type { Api } from "@/api/apiProvider";
import type { PromptListParams } from "@/api/prompts.api";
import { collectPages } from "@/lib/collect-pages";
import {
  accountKey,
  billingSubscriptionKey,
  brandsKey,
  capabilitiesKey,
  competitorKey,
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
export const brandsOptions = (api: Api, organizationId: string) =>
  queryOptions({
    queryKey: brandsKey(organizationId),
    queryFn: ({ signal }) =>
      collectPages((page) => api.brands.list({ page, pageSize: 100 }, { signal })),
  });
export const billingOptions = (api: Api, organizationId: string) =>
  queryOptions({
    queryKey: billingSubscriptionKey(organizationId),
    queryFn: ({ signal }) => api.billing.subscription({ signal }),
    staleTime: 10_000,
  });
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
