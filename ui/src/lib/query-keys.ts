export const promptKey = (brandID: string) => ["brands", brandID, "prompts"] as const;
export const competitorKey = (brandID: string) => ["brands", brandID, "competitors"] as const;
export const generationKey = (brandID: string) => ["brands", brandID, "generations"] as const;
