import type { VisibilityEngine } from "@/api/generated";
import type { Prompt } from "@/api/prompts.api";
import type { VisibilityResult } from "@/api/runs.api";
import { ENGINES } from "./data";
import {
  type ExplorationSearch,
  type ExplorationState,
  type PreviewBrand,
  RECORDED_PROMPT,
  resultForState,
} from "./exploration-fixtures";
import {
  HOME_BRANDS,
  HOME_LOGOS,
  type HomeAnswer,
  type HomeBrand,
  homeEvidence,
} from "./home-evidence";

export type EvidenceRecord = { answer: HomeAnswer; result: VisibilityResult; stopped: boolean };
export type PromptRun = {
  id: string;
  date: string;
  timestamp?: string;
  day: number;
  results: Partial<Record<VisibilityEngine, EvidenceRecord>>;
};
export type MentionStats = {
  mentioned: number;
  completed: number;
  rate: number | null;
  previousRate: number | null;
  changePoints: number | null;
};
export type ComparedBrand = {
  id: string;
  name: string;
  domain?: string;
  src?: string;
  measured: boolean;
};
export type PromptRow = {
  prompt: Prompt;
  past: boolean;
  stats: MentionStats;
  engines: { id: string; rate: number | null }[];
  topBrands: { brand: ComparedBrand; rate: number }[];
};
export type VisibilityScope = { days: number; engines?: VisibilityEngine[]; promptIds: string[] };
export const LAST_DAY = 89;

export function promptRuns(records: EvidenceRecord[], promptId: string): PromptRun[] {
  const runs = new Map<string, PromptRun>();

  for (const record of records) {
    if (record.result.promptId !== promptId) continue;

    const id = record.result.runId;
    const run = runs.get(id) ?? {
      id,
      date: record.answer.date,
      timestamp: id.startsWith("recorded:") ? record.result.fetchedAt?.toISOString() : undefined,
      day: record.answer.day,
      results: {},
    };
    run.results[record.result.engine] = record;
    runs.set(id, run);
  }

  return [...runs.values()].sort((a, b) => b.day - a.day || b.id.localeCompare(a.id));
}

export function runCompetitors(results: Partial<Record<VisibilityEngine, EvidenceRecord>>) {
  const found = new Map<string, { id: string; name: string }>();

  for (const record of Object.values(results)) {
    for (const appearance of record?.result.appearances ?? []) {
      const key = appearance.competitorId ?? appearance.name.toLocaleLowerCase();
      found.set(key, { id: key, name: appearance.name });
    }
  }

  return [...found.values()].sort((a, b) => a.name.localeCompare(b.name));
}

export function buildEvidence(brand: HomeBrand, scenario?: ExplorationState): EvidenceRecord[] {
  if (scenario === "empty" || scenario === "review") return [];

  const days = new Map<number, HomeAnswer[]>();
  for (const answer of homeEvidence(brand)) {
    if (
      brand === "deja" &&
      answer.prompt === RECORDED_PROMPT &&
      answer.day !== 84 &&
      answer.day !== 85
    )
      continue;

    const entries = days.get(answer.day) ?? [];
    entries.push(answer);
    days.set(answer.day, entries);
  }

  return [...days.values()].flatMap((answers) =>
    answers.reverse().map((answer, index) => ({
      answer,
      result: resultForState(answer, brand, index, answer.day === LAST_DAY ? scenario : undefined),
      stopped: answer.day === LAST_DAY && scenario === "partial",
    })),
  );
}

export function comparedBrands(brand: HomeBrand, data: PreviewBrand): ComparedBrand[] {
  return [
    { id: brand, name: data.settings.name, ...HOME_LOGOS[HOME_BRANDS[brand].name], measured: true },
    ...data.competitors
      .filter((item) => item.confirmed)
      .map((item) => ({
        id: item.id,
        name: item.name,
        domain: item.domain || undefined,
        measured: HOME_BRANDS[brand].rivals.some(
          (_, index) => item.id === `${brand}:competitor:${index}`,
        ),
      })),
  ];
}

export function completed(record: EvidenceRecord) {
  return (
    record.result.outcome === "mentioned" ||
    record.result.outcome === "cited" ||
    record.result.outcome === "miss"
  );
}

export function included(record: EvidenceRecord, brand: HomeBrand, target: string) {
  return target === brand
    ? record.result.outcome === "mentioned" || record.result.outcome === "cited"
    : record.result.appearances.some((item) => item.competitorId === target);
}

export function inScope(records: EvidenceRecord[], scope: VisibilityScope, previous = false) {
  const end = LAST_DAY + 1 - (previous ? scope.days : 0);
  const ids = new Set(scope.promptIds);

  return records.filter(
    ({ answer, result }) =>
      ids.has(result.promptId) &&
      (!scope.engines?.length || scope.engines.includes(result.engine)) &&
      answer.day >= end - scope.days &&
      answer.day < end,
  );
}

export function rateFor(records: EvidenceRecord[], brand: HomeBrand, target = brand as string) {
  const answers = records.filter(completed);
  const mentioned = answers.filter((record) => included(record, brand, target)).length;

  return {
    mentioned,
    completed: answers.length,
    rate: answers.length ? (mentioned / answers.length) * 100 : null,
  };
}

