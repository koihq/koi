import { describe, expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";
import { box, css } from "@/test/dom";
import { brand, me, prompt, run, tracking } from "@/test/fixtures";
import { renderRoute } from "@/test/render-route";
import { serveWorkspace, type Workspace, workspace } from "@/test/worker";

/** 121 brands with the last one active. */
function manyBrands(): Workspace {
  const brands = Array.from({ length: 121 }, (_, index) =>
    brand({
      id: `brand-${index}`,
      name: `Brand ${String(index).padStart(3, "0")}`,
      domain: `brand${index}.example`,
    }),
  );

  return serveWorkspace(
    workspace({
      me: me({ activeBrand: brands[120] }),
      brands,
      tracking: tracking({ startedAt: null }),
      prompts: [prompt({ id: "prompt", brandId: "brand-120" })],
    }),
  );
}

const switcher = (name: string) => page.getByLabelText(`Active brand: ${name}`, { exact: true });
const option = (name: RegExp) => page.getByRole("option", { name });
const search = () => page.getByPlaceholder("Search brands");
const brandLists = (state: Workspace) =>
  state.requests.filter((url) => url.pathname === "/api/v1/brands");

describe("brand switcher", () => {
  test("fetches only requested pages and searches before switching", async () => {
    const state = manyBrands();

    await renderRoute("/");
    await switcher("Brand 120").click();

    await expect.element(option(/Brand 000/)).toBeVisible();
    expect(brandLists(state)).toHaveLength(1);
    expect(brandLists(state)[0].searchParams.get("pageSize")).toBe("20");

    const popup = await css('[data-slot="combobox-content"]');
    await popup.getByRole("button", { name: "Next", exact: true }).click();

    await expect.element(option(/Brand 020/)).toBeVisible();
    expect(brandLists(state)).toHaveLength(2);
    expect(brandLists(state)[1].searchParams.get("page")).toBe("2");

    await search().fill("Brand 105");

    await expect.element(option(/Brand 105/)).toBeVisible();
    expect(brandLists(state)).toHaveLength(3);
    expect(brandLists(state)[2].searchParams.get("q")).toBe("Brand 105");
    expect(brandLists(state)[2].searchParams.get("page")).toBe("1");

    await userEvent.keyboard("{ArrowDown}{Enter}");

    await expect.element(switcher("Brand 105")).toBeVisible();
    expect(state.requests.filter((url) => url.pathname.endsWith("/activate"))).toHaveLength(1);
    expect(brandLists(state)).toHaveLength(3);
  });

  test("a failed activation keeps the current brand and can be retried", async () => {
    const state = manyBrands();
    state.fail.add("activate");

    await renderRoute("/");
    await switcher("Brand 120").click();
    await option(/Brand 000/).click();

    await expect
      .element(page.getByRole("alert").filter({ hasText: "Unable to switch brand." }))
      .toHaveTextContent("Unable to switch brand. Open the selector to try again.");
    await expect.element(switcher("Brand 120")).toBeVisible();

    state.fail.delete("activate");
    await switcher("Brand 120").click();
    await option(/Brand 000/).click();

    await expect.element(switcher("Brand 000")).toBeVisible();
  });

  test("search handles no results and a recoverable API failure", async () => {
    const state = manyBrands();

    await renderRoute("/");
    await switcher("Brand 120").click();
    await search().fill("Missing brand");

    await expect.element(page.getByText("No matching brands.")).toBeVisible();

    state.fail.add("brands");
    await search().fill("Brand 100");

    await expect.element(page.getByText("Brands could not be loaded.")).toBeVisible();

    state.fail.delete("brands");
    const popup = await css('[data-slot="combobox-content"]');
    await popup.getByRole("button", { name: "Retry" }).click();

    await expect.element(option(/Brand 100/)).toBeVisible();
  });

  test("remains usable at 320px", async () => {
    await page.viewport(320, 740);
    manyBrands();

    await renderRoute("/");
    await switcher("Brand 120").click();

    const popup = await css('[data-slot="combobox-content"]');
    await expect.element(popup.getByRole("option", { name: /Brand 000/ })).toBeVisible();
    await expect.element(popup.getByRole("button", { name: "Next", exact: true })).toBeVisible();

    const bounds = box(popup);
    expect(bounds.x).toBeGreaterThanOrEqual(0);
    expect(bounds.x + bounds.width).toBeLessThanOrEqual(320);

    await popup.getByRole("option", { name: /Brand 000/ }).click();

    await expect.element(switcher("Brand 000")).toBeVisible();
  });

  test("switching brand stops polling the previous workspace", async () => {
    const state = manyBrands();
    state.tracking = tracking();
    state.runs = [run({ brandId: "brand-120", state: "running" })];

    const previous = () =>
      state.requests.filter((url) => url.searchParams.get("brandId") === "brand-120").length;
    const runPolls = () =>
      state.requests.filter(
        (url) => url.pathname === "/api/v1/runs" && url.searchParams.get("brandId") === "brand-120",
      ).length;

    await renderRoute("/");

    // A running collection is polled while its brand is active.
    await expect.poll(runPolls, { timeout: 10_000 }).toBeGreaterThan(1);

    await switcher("Brand 120").click();
    await option(/Brand 000/).click();
    await expect.element(switcher("Brand 000")).toBeVisible();

    const count = previous();

    await new Promise((resolve) => setTimeout(resolve, 6_000));

    expect(previous()).toBe(count);
  });
});
