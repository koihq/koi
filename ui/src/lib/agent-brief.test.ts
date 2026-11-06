import { describe, expect, it } from "vitest";

import { agentBrief } from "@/lib/agent-brief";

describe("agentBrief", () => {
  it("lists rivals with citations, unique sources and the quoted answer", () => {
    const brief = agentBrief({
      brand: { name: "Koi", domain: "koi.dev" },
      prompt: "Best AEO tools?",
      engine: "ChatGPT",
      collectedAt: "Nov 2, 8:22 AM",
      outcome: "Cited",
      result: {
        mentionVerdict: null,
        rawAnswer: "Koi and Peec AI.\nBoth work.",
        sources: ["https://peec.ai/", "https://peec.ai/"],
        appearances: [
          {
            competitorId: "1",
            name: "Peec AI",
            confirmed: true,
            evidence: [
              { kind: "mention", value: "Peec AI" },
              { kind: "citation", value: "https://peec.ai/" },
            ],
          },
        ],
      },
    });

    expect(brief).toContain("# AI answer evidence: Best AEO tools?");
    expect(brief).toContain("- Brand: Koi (koi.dev)");
    expect(brief).toContain("- Peec AI (cited: https://peec.ai/)");
    expect(brief.match(/- https:\/\/peec\.ai\//g)).toHaveLength(1);
    expect(brief).toContain("> Koi and Peec AI.\n> Both work.");
  });

  it("states empty sections explicitly", () => {
    const brief = agentBrief({
      brand: { name: "Koi" },
      prompt: "Q",
      engine: "Gemini",
      collectedAt: "Nov 2",
      outcome: "Miss",
      result: { mentionVerdict: null, rawAnswer: "", sources: [], appearances: [] },
    });

    expect(brief).toContain("## Competitors in the answer\n- None");
    expect(brief).toContain("## Sources\n- None");
    expect(brief).toContain("> (empty)");
  });

  it("tells the agent when the engine doesn't recognize the brand", () => {
    const brief = agentBrief({
      brand: { name: "Koi" },
      prompt: "What is Koi?",
      engine: "ChatGPT",
      collectedAt: "Nov 2, 8:22 AM",
      outcome: "Not in the answer",
      result: {
        mentionVerdict: "no_info",
        rawAnswer: "I couldn't identify Koi.",
        sources: [],
        appearances: [],
      },
    });

    expect(brief).toContain(
      "- Recognition: The engine named the brand but said it does not know it.",
    );
    expect(brief).toContain(
      "make ChatGPT recognize Koi as the product it is, then recommend and cite Koi",
    );
  });
});
