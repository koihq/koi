import { describe, expect, it } from "vitest";

import { promptFormSchema } from "@/routes/_authenticated/_app/prompts/-types/prompt-form-schema";

describe("promptFormSchema", () => {
  it("validates and trims prompts", () => {
    expect(promptFormSchema.parse({ text: "  What is Koi?  " })).toEqual({ text: "What is Koi?" });
    expect(promptFormSchema.safeParse({ text: " " }).success).toBe(false);
  });
});
