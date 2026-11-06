/**
 * Shared date and number formatting. Every surface shows "Oct 4" for a day and adds the
 * year only when it differs from the current one.
 */
function withYear(date: Date, now: Date): Intl.DateTimeFormatOptions {
  return date.getFullYear() === now.getFullYear() ? {} : { year: "numeric" };
}

export function formatDay(date: Date, locale: string, now = new Date()): string {
  return date.toLocaleDateString(locale, {
    month: "short",
    day: "numeric",
    ...withYear(date, now),
  });
}

export function formatDayRange(start: Date, end: Date, locale: string, now = new Date()): string {
  return `${formatDay(start, locale, now)} – ${formatDay(end, locale, now)}`;
}

export function formatTime(date: Date, locale: string): string {
  return date.toLocaleTimeString(locale, { hour: "numeric", minute: "2-digit" });
}

export function formatDateTime(date: Date, locale: string, now = new Date()): string {
  return `${formatDay(date, locale, now)}, ${formatTime(date, locale)}`;
}

/** Percentage-point difference with a real minus sign, e.g. "+4.2 pp" or "−16.0 pp". */
export function formatPoints(value: number, locale: string): string {
  const sign = value > 0 ? "+" : value < 0 ? "−" : "";
  const digits = Math.abs(value).toLocaleString(locale, {
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
  });

  return `${sign}${digits} pp`;
}

export function formatRate(value: number, locale: string): string {
  return `${value.toLocaleString(locale, { minimumFractionDigits: 1, maximumFractionDigits: 1 })}%`;
}
