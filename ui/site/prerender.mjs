// Writes each rendered page into a copy of the built index.html, so crawlers read the copy without
// JavaScript. The home page replaces dist/index.html; every other page goes to dist/<path>/index.html.
import { mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { resolve } from "node:path";
import { pathToFileURL } from "node:url";

const site = import.meta.dirname;
const dist = resolve(site, "dist");
const serverDir = resolve(site, "dist-ssr");

const { pages, render } = await import(pathToFileURL(resolve(serverDir, "entry-server.js")).href);

const template = await readFile(resolve(dist, "index.html"), "utf8");

if (!template.includes("<!--app-->")) {
  throw new Error("site/index.html is missing the <!--app--> placeholder");
}

for (const page of pages) {
  const html = withMeta(template, page).replace("<!--app-->", render(page.path));
  const dir = resolve(dist, `.${page.path}`);

  await mkdir(dir, { recursive: true });
  await writeFile(resolve(dir, "index.html"), html);
}

await rm(serverDir, { recursive: true, force: true });

/** Gives a page other than home its own title, description and canonical address. */
function withMeta(html, page) {
  if (page.path === "/") return html;

  const url = `https://getkoi.co${page.path}`;
  const title = escapeAttribute(page.title);
  const description = escapeAttribute(page.description);

  return html
    .replace(/<title>[^<]*<\/title>/, `<title>${title}</title>`)
    .replace(/(<meta\s+name="description"\s+content=")[^"]*(")/, `$1${description}$2`)
    .replace(/(<meta\s+property="og:title"\s+content=")[^"]*(")/, `$1${title}$2`)
    .replace(/(<meta\s+property="og:description"\s+content=")[^"]*(")/, `$1${description}$2`)
    .replace(/(<meta\s+property="og:url"\s+content=")[^"]*(")/, `$1${url}$2`)
    .replace(/(<link\s+rel="canonical"\s+href=")[^"]*(")/, `$1${url}$2`);
}

function escapeAttribute(value) {
  return value.replaceAll("&", "&amp;").replaceAll('"', "&quot;").replaceAll("<", "&lt;");
}
