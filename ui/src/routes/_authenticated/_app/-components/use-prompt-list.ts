import { useQuery } from "@tanstack/react-query";
import { useApi } from "@/api/apiProvider";
import type { PromptListParams } from "@/api/prompts.api";
import { promptListOptions } from "@/lib/queries";
import { runKey } from "@/lib/query-keys";
import type { VisibilityScope } from "@/lib/visibility";

type PromptSearch = {
  page: number;
  pageSize: 10 | 20 | 50;
  q: string;
  sort: NonNullable<PromptListParams["sort"]>;
  descending: boolean;
  status: string;
};

export function usePromptList(
  brandID: string,
  scope: VisibilityScope,
  search: PromptSearch,
  collecting: boolean,
) {
  const api = useApi();
  const page = Math.max(1, search.page);
  const confirmed =
    search.status === "confirmed" ? true : search.status === "unconfirmed" ? false : undefined;
  const params = {
    page,
    pageSize: search.pageSize,
    q: search.q.trim() || undefined,
    sort: search.sort,
    descending: search.descending,
    scope,
  };
  const live = useQuery({
    ...promptListOptions(api, brandID, { ...params, confirmed }),
    enabled: search.status !== "past",
    refetchInterval: collecting ? 5_000 : false,
  });
  const history = useQuery({
    queryKey: [...runKey(brandID), "past-prompts", params],
    enabled: search.status === "past",
    queryFn: ({ signal }) => api.runs.historicalPrompts(brandID, params, { signal }),
    refetchInterval: collecting ? 5_000 : false,
  });
  const active = search.status === "past" ? history : live;

  return { active, live, history, page, totalPages: active.data?.totalPages ?? 0 };
}
