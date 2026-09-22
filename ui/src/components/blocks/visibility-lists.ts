import { z } from "zod";

import type { CompetitorRequest, CompetitorRes, PromptRes } from "@/api/generated";
import { validHostname } from "@/components/blocks/brand-form-schema";

export const promptFormSchema = z.object({
  text: z.string().trim().min(1),
});

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

export type CompetitorFormInput = z.input<typeof competitorFormSchema>;

export function competitorRequest(input: CompetitorFormInput): CompetitorRequest {
  return competitorFormSchema.parse(input);
}

export function listsReady(
  prompts: Pick<PromptRes, "confirmed">[],
  competitors: Pick<CompetitorRes, "confirmed">[],
): boolean {
  return (
    prompts.length > 0 &&
    competitors.length > 0 &&
    prompts.every((prompt) => prompt.confirmed) &&
    competitors.every((competitor) => competitor.confirmed)
  );
}

export function brandfetchLogoURL(domain: string, clientID: string | undefined): string | null {
  const id = clientID?.trim();
  if (!id || !domain) return null;
  return `https://cdn.brandfetch.io/domain/${encodeURIComponent(domain)}/w/64/h/64/fallback/404/type/icon?c=${encodeURIComponent(id)}`;
}

export function brandInitial(name: string, domain: string): string {
  return (name.trim() || domain.trim() || "?").charAt(0).toLocaleUpperCase();
}
