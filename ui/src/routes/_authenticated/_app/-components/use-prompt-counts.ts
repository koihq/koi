import { useQuery } from "@tanstack/react-query";
import { useApi } from "@/api/apiProvider";
import { keepPreviousDataForScope, promptListOptions } from "@/lib/queries";
import { promptKey } from "@/lib/query-keys";
import type { VisibilityScope } from "@/lib/visibility";
import { promptStatusFilters } from "./prompt-status";

/**
 * How many prompts each status tab holds. Each count is the `totalItems` of a one-page
 * request, so the tabs stay accurate without a dedicated counts endpoint. A count keeps its last
 * value while a new scope loads, so the tabs do not shrink and reflow the header meanwhile.
 */
export function usePromptCounts(brandID: string, scope: VisibilityScope) {
  const api = useApi();
  const base = { page: 1, pageSize: 10 as const, scope };
  const placeholderData = keepPreviousDataForScope(promptKey(brandID));
  const tracking = useQuery({
    ...promptListOptions(api, brandID, { ...base, ...promptStatusFilters.tracking }),
    placeholderData,
    select: (page) => page.totalItems,
  });
  const drafts = useQuery({
    ...promptListOptions(api, brandID, { ...base, ...promptStatusFilters.drafts }),
    placeholderData,
    select: (page) => page.totalItems,
  });
  const stopped = useQuery({
    ...promptListOptions(api, brandID, { ...base, ...promptStatusFilters.stopped }),
    placeholderData,
    select: (page) => page.totalItems,
  });

  return { tracking: tracking.data, drafts: drafts.data, stopped: stopped.data };
}
