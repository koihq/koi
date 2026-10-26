import type { VisibilityEngine, VisibilityReport } from "@/api/generated";

export const engines: { id: VisibilityEngine; name: string; domain: string }[] = [
  { id: "chatgpt", name: "ChatGPT", domain: "chatgpt.com" },
  { id: "perplexity", name: "Perplexity", domain: "perplexity.ai" },
  { id: "gemini", name: "Gemini", domain: "gemini.google.com" },
  { id: "google_ai_overviews", name: "Google AI Overviews", domain: "google.com" },
];

export type VisibilityScope = { days: 7 | 30 | 90; engines: VisibilityEngine[] };
export type { VisibilityReport };

export function rate(report?: VisibilityReport, engine?: VisibilityEngine, competitorID?: string) {
  if (competitorID)
    return report?.competitors.find((item) => item.id === competitorID)?.metric.rate ?? null;
  return engine ? (report?.engines[engine]?.rate ?? null) : (report?.metric.rate ?? null);
}

export function change(report?: VisibilityReport) {
  const current = report?.metric.rate;
  const previous = report?.previous.rate;
  return current == null || previous == null ? null : current - previous;
}

export function series(report?: VisibilityReport, competitorID?: string) {
  const days = competitorID
    ? report?.competitors.find((item) => item.id === competitorID)?.series
    : report?.series;
  return (days ?? []).map((day) => ({ date: day.date, rate: day.metric.rate }));
}
