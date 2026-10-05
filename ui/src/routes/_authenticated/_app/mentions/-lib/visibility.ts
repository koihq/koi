import type { VisibilityResult, VisibilityRun } from "@/api/runs.api";

export function isVisibilityActive(state: VisibilityRun["state"] | undefined) {
  return state === "queued" || state === "running";
}

export function visibilityPollDelay(state: VisibilityRun["state"] | undefined, failures = 0) {
  return isVisibilityActive(state) ? Math.min(30_000, 1000 * 2 ** Math.min(failures, 5)) : false;
}

export function visibilityReadiness(
  promptCount: number,
  unconfirmedPrompts: number,
  unconfirmedCompetitors: number,
) {
  return {
    needsPrompts: promptCount === 0,
    needsPromptReview: unconfirmedPrompts > 0,
    needsCompetitorReview: unconfirmedCompetitors > 0,
    ready: promptCount > 0 && unconfirmedPrompts === 0 && unconfirmedCompetitors === 0,
  };
}

export function groupVisibilityResults(results: VisibilityResult[]) {
  const rows = new Map<
    string,
    {
      promptID: string;
      promptText: string;
      promptPosition: number;
      results: Map<VisibilityResult["engine"], VisibilityResult>;
    }
  >();
  for (const result of results) {
    let row = rows.get(result.promptId);
    if (!row) {
      row = {
        promptID: result.promptId,
        promptText: result.promptText,
        promptPosition: result.promptPosition,
        results: new Map(),
      };
      rows.set(result.promptId, row);
    }
    row.results.set(result.engine, result);
  }
  return [...rows.values()].sort((a, b) => a.promptPosition - b.promptPosition);
}

export function safeSourceURL(value: string) {
  try {
    const url = new URL(value);
    return url.protocol === "https:" || url.protocol === "http:" ? url.href : undefined;
  } catch {
    return undefined;
  }
}
