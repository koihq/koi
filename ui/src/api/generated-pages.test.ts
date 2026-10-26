import { describe, expect, it, vi } from "vitest";

import {
  AuthApi,
  BrandsApi,
  CompetitorsApi,
  Configuration,
  PromptsApi,
  VisibilityApi,
} from "@/api/generated";

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

describe("generated analytics and account deserialization", () => {
  it("retains competitor identities, prompt state and the resolved or absent active brand", async () => {
    const metric = { completed: 0, cited: 0, mentioned: 0, rate: null };
    const competitor = {
      id: "competitor",
      name: "Other",
      domain: "other.example",
      metric,
      series: [],
    };
    const report = { metric, previous: metric, series: [], engines: {}, competitors: [competitor] };
    const fetchApi = vi
      .fn()
      .mockImplementation(async () => Response.json({ ...report, hasConfirmedPrompts: false }));
    const config = new Configuration({ basePath: "http://koi.test", fetchApi });

    const summary = await new VisibilityApi(config).getVisibilitySummary({ brandId: "brand" });
    expect(summary.hasConfirmedPrompts).toBe(false);
    expect(summary.competitors).toEqual([competitor]);

    const brand = {
      id: "brand",
      organizationId: "org",
      name: "Koi",
      domain: "koi.example",
      githubRepo: "",
      country: "IT",
      description: "",
    };

    for (const activeBrand of [brand, null]) {
      fetchApi.mockImplementation(async () =>
        Response.json({
          id: "user",
          email: "founder@example.com",
          organization: { id: "org", name: "Organization", role: "owner" },
          activeBrand,
        }),
      );

      expect((await new AuthApi(config).me()).activeBrand).toEqual(activeBrand);
    }
  });
});
