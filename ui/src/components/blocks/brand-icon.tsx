import { useState } from "react";

export function logoDevURL(domain: string, token: string | undefined): string | null {
  const key = token?.trim();
  if (!key || !domain) return null;

  return `https://img.logo.dev/${encodeURIComponent(domain)}?token=${encodeURIComponent(key)}&size=64&format=webp&fallback=404`;
}

export function brandInitial(name: string, domain: string): string {
  return (name.trim() || domain.trim() || "?").charAt(0).toLocaleUpperCase();
}

export function BrandIcon({ name, domain }: { name: string; domain: string }) {
  const logoURL = logoDevURL(domain, import.meta.env.VITE_LOGODEV_PUBLISHABLE_KEY);
  const [failedURL, setFailedURL] = useState<string>();
  const showLogo = logoURL !== null && failedURL !== logoURL;

  return (
    <span
      aria-hidden="true"
      className="relative flex size-8 shrink-0 items-center justify-center overflow-hidden rounded-md bg-muted text-xs font-medium text-muted-foreground"
    >
      {brandInitial(name, domain)}
      {showLogo ? (
        <img
          src={logoURL}
          alt=""
          className="absolute inset-0 size-full bg-background object-contain"
          onError={() => setFailedURL(logoURL)}
        />
      ) : null}
    </span>
  );
}
