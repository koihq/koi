export function safeSourceURL(value: string) {
  try {
    const url = new URL(value);

    return url.protocol === "https:" || url.protocol === "http:" ? url.href : undefined;
  } catch {
    return undefined;
  }
}
