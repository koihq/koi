export { cn } from "cn";

export function fieldErrorsFromZod<T extends string>(
  issues: { path: PropertyKey[] }[],
  names: Record<string, T | undefined>,
  message: (field: T) => string,
): Partial<Record<T, string>> {
  const next: Partial<Record<T, string>> = {};

  for (const issue of issues) {
    const field = names[String(issue.path[0])];
    if (field && !next[field]) next[field] = message(field);
  }

  return next;
}
