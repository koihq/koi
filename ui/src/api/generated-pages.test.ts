import { describe, expect, it, vi } from "vitest";

import { BrandsApi, CompetitorsApi, Configuration, PromptsApi } from "@/api/generated";

describe("generated list deserialization", () => {
  it("converts typed items and retains populated and empty page metadata", async () => {
    const page = {
      items: [],
      totalItems: 0,
      totalPages: 0,
      currentPage: 1,
      hasNext: false,
      hasPrev: false,
    };
    const configuration = (items: unknown[]) =>
      new Configuration({
        basePath: "http://koi.test",
        fetchApi: vi.fn().mockResolvedValue(
          new Response(
            JSON.stringify({
              ...page,
              items,
              totalItems: items.length,
              totalPages: items.length ? 1 : 0,
            }),
            { headers: { "Content-Type": "application/json" } },
          ),
        ),
      });

    const brands = await new BrandsApi(
      configuration([
        {
          id: "b",
          organization_id: "o",
          name: "Koi",
          domain: "koi.dev",
          github_repo: "",
          country: "IT",
          description: "",
        },
      ]),
    ).listBrands();
    expect(brands.items[0]).toMatchObject({ organizationId: "o", githubRepo: "" });

    const prompts = await new PromptsApi(
      configuration([
        {
          id: "p",
          brand_id: "b",
          text: "Best tools?",
          source: "manual",
          confirmed: true,
          inactive: false,
        },
      ]),
    ).listPrompts("b");
    expect(prompts.items[0]).toMatchObject({ brandId: "b", text: "Best tools?" });
    expect(prompts).toMatchObject({ currentPage: 1, totalItems: 1, totalPages: 1 });

    const competitors = await new CompetitorsApi(
      configuration([
        {
          id: "c",
          brand_id: "b",
          name: "Other",
          domain: "other.dev",
          github_repo: "",
          source: "manual",
          confirmed: true,
        },
      ]),
    ).listCompetitors("b");
    expect(competitors.items[0]).toMatchObject({ brandId: "b", githubRepo: "" });

    expect(await new PromptsApi(configuration([])).listPrompts("b")).toEqual(page);
    expect(await new BrandsApi(configuration([])).listBrands()).toEqual(page);
    expect(await new CompetitorsApi(configuration([])).listCompetitors("b")).toEqual(page);
  });
});
