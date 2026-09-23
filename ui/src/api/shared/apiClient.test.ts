import { describe, expect, it } from "vitest";

import { ResponseError } from "@/api/generated";
import { callApi, createApiClient } from "@/api/shared/apiClient";

describe("shared API client", () => {
  it("creates all generated resources from one composition boundary", () => {
    const client = createApiClient();

    expect(client.auth.login).toBeTypeOf("function");
    expect(client.brands.listBrands).toBeTypeOf("function");
    expect(client.competitors.listCompetitors).toBeTypeOf("function");
    expect(client.health.get).toBeTypeOf("function");
    expect(client.prompts.listPrompts).toBeTypeOf("function");
  });

  it("converts generated response errors into ApiError", async () => {
    const response = new Response(
      JSON.stringify({ error: { code: "not_found", message: "not found" } }),
      { status: 404, headers: { "Content-Type": "application/json" } },
    );

    await expect(
      callApi(() => Promise.reject(new ResponseError(response, "request failed"))),
    ).rejects.toMatchObject({ name: "ApiError", code: "not_found" });
  });
});
