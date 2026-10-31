import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "@/api/apiProvider";
import type { OAuthConsent } from "@/api/oauth.api";
import { Button } from "@/components/ui/button";
import { clearAccountCache } from "@/lib/queries";
import { AuthShell } from "@/routes/-auth-shell";
import { CredentialsForm } from "@/routes/-credentials-form";

export const Route = createFileRoute("/oauth/consent")({ component: Consent });

function Consent() {
  const api = useApi();
  const { t } = useTranslation("oauth");
  const queryClient = useQueryClient();
  const [consent, setConsent] = useState<OAuthConsent | null>(null);
  const [email, setEmail] = useState("");
  const [intent, setIntent] = useState<"sign-in" | "sign-up">("sign-in");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  async function load() {
    setError("");
    try {
      const data = await api.oauth.consent();
      setConsent(data);
      if (data.signedIn) setEmail((await api.auth.me()).email);
    } catch {
      setError(t("unavailable"));
    }
  }

  useEffect(() => {
    void api.oauth
      .consent()
      .then(async (data) => {
        setConsent(data);
        if (data.signedIn) setEmail((await api.auth.me()).email);
      })
      .catch(() => setError(t("unavailable")));
  }, [api, t]);

  async function decide(action: "approve" | "deny") {
    if (!consent || pending) return;
    setPending(true);
    setError("");
    try {
      const { redirectUrl } = await api.oauth.decide(consent.csrf, action);
      window.location.assign(redirectUrl);
    } catch {
      setError(t("failed"));
      setPending(false);
    }
  }

  return (
    <AuthShell>
      {error && (
        <p role="alert" className="mb-6 text-sm text-destructive">
          {error}
        </p>
      )}
      {!consent ? (
        <Button variant="outline" onClick={() => void load()}>
          {t("retry")}
        </Button>
      ) : !consent.signedIn ? (
        <>
          <p className="mb-6 text-sm text-muted-foreground">
            {t("signInHint", { client: consent.clientName })}
          </p>
          <CredentialsForm
            intent={intent}
            passwordComplete={intent === "sign-in" ? "current-password" : "new-password"}
            onSwitchIntent={() => setIntent(intent === "sign-in" ? "sign-up" : "sign-in")}
            onSubmit={async (body) => {
              if (intent === "sign-in") await api.auth.login(body);
              else await api.auth.signup(body);
              await clearAccountCache(queryClient);
              await load();
            }}
          />
        </>
      ) : (
        <>
          <h1 className="text-2xl font-medium tracking-tight">
            {t("title", { client: consent.clientName })}
          </h1>
          <p className="mt-3 text-sm text-muted-foreground">{t("signedIn", { email })}</p>
          <div className="mt-8 space-y-3 text-sm">
            <p className="font-medium">{t("permissionsTitle")}</p>
            {consent.scopes.includes("koi:read") && <p>{t("read")}</p>}
            {consent.scopes.includes("koi:write") && <p>{t("write")}</p>}
          </div>
          <p className="mt-6 break-all text-xs text-muted-foreground">
            {t("returnTo", { host: consent.redirectHost })}
          </p>
          <div className="mt-8 flex gap-3">
            <Button disabled={pending} onClick={() => void decide("approve")}>
              {t("allow")}
            </Button>
            <Button variant="outline" disabled={pending} onClick={() => void decide("deny")}>
              {t("deny")}
            </Button>
          </div>
        </>
      )}
    </AuthShell>
  );
}
