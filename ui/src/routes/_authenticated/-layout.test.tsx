import { describe, expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";
import { cleanup } from "vitest-browser-react";
import { box, fitsViewport, pointerAway } from "@/test/dom";
import { competitor, prompt, report, result, run, tracking } from "@/test/fixtures";
import { renderRoute } from "@/test/render-route";
import { serveWorkspace, workspace } from "@/test/worker";

const LONG = "An unusually long competitor name for checking wrapping";

function populated() {
  serveWorkspace(
    workspace({
      tracking: tracking({ startedAt: null }),
      prompts: [
        prompt({ text: "First question about analytics tools", visibility: report() }),
        prompt({
          id: "prompt-1",
          text: "Second question about analytics tools",
          visibility: report(),
        }),
      ],
      competitors: [competitor({ name: LONG })],
      runs: [run({ state: "succeeded" })],
      results: { run: [result({ engine: "chatgpt", rawAnswer: "Koi is mentioned here." })] },
    }),
  );
}

describe.each(["light", "dark"] as const)("layout in %s mode", (theme) => {
  test.each([
    1440, 960, 390, 320,
  ])("every app page fits %ipx without sideways scroll", async (width) => {
    await page.viewport(width, 1000);
    populated();

    for (const path of ["/", "/competitors", "/prompts/prompt-0", "/settings/brand"]) {
      await renderRoute(path, { theme });

      await expect.element(page.getByRole("main")).toBeVisible();
      await expect.element(page.getByText("Loading…", { exact: true })).not.toBeInTheDocument();
      await expect.poll(fitsViewport, { message: `${path} at ${width}px` }).toBe(true);

      await cleanup();
    }
  });

  test("settings align with the header", async () => {
    await page.viewport(1440, 1000);
    populated();

    await renderRoute("/settings/brand", { theme });
    await expect.element(page.getByRole("main")).toBeVisible();

    const main = box(page.getByRole("main"));
    const header = box(page.getByRole("banner"));

    expect(main.x).toBe(header.x);
    expect(main.width).toBe(header.width);
  });

  test("the account control keeps a 32px visible target with an extended hit area", async () => {
    populated();

    await renderRoute("/", { theme });

    const account = page.getByRole("button", { name: "Account", exact: true });
    await expect.element(account).toBeVisible();

    const style = getComputedStyle(account.element());
    expect(style.width).toBe("32px");
    expect(style.height).toBe("32px");
    expect(getComputedStyle(account.element(), "::after").inset).toBe("-4px");

    await pointerAway();
    expect(getComputedStyle(account.element()).backgroundColor).toBe("rgba(0, 0, 0, 0)");

    // The transparent hit extension, outside the visible 32px circle, belongs to the control.
    const bounds = box(account);
    const outside = document.elementFromPoint(bounds.right + 2, bounds.y + bounds.height / 2);
    expect(account.element().contains(outside)).toBe(true);

    await account.click();

    await expect
      .element(page.getByRole("menuitem", { name: "Sign out", exact: true }))
      .toBeVisible();
    await userEvent.keyboard("{Escape}");
    await expect.element(account).toHaveFocus();
  });
});
