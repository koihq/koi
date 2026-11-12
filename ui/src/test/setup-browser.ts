import "@/config/zod";
import "@/config/i18n";
// Components render with the app's CSS; routes also get it from the root route.
import "@/styles.css";

import { afterAll, afterEach, beforeAll, expect, vi } from "vitest";
import { cleanup } from "vitest-browser-react";

import { clearRenderedQueries } from "@/test/render-route";
import { worker } from "@/test/worker";

// Vitest identifies the test frame by its URL, and routes under test replace it.
const frameUrl = window.location.href;

// MSW 3 only logs a request without a handler and lets it through, so record it and fail.
const unhandled: string[] = [];

// Requests a test already sent still belong to its handlers, so cleanup waits for them.
let inFlight = 0;
worker.events.on("request:start", () => {
  inFlight += 1;
});
worker.events.on("request:end", () => {
  inFlight -= 1;
});

async function settleNetwork() {
  await new Promise((resolve) => setTimeout(resolve, 20));
  await vi.waitFor(() => expect(inFlight).toBe(0), { timeout: 2_000 });
}

beforeAll(async () => {
  await worker.start({
    quiet: true,
    onUnhandledFrame({ frame, defaults }) {
      const { request } = frame.data as { request?: Request };
      const url = request ? new URL(request.url) : null;

      // A request the test already cancelled, for example on cleanup, is not a missing handler.
      if (url?.pathname.startsWith("/api/") && !request?.signal.aborted) {
        unhandled.push(`${request?.method} ${url.pathname}${url.search}`);
        defaults.error();
      }
    },
  });
});

afterEach(async () => {
  await cleanup();
  clearRenderedQueries();
  await settleNetwork();
  window.history.replaceState(null, "", frameUrl);
  window.scrollTo(0, 0);
  worker.resetHandlers();
  vi.useRealTimers();

  const requests = unhandled.splice(0);
  expect(requests, "API requests without an MSW handler").toEqual([]);
});

afterAll(() => {
  worker.stop();
});
