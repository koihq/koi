import { http } from "msw";
import { describe, expect, test } from "vitest";
import { page } from "vitest/browser";
import { box, gate, layoutSettled, opacity } from "@/test/dom";
import { competitor, tracking } from "@/test/fixtures";
import { renderRoute } from "@/test/render-route";
import { serveWorkspace, type Workspace, worker, workspace } from "@/test/worker";

/** 25 competitors, the first one unconfirmed and with a name long enough to wrap. */
function rivals(): Workspace {
  return serveWorkspace(
    workspace({
      tracking: tracking(),
      competitors: Array.from({ length: 25 }, (_, index) =>
        competitor({
          id: `rival-${index}`,
          name:
            index === 0
              ? "An unusually long competitor name for checking wrapping"
              : `Rival ${String(index).padStart(2, "0")}`,
          domain: `rival${index}.example`,
          confirmed: index !== 0,
        }),
      ),
    }),
  );
}

const panel = () => page.getByRole("region", { name: "Competitors" });

describe("competitors", () => {
  test("status filters live in the URL and reset the page", async () => {
    await page.viewport(1440, 1000);
    const state = rivals();

    const { router } = await renderRoute("/competitors?page=2");
    await expect.element(panel().getByRole("table")).toBeVisible();

    await page
      .getByRole("group", { name: "Filter by status" })
      .getByRole("button", { name: /Confirmed/ })
      .click();

    await expect.poll(() => router.state.location.search).toMatchObject({ status: "confirmed" });
    expect(router.state.location.search).not.toHaveProperty("page", 2);
    await expect
      .poll(() =>
        state.requests
          .filter((url) => url.pathname === "/api/v1/competitors")
          .at(-1)
          ?.searchParams.get("confirmed"),
      )
      .toBe("true");
  });

  test("search keeps the current table while its request is pending", async () => {
    await page.viewport(1440, 1000);
    rivals();

    await renderRoute("/competitors");

    const table = panel().getByRole("table");
    await expect.element(table).toBeVisible();
    await expect.element(panel()).toHaveAttribute("aria-busy", "false");

    await layoutSettled(panel());
    const height = box(panel()).height;
    const hold = gate();

    worker.use(http.get("/api/v1/competitors", () => hold.opened));

    try {
      await panel().getByRole("button", { name: "Search competitors" }).click();
      const field = panel().getByRole("searchbox", { name: "Search competitors" });
      await field.fill("Rival");

      await expect.element(panel()).toHaveAttribute("aria-busy", "true");
      await expect.element(table).toBeVisible();
      await expect.poll(() => opacity(table.element())).toBe(0.5);
      expect(opacity(field.element())).toBe(1);
      expect(box(panel()).height).toBe(height);
    } finally {
      hold.release();
    }

    await expect.element(panel()).toHaveAttribute("aria-busy", "false");
    await expect.poll(() => opacity(table.element())).toBe(1);
  });

  test.each([1440, 960, 390, 320])("rows stay compact and the page fits at %ipx", async (width) => {
    await page.viewport(width, 1000);
    rivals();

    await renderRoute("/competitors");

    const name = "An unusually long competitor name for checking wrapping";
    const shown = () =>
      [...document.querySelectorAll("tr, li")].find(
        (row) => row.textContent?.includes(name) && row.getBoundingClientRect().height > 0,
      );
    await expect.poll(shown).toBeTruthy();

    const row = shown() as Element;
    expect(row.getBoundingClientRect().height).toBeLessThan(160);
    expect(document.documentElement.scrollWidth).toBeLessThanOrEqual(window.innerWidth);
  });
});
