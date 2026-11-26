import type { Page } from "./pages";

export const origin = "https://getkoi.co";

/** Wide enough for every large card; the image itself is public/og.png. */
const image = { url: `${origin}/og.png`, width: 1200, height: 630 };

export function pageUrl(path: string) {
  return `${origin}${path}`;
}

/**
 * Everything a crawler reads in <head>, written into the HTML at build time. A page with
 * `noindex` gets no canonical address, so nothing points search engines at it.
 */
export async function headFor(page: Page) {
  const url = pageUrl(page.path);
  const title = page.social?.title ?? page.title;
  const description = page.social?.description ?? page.description;

  const tags = [
    `<title>${escapeHtml(page.title)}</title>`,
    meta("name", "description", page.description),
    meta("property", "og:title", title),
    meta("property", "og:description", description),
    meta("property", "og:type", page.updated ? "article" : "website"),
    meta("property", "og:site_name", "Koi"),
    meta("property", "og:image", image.url),
    meta("property", "og:image:width", String(image.width)),
    meta("property", "og:image:height", String(image.height)),
    meta("name", "twitter:card", "summary_large_image"),
    ...(page.noindex
      ? [meta("name", "robots", "noindex")]
      : [meta("property", "og:url", url), `<link rel="canonical" href="${escapeHtml(url)}" />`]),
    ...((await page.jsonLd?.()) ?? []).map(
      (schema) =>
        `<script type="application/ld+json">${JSON.stringify(schema).replaceAll("<", "\\u003c")}</script>`,
    ),
  ];

  return tags.join("\n    ");
}

export function sitemapFor(pages: Page[]) {
  const urls = pages.map((page) => {
    const lastmod = page.updated ? `\n    <lastmod>${page.updated}</lastmod>` : "";

    return `  <url>\n    <loc>${escapeHtml(pageUrl(page.path))}</loc>${lastmod}\n  </url>`;
  });

  return `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${urls.join("\n")}
</urlset>
`;
}

function meta(kind: "name" | "property", key: string, value: string) {
  return `<meta ${kind}="${key}" content="${escapeHtml(value)}" />`;
}

function escapeHtml(value: string) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll('"', "&quot;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}
