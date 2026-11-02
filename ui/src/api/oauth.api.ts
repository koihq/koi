import type { OAuthConsent, OauthApi } from "@/api/generated";

export type { OAuthConsent };

export function createOAuthApi(api: OauthApi) {
  return {
    consent: (): Promise<OAuthConsent> => api.getOAuthConsent({ cache: "no-store" }),
    decide: (csrf: string, action: "approve" | "deny") =>
      api.decideOAuthConsent({ oAuthConsentDecision: { csrf, action } }),
  };
}
