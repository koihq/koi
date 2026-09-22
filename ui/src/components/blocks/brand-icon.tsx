import { useState } from "react";

import { brandfetchLogoURL, brandInitial } from "@/components/blocks/visibility-lists";

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
