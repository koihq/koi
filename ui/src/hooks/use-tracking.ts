import { useQuery } from "@tanstack/react-query";

import { useApi } from "@/api/apiProvider";
import { trackingKey } from "@/lib/query-keys";

export function useTracking(brandID: string) {
  const api = useApi();

  return useQuery({
    queryKey: trackingKey(brandID),
    queryFn: ({ signal }) => api.runs.tracking(brandID, { signal }),
    refetchInterval: (query) => (query.state.data?.startedAt ? 60_000 : 5_000),
  });
}
