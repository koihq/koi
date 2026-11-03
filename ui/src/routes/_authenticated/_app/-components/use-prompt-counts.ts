import { useQuery } from "@tanstack/react-query";
import { useApi } from "@/api/apiProvider";
import { promptListOptions } from "@/lib/queries";
import { runKey } from "@/lib/query-keys";
import type { VisibilityScope } from "@/lib/visibility";

/**
 * How many prompts each status tab holds. Each count is the `totalItems` of a one-page
 * request, so the tabs stay accurate without a dedicated counts endpoint.
 */
export function usePromptCounts(brandID: string, scope: VisibilityScope) {
  const api = useApi();
  const base = { page: 1, pageSize: 10 as const, scope };
  const confirmed = useQuery({
    ...promptListOptions(api, brandID, { ...base, confirmed: true }),
    select: (page) => page.totalItems,
  });
  const unconfirmed = useQuery({
    ...promptListOptions(api, brandID, { ...base, confirmed: false }),
    select: (page) => page.totalItems,
  });
  const past = useQuery({
    queryKey: [...runKey(brandID), "past-prompts", base],
    queryFn: ({ signal }) => api.runs.historicalPrompts(brandID, base, { signal }),
    select: (page) => page.totalItems,
  });
  const all =
    confirmed.data === undefined || unconfirmed.data === undefined
      ? undefined
      : confirmed.data + unconfirmed.data;

  return { all, confirmed: confirmed.data, unconfirmed: unconfirmed.data, past: past.data };
}
