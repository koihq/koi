import { useState } from "react";

export function brandfetchLogoURL(domain: string, clientID: string | undefined): string | null {
  const id = clientID?.trim();
  if (!id || !domain) return null;
  return `https://cdn.brandfetch.io/domain/${encodeURIComponent(domain)}/w/64/h/64/fallback/404/type/icon?c=${encodeURIComponent(id)}`;
}

export function brandInitial(name: string, domain: string): string {
  return (name.trim() || domain.trim() || "?").charAt(0).toLocaleUpperCase();
}

export function BrandIcon({ name, domain }: { name: string; domain: string }) {
  const logoURL = brandfetchLogoURL(domain, import.meta.env.VITE_BRANDFETCH_CLIENT_ID);
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
