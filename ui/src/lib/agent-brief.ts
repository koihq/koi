import type { VisibilityResult } from "@/api/generated";

export type AgentBriefInput = {
  brand: { name: string; domain?: string | null };
  prompt: string;
  engine: string;
  collectedAt: string;
  outcome: string;
  result: Pick<VisibilityResult, "rawAnswer" | "sources" | "appearances">;
};

function quote(text: string) {
  return text
    .trim()
    .split("\n")
    .map((line) => `> ${line}`.trimEnd())
    .join("\n");
}

/**
 * Markdown handed to the founder's own coding agent: the evidence from one engine answer
 * and the task it supports. Kept in English because it is agent input, not interface copy.
 */
export function agentBrief({
  brand,
  prompt,
  engine,
  collectedAt,
  outcome,
  result,
}: AgentBriefInput) {
  const rivals = result.appearances.map((appearance) => {
    const citations = appearance.evidence
      .filter((item) => item.kind === "citation")
      .map((item) => item.value);

    return citations.length
      ? `- ${appearance.name} (cited: ${citations.join(", ")})`
      : `- ${appearance.name}`;
  });
  const sources = [...new Set(result.sources)].map((source) => `- ${source}`);

  return [
    `# AI answer evidence: ${prompt}`,
    "",
    `- Brand: ${brand.name}${brand.domain ? ` (${brand.domain})` : ""}`,
    `- Engine: ${engine}`,
    `- Collected: ${collectedAt}`,
    `- Brand outcome: ${outcome}`,
    "",
    "## Competitors in the answer",
    ...(rivals.length ? rivals : ["- None"]),
    "",
    "## Sources",
    ...(sources.length ? sources : ["- None"]),
    "",
    "## Answer",
    result.rawAnswer.trim() ? quote(result.rawAnswer) : "> (empty)",
    "",
    "## Task",
    `Propose specific content or page changes that would make ${engine} recommend and cite ${brand.name} for this question. Ground each change in the sources above.`,
    "",
  ].join("\n");
}
