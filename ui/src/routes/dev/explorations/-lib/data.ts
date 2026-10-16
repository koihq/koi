import type { VisibilityEngine } from "@/api/generated";

export const ENGINES: { id: VisibilityEngine; name: string; short: string; domain: string }[] = [
  { id: "chatgpt", name: "ChatGPT", short: "C", domain: "chatgpt.com" },
  { id: "perplexity", name: "Perplexity", short: "P", domain: "perplexity.ai" },
  { id: "gemini", name: "Gemini", short: "G", domain: "gemini.google.com" },
  { id: "google_ai_overviews", name: "Google AI Overviews", short: "A", domain: "google.com" },
];

export function hash(value: string): number {
  let result = 2166136261;
  for (const character of value) result = Math.imul(result ^ character.charCodeAt(0), 16777619);
  return result >>> 0;
}
