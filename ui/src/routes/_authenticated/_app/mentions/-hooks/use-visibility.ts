import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { useApi } from "@/api/apiProvider";
import { competitorKey, promptKey, runKey } from "@/lib/query-keys";
import {
  isVisibilityActive,
  visibilityPollDelay,
  visibilityReadiness,
} from "@/routes/_authenticated/_app/mentions/-lib/visibility";

export function useVisibility(
  brandID: string,
  selectedRunID: string | undefined,
  page: number,
  onCreated: (runID: string) => void,
) {
  const api = useApi();
  const queryClient = useQueryClient();
  const runs = useQuery({
    queryKey: runKey(brandID),
    queryFn: ({ signal }) => api.runs.list(brandID, 1, 10, { signal }),
    retry: false,
    refetchInterval: (query) =>
      visibilityPollDelay(query.state.data?.items[0]?.state, query.state.fetchFailureCount),
  });
  const runID = selectedRunID || runs.data?.items[0]?.id;
  const status = useQuery({
    queryKey: [...runKey(brandID), runID],
    queryFn: ({ signal }) => {
      if (!runID) throw new Error("Missing visibility run");
      return api.runs.getById(brandID, runID, { signal });
    },
    enabled: !!runID,
    initialData: () => runs.data?.items.find((run) => run.id === runID),
    retry: false,
    refetchInterval: (query) =>
      visibilityPollDelay(query.state.data?.state, query.state.fetchFailureCount),
  });
  const run = status.data;
  const results = useQuery({
    queryKey: [
      ...runKey(brandID),
      runID,
      "results",
      { page, completed: run?.completedResults, state: run?.state },
    ],
    queryFn: ({ signal }) => {
      if (!runID) throw new Error("Missing visibility run");
      return api.runs.listResults(brandID, runID, page, 40, { signal });
    },
    enabled: !!run,
    retry: false,
    placeholderData: (previous, query) =>
      query?.queryKey[3] === runID ? keepPreviousData(previous) : undefined,
  });
  const prompts = useQuery({
    queryKey: [...promptKey(brandID), { page: 1, pageSize: 10 }],
    queryFn: ({ signal }) => api.prompts.list(brandID, { page: 1, pageSize: 10 }, { signal }),
    retry: false,
  });
  const unconfirmedPrompts = useQuery({
    queryKey: [...promptKey(brandID), { page: 1, pageSize: 10, confirmed: false }],
    queryFn: ({ signal }) =>
      api.prompts.list(brandID, { page: 1, pageSize: 10, confirmed: false }, { signal }),
    retry: false,
  });
  const unconfirmedCompetitors = useQuery({
    queryKey: [...competitorKey(brandID), { page: 1, pageSize: 10, confirmed: false }],
    queryFn: ({ signal }) =>
      api.competitors.list(brandID, { page: 1, pageSize: 10, confirmed: false }, { signal }),
    retry: false,
  });
  const readinessLoaded =
    prompts.isSuccess && unconfirmedPrompts.isSuccess && unconfirmedCompetitors.isSuccess;
  const readinessError =
    prompts.isError || unconfirmedPrompts.isError || unconfirmedCompetitors.isError;
  const readiness = visibilityReadiness(
    prompts.data?.totalItems ?? 0,
    unconfirmedPrompts.data?.totalItems ?? 0,
    unconfirmedCompetitors.data?.totalItems ?? 0,
  );
  const create = useMutation({
    mutationFn: () => api.runs.create(brandID),
    onSuccess: (accepted) => {
      queryClient.setQueryData([...runKey(brandID), accepted.id], accepted);
      onCreated(accepted.id);
      void queryClient.invalidateQueries({ queryKey: runKey(brandID), exact: true });
    },
    onError: () => {
      void queryClient.invalidateQueries({ queryKey: promptKey(brandID) });
      void queryClient.invalidateQueries({ queryKey: competitorKey(brandID) });
    },
  });
  const busy =
    create.isPending ||
    isVisibilityActive(run?.state) ||
    isVisibilityActive(runs.data?.items[0]?.state);
  const canStart = readinessLoaded && !readinessError && readiness.ready && runs.isSuccess && !busy;
  const retryReadiness = () =>
    Promise.all([
      prompts.refetch(),
      unconfirmedPrompts.refetch(),
      unconfirmedCompetitors.refetch(),
    ]);
  return {
    runs,
    status,
    run,
    runID,
    results,
    create,
    busy,
    canStart,
    readiness,
    readinessLoaded,
    readinessError,
    retryReadiness,
  };
}
