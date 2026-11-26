import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import catalog from "../../pkg/pricing/catalog.json";
import { koiTopics } from "./alternatives/comparison";
import { alternatives, comparePages, comparisonsPage, openSourcePage } from "./alternatives/list";
import { koiPrice, type Tool, tools } from "./alternatives/tools";
import { copy } from "./copy";
import { headFor, pageUrl, sitemapFor } from "./head";
import { business } from "./legal/business";
import { notFound, type Page, pageFor, pages } from "./pages";

const markupOf = async (page: Page) => renderToStaticMarkup(await page.load());
/** Copy as React writes it, apostrophes and all. */
const expectShown = (markup: string, text: string) =>
  expect(markup).toContain(renderToStaticMarkup(text));
const schemasIn = (head: string) =>
  [...head.matchAll(/<script type="application\/ld\+json">(.*?)<\/script>/g)].map((match) =>
    JSON.parse(match[1]),
  );

describe("pageFor", () => {
  it("serves a page with or without its trailing slash", () => {
    expect(pageFor("/privacy").path).toBe("/privacy");
    expect(pageFor("/privacy/").path).toBe("/privacy");
    expect(pageFor("/terms/").path).toBe("/terms");
  });

  it("serves the home page, and the not found page for any other address", () => {
    expect(pageFor("/").path).toBe("/");
    expect(pageFor("/missing")).toBe(notFound);
  });
});

describe("every page", () => {
  it("has its own address, title and description", () => {
    for (const key of ["path", "title", "description"] as const) {
      expect(new Set(pages.map((page) => page[key])).size).toBe(pages.length);
    }
  });

  it.each([
    ...pages,
    notFound,
  ])("$path fits a search result, has one heading and links to the comparisons", async (page) => {
    expect(page.title.length).toBeGreaterThan(0);
    expect(page.title.length).toBeLessThanOrEqual(60);
    expect(page.description.length).toBeGreaterThan(0);
    expect(page.description.length).toBeLessThanOrEqual(160);

    const markup = await markupOf(page);

    expect(markup.match(/<h1[\s>]/g)).toHaveLength(1);
    for (const { path } of comparePages) expect(markup).toContain(`href="${path}"`);
  });
});

describe("headFor", () => {
  it.each(pages)("$path points search engines and shared links at itself", async (page) => {
    const head = await headFor(page);

    expect(head).toContain(`<link rel="canonical" href="${pageUrl(page.path)}" />`);
    expect(head).toContain(`property="og:url" content="${pageUrl(page.path)}"`);
    expect(head).toContain('property="og:image" content="https://getkoi.co/og.png"');
    expect(head).not.toContain("noindex");
  });

  it("describes the product on the home page only", async () => {
    const types = async (page: Page) =>
      schemasIn(await headFor(page)).map((schema) => schema["@type"]);

    expect(await types(pageFor("/"))).toEqual(["SoftwareApplication", "FAQPage"]);
    expect(await types(pageFor("/privacy"))).toEqual([]);
    expect(await types(pageFor("/alternatives"))).toEqual(["BreadcrumbList"]);
    for (const { path } of [...alternatives, openSourcePage]) {
      expect(await types(pageFor(path))).toEqual(["Article", "BreadcrumbList", "FAQPage"]);
    }
  });

  it("keeps the not found page out of search", async () => {
    const head = await headFor(notFound);

    expect(head).toContain('name="robots" content="noindex"');
    expect(head).not.toContain("canonical");
  });

  it("escapes the copy", async () => {
    const head = await headFor({
      ...notFound,
      title: 'A "<b>" & B',
      jsonLd: async () => [{ name: "</script><script>" }],
    });

    expect(head).toContain("<title>A &quot;&lt;b&gt;&quot; &amp; B</title>");
    expect(schemasIn(head)).toEqual([{ name: "</script><script>" }]);
    expect(head).not.toContain("</script><script>");
  });
});

describe("sitemapFor", () => {
  it("lists every page, with the date its content changed", () => {
    const sitemap = sitemapFor([...pages, { ...notFound, path: "/dated", updated: "2026-11-25" }]);
    const listed = [...sitemap.matchAll(/<loc>(.*?)<\/loc>/g)].map((match) => match[1]);

    expect(listed).toEqual([...pages.map((page) => pageUrl(page.path)), pageUrl("/dated")]);
    expect(sitemap).toContain(`${pageUrl("/dated")}</loc>\n    <lastmod>2026-11-25</lastmod>`);
  });
});

describe("alternatives pages", () => {
  it.each(
    alternatives,
  )("$path compares Koi with one rival, says where it is ahead and cites both", async ({
    path,
    rival,
    content,
    updated,
  }) => {
    const page = pageFor(path);
    const markup = await markupOf(page);
    const { rows, topics, questions } = await content();
    const shows = (text: string) => expectShown(markup, text);

    expect(page.updated).toBe(updated);
    shows(copy.alternatives.disclosure);

    // A table where Koi always wins is not a comparison.
    const ahead = rows.filter(({ koi, rival }) => koi.mark === "no" && rival.mark === "yes");
    expect(ahead.length).toBeGreaterThanOrEqual(3);

    for (const { feature } of rows) shows(feature);
    for (const topic of Object.keys(topics) as (keyof typeof topics)[]) {
      shows(koiTopics[topic]);
      shows(topics[topic]);
    }
    for (const { q } of questions) shows(q);

    for (const tool of [tools.koi, tools[rival]]) {
      for (const source of tool.sources) expect(markup).toContain(`href="${source.url}"`);
      // Facts cannot be newer than the page that shows them.
      expect(tool.checked <= updated).toBe(true);
    }
  });

  it("the open source page shows every project, its code and who it suits", async () => {
    const markup = await markupOf(pageFor(openSourcePage.path));
    const { projects, questions } = await openSourcePage.content();

    expect(projects[0].tool).toBe("koi");
    expectShown(markup, copy.alternatives.openSource.disclosure);

    for (const project of projects) {
      const tool: Tool = tools[project.tool];
      const repository = tool.sources.find((source) =>
        source.url.startsWith("https://github.com/"),
      );

      expectShown(markup, project.summary);
      expectShown(markup, project.pick);
      expect(markup).toContain(`href="${repository?.url}"`);
      expect(tool.checked <= openSourcePage.updated).toBe(true);
    }
    for (const { q } of questions) expectShown(markup, q);
  });

  it("the index sums up every comparison", async () => {
    const markup = await markupOf(pageFor(comparisonsPage.path));

    for (const page of [...alternatives, openSourcePage]) expectShown(markup, page.summary);
  });

  it.each(Object.entries<Tool>(tools))("%s has a date and a source", (_, tool) => {
    expect(tool.checked).toMatch(/^\d{4}-\d{2}-\d{2}$/);
    expect(tool.sources.length).toBeGreaterThan(0);
  });

  it("prices Koi from the catalog", () => {
    for (const option of catalog.options) {
      expect(koiPrice(option.prompts)).toBe(`€${option.euros}`);
    }
    expect(() => koiPrice(16)).toThrow();
  });
});

describe("legal pages", () => {
  it.each(["/privacy", "/terms"])("%s names who runs Koi Cloud and links back", async (path) => {
    const markup = await markupOf(pageFor(path));

    expect(markup).toContain(business.vat);
    expect(markup).toContain(`mailto:${business.email}`);
    expect(markup).toContain('href="/#pricing"');
  });
});
