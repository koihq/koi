import type { BillingApi } from "@/api/generated";

export function createBillingApi(api: BillingApi) {
  return {
    catalog: (init?: RequestInit) => api.billingCatalog(init),
    quote: (prompts: number) => api.billingQuote({ prompts }),
    subscription: (init?: RequestInit) => api.billingSubscription(init),
    checkout: (prompts: number) => api.billingCheckout({ billingAllowanceRequest: { prompts } }),
    changeAllowance: (prompts: number, idempotencyKey: string) =>
      api.billingChangeAllowance({ idempotencyKey, billingAllowanceRequest: { prompts } }),
    portal: () => api.billingPortal(),
  };
}
