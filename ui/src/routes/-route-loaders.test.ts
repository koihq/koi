import { QueryClient } from "@tanstack/react-query";
import { isRedirect } from "@tanstack/react-router";
import { describe, expect, it, vi } from "vitest";

import type { Api } from "@/api/apiProvider";
import { ApiError } from "@/api/errors";
import { loadAccountBrand, requireAccount } from "@/routes/-route-loaders";

describe("account route loading", () => {
  it("uses the resolved brand without listing brands, including an account with no brands", async () => {
    const list = vi.fn();
    const brand = { id: "brand-a", name: "Koi", organizationId: "org" };

    for (const activeBrand of [brand, null]) {
      const me = { id: "user", organization: { id: "org" }, activeBrand };
      const api = {
        auth: { me: vi.fn().mockResolvedValue(me) },
        brands: { list },
        capabilities: { get: vi.fn().mockResolvedValue({ cloudBilling: false }) },
      } as unknown as Api;

      const loaded = await loadAccountBrand(api, new QueryClient());
      expect(loaded.brand).toBe(activeBrand);
      expect(loaded.me).toBe(me);
      expect(list).not.toHaveBeenCalled();
    }
  });

  it("redirects only unauthorized responses", async () => {
    for (const error of [
      new ApiError("unauthorized"),
      new ApiError("internal"),
      new Error("offline"),
    ]) {
      const api = { auth: { me: vi.fn().mockRejectedValue(error) } } as unknown as Api;
      const caught = await requireAccount(api, new QueryClient()).catch(
        (failure: unknown) => failure,
      );

      if (error instanceof ApiError && error.code === "unauthorized") {
        expect(isRedirect(caught)).toBe(true);
      } else {
        expect(caught).toBe(error);
      }
    }
  });
});
