/**
 * Answers the app's API calls from responses recorded on a real Koi instance, so the landing shows
 * the product itself. Reads replay. Anything that would change data is refused, and the page around
 * the demo is told, so it can explain.
 */

type Fixture = {
  recordedAt: string;
  routes: Record<string, { status: number; body: string }>;
  bodies: Record<string, string>;
};

export const BLOCKED_MESSAGE = "koi-demo:blocked";

async function load(url: string): Promise<Fixture> {
  const response = await fetch(url);
  const bytes = new Uint8Array(await response.arrayBuffer());
  const gzipped = bytes[0] === 0x1f && bytes[1] === 0x8b;
  const text = gzipped
    ? await new Response(
        new Blob([bytes]).stream().pipeThrough(new DecompressionStream("gzip")),
      ).text()
    : new TextDecoder().decode(bytes);

  return JSON.parse(text);
}

/** The recorded call with the same path whose query shares the most parameters with this one. */
function closest(fixture: Fixture, url: URL) {
  const exact = fixture.routes[url.pathname + url.search];

  if (exact) {
    return exact;
  }

  const wanted = new Set([...url.searchParams.entries()].map(([key, value]) => `${key}=${value}`));
  let best: { status: number; body: string } | undefined;
  let bestScore = -1;

  for (const [key, route] of Object.entries(fixture.routes)) {
    const recorded = new URL(key, url.origin);

    if (recorded.pathname !== url.pathname) {
      continue;
    }

    const score = [...recorded.searchParams.entries()].filter(([name, value]) =>
      wanted.has(`${name}=${value}`),
    ).length;

    if (score > bestScore) {
      best = route;
      bestScore = score;
    }
  }

  return best;
}

function json(body: string, status: number) {
  return new Response(body, { status, headers: { "Content-Type": "application/json" } });
}

export async function installReplay(url: string) {
  const fixture = await load(url);
  const original = window.fetch.bind(window);

  window.fetch = async (input, init) => {
    const request = new Request(input, init);
    const target = new URL(request.url, window.location.href);

    if (!target.pathname.startsWith("/api/") && target.pathname !== "/health") {
      return original(input, init);
    }

    if (request.method !== "GET") {
      window.parent.postMessage({ type: BLOCKED_MESSAGE }, "*");

      return json(
        JSON.stringify({ error: { code: "demo", message: "This is a demo with example data." } }),
        403,
      );
    }

    const route = closest(fixture, target);

    if (!route) {
      return json(
        JSON.stringify({ error: { code: "not_found", message: "Not in the demo." } }),
        404,
      );
    }

    return json(fixture.bodies[route.body], route.status);
  };
}
