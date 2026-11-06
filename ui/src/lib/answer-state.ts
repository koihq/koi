import type { VisibilityEngine, VisibilityResult, VisibilityRunStatus } from "@/api/generated";

export type AnswerState =
  | NonNullable<VisibilityResult["outcome"]>
  | "pending"
  | "not-collected"
  | "not-requested";

export function isCollecting(run?: Pick<VisibilityRunStatus, "state">) {
  return run?.state === "queued" || run?.state === "running";
}

export function getAnswerState(
  run: Pick<VisibilityRunStatus, "state" | "engines">,
  engine: VisibilityEngine,
  result?: Pick<VisibilityResult, "outcome" | "errorCode">,
): AnswerState {
  if (result?.errorCode || result?.outcome === "error") return "error";
  if (result?.outcome) return result.outcome;
  if (!run.engines.includes(engine)) return "not-requested";
  return isCollecting(run) ? "pending" : "not-collected";
}

export function answerMessageKey(state: AnswerState) {
  return state === "cited" || state === "mentioned" || state === "miss" ? "empty" : state;
}
