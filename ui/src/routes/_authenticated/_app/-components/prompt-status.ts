import type { Prompt } from "@/api/prompts.api";

export const promptStatuses = ["tracking", "drafts", "stopped"] as const;

export type PromptStatus = (typeof promptStatuses)[number];

/** Each prompt belongs to exactly one tab. */
export const promptStatusFilters: Record<
  PromptStatus,
  { confirmed?: boolean; inactive?: boolean }
> = {
  tracking: { confirmed: true, inactive: false },
  drafts: { confirmed: false },
  stopped: { inactive: true },
};

export type PromptBulkKind = "confirm" | "stop" | "resume" | "delete";

type PromptState = Pick<Prompt, "confirmed" | "inactive" | "hasResults">;

const eligibility: Record<PromptBulkKind, (prompt: PromptState) => boolean> = {
  confirm: (prompt) => !prompt.confirmed,
  stop: (prompt) => prompt.confirmed && !prompt.inactive && prompt.hasResults,
  resume: (prompt) => prompt.inactive,
  delete: (prompt) => !prompt.hasResults,
};

/** Whether an action applies to a prompt: one with saved answers is stopped, never deleted. */
export function canApply(kind: PromptBulkKind, prompt: PromptState) {
  return eligibility[kind](prompt);
}
