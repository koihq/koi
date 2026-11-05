export type TrackingStartView = "unavailable" | "drafting" | "empty" | "ready";

/** What the first-run panel offers, given the lists the brand has so far. */
export function trackingStartView({
  collectionAvailable,
  drafting,
  prompts,
}: {
  collectionAvailable: boolean;
  drafting: boolean;
  prompts: number;
}): TrackingStartView {
  if (!collectionAvailable) return "unavailable";
  if (drafting) return "drafting";

  return prompts > 0 ? "ready" : "empty";
}
