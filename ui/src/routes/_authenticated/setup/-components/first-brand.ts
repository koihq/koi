/** Splits the single setup field into the identifiers a brand stores. */
export function parseBrandSite(value: string): { domain: string; githubRepo: string } {
  const site = value.trim();
  if (!site) return { domain: "", githubRepo: "" };

  const withScheme = /^[a-z][a-z0-9+.-]*:\/\//i.test(site) ? site : `https://${site}`;

  let url: URL;
  try {
    url = new URL(withScheme);
  } catch {
    return { domain: site, githubRepo: "" };
  }

  const host = url.hostname.toLowerCase().replace(/^www\./, "");
  if (host === "github.com") {
    return { domain: "", githubRepo: `https://github.com${url.pathname.replace(/\/$/, "")}` };
  }

  return { domain: host, githubRepo: "" };
}

/** Suggests a name to edit; answers are matched on it, so the form never hides it. */
export function brandNameFromSite(value: string): string {
  const { domain, githubRepo } = parseBrandSite(value);
  const labels = domain.split(".");

  // "acme.com" and "app.acme.co.uk" both name Acme.
  const label = githubRepo
    ? (githubRepo.split("/").at(-1) ?? "")
    : labels.length > 2 && (labels.at(-2)?.length ?? 0) <= 3
      ? (labels.at(-3) ?? "")
      : (labels.at(-2) ?? labels[0]);

  return label.charAt(0).toUpperCase() + label.slice(1);
}

export function countryFromLocale(locale: string): string {
  try {
    return new Intl.Locale(locale).region ?? "";
  } catch {
    return "";
  }
}

const PERSONAL_EMAIL_DOMAINS = new Set([
  "aol.com",
  "duck.com",
  "fastmail.com",
  "gmail.com",
  "gmx.com",
  "gmx.de",
  "googlemail.com",
  "hey.com",
  "hotmail.com",
  "hotmail.it",
  "icloud.com",
  "libero.it",
  "live.com",
  "mac.com",
  "me.com",
  "msn.com",
  "outlook.com",
  "pm.me",
  "proton.me",
  "protonmail.com",
  "tiscali.it",
  "tutanota.com",
  "virgilio.it",
  "yahoo.com",
  "yahoo.it",
  "yandex.com",
]);

/** A work address usually names the site, which saves the only thing setup has to ask. */
export function siteFromEmail(email: string): string {
  const domain = email.split("@")[1]?.trim().toLowerCase() ?? "";

  return domain.includes(".") && !PERSONAL_EMAIL_DOMAINS.has(domain) ? domain : "";
}
