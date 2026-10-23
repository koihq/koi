import { afterEach, describe, expect, it, vi } from "vitest";
import { createApi } from "@/api/apiProvider";

afterEach(() => vi.unstubAllGlobals());

describe("shared API errors", () => {
  it("normalizes HTTP errors for every generated resource", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation(
        async () =>
          new Response(
            JSON.stringify({
              error: { code: "not_found", message: "not found", requestId: "req-1" },
            }),
            { status: 404 },
          ),
      ),
    );
    const api = createApi();
    await expect(api.prompts.getById("p")).rejects.toMatchObject({
      name: "ApiError",
      code: "not_found",
      requestId: "req-1",
    });
    await expect(api.brands.getById("b")).rejects.toMatchObject({
      name: "ApiError",
      code: "not_found",
    });
  });

  it("uses an internal error for unreadable error responses", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("bad gateway", { status: 502 })));
    await expect(createApi().auth.me()).rejects.toMatchObject({ code: "internal" });
  });
});
