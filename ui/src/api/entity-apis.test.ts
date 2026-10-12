import { describe, expect, it, vi } from "vitest";

import { createAuthApi } from "@/api/auth.api";
import { createBrandsApi } from "@/api/brands.api";
import { createCompetitorsApi } from "@/api/competitors.api";
import type {
  AuthApi,
  BrandPage,
  BrandsApi,
  CompetitorsApi,
  GenerationsApi,
  HealthApi,
  PromptsApi,
} from "@/api/generated";
import { createGenerationsApi } from "@/api/generations.api";
import { createHealthApi } from "@/api/health.api";
import { createPromptsApi } from "@/api/prompts.api";

const page: BrandPage = {
  items: [],
  totalItems: 0,
  totalPages: 0,
  currentPage: 1,
  hasNext: false,
  hasPrev: false,
};

describe("entity API wrappers", () => {
  it("wraps generated auth methods", async () => {
    const account = {
      id: "user-id",
      email: "founder@example.com",
      organization: { id: "org-id", name: "Koi" },
    };

    const client = {
      login: vi.fn().mockResolvedValue(account),
      logout: vi.fn(),
      me: vi.fn(),
      signup: vi.fn(),
    } as unknown as AuthApi;

    const api = createAuthApi(client);
    const credentials = { email: account.email, password: "password123" };

    await expect(api.login(credentials)).resolves.toEqual(account);
    expect(client.login).toHaveBeenCalledWith(credentials);
  });

  it("wraps the generated health method", async () => {
    const health = { status: "ok", db: "ok" };
    const client = { get: vi.fn().mockResolvedValue(health) } as unknown as HealthApi;

    await expect(createHealthApi(client).get()).resolves.toEqual(health);
    expect(client.get).toHaveBeenCalledOnce();
  });

  it("sends generation modes and uses the setup key only when supplied", async () => {
    const accepted = { id: "run-id", mode: "initial", state: "queued", statusUrl: "/status" };

    const client = {
      createGeneration: vi.fn().mockResolvedValue(accepted),
      listGenerations: vi.fn().mockResolvedValue({ ...page, items: [] }),
      getGeneration: vi.fn().mockResolvedValue({ ...accepted, attempt: 0 }),
    } as unknown as GenerationsApi;

    const api = createGenerationsApi(client);

    await api.create("brand-id", "initial", "setup-brand-id");
    await api.create("brand-id", "prompts");
    await api.create("brand-id", "competitors");
    await api.list("brand-id");
    await api.getById("brand-id", "run-id");

    expect(client.createGeneration).toHaveBeenNthCalledWith(
      1,
      "brand-id",
      { mode: "initial" },
      "setup-brand-id",
    );
    expect(client.createGeneration).toHaveBeenNthCalledWith(
      2,
      "brand-id",
      { mode: "prompts" },
      undefined,
    );
    expect(client.createGeneration).toHaveBeenNthCalledWith(
      3,
      "brand-id",
      { mode: "competitors" },
      undefined,
    );
    expect(client.listGenerations).toHaveBeenCalledWith("brand-id", 1, 1, undefined);
    expect(client.getGeneration).toHaveBeenCalledWith("brand-id", "run-id", undefined);
  });

  it("exposes generic brand method names", async () => {
    const client = {
      listBrands: vi.fn().mockResolvedValue(page),
      getBrand: vi.fn(),
      createBrand: vi.fn(),
      updateBrand: vi.fn(),
      listBrandCountries: vi.fn(),
    } as unknown as BrandsApi;

    const api = createBrandsApi(client);

    await expect(api.list({ page: 2, pageSize: 5 })).resolves.toEqual(page);
    expect(client.listBrands).toHaveBeenCalledWith(2, 5);
  });

  it("keeps generated prompt names behind the wrapper", async () => {
    const promptPage = {
      ...page,
      items: [
        {
          id: "prompt-id",
          brandId: "brand",
          text: "What is Koi?",
          source: "drafted",
          confirmed: false,
        },
      ],
      totalItems: 1,
      totalPages: 1,
    };

    const client = {
      listPrompts: vi.fn().mockResolvedValue(promptPage),
      getPrompt: vi.fn(),
      createPrompt: vi.fn(),
      updatePrompt: vi.fn(),
      deletePrompt: vi.fn(),
      confirmPrompt: vi.fn(),
      confirmAllPrompts: vi.fn().mockResolvedValue(undefined),
    } as unknown as PromptsApi;

    const api = createPromptsApi(client);

    const result = await api.list("brand", {
      page: 3,
      pageSize: 10,
      q: "best tools",
      confirmed: false,
    });

    await api.confirmAll("brand");
    expect(result.items[0]).toMatchObject({ brandId: "brand", text: "What is Koi?" });
    expect(client.listPrompts).toHaveBeenCalledWith("brand", 3, 10, "best tools", false, undefined);
    expect(client.confirmAllPrompts).toHaveBeenCalledWith("brand");
  });

  it("keeps generated competitor names behind the wrapper", async () => {
    const client = {
      listCompetitors: vi.fn().mockResolvedValue(page),
      getCompetitor: vi.fn(),
      createCompetitor: vi.fn(),
      deleteCompetitor: vi.fn(),
      confirmCompetitor: vi.fn(),
      confirmAllCompetitors: vi.fn(),
    } as unknown as CompetitorsApi;

    const api = createCompetitorsApi(client);

    await api.list("brand", { page: 1, pageSize: 10, q: "acme", confirmed: true });
    expect(client.listCompetitors).toHaveBeenCalledWith("brand", 1, 10, "acme", true, undefined);
  });
});
