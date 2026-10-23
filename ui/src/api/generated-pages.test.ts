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
          organizationId: "o",
          name: "Koi",
          domain: "koi.dev",
          githubRepo: "",
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
          brandId: "b",
          text: "Best tools?",
          source: "manual",
          confirmed: true,
          inactive: false,
        },
      ]),
    ).listPrompts({ brandId: "b" });
    expect(prompts.items[0]).toMatchObject({ brandId: "b", text: "Best tools?" });
    expect(prompts).toMatchObject({ currentPage: 1, totalItems: 1, totalPages: 1 });

    const competitors = await new CompetitorsApi(
      configuration([
        {
          id: "c",
          brandId: "b",
          name: "Other",
          domain: "other.dev",
          githubRepo: "",
          source: "manual",
          confirmed: true,
        },
      ]),
    ).listCompetitors({ brandId: "b" });
    expect(competitors.items[0]).toMatchObject({ brandId: "b", githubRepo: "" });

    expect(await new PromptsApi(configuration([])).listPrompts({ brandId: "b" })).toEqual(page);
    expect(await new BrandsApi(configuration([])).listBrands()).toEqual(page);
    expect(await new CompetitorsApi(configuration([])).listCompetitors({ brandId: "b" })).toEqual(
      page,
    );
  });
});
