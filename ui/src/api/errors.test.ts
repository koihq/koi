import { describe, expect, it } from "vitest";

import { displayMessage, envelopeMiddleware } from "@/api/errors";

describe("displayMessage", () => {
  it("maps field codes", () => {
    expect(displayMessage("validation", [{ field: "password", code: "min_length" }])).toBe(
      "Password must be at least 8 characters",
    );
    expect(displayMessage("validation", [{ field: "email", code: "invalid" }])).toBe(
      "Enter a valid email",
    );
  });

  it("maps envelope codes", () => {
    expect(displayMessage("invalid_credentials")).toBe("Invalid email or password");
    expect(displayMessage("email_taken")).toBe("That email is already registered");
  });
});

describe("envelopeMiddleware", () => {
  it("throws ApiError with UI copy", async () => {
    const response = new Response(
      JSON.stringify({ error: { code: "invalid_credentials", message: "invalid credentials" } }),
      { status: 401, headers: { "Content-Type": "application/json" } },
    );
    await expect(
      envelopeMiddleware.post?.({
        fetch,
        url: "/api/v1/auth/login",
        init: { method: "POST" },
        response,
      }),
    ).rejects.toMatchObject({
      name: "ApiError",
      message: "Invalid email or password",
    });
  });
});
