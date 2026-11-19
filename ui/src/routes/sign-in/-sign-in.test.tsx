import { describe, expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderRoute } from "@/test/render-route";
import { serveWorkspace, workspace } from "@/test/worker";

const google = () => page.getByRole("link", { name: "Continue with Google" });

describe("sign-in with Google", () => {
  test("offers Google above the form and returns to the page that asked", async () => {
    serveWorkspace(workspace({ providers: { google: true } }));

    await renderRoute("/sign-up");

    await expect
      .element(google())
      .toHaveAttribute("href", "/api/v1/auth/google/start?returnTo=%2F");
    await expect.element(page.getByText("or", { exact: true })).toBeVisible();

    const button = google().element().getBoundingClientRect();
    const email = page.getByLabelText("Email").element().getBoundingClientRect();
    expect(button.bottom).toBeLessThan(email.top);
  });

  test("keeps the email form alone when the instance has no Google client", async () => {
    serveWorkspace(workspace());

    await renderRoute("/sign-in");

    await expect.element(page.getByLabelText("Email")).toBeVisible();
    await expect.element(google()).not.toBeInTheDocument();
    await expect.element(page.getByText("or", { exact: true })).not.toBeInTheDocument();
  });

  test("explains a failed Google sign-in until the next attempt", async () => {
    serveWorkspace(workspace({ providers: { google: true } }));

    await renderRoute("/sign-in?error=google_unverified");

    await expect
      .element(page.getByText("Google hasn't verified this email.", { exact: false }))
      .toBeVisible();
    await expect.element(page.getByLabelText("Email")).not.toHaveAttribute("aria-invalid");
  });

  test("ignores an error code it does not know", async () => {
    serveWorkspace(workspace({ providers: { google: true } }));

    await renderRoute("/sign-in?error=%3Cscript%3E");

    await expect.element(google()).toBeVisible();
    await expect.element(page.getByText("Google", { exact: false }).nth(1)).not.toBeInTheDocument();
  });
});
