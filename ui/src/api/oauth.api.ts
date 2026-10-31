import { apiErrorFromResponse } from "@/api/errors";

export type OAuthConsent = {
  clientName: string;
  scopes: string[];
  signedIn: boolean;
  csrf: string;
  redirectHost: string;
};

async function jsonResponse<T>(response: Response): Promise<T> {
  if (!response.ok) throw await apiErrorFromResponse(response);
  return response.json() as Promise<T>;
}

export function createOAuthApi() {
  return {
    consent: () =>
      fetch("/api/v1/oauth/consent", { credentials: "include", cache: "no-store" }).then(
        jsonResponse<OAuthConsent>,
      ),
    decide: (csrf: string, action: "approve" | "deny") =>
      fetch("/api/v1/oauth/consent", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ csrf, action }),
      }).then(jsonResponse<{ redirectUrl: string }>),
  };
}
