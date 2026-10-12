import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";

import { useApi } from "@/api/apiProvider";
import type { Generation, GenerationMode } from "@/api/generations.api";
import { competitorKey, generationKey, promptKey, trackingKey } from "@/lib/query-keys";

export function generationPollDelay(
  state: Generation["state"] | undefined,
  failures: number,
  jitter: number,
) {
  if (state !== "queued" && state !== "running") return 60_000;

  return Math.min(60_000, 15_000 * 2 ** Math.min(failures, 2) + jitter);
}

export function useGeneration(brandID: string) {
  const api = useApi();
  const queryClient = useQueryClient();
  const [jitter] = useState(() => Math.floor(Math.random() * 3000));
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
    refetchIntervalInBackground: false,
    refetchInterval: (query) =>
      generationPollDelay(query.state.data?.items[0]?.state, query.state.fetchFailureCount, jitter),
  });

  const run = runs.data?.items[0];

  useEffect(() => {
    if (!run || (run.state !== "succeeded" && run.state !== "failed")) return;

    if (lastTerminal.current === run.id) return;

    lastTerminal.current = run.id;
    void queryClient.invalidateQueries({ queryKey: trackingKey(brandID) });
    if (run.state === "succeeded") {
      void queryClient.invalidateQueries({ queryKey: promptKey(brandID) });
      void queryClient.invalidateQueries({ queryKey: competitorKey(brandID) });
    }
  }, [brandID, queryClient, run]);

  const create = useMutation({
    mutationFn: (mode: GenerationMode) => api.generations.create(brandID, mode),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: generationKey(brandID), exact: true }),
  });

  return { capabilities, runs, run, create };
}
