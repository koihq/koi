export type AnswerMark = "brand" | "rival";

export type AnswerSegment = { text: string; mark?: AnswerMark };

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

/**
 * Splits an engine answer into plain text and named brands so the reader sees at a glance
 * where the brand and its competitors appear. Matching is case-insensitive on word edges.
 */
export function markAnswer(
  text: string,
  brand: string,
  rivals: readonly string[],
): AnswerSegment[] {
  const marks = new Map<string, AnswerMark>();

  for (const rival of rivals) {
    if (rival.trim()) marks.set(rival.trim().toLowerCase(), "rival");
  }
  if (brand.trim()) marks.set(brand.trim().toLowerCase(), "brand");

  if (!text || marks.size === 0) return [{ text }];

  const names = [...marks.keys()].sort((left, right) => right.length - left.length);
  const pattern = new RegExp(
    `(?<![\\p{L}\\p{N}])(${names.map(escapeRegExp).join("|")})(?![\\p{L}\\p{N}])`,
    "giu",
  );
  const segments: AnswerSegment[] = [];
  let index = 0;

  for (const match of text.matchAll(pattern)) {
    const start = match.index ?? 0;

    if (start > index) segments.push({ text: text.slice(index, start) });
    segments.push({ text: match[0], mark: marks.get(match[0].toLowerCase()) });
    index = start + match[0].length;
  }

  if (index < text.length) segments.push({ text: text.slice(index) });

  return segments;
}
