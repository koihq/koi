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
  return code >>> 0;
}

function rateForDay(brandID: string, promptID: string, engine: string, day: number) {
  const base = 24 + (hash(`${brandID}:${promptID}`) % 43);
  const engineOffset = (hash(engine) % 19) - 9;
  const wave =
    Math.sin(day / 8 + (hash(promptID) % 17)) * 8 + Math.sin(day / 2.6 + (hash(brandID) % 11)) * 10;
  const variation = (hash(`${brandID}:${promptID}:${engine}:${day}`) % 17) - 8;
  const trend = day * 0.075;
  return Math.max(3, Math.min(96, base + engineOffset + wave + variation + trend));
}

export function mockSeries(
  brandID: string,
  prompts: Prompt[],
  scope: VisibilityScope,
  competitor?: Competitor,
) {
  const included = prompts.filter((prompt) => prompt.confirmed);
  if (!included.length) return [];

  const selectedEngines = scope.engines.length ? scope.engines : engines.map((engine) => engine.id);
  const target = competitor?.id ?? brandID;

  return Array.from({ length: scope.days }, (_, index) => {
    const day = 90 - scope.days + index;
    const values = included.flatMap((prompt) =>
      selectedEngines.map((engine) => rateForDay(target, prompt.id, engine, day)),
    );
    const rate = values.reduce((sum, value) => sum + value, 0) / values.length;
    const date = new Date();
    date.setHours(12, 0, 0, 0);
    date.setDate(date.getDate() - (scope.days - 1 - index));

    return { date, rate };
  });
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

  const included = prompts.filter((prompt) => prompt.confirmed);
  const selectedEngines = scope.engines.length ? scope.engines : engines.map((engine) => engine.id);
  const target = competitor?.id ?? brandID;
  let total = 0;

  for (let day = 90 - scope.days * 2; day < 90 - scope.days; day += 1) {
    for (const prompt of included) {
      for (const engine of selectedEngines) {
        total += rateForDay(target, prompt.id, engine, day);
      }
    }
  }

  const previous = total / (scope.days * included.length * selectedEngines.length);
  return current - previous;
}
