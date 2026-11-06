import { createFileRoute, Link } from "@tanstack/react-router";
import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";

import { useApi } from "@/api/apiProvider";
import { ApiError, messageFromUnknown } from "@/api/errors";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { AuthShell } from "@/routes/-auth-shell";

export const Route = createFileRoute("/forgot-password/")({
  component: ForgotPassword,
});

function ForgotPassword() {
  const { t } = useTranslation("auth");
  const api = useApi();
  const [sentTo, setSentTo] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const email = String(new FormData(event.currentTarget).get("email") ?? "").trim();
    setError(null);
    setPending(true);

    try {
      await api.auth.forgotPassword(email);
      setSentTo(email);
    } catch (err) {
      setError(
        err instanceof ApiError && err.code === "feature_unavailable"
          ? t("forgot.unavailable")
          : messageFromUnknown(err),
      );
    } finally {
      setPending(false);
    }
  }

  return (
    <AuthShell>
      {sentTo ? (
        <>
          <h1 className="text-2xl font-semibold tracking-tight">{t("forgot.sentTitle")}</h1>
          <p className="mt-4 text-sm text-muted-foreground">
            {t("forgot.sentBody", { email: sentTo })}
          </p>
        </>
      ) : (
        <>
          <h1 className="text-2xl font-semibold tracking-tight">{t("forgot.title")}</h1>
          <form className="mt-8" onSubmit={handleSubmit}>
            <FieldGroup>
              <Field data-invalid={error ? true : undefined}>
                <FieldLabel htmlFor="email">{t("fields.email")}</FieldLabel>
                <Input
                  id="email"
                  name="email"
                  type="email"
                  autoComplete="email"
                  autoFocus
                  required
                  aria-invalid={error ? true : undefined}
                />
              </Field>
              <FieldError>{error}</FieldError>
              <Button type="submit" size="lg" className="w-full" disabled={pending}>
                {pending ? t("working", { ns: "common" }) : t("forgot.submit")}
              </Button>
            </FieldGroup>
          </form>
        </>
      )}
      <p className="mt-6 text-sm">
        <Link to="/sign-in" className="text-foreground underline-offset-4 hover:underline">
          {t("backToSignIn")}
        </Link>
      </p>
    </AuthShell>
  );
}
