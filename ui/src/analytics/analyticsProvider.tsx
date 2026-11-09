import { createContext, type ReactNode, useContext, useEffect, useMemo } from "react";

import type { Analytics, AnalyticsConfig } from "@/analytics/analytics";
import { noopAnalytics } from "@/analytics/noop.adapter";
import { createPostHogAnalytics } from "@/analytics/posthog.adapter";

export function createAnalytics(config: AnalyticsConfig | null | undefined): Analytics {
  return config ? createPostHogAnalytics(config) : noopAnalytics;
}

const AnalyticsContext = createContext<Analytics>(noopAnalytics);

/** Identifies the signed-in account. Without a configuration the app sends nothing. */
export function AnalyticsProvider({
  config,
  user,
  children,
}: {
  config: AnalyticsConfig | null | undefined;
  user: { id: string; email: string };
  children: ReactNode;
}) {
  const key = config?.key;
  // biome-ignore lint/correctness/useExhaustiveDependencies: the key identifies the configuration.
  const analytics = useMemo(() => createAnalytics(config), [key]);

  useEffect(() => {
    analytics.identify({ id: user.id, email: user.email });
  }, [analytics, user.id, user.email]);

  return <AnalyticsContext.Provider value={analytics}>{children}</AnalyticsContext.Provider>;
}

/** Outside a provider, as on the sign-in screens, analytics are the no-op adapter. */
export function useAnalytics() {
  return useContext(AnalyticsContext);
}
