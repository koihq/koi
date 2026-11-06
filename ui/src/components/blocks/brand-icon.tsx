import { useQuery } from "@tanstack/react-query";
import { cn } from "cn";
import { useState } from "react";
import { useApi } from "@/api/apiProvider";
import { useLogoInfo } from "@/lib/logo-info";
import { capabilitiesOptions } from "@/lib/queries";

const LOGO_SIZE = 64;

/**
 * Logo.dev CDN URL. Logos always come in their light variant: every logo sits on its own
 * disc (its background, or white), so the page theme never changes how it is drawn.
 */
export function logoDevURL(domain: string, token: string | undefined): string | null {
  const key = token?.trim();
  if (!key || !domain) return null;

  return `https://img.logo.dev/${encodeURIComponent(domain)}?token=${encodeURIComponent(key)}&size=${LOGO_SIZE}&format=webp&fallback=404&theme=light`;
}

function useLogoURL(domain: string, enabled: boolean): string | null {
  const capabilities = useQuery({ ...capabilitiesOptions(useApi()), enabled });

  return enabled ? logoDevURL(domain, capabilities.data?.logoDevPublishableKey) : null;
}

/**
 * The brand color read from a domain's logo, or null for monochrome logos, missing logos
 * and instances without Logo.dev. Callers fall back to neutral tokens.
 */
export function useBrandColor(domain: string | undefined): string | null {
  const url = useLogoURL(domain ?? "", Boolean(domain));

  return useLogoInfo(url)?.color ?? null;
}

export function brandInitial(name: string, domain: string): string {
  return (name.trim() || domain.trim() || "?").charAt(0).toLocaleUpperCase();
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
  const lookupURL = useLogoURL(domain, lookupLogo && !src);
  const logoURL = src ?? lookupURL;
  const [failedURL, setFailedURL] = useState<string>();
  const showLogo = logoURL !== null && failedURL !== logoURL;
  const shape = useLogoInfo(showLogo ? logoURL : null)?.shape;

  return (
    <span
      aria-hidden="true"
      data-slot="brand-icon"
      data-shape={showLogo ? shape : undefined}
      className={cn(
        "relative flex size-8 shrink-0 items-center justify-center overflow-hidden rounded-full text-xs font-medium text-muted-foreground",
        !showLogo && "bg-muted",
        showLogo && shape === "symbol" && "bg-white",
        "ring-1 ring-black/10 ring-inset dark:ring-white/10",
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
          onError={() => setFailedURL(logoURL)}
        />
      ) : (
        brandInitial(name, domain)
      )}
    </span>
  );
}
