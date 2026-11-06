import { useQuery } from "@tanstack/react-query";
import { useApi } from "@/api/apiProvider";
import { competitorKey } from "@/lib/query-keys";

/** How many competitors each status tab holds, from one-page requests. */
export function useCompetitorCounts(brandID: string) {
  const api = useApi();

  const count = (confirmed: boolean) => ({
    queryKey: [...competitorKey(brandID), { page: 1, pageSize: 10, confirmed }],
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      api.competitors.list({ brandId: brandID, page: 1, pageSize: 10, confirmed }, { signal }),
  });
  const confirmed = useQuery({ ...count(true), select: (page) => page.totalItems });
  const unconfirmed = useQuery({ ...count(false), select: (page) => page.totalItems });
  const all =
    confirmed.data === undefined || unconfirmed.data === undefined
      ? undefined
      : confirmed.data + unconfirmed.data;

  return { all, confirmed: confirmed.data, unconfirmed: unconfirmed.data };
}
