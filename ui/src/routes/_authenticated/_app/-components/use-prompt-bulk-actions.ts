import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useApi } from "@/api/apiProvider";
import type { Prompt } from "@/api/prompts.api";
import { invalidateProduct } from "@/lib/queries";
import { canApply, type PromptBulkKind } from "./prompt-status";

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

  async function run(kind: PromptBulkKind) {
    const eligible = targets.filter((prompt) => canApply(kind, prompt));
    if (!eligible.length) return false;

    setPending(true);
    setFailed(false);
    try {
      const failedIDs = await runPromptBatch(eligible, (prompt) => {
        if (kind === "confirm") return api.prompts.confirm(prompt.id);
        if (kind === "delete") return api.prompts.delete(prompt.id);

        return api.prompts.update(prompt.id, { inactive: kind === "stop" });
      });
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
