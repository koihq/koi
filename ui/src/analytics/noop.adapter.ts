import type { Analytics } from "@/analytics/analytics";

/** Self-hosted instances, local development and tests send nothing. */
export const noopAnalytics: Analytics = {
  enabled: false,
  start: () => undefined,
  identify: () => undefined,
  reset: () => undefined,
  setConsent: () => undefined,
  startReplay: () => undefined,
  stopReplay: () => undefined,
  openSupport: () => undefined,
};
