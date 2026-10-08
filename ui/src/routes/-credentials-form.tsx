import { Link } from "@tanstack/react-router";
import { type FormEvent, useState } from "react";
import { Trans, useTranslation } from "react-i18next";

import { messageFromUnknown } from "@/api/errors";
import type { CredentialsReq } from "@/api/generated";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";

export function CredentialsForm({
  intent,
  passwordComplete,
  onSubmit,
}: {
  intent: "sign-in" | "sign-up";
  passwordComplete: "current-password" | "new-password";
  onSubmit: (body: CredentialsReq) => Promise<void>;
}) {
  const { t } = useTranslation("auth");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    const email = String(data.get("email") ?? "");
    const password = String(data.get("password") ?? "");
    setError(null);
    setPending(true);

    try {
      await onSubmit({ email, password });
    } catch (err) {
      setError(messageFromUnknown(err));
    } finally {
      setPending(false);
    }
  }

  return (
    <>
      <h1 className="text-2xl font-medium tracking-tight">
        {intent === "sign-in" ? t("signIn.title") : t("signUp.title")}
      </h1>
      <form className="mt-8" onSubmit={handleSubmit}>
        <FieldGroup>
          <Field data-invalid={error ? true : undefined}>
            <FieldLabel htmlFor="email">{t("fields.email")}</FieldLabel>
            <Input
              id="email"
              name="email"
              type="email"
              autoComplete="email"
              required
              aria-invalid={error ? true : undefined}
            />
          </Field>
          <Field data-invalid={error ? true : undefined}>
            <FieldLabel htmlFor="password">{t("fields.password")}</FieldLabel>
            <Input
              id="password"
              name="password"
              type="password"
              autoComplete={passwordComplete}
              required
              minLength={8}
              aria-invalid={error ? true : undefined}
            />
          </Field>
          <FieldError>{error}</FieldError>
          <Button type="submit" size="lg" className="w-full" disabled={pending}>
            {pending
              ? t("working", { ns: "common" })
              : intent === "sign-in"
                ? t("signIn.submit")
                : t("signUp.submit")}
          </Button>
        </FieldGroup>
      </form>
      <p className="mt-6 text-sm text-muted-foreground">
        <Trans
          t={t}
          i18nKey={intent === "sign-in" ? "signIn.alt" : "signUp.alt"}
          components={{
            action: (
              <Link
                to={intent === "sign-in" ? "/sign-up" : "/sign-in"}
                className="text-foreground underline-offset-4 hover:underline"
              />
            ),
          }}
        />
      </p>
    </>
  );
}
