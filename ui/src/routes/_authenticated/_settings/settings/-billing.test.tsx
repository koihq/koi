import { HttpResponse, http } from "msw";
import { describe, expect, test, vi } from "vitest";
import { page } from "vitest/browser";

import { billingCatalog, capabilities, subscription } from "@/test/fixtures";
import { renderRoute } from "@/test/render-route";
import { apiError, serveWorkspace, worker, workspace } from "@/test/worker";

const allowance = () => page.getByRole("slider", { name: "Tracked prompts" });

async function expectAllowance(prompts: number) {
  await expect.element(allowance()).toHaveAttribute("aria-valuetext", `${prompts} prompts`);
}

async function chooseAllowance(prompts: number) {
  const index = billingCatalog().options.findIndex((option) => option.prompts === prompts);

  await allowance().fill(String(index));
  await expectAllowance(prompts);
}

describe("billing settings", () => {
  test("starts from the current allowance and offers a change only after a new choice", async () => {
    serveWorkspace(workspace({ capabilities: capabilities({ cloudBilling: true }) }));

    await renderRoute("/settings/billing");

    await expectAllowance(25);
    await expect.element(page.getByText("25 tracked prompts")).toBeVisible();
    await expect.element(page.getByText("Active", { exact: true })).toBeVisible();
    await expect
      .element(page.getByRole("button", { name: "Change allowance" }))
      .not.toBeInTheDocument();

    await chooseAllowance(50);

    await expect.element(page.getByText("€45 → €75 / month")).toBeVisible();
    await expect.element(page.getByRole("button", { name: "Change allowance" })).toBeEnabled();
  });

  test("cancels a pending downgrade with an idempotency key", async () => {
    const state = serveWorkspace(workspace({ capabilities: capabilities({ cloudBilling: true }) }));
    state.billing.subscription = subscription({ pendingQuantity: 15 });

    const requests: { body: unknown; key: string | null }[] = [];

    worker.use(
      http.post("/api/v1/billing/change-allowance", async ({ request }) => {
        requests.push({ body: await request.json(), key: request.headers.get("idempotency-key") });
        state.billing.subscription = subscription();

        return HttpResponse.json({ quantity: 25, effective: "current" });
      }),
    );

    await renderRoute("/settings/billing");

    await expectAllowance(15);
    await expect
      .element(page.getByText("15 prompts scheduled for the next billing cycle."))
      .toBeVisible();

    await page.getByRole("button", { name: "Keep current allowance" }).click();

    await expect.element(page.getByText("Your current allowance has been kept.")).toBeVisible();
    await expectAllowance(25);
    expect(requests).toEqual([{ body: { prompts: 25 }, key: expect.any(String) }]);
  });

  test("shows payment recovery and updates after confirmation", async () => {
    const state = serveWorkspace(workspace({ capabilities: capabilities({ cloudBilling: true }) }));
    state.billing.subscription = subscription({
      pendingUpgradeQuantity: 50,
      paymentUrl: "https://invoice.stripe.com/payment",
    });

    await renderRoute("/settings/billing");

    await expect
      .element(page.getByText("An upgrade to 50 prompts is awaiting payment."))
      .toBeVisible();
    await expect
      .element(page.getByRole("link", { name: "Complete payment" }))
      .toHaveAttribute("href", "https://invoice.stripe.com/payment");
    await expect.element(allowance()).toBeDisabled();

    state.billing.subscription = subscription({ quantity: 50 });
    state.billing.promptLimit = 50;

    // The page polls every 2 seconds while an upgrade awaits payment.
    await expect.element(page.getByText("50 tracked prompts"), { timeout: 5_000 }).toBeVisible();
    await expect
      .element(page.getByRole("link", { name: "Complete payment" }))
      .not.toBeInTheDocument();
    await expect.element(allowance()).toBeEnabled();
  });

  test("stops waiting automatically and offers a manual refresh", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true, toFake: ["setTimeout", "clearTimeout", "Date"] });

    const state = serveWorkspace(workspace({ capabilities: capabilities({ cloudBilling: true }) }));
    state.billing.promptLimit = 0;

    await renderRoute("/settings/billing?checkout=success");

    await expect
      .element(
        page.getByText("Your payment or subscription change is being confirmed.", { exact: false }),
      )
      .toBeVisible();

    await vi.advanceTimersByTimeAsync(61_000);
    await expect
      .element(page.getByText("Confirmation is taking longer than expected.", { exact: false }))
      .toBeVisible();

    state.billing.promptLimit = 25;
    await page.getByRole("button", { name: "Refresh status" }).click();

    await expect.element(page.getByRole("link", { name: "Continue to prompts" })).toBeVisible();
  });

  test("preserves useful billing errors and keeps the retry key", async () => {
    const state = serveWorkspace(workspace({ capabilities: capabilities({ cloudBilling: true }) }));
    state.billing.subscription = subscription({ pendingQuantity: 15 });

    const keys: (string | null)[] = [];

    worker.use(
      http.post("/api/v1/billing/change-allowance", ({ request }) => {
        keys.push(request.headers.get("idempotency-key"));

        return apiError(409, "billing_conflict");
      }),
    );

    await renderRoute("/settings/billing");

    const keep = page.getByRole("button", { name: "Keep current allowance" });
    await keep.click();
    await expect
      .element(
        page.getByText("Open billing management to resolve your subscription", { exact: false }),
      )
      .toBeVisible();

    await keep.click();
    await expect.poll(() => keys.length).toBe(2);
    expect(keys[0]).toBeTruthy();
    expect(keys[0]).toBe(keys[1]);
  });

  test("keeps controls visible on a narrow viewport", async () => {
    await page.viewport(320, 740);

    const state = serveWorkspace(workspace({ capabilities: capabilities({ cloudBilling: true }) }));
    state.billing.subscription = subscription({ pendingQuantity: 15 });

    await renderRoute("/settings/billing");

    await expect
      .element(page.getByRole("button", { name: "Keep current allowance" }))
      .toBeVisible();
    await expect.element(page.getByRole("button", { name: "Manage billing" })).toBeVisible();
    expect(document.documentElement.scrollWidth).toBeLessThanOrEqual(320);
  });

  test("changes the future allowance during trial", async () => {
    const state = serveWorkspace(workspace({ capabilities: capabilities({ cloudBilling: true }) }));
    state.billing.subscription = subscription({ status: "trialing" });
    state.billing.promptLimit = 10;

    const bodies: unknown[] = [];

    worker.use(
      http.post("/api/v1/billing/change-allowance", async ({ request }) => {
        bodies.push(await request.json());
        state.billing.subscription = subscription({ status: "trialing", quantity: 50 });

        return HttpResponse.json({ quantity: 50, effective: "trial_end" }, { status: 202 });
      }),
    );

    await renderRoute("/settings/billing");

    await expect
      .element(
        page.getByText("Your trial end date and 10-prompt allowance stay the same.", {
          exact: false,
        }),
      )
      .toBeVisible();

    await chooseAllowance(50);
    await page.getByRole("button", { name: "Change allowance" }).click();

    await expect
      .element(page.getByText("Your selected allowance will start when your trial ends."))
      .toBeVisible();
    await expect
      .element(
        page.getByText(
          "Your trial includes 10 prompts. Your 50-prompt allowance starts when it ends.",
        ),
      )
      .toBeVisible();
    expect(bodies).toEqual([{ prompts: 50 }]);
  });

  test("starts the plan during trial after confirming the charge", async () => {
    const state = serveWorkspace(workspace({ capabilities: capabilities({ cloudBilling: true }) }));
    state.billing.subscription = subscription({ status: "trialing" });
    state.billing.promptLimit = 10;

    const keys: (string | null)[] = [];

    worker.use(
      http.post("/api/v1/billing/start-plan", ({ request }) => {
        keys.push(request.headers.get("idempotency-key"));
        state.billing.subscription = subscription();
        state.billing.promptLimit = 25;

        return HttpResponse.json({ quantity: 25, effective: "started" });
      }),
    );

    await renderRoute("/settings/billing");

    await page.getByRole("button", { name: "Start plan now" }).click();
    await expect
      .element(page.getByText("€45 is charged to your saved card", { exact: false }))
      .toBeVisible();
    await page.getByRole("button", { name: "Pay €45 now" }).click();

    await expect.element(page.getByText("Your plan is active.")).toBeVisible();
    await expect.element(page.getByText("Active", { exact: true })).toBeVisible();
    await expect
      .element(page.getByRole("button", { name: "Start plan now" }))
      .not.toBeInTheDocument();
    expect(keys).toEqual([expect.any(String)]);
  });

  test("keeps the trial and offers recovery when the payment fails", async () => {
    const state = serveWorkspace(workspace({ capabilities: capabilities({ cloudBilling: true }) }));
    state.billing.subscription = subscription({ status: "trialing" });
    state.billing.promptLimit = 10;

    const expiresAt = new Date(Date.now() + 23 * 60 * 60 * 1000).toISOString();

    worker.use(
      http.post("/api/v1/billing/start-plan", () => {
        state.billing.subscription = subscription({
          status: "trialing",
          pendingUpgradeQuantity: 25,
          pendingUpgradeExpiresAt: expiresAt,
          paymentUrl: "https://invoice.stripe.com/payment",
        });

        return HttpResponse.json(
          {
            quantity: 25,
            effective: "after_payment",
            paymentUrl: "https://invoice.stripe.com/payment",
            expiresAt,
          },
          { status: 202 },
        );
      }),
    );

    await renderRoute("/settings/billing");

    await page.getByRole("button", { name: "Start plan now" }).click();
    await page.getByRole("button", { name: "Pay €45 now" }).click();

    await expect
      .element(page.getByText("Your payment didn't go through", { exact: false }))
      .toBeVisible();
    await expect
      .element(page.getByRole("link", { name: "Complete payment" }))
      .toHaveAttribute("href", "https://invoice.stripe.com/payment");
    await expect.element(page.getByRole("button", { name: "Manage billing" })).toBeVisible();
    await expect
      .element(page.getByRole("button", { name: "Start plan now" }))
      .not.toBeInTheDocument();
    await expect.element(page.getByText("Trial", { exact: true })).toBeVisible();
    await expect
      .element(page.getByText("being confirmed", { exact: false }))
      .not.toBeInTheDocument();
  });

  test("explains an expired upgrade and allows another attempt", async () => {
    const state = serveWorkspace(workspace({ capabilities: capabilities({ cloudBilling: true }) }));
    state.billing.subscription = subscription({
      pendingUpgradeQuantity: 50,
      pendingUpgradeExpiresAt: new Date(Date.now() - 1000).toISOString(),
      paymentUrl: "https://invoice.stripe.com/expired",
    });

    await renderRoute("/settings/billing");

    await expect
      .element(page.getByText("The upgrade payment expired.", { exact: false }))
      .toBeVisible();
    await expect
      .element(page.getByRole("link", { name: "Complete payment" }))
      .not.toBeInTheDocument();

    await chooseAllowance(50);

    await expect.element(page.getByRole("button", { name: "Change allowance" })).toBeEnabled();
  });
});
