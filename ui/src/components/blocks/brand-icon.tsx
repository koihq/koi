import { useQuery } from "@tanstack/react-query";
import { cn } from "cn";
import { type SyntheticEvent, useState } from "react";
import { useApi } from "@/api/apiProvider";
import { classifyLogo, type LogoShape } from "@/lib/logo-shape";
import { capabilitiesOptions } from "@/lib/queries";

const LOGO_SIZE = 64;

/** Shapes already read from a logo, by URL, so each asset is measured once per session. */
const shapes = new Map<string, LogoShape>();

/**
 * Logo.dev CDN URL. Logos always come in their light variant: every logo sits on its own
 * disc (its background, or white), so the page theme never changes how it is drawn.
 */
export function logoDevURL(domain: string, token: string | undefined): string | null {
  const key = token?.trim();
  if (!key || !domain) return null;

  return `https://img.logo.dev/${encodeURIComponent(domain)}?token=${encodeURIComponent(key)}&size=${LOGO_SIZE}&format=webp&fallback=404&theme=light`;
}

export function brandInitial(name: string, domain: string): string {
  return (name.trim() || domain.trim() || "?").charAt(0).toLocaleUpperCase();
}

function measure(image: HTMLImageElement): LogoShape {
  try {
    const canvas = document.createElement("canvas");
    const width = image.naturalWidth || LOGO_SIZE;
    const height = image.naturalHeight || LOGO_SIZE;

    canvas.width = width;
    canvas.height = height;

    const context = canvas.getContext("2d", { willReadFrequently: true });
    if (!context) return "symbol";

    context.drawImage(image, 0, 0, width, height);

    return classifyLogo(context.getImageData(0, 0, width, height).data, width, height);
  } catch {
    return "symbol";
  }
}

/**
 * A brand as a round coin: its logo when Logo.dev has one, otherwise its initial.
 * Logos with their own background or round badge fill the coin; transparent glyphs sit
 * smaller on a white disc. The shape is read from the image, not from a list of brands.
 */
export function BrandIcon({
  name,
  domain = "",
  src,
  lookupLogo = true,
  className,
}: {
  name: string;
  domain?: string;
  src?: string;
  lookupLogo?: boolean;
  className?: string;
}) {
  const capabilities = useQuery({ ...capabilitiesOptions(useApi()), enabled: lookupLogo && !src });
  const logoURL =
    src ?? (lookupLogo ? logoDevURL(domain, capabilities.data?.logoDevPublishableKey) : null);
  const [failedURL, setFailedURL] = useState<string>();
  const [measured, setMeasured] = useState<{ url: string; shape: LogoShape }>();
  const showLogo = logoURL !== null && failedURL !== logoURL;
  const shape =
    logoURL === null
      ? undefined
      : (shapes.get(logoURL) ?? (measured?.url === logoURL ? measured.shape : undefined));

  function resolve(image: HTMLImageElement) {
    if (!logoURL || measured?.url === logoURL) return;

    const next = shapes.get(logoURL) ?? measure(image);

    shapes.set(logoURL, next);
    setMeasured({ url: logoURL, shape: next });
  }

  function onLoad(event: SyntheticEvent<HTMLImageElement>) {
    resolve(event.currentTarget);
  }

  /** Images already in the browser cache can finish before React attaches `onLoad`. */
  function attach(image: HTMLImageElement | null) {
    if (image?.complete && image.naturalWidth > 0) resolve(image);
  }

  return (
    <span
      aria-hidden="true"
      data-slot="brand-icon"
      data-shape={showLogo ? shape : undefined}
      className={cn(
        "relative flex size-8 shrink-0 items-center justify-center overflow-hidden rounded-full text-xs font-medium text-muted-foreground",
        !showLogo && "bg-muted",
        showLogo && shape === "symbol" && "bg-white",
        showLogo && "ring-1 ring-black/10 ring-inset dark:ring-white/10",
        className,
      )}
    >
      {showLogo ? (
        <img
          src={logoURL}
          alt=""
          crossOrigin="anonymous"
          className={cn(
            "block",
            shape === undefined && "invisible",
            shape === "symbol" ? "size-[62%] object-contain" : "size-full object-cover",
          )}
          ref={attach}
          onLoad={onLoad}
          onError={() => setFailedURL(logoURL)}
        />
      ) : (
        brandInitial(name, domain)
      )}
    </span>
  );
}
