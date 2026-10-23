import { expect, test } from "@playwright/test";

test("an open form keeps its explicit brand when another tab changes the preference", async ({
  context,
}) => {
  const brands = [
    {
      id: "brand-a",
      organizationId: "org",
      name: "Brand A",
      domain: "a.example",
      githubRepo: "",
      country: "IT",
      description: "First",
    },
    {
      id: "brand-b",
      organizationId: "org",
      name: "Brand B",
      domain: "b.example",
      githubRepo: "",
      country: "IT",
      description: "Second",
    },
  ];
  let activeBrandId = brands[0].id;
  const updates: string[] = [];
  await context.route("**/api/v1/**", async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    let json: unknown;
    if (path === "/api/v1/auth/me")
      json = {
        id: "user",
        email: "founder@example.com",
        organization: { id: "org", name: "Organization", role: "owner" },
        activeBrand: brands.find((brand) => brand.id === activeBrandId),
      };
    else if (path === "/api/v1/brands")
      json = {
        items: brands,
        totalItems: 2,
        totalPages: 1,
        currentPage: 1,
        hasNext: false,
        hasPrev: false,
      };
    else if (path === "/api/v1/capabilities")
      json = {
        cloudBilling: false,
        aiDrafting: false,
        competitorSearch: false,
        liveCollection: false,
      };
    else if (path === "/api/v1/brands/countries") json = [{ code: "IT", name: "Italy" }];
    else if (path.endsWith("/activate")) {
      activeBrandId = path.split("/")[4];
      json = brands.find((brand) => brand.id === activeBrandId);
    } else if (request.method() === "PATCH" && path.startsWith("/api/v1/brands/")) {
      const id = path.split("/")[4];
      updates.push(id);
      json = Object.assign(brands.find((brand) => brand.id === id) ?? {}, request.postDataJSON());
    } else {
      await route.fulfill({
        status: 404,
        json: { error: { code: "not_found", message: "not found" } },
      });
      return;
    }
    await route.fulfill({ json });
  });
  const first = await context.newPage();
  const second = await context.newPage();
  await first.goto("/settings/brand");
  await expect(first.getByRole("textbox", { name: "Brand name" })).toHaveValue("Brand A");
  await second.goto("/settings/brand");
  await second.evaluate(async () => {
    await fetch("/api/v1/brands/brand-b/activate", { method: "POST" });
  });
  await first.getByRole("textbox", { name: "Description" }).fill("Edited in first tab");
  await first.getByRole("button", { name: "Save changes" }).click();
  await expect.poll(() => updates).toEqual(["brand-a"]);
});
