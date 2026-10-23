import { QueryClient } from "@tanstack/react-query";
import { isRedirect } from "@tanstack/react-router";
import { describe, expect, it, vi } from "vitest";

import type { Api } from "@/api/apiProvider";
import { ApiError } from "@/api/errors";
import { requireAccount } from "@/routes/-route-loaders";

describe("account route loading", () => {
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
