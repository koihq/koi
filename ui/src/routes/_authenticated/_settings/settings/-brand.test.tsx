import { describe, expect, test } from "vitest";
import { page } from "vitest/browser";

import { billingStatus, brand, capabilities, me } from "@/test/fixtures";
import { renderRoute } from "@/test/render-route";
import { serveWorkspace, workspace } from "@/test/worker";

const first = brand({ id: "brand-a", name: "Brand A", domain: "a.example", description: "First" });
const second = brand({
  id: "brand-b",
  name: "Brand B",
  domain: "b.example",
  description: "Second",
});

describe("brand settings", () => {
  test("an open form keeps its explicit brand when another tab changes the preference", async () => {
    await page.viewport(1440, 1000);
    const state = serveWorkspace(
      workspace({ me: me({ activeBrand: first }), brands: [first, second] }),
    );

    await renderRoute("/settings/brand");

    await page
      .getByRole("row", { name: /^Brand A / })
      .getByRole("button", { name: "Edit", exact: true })
      .click();

    const form = page.getByRole("dialog");
    await expect.element(form.getByRole("textbox", { name: "Brand name" })).toHaveValue("Brand A");

    // Another tab switches the session's active brand.
    state.me = { ...state.me, activeBrand: second };

    await form.getByRole("textbox", { name: "Description" }).fill("Edited in first tab");
    await form.getByRole("button", { name: "Save changes", exact: true }).click();

    // The save goes to the brand the form was opened for, not to the new preference.
    await expect
      .poll(() => state.brands.map((item) => item.description))
      .toEqual(["Edited in first tab", "Second"]);
  });

  test("hides adding a second brand during the free check", async () => {
    serveWorkspace(
      workspace({
        capabilities: capabilities({ cloudBilling: true }),
        billing: billingStatus({
          subscription: null,
          promptLimit: 5,
          firstLook: true,
          trialAvailable: true,
        }),
      }),
    );

    await renderRoute("/settings/brand");

    await expect.element(page.getByRole("heading", { name: "Brands" })).toBeVisible();
    await expect
      .element(page.getByRole("button", { name: "Create brand" }))
      .not.toBeInTheDocument();
  });

  test("offers creating a brand on a paid plan", async () => {
    serveWorkspace(workspace({ capabilities: capabilities({ cloudBilling: true }) }));

    await renderRoute("/settings/brand");

    await expect.element(page.getByRole("button", { name: "Create brand" })).toBeVisible();
  });
});
