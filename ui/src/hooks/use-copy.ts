import { useEffect, useState } from "react";

const COPIED_FOR_MS = 2_000;

export function useCopy() {
  const [copied, setCopied] = useState(false);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    if (!copied) return;

    const timer = window.setTimeout(() => setCopied(false), COPIED_FOR_MS);

    return () => window.clearTimeout(timer);
  }, [copied]);

  async function copy(value: string) {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setFailed(false);
    } catch {
      setFailed(true);
    }
  }

  return { copy, copied, failed };
}
