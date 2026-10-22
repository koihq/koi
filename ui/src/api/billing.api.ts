import type { BillingApi } from "@/api/generated";
import { callApi } from "@/api/shared/apiClient";

export function createBillingApi(api: BillingApi) {
  return {
    catalog: () => callApi(() => api.billingCatalog()),
    quote: (prompts: number) => callApi(() => api.billingQuote(prompts)),
    subscription: () => callApi(() => api.billingSubscription()),
    checkout: (prompts: number) => callApi(() => api.billingCheckout({ prompts })),
    changeAllowance: (prompts: number, idempotencyKey: string) =>
      callApi(() => api.billingChangeAllowance(idempotencyKey, { prompts })),
    portal: () => callApi(() => api.billingPortal()),
  };
}
