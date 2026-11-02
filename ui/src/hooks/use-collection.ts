import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { useApi } from "@/api/apiProvider";
import type { VisibilityRunStatus } from "@/api/generated";
import { useTracking } from "@/hooks/use-tracking";
import { isCollecting } from "@/lib/answer-state";
import { latestRunOptions } from "@/lib/queries";
import { promptKey, runKey } from "@/lib/query-keys";

export function useRefreshAfterCollection(brandID: string, runs: VisibilityRunStatus[]) {
  const queryClient = useQueryClient();
  const previous = useRef({ brandID, active: new Set<string>() });

  useEffect(() => {
    const current = runs;
    const finished =
      previous.current.brandID === brandID &&
      current.some((run) => previous.current.active.has(run.id) && !isCollecting(run));
    previous.current = {
      brandID,
      active: new Set(current.filter(isCollecting).map((run) => run.id)),
    };

    if (finished) {
      void queryClient.invalidateQueries({ queryKey: promptKey(brandID) });
      void queryClient.invalidateQueries({
        queryKey: runKey(brandID),
        predicate: (query) => query.queryKey[3] !== "latest",
      });
    }
  }, [brandID, queryClient, runs]);
}

export function useCollection(brandID: string) {
  const api = useApi();
  const tracking = useTracking(brandID);
  const latest = useQuery({
    ...latestRunOptions(api, brandID),
    enabled: Boolean(tracking.data?.startedAt),
  });
  useRefreshAfterCollection(brandID, latest.data?.items ?? []);

  return { collecting: isCollecting(latest.data?.items[0]), latest };
}
