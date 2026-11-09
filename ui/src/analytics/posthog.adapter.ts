import type { PostHog } from "posthog-js";

import type { Analytics, AnalyticsConfig } from "@/analytics/analytics";
import { readConsent, writeConsent } from "@/analytics/consent";

/**
 * The only module that knows PostHog. The SDK loads on first use, so an instance without analytics
 * never downloads it, and calls made before it loads wait for it.
 */
export function createPostHogAnalytics(config: AnalyticsConfig): Analytics {
  const excluded = new Set(config.excludedEmails.map((email) => email.toLowerCase()));
  let client: Promise<PostHog> | null = null;

  const load = () => {
    client ??= import("posthog-js").then(({ default: posthog }) => {
      if (!posthog.__loaded) {
        posthog.init(config.key, {
          api_host: config.apiHost,
          ui_host: config.uiHost || null,
          defaults: "2026-08-30",
          person_profiles: "identified_only",
          cross_subdomain_cookie: true,
          cookieless_mode: "on_reject",
          opt_out_capturing_by_default: readConsent() !== "granted",
          capture_pageview: "history_change",
          disable_session_recording: true,
          disable_surveys: true,
          session_recording: { maskAllInputs: true, maskTextSelector: "[data-ph-mask]" },
        });
      }

      return posthog;
    });

    return client;
  };

  const run = (action: (posthog: PostHog) => void) => {
    void load()
      .then(action)
      .catch(() => undefined);
  };

  return {
    enabled: true,
    start: () => run(() => undefined),
    identify: (user) => {
      run((posthog) => {
        // The team's own accounts send nothing, here and on the landing that shares the cookie.
        if (excluded.has(user.email.toLowerCase())) {
          posthog.opt_out_capturing();

          return;
        }

        posthog.identify(user.id, { email: user.email });
      });
    },
    reset: () => run((posthog) => posthog.reset()),
    setConsent: (choice) => {
      writeConsent(choice);
      run((posthog) =>
        choice === "granted" ? posthog.opt_in_capturing() : posthog.opt_out_capturing(),
      );
    },
    startReplay: () => {
      if (readConsent() !== "granted") return;

      run((posthog) => posthog.startSessionRecording());
    },
    stopReplay: () => run((posthog) => posthog.stopSessionRecording()),
    openSupport: () => run((posthog) => posthog.conversations?.open()),
  };
}
