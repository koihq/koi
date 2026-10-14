import { describe, expect, it } from "vitest";
import { splitHomeBrandText } from "./home-brand-text";

describe("Home brand mentions", () => {
  it("recognizes brands and engines without changing prose or matching inside words", () => {
    const text =
      "Tally, TYPEFORM and Kit: a toolkit for ChatGPT and zsh-autosuggestions. Tally’s results.";
    const parts = splitHomeBrandText(text);

    expect(parts.map((part) => part.text).join("")).toBe(text);
    expect(parts.filter((part) => part.brand).map((part) => part.brand?.name)).toEqual([
      "Tally",
      "Typeform",
      "Kit",
      "ChatGPT",
      "zsh-autosuggestions",
      "Tally",
    ]);
    expect(new Set(parts.map((part) => part.offset)).size).toBe(parts.length);
    expect(splitHomeBrandText("")).toEqual([]);
  });
});
