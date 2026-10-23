import { QueryClient } from "@tanstack/react-query";
import { afterEach, describe, expect, it, vi } from "vitest";
import { type Api, createApi } from "@/api/apiProvider";
import {
  brandsOptions,
  capabilitiesOptions,
  clearAccountCache,
  promptListOptions,
} from "@/lib/queries";
import { runPromptBatch } from "@/routes/_authenticated/_app/-components/use-prompt-bulk-actions";

describe("query isolation and batch failures", () => {
  it("deduplicates shared capabilities and retains explicit brand cache keys", async () => {
    const client = new QueryClient();
    const get = vi.fn().mockResolvedValue({ aiDrafting: false });
    const list = vi.fn().mockImplementation(async ({ brandId }) => ({ items: [{ brandId }] }));
    const api = { capabilities: { get }, prompts: { list } } as unknown as Api;
    await Promise.all([
      client.fetchQuery(capabilitiesOptions(api)),
      client.fetchQuery(capabilitiesOptions(api)),
    ]);
    expect(get).toHaveBeenCalledOnce();
    const a = promptListOptions(api, "a", {});
    const b = promptListOptions(api, "b", {});
    await Promise.all([client.fetchQuery(a), client.fetchQuery(b)]);
    expect(client.getQueryData(a.queryKey)).toEqual({ items: [{ brandId: "a" }] });
    expect(client.getQueryData(b.queryKey)).toEqual({ items: [{ brandId: "b" }] });
    await clearAccountCache(client);
    expect(client.getQueryCache().getAll()).toEqual([]);
  });

  it("retains only failed IDs after independent batch actions", async () => {
    const targets = [{ id: "saved" }, { id: "failed" }, { id: "saved-too" }];
    const failed = await runPromptBatch(targets, async (item) => {
      if (item.id === "failed") throw new Error("network failure");
    });
    expect(failed).toEqual(["failed"]);
  });
});

afterEach(() => vi.unstubAllGlobals());

describe("account changes during pagination", () => {
  it("aborts brand requests and stops collecting subsequent pages", async () => {
    const client = new QueryClient();
    let requestSignal: AbortSignal | undefined;
    const fetch = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            items: [
              {
                id: "brand-a",
                organizationId: "org-a",
                name: "A",
                domain: "a.example",
                githubRepo: "",
                country: "IT",
                description: "",
              },
            ],
            hasNext: true,
            hasPrev: false,
            currentPage: 1,
            totalPages: 3,
            totalItems: 201,
          }),
          { headers: { "Content-Type": "application/json" } },
        ),
      )
      .mockImplementation((_address, init?: RequestInit) => {
        requestSignal = init?.signal ?? undefined;
        return new Promise((_resolve, reject) => {
          requestSignal?.addEventListener(
            "abort",
            () => reject(new DOMException("Aborted", "AbortError")),
            { once: true },
          );
        });
      });
    vi.stubGlobal("fetch", fetch);
    const pending = client.fetchQuery(brandsOptions(createApi(), "org-a"));
    const settled = pending.catch((error: unknown) => error);
    await vi.waitFor(() => expect(fetch).toHaveBeenCalledTimes(2));

    await clearAccountCache(client);
    await settled;

    expect(requestSignal?.aborted).toBe(true);
    expect(fetch.mock.calls[0][1].signal).toBe(requestSignal);
    expect(fetch).toHaveBeenCalledTimes(2);
    expect(client.getQueryCache().getAll()).toEqual([]);
  });
});
