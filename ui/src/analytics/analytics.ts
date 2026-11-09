import type { BrowserAnalytics } from "@/api/generated";

/** What the instance sends to its analytics service, from `/capabilities`. */
export type AnalyticsConfig = BrowserAnalytics;

export type ConsentChoice = "granted" | "denied";

/**
 * Product analytics as the app sees it. Only the adapter knows the vendor, so replacing it means
 * writing a new adapter and nothing else.
 */
export interface Analytics {
  /** False for the no-op adapter: nothing to consent to, no support chat. */
  readonly enabled: boolean;
  /** Loads the SDK, which records the page view. Visitors stay anonymous until they sign in. */
  start(): void;
  identify(user: { id: string; email: string }): void;
  reset(): void;
  setConsent(choice: ConsentChoice): void;
  /** Records the session from here on. Only with consent, and only where setup is studied. */
  startReplay(): void;
  stopReplay(): void;
  openSupport(): void;
}
