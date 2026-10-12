import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";

import { useApi } from "@/api/apiProvider";
import { runKey, runResultsKey } from "@/lib/query-keys";
import { visibilityPollDelay } from "@/routes/_authenticated/_app/mentions/-lib/visibility";

export function useVisibility(brandID: string, selectedRunID: string | undefined, page: number) {
  const api = useApi();
  const queryClient = useQueryClient();

  const runs = useQuery({
    queryKey: runKey(brandID),
    queryFn: ({ signal }) => api.runs.list(brandID, 1, 10, { signal }),
    retry: false,
    refetchInterval: (query) =>
      visibilityPollDelay(query.state.data?.items[0]?.state, query.state.fetchFailureCount) ||
      60_000,
  });

  const runID = selectedRunID || runs.data?.items[0]?.id;
  const listedRun = runs.data?.items.find((item) => item.id === runID);
  const needsStatus = !!runID && !listedRun;

  const status = useQuery({
    queryKey: [...runKey(brandID), runID],
    queryFn: ({ signal }) => {
      if (!runID) throw new Error("Missing visibility run");

      return api.runs.getById(brandID, runID, { signal });
    },
    enabled: needsStatus,
    retry: false,
    refetchInterval: (query) =>
      visibilityPollDelay(query.state.data?.state, query.state.fetchFailureCount),
  });

  const run = listedRun ?? status.data;
  const completedResults = run?.completedResults;
  const runState = run?.state;

  const results = useQuery({
    queryKey: [...runResultsKey(brandID, runID), { page, pageSize: 40 }],
    queryFn: ({ signal }) => {
      if (!runID) throw new Error("Missing visibility run");

      return api.runs.listResults(brandID, runID, page, 40, { signal });
    },
    enabled: !!run,
    retry: false,
    placeholderData: (previous, query) =>
      query?.queryKey[1] === brandID && query.queryKey[3] === runID ? previous : undefined,
  });

  useEffect(() => {
    if (!runID || completedResults === undefined || runState === undefined) return;

    void queryClient.invalidateQueries({ queryKey: runResultsKey(brandID, runID) });
  }, [brandID, queryClient, runID, completedResults, runState]);

  return { runs, status, needsStatus, run, runID, results };
}
