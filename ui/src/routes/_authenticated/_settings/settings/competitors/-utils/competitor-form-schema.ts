import { z } from "zod";

import { validHostname } from "@/components/blocks/brand-form-schema";

export const competitorFormSchema = z.object({
  name: z.string().trim().min(1),
  domain: z
    .string()
    .trim()
    .transform((value) => value.toLowerCase())
    .refine(validHostname),
  aliases: z.string().transform((value) => {
    const seen = new Set<string>();
    return value
      .split(",")
      .map((alias) => alias.trim().replace(/\s+/g, " "))
      .filter((alias) => {
        const key = alias.toLowerCase();
        if (!alias || seen.has(key)) return false;
        seen.add(key);
        return true;
      });
  }),
});
