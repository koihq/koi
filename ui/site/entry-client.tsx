import { createRoot, hydrateRoot } from "react-dom/client";

import { createAnalytics } from "@/analytics/analyticsProvider";
import { ConsentBanner } from "@/components/blocks/consent-banner";
import { copy } from "./copy";
import { pageFor } from "./pages";
import "./site.css";

const root = document.getElementById("root");
const page = pageFor(window.location.pathname);

void mount();

startAnalytics(import.meta.env.VITE_POSTHOG_KEY);

async function mount() {
  const element = await page.load();

  if (root?.firstElementChild) {
    hydrateRoot(root, element);
  } else if (root) {
    // The dev server serves the template without the prerendered page.
    createRoot(root).render(element);
    document.title = page.title;
  }
}

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
