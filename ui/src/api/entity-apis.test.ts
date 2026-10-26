import { afterEach, describe, expect, it, vi } from "vitest";
import { createApi } from "@/api/apiProvider";

afterEach(() => vi.unstubAllGlobals());

function mockResponse(body: unknown) {
  const fetch = vi
    .fn()
    .mockImplementation(
      async () =>
        new Response(JSON.stringify(body), { headers: { "Content-Type": "application/json" } }),
    );
  vi.stubGlobal("fetch", fetch);
  return fetch;
}

const emptyPage = {
  items: [],
  totalItems: 0,
  totalPages: 0,
  currentPage: 1,
  hasNext: false,
  hasPrev: false,
};

describe("API wire contract", () => {
  it("sends explicit brand scope and filters on flat collections", async () => {
    const fetch = mockResponse(emptyPage);
    const api = createApi();
    await expect(
      api.prompts.list({ brandId: "brand-a", page: 2, scope: { days: 7, engines: ["chatgpt"] } }),
    ).resolves.toEqual(emptyPage);
    const [address] = fetch.mock.calls[0];
    const url = new URL(address, "http://koi.test");
    expect(url.pathname).toBe("/api/v1/prompts");
    expect(Object.fromEntries(url.searchParams)).toMatchObject({
      brandId: "brand-a",
      page: "2",
      days: "7",
      engines: "chatgpt",
    });
  });

  it("puts creation scope in the body and identifies updates by resource ID", async () => {
    const fetch = mockResponse({
      id: "p",
      brandId: "brand-a",
      text: "Question",
      source: "manual",
      confirmed: true,
      inactive: false,
    });
    const api = createApi();
    await api.prompts.create({ brandId: "brand-a", text: "Question" });
    expect(fetch.mock.calls[0][0]).toBe("/api/v1/prompts");
    expect(JSON.parse(fetch.mock.calls[0][1].body)).toEqual({
      brandId: "brand-a",
      text: "Question",
    });
    await api.prompts.update("p", { text: "Changed" });
    expect(fetch.mock.calls[1][0]).toBe("/api/v1/prompts/p");
    expect(JSON.parse(fetch.mock.calls[1][1].body)).toEqual({ text: "Changed" });
    expect(fetch.mock.calls[1][1].credentials).toBe("include");
  });

  it("retains generation idempotency and cancellation", async () => {
    const fetch = mockResponse({
      id: "g",
      mode: "initial",
      state: "queued",
      statusUrl: "/api/v1/generations/g",
    });
    const api = createApi();
    await api.generations.create({ brandId: "brand-a", mode: "initial" }, "setup-brand-a");
    expect(fetch.mock.calls[0][1].headers["Idempotency-Key"]).toBe("setup-brand-a");
    expect(JSON.parse(fetch.mock.calls[0][1].body)).toEqual({
      brandId: "brand-a",
      mode: "initial",
    });
    const controller = new AbortController();
    await api.generations.getById("g", { signal: controller.signal });
    expect(fetch.mock.calls[1][0]).toBe("/api/v1/generations/g");
    expect(fetch.mock.calls[1][1].signal).toBe(controller.signal);
  });
});
