export const promptKey = (brandID: string) => ["brands", brandID, "prompts"] as const;
export const competitorKey = (brandID: string) => ["brands", brandID, "competitors"] as const;
export const generationKey = (brandID: string) => ["brands", brandID, "generations"] as const;
export const runKey = (brandID: string) => ["brands", brandID, "runs"] as const;

export const runResultsKey = (brandID: string, runID: string | undefined) =>
  [...runKey(brandID), runID, "results"] as const;
export const trackingKey = (brandID: string) => ["brands", brandID, "tracking"] as const;
