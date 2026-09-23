export function listsReady(
  prompts: { totalItems: number },
  competitors: { totalItems: number },
  unconfirmedPrompts: { totalItems: number },
  unconfirmedCompetitors: { totalItems: number },
): boolean {
  return (
    prompts.totalItems > 0 &&
    competitors.totalItems > 0 &&
    unconfirmedPrompts.totalItems === 0 &&
    unconfirmedCompetitors.totalItems === 0
  );
}
