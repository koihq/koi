import { z } from "zod";

export const promptFormSchema = z.object({ text: z.string().trim().min(1) });
