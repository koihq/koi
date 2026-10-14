import { cn } from "cn";
import { useState } from "react";

export function logoDevURL(domain: string, token: string | undefined): string | null {
  const key = token?.trim();
  if (!key || !domain) return null;

  return `https://img.logo.dev/${encodeURIComponent(domain)}?token=${encodeURIComponent(key)}&size=64&format=webp&fallback=404`;
}

export function brandInitial(name: string, domain: string): string {
  return (name.trim() || domain.trim() || "?").charAt(0).toLocaleUpperCase();
}

export function BrandIcon({
  name,
  domain = "",
  src,
  className,
}: {
  name: string;
  domain?: string;
  src?: string;
  className?: string;
}) {
  const logoURL = src ?? logoDevURL(domain, import.meta.env.VITE_LOGODEV_PUBLISHABLE_KEY);
  const [failedURL, setFailedURL] = useState<string>();
  const showLogo = logoURL !== null && failedURL !== logoURL;

  return (
    <span
      aria-hidden="true"
      className={cn(
        "relative flex size-8 shrink-0 items-center justify-center overflow-hidden rounded-full text-xs font-medium text-muted-foreground",
        !showLogo && "bg-muted",
        className,
      )}
    >
      {showLogo ? (
        <img
          src={logoURL}
          alt=""
          className="block size-full rounded-full object-contain"
          onError={() => setFailedURL(logoURL)}
        />
      ) : (
        brandInitial(name, domain)
      )}
    </span>
  );
}
