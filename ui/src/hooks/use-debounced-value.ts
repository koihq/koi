import { useEffect, useState } from "react";

export function useDebouncedValue(value: string, delay = 300): string {
  const [debounced, setDebounced] = useState(value);

  useEffect(() => {
    const id = window.setTimeout(() => setDebounced(value), delay);

    return () => window.clearTimeout(id);
  }, [delay, value]);

  return debounced;
}
