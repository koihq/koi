import { cn } from "cn";
import { useState } from "react";
import { useDarkMode } from "@/hooks/use-dark-mode";

type LogoAppearance = {
  fit?: "image" | "symbol";
  background?: string;
  theme?: "light" | "dark";
};

// These assets already contain a background or a circular badge. Keep them edge to edge.
const logoOverrides: Record<string, LogoAppearance & { domain?: string }> = {
  "amplitude.com": { fit: "image" },
  "beehiiv.com": { fit: "image" },
  "fillout.com": { fit: "image" },
  "gitbook.com": { fit: "image" },
  "jotform.com": { fit: "image" },
  "kit.com": { fit: "image" },
  "mintlify.com": { fit: "image" },
  "mixpanel.com": { fit: "image" },
  "peec.ai": { fit: "image" },
  "posthog.com": { fit: "image" },
  "readme.com": { domain: "readme.io", fit: "image" },
  "readme.io": { fit: "image" },
  "substack.com": { fit: "image" },
  "tryprofound.com": { fit: "image" },
};

export function logoDevURL(
  domain: string,
  token: string | undefined,
  theme: "light" | "dark" = "light",
): string | null {
  const key = token?.trim();
  if (!key || !domain) return null;

  const logoDomain = logoOverrides[domain.toLowerCase()]?.domain ?? domain;

  return `https://img.logo.dev/${encodeURIComponent(logoDomain)}?token=${encodeURIComponent(key)}&size=64&format=webp&fallback=404&theme=${theme}`;
}

export function brandInitial(name: string, domain: string): string {
  return (name.trim() || domain.trim() || "?").charAt(0).toLocaleUpperCase();
}

export function BrandIcon({
  name,
  domain = "",
  src,
  appearance,
  className,
}: {
  name: string;
  domain?: string;
  src?: string;
  appearance?: LogoAppearance;
  className?: string;
}) {
  const dark = useDarkMode();
  const {
    fit = "symbol",
    background,
    theme,
  } = { ...logoOverrides[domain.toLowerCase()], ...appearance };
  const logoURL =
    src ??
    logoDevURL(
      domain,
      import.meta.env.VITE_LOGODEV_PUBLISHABLE_KEY,
      theme ?? (dark ? "dark" : "light"),
    );
  const [failedURL, setFailedURL] = useState<string>();
  const showLogo = logoURL !== null && failedURL !== logoURL;

  return (
    <span
      aria-hidden="true"
      data-slot="brand-icon"
      data-fit={showLogo ? fit : undefined}
      style={showLogo && background ? { backgroundColor: background } : undefined}
      className={cn(
        "relative flex size-8 shrink-0 items-center justify-center rounded-full text-xs font-medium text-muted-foreground",
        showLogo && fit === "image" && "overflow-hidden",
        !showLogo && "bg-muted",
        className,
      )}
    >
      {showLogo ? (
        <img
          src={logoURL}
          alt=""
          className={cn(
            "block object-contain",
            fit === "image" ? "size-full rounded-full" : "size-3/4",
          )}
          onError={() => setFailedURL(logoURL)}
        />
      ) : (
        brandInitial(name, domain)
      )}
    </span>
  );
}