export function summarize(
  records: EvidenceRecord[],
  brand: HomeBrand,
  scope: VisibilityScope,
  target: ComparedBrand,
): MentionStats {
  const current = rateFor(target.measured ? inScope(records, scope) : [], brand, target.id);
  const previous = rateFor(target.measured ? inScope(records, scope, true) : [], brand, target.id);

  return {
    ...current,
    previousRate: previous.rate,
    changePoints:
      current.rate !== null && previous.rate !== null ? current.rate - previous.rate : null,
  };
}

export function gapChange(you: MentionStats, other: MentionStats) {
  return you.changePoints === null || other.changePoints === null
    ? null
    : you.changePoints - other.changePoints;
}

export function dailySeries(
  records: EvidenceRecord[],
  brand: HomeBrand,
  scope: VisibilityScope,
  target: ComparedBrand,
) {
  const promptIds = new Set(scope.promptIds);
  const selected = records.filter(
    ({ result }) =>
      promptIds.has(result.promptId) &&
      (!scope.engines?.length || scope.engines.includes(result.engine)),
  );
  const byDay = new Map<number, EvidenceRecord[]>();
  for (const record of selected) {
    const entries = byDay.get(record.answer.day) ?? [];
    entries.push(record);
    byDay.set(record.answer.day, entries);
  }

  const firstDay = LAST_DAY + 1 - scope.days;
  let previousRate: number | null = null;

  for (let day = firstDay - 1; day >= 0; day -= 1) {
    const rate = rateFor(target.measured ? (byDay.get(day) ?? []) : [], brand, target.id).rate;
    if (rate === null) continue;

    previousRate = rate;
    break;
  }

  return Array.from({ length: scope.days }, (_, index) => {
    const day = firstDay + index;
    const stats = rateFor(target.measured ? (byDay.get(day) ?? []) : [], brand, target.id);
    const changePoints =
      stats.rate !== null && previousRate !== null ? stats.rate - previousRate : null;

    if (stats.rate !== null) previousRate = stats.rate;

    return {
      day,
      date: collectionDate(day),
      ...stats,
      changePoints,
    };
  });
}

export function collectionDate(day: number) {
  return new Date(Date.UTC(2026, 6, 16 + day)).toISOString().slice(0, 10);
}

export function promptRows(
  records: EvidenceRecord[],
  brand: HomeBrand,
  data: PreviewBrand,
  days: number,
  engines?: VisibilityEngine[],
): PromptRow[] {
  const prompts = new Map(data.prompts.map((prompt) => [prompt.id, { prompt, past: false }]));
  for (const { result } of records) {
    if (!prompts.has(result.promptId))
      prompts.set(result.promptId, {
        prompt: {
          id: result.promptId,
          brandId: brand,
          text: result.promptText,
          source: "manual",
          confirmed: true,
        },
        past: true,
      });
  }
  const brands = comparedBrands(brand, data);

  return [...prompts.values()].map(({ prompt, past }) => {
    const scope = { days, engines, promptIds: [prompt.id] };
    const scores = brands.map((item) => ({
      item,
      rate: summarize(records, brand, scope, item).rate,
    }));
    const topBrands = scores
      .filter(
        (item): item is { item: ComparedBrand; rate: number } =>
          item.rate !== null && item.rate > 0,
      )
      .sort((a, b) => b.rate - a.rate)
      .slice(0, 4)
      .map(({ item, rate }) => ({ brand: item, rate }));

    return {
      prompt,
      past,
      stats: summarize(records, brand, scope, brands[0]),
      engines: ENGINES.filter((item) => !engines?.length || engines.includes(item.id)).map(
        (item) => ({
          id: item.id,
          rate: summarize(records, brand, { ...scope, engines: [item.id] }, brands[0]).rate,
        }),
      ),
      topBrands,
    };
  });
}

export function filterPromptRows(rows: PromptRow[], search: ExplorationSearch) {
  const query = (search.q ?? "").trim().toLowerCase();
  const status = search.status ?? "all";
  const filtered = rows.filter(
    ({ prompt, past }) =>
      prompt.text.toLowerCase().includes(query) &&
      (status === "past"
        ? past
        : !past && (status === "all" || prompt.confirmed === (status === "confirmed"))),
  );
  if (!search.sort || search.sort === "original") return filtered;

  return filtered.sort((a, b) => {
    if (search.sort === "prompt")
      return a.prompt.text.localeCompare(b.prompt.text) * (search.descending ? -1 : 1);
    const left = search.sort === "rate" ? a.stats.rate : a.stats.changePoints;
    const right = search.sort === "rate" ? b.stats.rate : b.stats.changePoints;
    if (left === null) return right === null ? 0 : 1;
    if (right === null) return -1;
    return (left - right) * (search.descending ? -1 : 1);
  });
}

export function percent(value: number | null) {
  return value === null ? "—" : `${Number(value.toFixed(1))}%`;
}

export function points(value: number) {
  return Number(Math.abs(value).toFixed(1)).toString();
}
