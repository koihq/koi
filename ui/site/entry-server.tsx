import { renderToString } from "react-dom/server";

import { headFor, sitemapFor } from "./head";
import { notFound, type Page, pages } from "./pages";

export { notFound, pages };

export const sitemap = sitemapFor(pages);

export async function render(page: Page) {
  return { head: await headFor(page), body: renderToString(await page.load()) };
}
