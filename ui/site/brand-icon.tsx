import { cn } from "cn";
import { useLogoInfo } from "@/lib/logo-info";
import mark from "./assets/koi-mark-64.png";
import athena from "./assets/logos/rivals/athenahq.png";
import otterly from "./assets/logos/rivals/otterly.png";
import peec from "./assets/logos/rivals/peec.png";
import rankscale from "./assets/logos/rivals/rankscale.png";
import scrunch from "./assets/logos/rivals/scrunch.png";
import profound from "./assets/logos/rivals/tryprofound.png";

const logos: Record<string, string> = {
  "getkoi.co": mark,
  "tryprofound.com": profound,
  "peec.ai": peec,
  "otterly.ai": otterly,
  "scrunch.com": scrunch,
  "athenahq.ai": athena,
  "rankscale.ai": rankscale,
};

/** Marks without their own background sit smaller on a white disc. The rest fill the coin. */
const symbols = new Set(["getkoi.co", "athenahq.ai", "rankscale.ai"]);

/** No Logo.dev on the landing: every logo it shows is saved beside the page. */
export function logoDevURL(_domain: string, _token: string | undefined): string | null {
  return null;
}

/** The brand color read from the saved logo, as the app reads it from Logo.dev. Client only. */
export function useBrandColor(domain: string | undefined): string | null {
  return useLogoInfo(domain ? (logos[domain] ?? null) : null)?.color ?? null;
}

export function brandInitial(name: string, domain: string): string {
  return (name.trim() || domain.trim() || "?").charAt(0).toLocaleUpperCase();
}

/**
 * Stands in for the app's `BrandIcon` on the landing and in its demo, through an alias in
 * `vite.site.config.ts`. Same coin, with logos saved beside the page rather than fetched from
 * Logo.dev, and its shape known ahead, so the server can render it.
 */
export function BrandIcon({
  name,
  domain = "",
  className,
}: {
  name: string;
  domain?: string;
  src?: string;
  lookupLogo?: boolean;
  className?: string;
}) {
  const logo = logos[domain];
  const symbol = symbols.has(domain);

  return (
    <span
      aria-hidden="true"
      data-slot="brand-icon"
      className={cn(
        "relative flex size-8 shrink-0 items-center justify-center overflow-hidden rounded-full text-xs font-medium text-muted-foreground",
        !logo && "bg-muted",
        logo && symbol && "bg-white",
        "ring-1 ring-black/10 ring-inset dark:ring-white/10",
        className,
      )}
    >
      {logo ? (
        <img
          src={logo}
          alt=""
          className={symbol ? "block size-[62%] object-contain" : "block size-full object-cover"}
        />
      ) : (
        brandInitial(name, domain)
      )}
    </span>
  );
}
