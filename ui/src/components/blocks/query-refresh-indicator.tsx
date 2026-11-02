export function QueryRefreshIndicator({ active, label }: { active: boolean; label: string }) {
  if (!active) return null;

  return <output className="sr-only">{label}</output>;
}
