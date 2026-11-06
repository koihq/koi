import { useEffect, useSyncExternalStore } from "react";
import { classifyLogo, dominantColor, type LogoShape } from "@/lib/logo-shape";

/** What Koi reads from a logo's own pixels: how it sits in a coin, and its brand color. */
export type LogoInfo = { shape: LogoShape; color: string | null };

const infos = new Map<string, LogoInfo>();
const loading = new Set<string>();
const listeners = new Set<() => void>();

function publish(url: string, info: LogoInfo) {
  infos.set(url, info);
  loading.delete(url);

  for (const listener of listeners) listener();
}

function subscribe(listener: () => void) {
  listeners.add(listener);

  return () => listeners.delete(listener);
}

function measure(image: HTMLImageElement): LogoInfo {
  try {
    const width = image.naturalWidth;
    const height = image.naturalHeight;
    const canvas = document.createElement("canvas");

    canvas.width = width;
    canvas.height = height;

    const context = canvas.getContext("2d", { willReadFrequently: true });
    if (!context || width === 0 || height === 0) return { shape: "symbol", color: null };

    context.drawImage(image, 0, 0, width, height);

    const { data } = context.getImageData(0, 0, width, height);

    return { shape: classifyLogo(data, width, height), color: dominantColor(data, width, height) };
  } catch {
    return { shape: "symbol", color: null };
  }
}

/**
 * Loads a logo once per session, off screen, and records its shape and color. The visible
 * `<img>` shares the browser cache entry because both use `crossOrigin="anonymous"`.
 */
function ensure(url: string) {
  if (infos.has(url) || loading.has(url)) return;

  loading.add(url);

  const image = new Image();

  image.crossOrigin = "anonymous";
  image.onload = () => publish(url, measure(image));
  image.onerror = () => publish(url, { shape: "symbol", color: null });
  image.src = url;
}

/** Shape and color of the logo at `url`; undefined until the image has been read. */
export function useLogoInfo(url: string | null): LogoInfo | undefined {
  const info = useSyncExternalStore(subscribe, () => (url ? infos.get(url) : undefined));

  useEffect(() => {
    if (url) ensure(url);
  }, [url]);

  return info;
}
