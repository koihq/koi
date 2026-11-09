// Writes the rendered page into the built index.html, so crawlers read the copy without JavaScript.
import { readFile, rm, writeFile } from "node:fs/promises";
import { resolve } from "node:path";
import { pathToFileURL } from "node:url";

const site = import.meta.dirname;
const htmlPath = resolve(site, "dist/index.html");
const serverDir = resolve(site, "dist-ssr");

const { render } = await import(pathToFileURL(resolve(serverDir, "entry-server.js")).href);

const template = await readFile(htmlPath, "utf8");

if (!template.includes("<!--app-->")) {
  throw new Error("site/index.html is missing the <!--app--> placeholder");
}

await writeFile(htmlPath, template.replace("<!--app-->", render()));
await rm(serverDir, { recursive: true, force: true });
