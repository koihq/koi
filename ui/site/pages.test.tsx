import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { business } from "./legal/business";
import { pageFor, pages } from "./pages";

describe("pageFor", () => {
  it("serves a page with or without its trailing slash", () => {
    expect(pageFor("/privacy").path).toBe("/privacy");
    expect(pageFor("/privacy/").path).toBe("/privacy");
    expect(pageFor("/terms/").path).toBe("/terms");
  });

  it("falls back to the home page", () => {
    expect(pageFor("/").path).toBe("/");
    expect(pageFor("/missing").path).toBe("/");
  });
});

describe("legal pages", () => {
  it.each(["/privacy", "/terms"])("%s names who runs Koi Cloud and links back", (path) => {
    const page = pages.find((candidate) => candidate.path === path);
    const markup = renderToStaticMarkup(page?.render());

    expect(page?.title).not.toBe("");
    expect(markup).toContain(business.vat);
    expect(markup).toContain(`mailto:${business.email}`);
    expect(markup).toContain('href="/#pricing"');
  });
});
