import { useQuery } from "@tanstack/react-query";
import { useApi } from "@/api/apiProvider";
import type { PromptListParams } from "@/api/prompts.api";
import { keepPreviousDataForScope, promptListOptions } from "@/lib/queries";
import { promptKey } from "@/lib/query-keys";
import type { VisibilityScope } from "@/lib/visibility";
import { type PromptStatus, promptStatusFilters } from "./prompt-status";

type PromptSearch = {
  page: number;
  pageSize: 10 | 20 | 50;
  q: string;
  sort: NonNullable<PromptListParams["sort"]>;
  descending: boolean;
  status: PromptStatus;
};

export function usePromptList(
  brandID: string,
  scope: VisibilityScope,
  search: PromptSearch,
  collecting: boolean,
  listing: { allStatuses: boolean; enabled: boolean },
) {
  const api = useApi();
  const page = Math.max(1, search.page);
  const active = useQuery({
    ...promptListOptions(api, brandID, {
      ...(listing.allStatuses ? {} : promptStatusFilters[search.status]),
      page,
      pageSize: search.pageSize,
      q: search.q.trim() || undefined,
      sort: search.sort,
      descending: search.descending,
      scope,
    }),
    enabled: listing.enabled,
    placeholderData: keepPreviousDataForScope(promptKey(brandID)),
    refetchInterval: collecting ? 5_000 : false,
  });

  return { active, page, totalPages: active.data?.totalPages ?? 0 };
}
