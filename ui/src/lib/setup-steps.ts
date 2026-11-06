export type SetupStep = "choice" | "website" | "confirm" | "agent_connect" | "agent_prompt";

/**
 * Both paths have the same length, and on Koi Cloud they end on the first answers, so finishing
 * setup and seeing the result are one step. Without an agent address there is nothing to choose.
 */
export function setupProgress(
  step: SetupStep | "first_look",
  { canChoose, cloud }: { canChoose: boolean; cloud: boolean },
) {
  const order: Record<SetupStep | "first_look", number> = {
    choice: 1,
    website: 2,
    agent_connect: 2,
    confirm: 3,
    agent_prompt: 3,
    first_look: 4,
  };
  const skipped = canChoose ? 0 : 1;

  return { current: order[step] - skipped, total: (cloud ? 4 : 3) - skipped };
}
