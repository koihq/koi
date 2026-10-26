import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useApi } from "@/api/apiProvider";
import type { Prompt } from "@/api/prompts.api";
import { invalidateProduct } from "@/lib/queries";

export async function runPromptBatch<T extends { id: string }>(
  targets: T[],
  run: (target: T) => Promise<unknown>,
) {
  const results = await Promise.allSettled(targets.map(run));
  return targets
    .filter((_, index) => results[index].status === "rejected")
    .map((target) => target.id);
}

export function usePromptBulkActions(
  brandID: string,
  targets: Prompt[],
  selectFailures: (ids: string[]) => void,
) {
  const api = useApi();
  const queryClient = useQueryClient();
  const [pending, setPending] = useState(false);
  const [failed, setFailed] = useState(false);

  async function run(kind: "confirm" | "delete") {
    const eligible = targets.filter((prompt) => kind === "delete" || !prompt.confirmed);
    if (!eligible.length) return false;

    setPending(true);
    setFailed(false);
    try {
      const failedIDs = await runPromptBatch(eligible, (prompt) =>
        kind === "confirm" ? api.prompts.confirm(prompt.id) : api.prompts.delete(prompt.id),
      );
      selectFailures(failedIDs);
      setFailed(failedIDs.length > 0);
      await invalidateProduct(queryClient, brandID, "prompts");
      return failedIDs.length === 0;
    } catch {
      setFailed(true);
      return false;
    } finally {
      setPending(false);
    }
  }

  return { run, pending, failed };
}
