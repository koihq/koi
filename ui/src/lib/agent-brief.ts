import type { VisibilityResult } from "@/api/generated";

export type AgentBriefInput = {
  brand: { name: string; domain?: string | null };
  prompt: string;
  engine: string;
  collectedAt: string;
  outcome: string;
  result: Pick<VisibilityResult, "rawAnswer" | "sources" | "appearances" | "mentionVerdict">;
};

const RECOGNITION: Partial<Record<NonNullable<VisibilityResult["mentionVerdict"]>, string>> = {
  no_info: "The engine named the brand but said it does not know it.",
  other_company: "The answer is about a different company with this name.",
  other_meaning: "The answer uses this name with an unrelated meaning.",
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
  const recognition = result.mentionVerdict ? RECOGNITION[result.mentionVerdict] : undefined;

  return [
    `# AI answer evidence: ${prompt}`,
    "",
    `- Brand: ${brand.name}${brand.domain ? ` (${brand.domain})` : ""}`,
    `- Engine: ${engine}`,
    `- Collected: ${collectedAt}`,
    `- Brand outcome: ${outcome}`,
    ...(recognition ? [`- Recognition: ${recognition}`] : []),
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
    `Propose specific content or page changes that would make ${engine} ${recognition ? `recognize ${brand.name} as the product it is, then ` : ""}recommend and cite ${brand.name} for this question. Ground each change in the sources above.`,
    "",
  ].join("\n");
}
