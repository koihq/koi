import type { VisibilityEngine, VisibilityResult } from "@/api/generated";

export type FirstLookVerdict = {
  kind: "rival" | "cited" | "mentioned" | "absent";
  engine: VisibilityEngine;
  competitor?: string;
  result: VisibilityResult;
};

/** Picks the one answer worth leading with: who won instead, then how the brand appeared. */
export function firstLookVerdict(results: VisibilityResult[]): FirstLookVerdict | null {
  const lost = results.find((result) => result.outcome === "miss" && result.appearances.length > 0);
  if (lost) {
    return {
      kind: "rival",
      engine: lost.engine,
      competitor: lost.appearances[0].name,
      result: lost,
    };
  }

  for (const [outcome, kind] of [
    ["cited", "cited"],
    ["mentioned", "mentioned"],
    ["miss", "absent"],
  ] as const) {
    const result = results.find((candidate) => candidate.outcome === outcome);
    if (result) return { kind, engine: result.engine, result };
  }

  return null;
}

export type FirstLookStep =
  | { kind: "rival"; competitor: string; count: number; total: number }
  | { kind: "source"; source: string; count: number }
  | { kind: "prompt"; prompt: string; count: number };

function topEntry(counts: Map<string, number>): [string, number] | null {
  let top: [string, number] | null = null;

  // Map keeps first-seen order, so ties go to what the answers mentioned first.
  for (const entry of counts) {
    if (!top || entry[1] > top[1]) top = entry;
  }

  return top;
}

function increment(counts: Map<string, number>, key: string) {
  counts.set(key, (counts.get(key) ?? 0) + 1);
}

function hostname(source: string): string | null {
  try {
    return new URL(source).hostname.replace(/^www\./, "");
  } catch {
    return null;
  }
}

/**
 * Where to start, computed only from answers that left the brand out. Each step counts what was
 * observed; a brand that appears everywhere gets no steps rather than invented advice.
 */
export function firstLookPlan(results: VisibilityResult[], brandDomain: string): FirstLookStep[] {
  const missed = results.filter((result) => result.outcome === "miss");
  const rivals = new Map<string, number>();
  const sources = new Map<string, number>();
  const prompts = new Map<string, number>();

  for (const result of missed) {
    increment(prompts, result.promptText);

    for (const name of new Set(result.appearances.map((appearance) => appearance.name))) {
      increment(rivals, name);
    }

    const hosts = new Set(result.sources.map(hostname));
    for (const host of hosts) {
      if (host && host !== brandDomain) increment(sources, host);
    }
  }

  const steps: FirstLookStep[] = [];
  const rival = topEntry(rivals);
  const source = topEntry(sources);
  const prompt = topEntry(prompts);

  if (rival) {
    steps.push({ kind: "rival", competitor: rival[0], count: rival[1], total: missed.length });
  }
  if (source) steps.push({ kind: "source", source: source[0], count: source[1] });
  if (prompt) steps.push({ kind: "prompt", prompt: prompt[0], count: prompt[1] });

  return steps;
}

/** Prompts in the order the check asks them, each with its answers so far. */
export function firstLookRows(results: VisibilityResult[]) {
  const rows = new Map<string, { id: string; text: string; answers: VisibilityResult[] }>();

  for (const result of results) {
    const row = rows.get(result.promptId) ?? {
      id: result.promptId,
      text: result.promptText,
      answers: [],
    };

    row.answers.push(result);
    rows.set(result.promptId, row);
  }

  return [...rows.values()];
}

/** What to hand to an agent. It works with or without the Koi skills installed. */
export function agentHandoffPrompt(brand: string) {
  return `Prepare my Koi visibility brief for ${brand} using the Koi MCP server. Then look at this repository and propose the changes you would make to close the gaps it finds.`;
}

const VERDICT_RANK: Record<FirstLookVerdict["kind"], number> = {
  absent: 0,
  mentioned: 1,
  cited: 2,
  rival: 3,
};

/** Keeps the headline still as answers arrive; only a stronger finding may replace it. */
export function keepVerdict(
  held: FirstLookVerdict | null,
  next: FirstLookVerdict | null,
): FirstLookVerdict | null {
  if (!held || !next) return next ?? held;

  return VERDICT_RANK[next.kind] > VERDICT_RANK[held.kind] ? next : held;
}

/** How many real answers named the brand. Answers that never came back are not counted. */
export function firstLookScore(results: VisibilityResult[]) {
  const answered = results.filter(
    (result) =>
      result.outcome === "cited" || result.outcome === "mentioned" || result.outcome === "miss",
  );

  return {
    seen: answered.filter((result) => result.outcome !== "miss").length,
    total: answered.length,
  };
}

function nameMatcher(names: readonly string[]) {
  const usable = names.map((name) => name.trim()).filter(Boolean);
  if (usable.length === 0) return null;

  const escaped = usable.map((name) => name.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"));

  return new RegExp(`(?<![\\p{L}\\p{N}])(${escaped.join("|")})(?![\\p{L}\\p{N}])`, "iu");
}

/**
 * The paragraph that proves the verdict: the first naming a rival, otherwise the brand, otherwise
 * the opening. Answers can be long, so the rest stays behind a disclosure.
 */
export function answerPassage(text: string, brand: string, rivals: readonly string[]) {
  const paragraphs = text
    .split(/\n+/)
    .map((paragraph) => paragraph.trim())
    .filter(Boolean);

  for (const names of [rivals, [brand]]) {
    const matcher = nameMatcher(names);
    const found = matcher && paragraphs.find((paragraph) => matcher.test(paragraph));
    if (found) return { passage: found, trimmed: paragraphs.length > 1 };
  }

  return { passage: paragraphs[0] ?? "", trimmed: paragraphs.length > 1 };
}
