import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";

import { useApi } from "@/api/apiProvider";
import type { Generation, GenerationMode } from "@/api/generations.api";
import { competitorKey, generationKey, promptKey } from "@/lib/query-keys";


export function generationPollDelay(
  state: Generation["state"] | undefined,
  failures: number,
  jitter: number,
) {
  if (state === "succeeded" || state === "failed") return false;
  return Math.min(60_000, 15_000 * 2 ** Math.min(failures, 2) + jitter);
}

export function useGeneration(brandID: string) {
  const api = useApi();
  const queryClient = useQueryClient();
  const [jitter] = useState(() => Math.floor(Math.random() * 3000));
  const [started, setStarted] = useState<Generation>();
  const lastTerminal = useRef<string | undefined>(undefined);
  const capabilities = useQuery({
    queryKey: ["capabilities"],
    queryFn: api.capabilities.get,
    staleTime: Number.POSITIVE_INFINITY,
    retry: false,
  });
  const runs = useQuery({
    queryKey: generationKey(brandID),
    queryFn: ({ signal }) => api.generations.list(brandID, { signal }),
    retry: false,
  });
  const listed = runs.data?.items[0];
  const latest = started && started.id !== listed?.id ? started : listed;
  const active = latest?.state === "queued" || latest?.state === "running";
  const status = useQuery({
    queryKey: [...generationKey(brandID), latest?.id],
    queryFn: ({ signal }) => {
      if (!latest) throw new Error("Missing generation run");
      return api.generations.getById(brandID, latest.id, { signal });
    },
    enabled: active,
    initialData: latest,
    retry: false,
    refetchIntervalInBackground: false,
    refetchInterval: (query) =>
      generationPollDelay(query.state.data?.state, query.state.fetchFailureCount, jitter),
  });
  const run = status.data ?? latest;

  useEffect(() => {
    if (!run || (run.state !== "succeeded" && run.state !== "failed")) return;
    if (started?.id === run.id && started.state !== run.state) setStarted(run);
    if (lastTerminal.current === run.id) return;
    lastTerminal.current = run.id;
    void queryClient.invalidateQueries({ queryKey: generationKey(brandID), exact: true });
    if (run.state === "succeeded") {
      void queryClient.invalidateQueries({ queryKey: promptKey(brandID) });
      void queryClient.invalidateQueries({ queryKey: competitorKey(brandID) });
    }
  }, [brandID, queryClient, run, started?.id, started?.state]);

  const create = useMutation({
    mutationFn: (mode: GenerationMode) => api.generations.create(brandID, mode),
    onSuccess: (accepted) => {
      setStarted({
        id: accepted.id,
        mode: accepted.mode,
        state: accepted.state,
        attempt: 0,
        promptCount: 0,
        competitorCount: 0,
        createdAt: new Date(),
      });
    },
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: generationKey(brandID), exact: true }),
  });

  return { capabilities, runs, status, run, create };
}
