import { describe, expect, it } from "vitest";

import { apiErrorFromResponse, displayMessage } from "@/api/errors";
import i18n from "@/config/i18n";

describe("displayMessage", () => {
  it("maps field codes", () => {
    expect(displayMessage("validation", [{ field: "password", code: "min_length" }])).toBe(
      i18n.t("fields.password.min_length", { ns: "errors" }),
    );
    expect(displayMessage("validation", [{ field: "email", code: "invalid" }])).toBe(
      i18n.t("fields.email.invalid", { ns: "errors" }),
    );
  });

  it("maps envelope codes", () => {
    expect(displayMessage("invalid_credentials")).toBe(
      i18n.t("codes.invalid_credentials", { ns: "errors" }),
    );
    expect(displayMessage("email_taken")).toBe(i18n.t("codes.email_taken", { ns: "errors" }));
  });
});

describe("apiErrorFromResponse", () => {
  it("creates ApiError with code and fields, not UI copy", async () => {
    const response = new Response(
      JSON.stringify({ error: { code: "invalid_credentials", message: "invalid credentials" } }),
      { status: 401, headers: { "Content-Type": "application/json" } },
    );

    await expect(apiErrorFromResponse(response)).resolves.toMatchObject({
      name: "ApiError",
      code: "invalid_credentials",
      message: "invalid_credentials",
    });
  });
});
