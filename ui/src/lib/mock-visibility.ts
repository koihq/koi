import type { Competitor } from "@/api/competitors.api";
import type { VisibilityEngine } from "@/api/generated";
import type { Prompt } from "@/api/prompts.api";

export const engines: { id: VisibilityEngine; name: string }[] = [
  { id: "chatgpt", name: "ChatGPT" },
  { id: "perplexity", name: "Perplexity" },
  { id: "gemini", name: "Gemini" },
  { id: "google_ai_overviews", name: "Google AI Overviews" },
];

export type VisibilityScope = {
  days: 7 | 30 | 90;
  engines: VisibilityEngine[];
};

function hash(value: string) {
  let code = 2166136261;
  for (const character of value) {
    code ^= character.charCodeAt(0);
    code = Math.imul(code, 16777619);
  }

  code ^= code >>> 16;
  code = Math.imul(code, 0x7feb352d);
  code ^= code >>> 15;
  code = Math.imul(code, 0x846ca68b);
  code ^= code >>> 16;
  return code >>> 0;
}

function noise(key: string) {
  return (hash(key) / 0xffffffff) * 2 - 1;
}

function smoothNoise(key: string, day: number, span: number) {
  const segment = Math.floor(day / span);
  const progress = day / span - segment;
  const blend = progress * progress * (3 - 2 * progress);
  return noise(`${key}:${segment}`) * (1 - blend) + noise(`${key}:${segment + 1}`) * blend;
}

function createSeries(
  brandID: string,
  prompts: Prompt[],
  scope: VisibilityScope,
  competitor: Competitor | undefined,
  daysAgo: number,
) {
  const included = prompts.filter((prompt) => prompt.confirmed);
  if (!included.length) return [];

  const selectedEngines = scope.engines.length ? scope.engines : engines.map((engine) => engine.id);
  const target = competitor?.id ?? brandID;
  const samples = included.flatMap((prompt) =>
    selectedEngines.map((engine) => ({
      key: `${target}:${prompt.id}:${engine}`,
      base: 35 + (hash(`${target}:${prompt.id}`) % 31) + (hash(engine) % 11) - 5,
    })),
  );
  const baseline = samples.reduce((sum, sample) => sum + sample.base, 0) / samples.length;
  const today = new Date();
  today.setHours(12, 0, 0, 0);

  return Array.from({ length: scope.days }, (_, index) => {
    const date = new Date(today);
    date.setDate(today.getDate() - daysAgo - (scope.days - 1 - index));
    const day = Math.floor(
      Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()) / 86_400_000,
    );
    const trend =
      smoothNoise(`${target}:long`, day, 43) * 12 +
      smoothNoise(`${target}:short`, day, 13) * 5 +
      noise(`${target}:day:${day}`) * 1.6;
    const sampleVariation =
      samples.reduce((sum, sample) => sum + noise(`${sample.key}:${day}`) * 2, 0) / samples.length;
    const rate = Math.max(3, Math.min(96, baseline + trend + sampleVariation));

    return { date, rate };
  });
}

export function mockSeries(
  brandID: string,
  prompts: Prompt[],
  scope: VisibilityScope,
  competitor?: Competitor,
) {
  return createSeries(brandID, prompts, scope, competitor, 0);
}

export function mockRate(
  brandID: string,
  prompts: Prompt[],
  scope: VisibilityScope,
  competitor?: Competitor,
) {
  const series = mockSeries(brandID, prompts, scope, competitor);
  if (!series.length) return null;

  return series.reduce((sum, point) => sum + point.rate, 0) / series.length;
}

export function mockChangePoints(
  brandID: string,
  prompts: Prompt[],
  scope: VisibilityScope,
  competitor?: Competitor,
) {
  const current = mockRate(brandID, prompts, scope, competitor);
  if (current === null) return null;

  const previousSeries = createSeries(brandID, prompts, scope, competitor, scope.days);
  const previous =
    previousSeries.reduce((sum, point) => sum + point.rate, 0) / previousSeries.length;
  return current - previous;
}
