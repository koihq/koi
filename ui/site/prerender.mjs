// Writes each rendered page into a copy of the built index.html, so crawlers read the copy without
// JavaScript. The home page replaces dist/index.html; every other page goes to dist/<path>/index.html.
// The sitemap comes from the same list, so a page cannot be published without being in it.
import { mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { resolve } from "node:path";
import { pathToFileURL } from "node:url";

const site = import.meta.dirname;
const dist = resolve(site, "dist");
const serverDir = resolve(site, "dist-ssr");

const { notFound, pages, render, sitemap } = await import(
  pathToFileURL(resolve(serverDir, "entry-server.js")).href
);

const template = await readFile(resolve(dist, "index.html"), "utf8");

for (const placeholder of ["<!--head-->", "<!--app-->"]) {
  if (!template.includes(placeholder)) {
    throw new Error(`site/index.html is missing the ${placeholder} placeholder`);
  }
}

for (const page of pages) {
  const dir = resolve(dist, `.${page.path}`);

  await mkdir(dir, { recursive: true });
  await writeFile(resolve(dir, "index.html"), await htmlFor(page));
}

await writeFile(resolve(dist, "404.html"), await htmlFor(notFound));
await writeFile(resolve(dist, "sitemap.xml"), sitemap);

await rm(serverDir, { recursive: true, force: true });

async function htmlFor(page) {
  if (!page.title || !page.description) {
    throw new Error(`${page.path} has no title or description`);
  }

  const { head, body } = await render(page);

  // Functions, because a replacement string would read "$" in the copy as a pattern.
  return template.replace("<!--head-->", () => head).replace("<!--app-->", () => body);
}
