import { useQuery } from "@tanstack/react-query";
import type { TFunction } from "i18next";
import { useTranslation } from "react-i18next";

import { useApi } from "@/api/apiProvider";
import { displayMessage } from "@/api/errors";
import googleLogo from "@/assets/google-g.svg";
import { Button } from "@/components/ui/button";
import { authProvidersOptions } from "@/lib/queries";

/** The codes the Google callback sends back to /sign-in. */
export const googleErrorCodes = [
  "google_cancelled",
  "google_unverified",
  "google_failed",
  "signup_closed",
] as const;

export type GoogleErrorCode = (typeof googleErrorCodes)[number];

export function parseGoogleError(value: unknown): GoogleErrorCode | undefined {
  return googleErrorCodes.find((code) => code === value);
}

export function googleErrorMessage(t: TFunction<"auth">, code: GoogleErrorCode): string {
  switch (code) {
    case "google_cancelled":
      return t("google.errors.cancelled");
    case "google_unverified":
      return t("google.errors.unverified");
    case "google_failed":
      return t("google.errors.failed");
    case "signup_closed":
      return displayMessage(code);
  }
}

/**
 * "Continue with Google" and the "or" before the email form. A link rather than a request:
 * Google's consent screen needs a full page navigation. Renders nothing without a Google client.
 */
export function GoogleSignIn({ returnTo }: { returnTo: string }) {
  const { t } = useTranslation("auth");
  const api = useApi();
  const { data } = useQuery(authProvidersOptions(api));

  if (!data?.google) return null;

  const href = `/api/v1/auth/google/start?${new URLSearchParams({ returnTo })}`;

  return (
    <>
      <Button
        variant="outline"
        size="lg"
        className="w-full"
        nativeButton={false}
        role="link"
        render={
          <a href={href}>
            <img src={googleLogo} alt="" className="size-4" />
            <span className="[text-box:trim-both_cap_alphabetic]">{t("google.continue")}</span>
          </a>
        }
      />
      <p className="text-center text-xs text-muted-foreground">{t("google.or")}</p>
    </>
  );
}
