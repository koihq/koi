import { reopenConsent } from "@/analytics/consent";
import { copy } from "../copy";

/** Analytics run only on the production landing, so only there is a choice to change. */
export const analyticsEnabled = Boolean(import.meta.env.VITE_POSTHOG_KEY);

export function CookieSettingsButton({ className }: { className?: string }) {
  if (!analyticsEnabled) return null;

  return (
    <button type="button" className={className} onClick={reopenConsent}>
      {copy.legal.cookieSettings}
    </button>
  );
}
