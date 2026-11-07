import { z } from "zod";

export const PROMPT_TEXT_MAX = 4000;

export const promptFormSchema = z.object({ text: z.string().trim().min(1).max(PROMPT_TEXT_MAX) });
