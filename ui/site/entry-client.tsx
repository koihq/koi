import { createRoot, hydrateRoot } from "react-dom/client";

import { createAnalytics } from "@/analytics/analyticsProvider";
import { ConsentBanner } from "@/components/blocks/consent-banner";
import { App } from "./app";
import { copy } from "./copy";
import "./site.css";

const root = document.getElementById("root");

if (root?.firstElementChild) {
  hydrateRoot(root, <App />);
} else if (root) {
  // The dev server serves the template without the prerendered page.
  createRoot(root).render(<App />);
}

startAnalytics(import.meta.env.VITE_POSTHOG_KEY);

/** Only the production build has a key; the prerendered page never runs this. */
function startAnalytics(key: string | undefined) {
  if (!key) return;

  const analytics = createAnalytics({
    key,
    apiHost: "/ingest",
    uiHost: "https://eu.posthog.com",
    excludedEmails: [],
  });
  analytics.start();

  const banner = document.createElement("div");
  document.body.append(banner);

  createRoot(banner).render(
    <ConsentBanner
      message={copy.consent.message}
      accept={copy.consent.accept}
      reject={copy.consent.reject}
      onChoose={analytics.setConsent}
    />,
  );
}
