import type { ConsentChoice } from "@/analytics/analytics";

const cookieName = "koi_consent";
const maxAgeSeconds = 180 * 24 * 60 * 60;

/** The landing and the app share one choice through a cookie on the parent domain. */
export function readConsent(cookie = document.cookie): ConsentChoice | null {
  for (const part of cookie.split(";")) {
    const [name, value] = part.trim().split("=");

    if (name === cookieName && (value === "granted" || value === "denied")) {
      return value;
    }
  }

  return null;
}

export function writeConsent(choice: ConsentChoice, hostname = window.location.hostname) {
  const domain = consentDomain(hostname);
  const secure = window.location.protocol === "https:" ? "; Secure" : "";

  // biome-ignore lint/suspicious/noDocumentCookie: the Cookie Store API is missing in Safari.
  document.cookie = `${cookieName}=${choice}; Max-Age=${maxAgeSeconds}; Path=/; SameSite=Lax${domain}${secure}`;
}

/** Shows the banner again so a visitor can change an earlier choice, as the law requires. */
export const consentReopenEvent = "koi:consent-reopen";

export function reopenConsent() {
  window.dispatchEvent(new Event(consentReopenEvent));
}

/** `app.getkoi.co` and `getkoi.co` share `.getkoi.co`; localhost keeps a host-only cookie. */
export function consentDomain(hostname: string) {
  const labels = hostname.split(".");

  if (labels.length < 2 || /^[\d.]+$/.test(hostname)) {
    return "";
  }

  return `; Domain=.${labels.slice(-2).join(".")}`;
}
