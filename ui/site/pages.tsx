import type { ReactNode } from "react";

import { alternatives, comparisonsPage, openSourcePage } from "./alternatives/list";
import { copy } from "./copy";
import { pageUrl } from "./head";
import { SiteLayout } from "./layout";
import { articleSchema, breadcrumbSchema, faqSchema, softwareSchema } from "./schema";

export type Page = {
  /** The path the page is served at, and the folder its prerendered HTML is written to. */
  path: string;
  title: string;
  description: string;
  /** What a shared link shows, when it should differ from the search result. */
  social?: { title: string; description: string };
  /** The ISO date the content last changed, for the sitemap and the article data. */
  updated?: string;
  /** Structured data. Only the build reads it. */
  jsonLd?: () => Promise<object[]>;
  noindex?: boolean;
  /** Loaded when the page is opened, so a page of text never downloads the home page's demos. */
  load: () => Promise<ReactNode>;
};

/** Koi, then the index of comparisons, then the page itself. */
function trailTo(page: { path: string; name: string }) {
  const index = { name: comparisonsPage.name, url: pageUrl(comparisonsPage.path) };
  const here = { name: page.name, url: pageUrl(page.path) };

  return breadcrumbSchema([{ name: "Koi", url: pageUrl("/") }, index, here]);
}

function alternativesPage({ name, rival, content, ...page }: (typeof alternatives)[number]): Page {
  return {
    ...page,
    jsonLd: async () => [
      articleSchema({ ...page, url: pageUrl(page.path) }),
      trailTo({ ...page, name }),
      faqSchema((await content()).questions),
    ],
    load: async () => {
      const { ComparisonPage } = await import("./alternatives/alternatives-page");

      return (
        <SiteLayout>
          <ComparisonPage rival={rival} content={await content()} />
        </SiteLayout>
      );
    },
  };
}

export const pages: Page[] = [
  {
    path: "/",
    ...copy.meta.home,
    social: copy.meta.homeSocial,
    jsonLd: async () => [softwareSchema, faqSchema(copy.faq.items)],
    load: async () => {
      const { App } = await import("./app");

      return <App />;
    },
  },
  {
    ...comparisonsPage,
    jsonLd: async () => [
      breadcrumbSchema([
        { name: "Koi", url: pageUrl("/") },
        { name: comparisonsPage.name, url: pageUrl(comparisonsPage.path) },
      ]),
    ],
    load: async () => {
      const { ComparisonsPage } = await import("./alternatives/comparisons-page");

      return (
        <SiteLayout>
          <ComparisonsPage />
        </SiteLayout>
      );
    },
  },
  ...alternatives.map(alternativesPage),
  {
    ...openSourcePage,
    jsonLd: async () => [
      articleSchema({ ...openSourcePage, url: pageUrl(openSourcePage.path) }),
      trailTo(openSourcePage),
      faqSchema((await openSourcePage.content()).questions),
    ],
    load: async () => {
      const { OpenSourcePage } = await import("./alternatives/open-source-page");

      return (
        <SiteLayout>
          <OpenSourcePage content={await openSourcePage.content()} />
        </SiteLayout>
      );
    },
  },
  {
    path: "/privacy",
    ...copy.meta.privacy,
    load: async () => {
      const { Privacy } = await import("./legal/privacy");

      return (
        <SiteLayout>
          <Privacy />
        </SiteLayout>
      );
    },
  },
  {
    path: "/terms",
    ...copy.meta.terms,
    load: async () => {
      const { Terms } = await import("./legal/terms");

      return (
        <SiteLayout>
          <Terms />
        </SiteLayout>
      );
    },
  },
];

/** Served for any other address. It is not in `pages`, so it stays out of the sitemap. */
export const notFound: Page = {
  path: "/404",
  ...copy.meta.notFound,
  noindex: true,
  load: async () => {
    const { NotFound } = await import("./not-found");

    return (
      <SiteLayout>
        <NotFound />
      </SiteLayout>
    );
  },
};

/** `/privacy/` and `/privacy` are the same page. */
export function pageFor(pathname: string) {
  const path = pathname.replace(/\/+$/, "") || "/";

  return pages.find((page) => page.path === path) ?? notFound;
}
