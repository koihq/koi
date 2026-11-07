import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link, useLocation } from "@tanstack/react-router";
import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";

import { useApi } from "@/api/apiProvider";
import { ApiError, messageFromUnknown } from "@/api/errors";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { clearAccountCache } from "@/lib/queries";
import { AuthShell } from "@/routes/-auth-shell";

export const Route = createFileRoute("/reset-password/")({
  component: ResetPassword,
});

function ResetPassword() {
  const { t } = useTranslation("auth");
  const api = useApi();
  const queryClient = useQueryClient();
  // The token travels in the fragment, which browsers do not send to servers or put in logs.
  const token = useLocation({ select: (location) => location.hash });
  const [done, setDone] = useState(false);
  const [linkInvalid, setLinkInvalid] = useState(!token);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const password = String(new FormData(event.currentTarget).get("password") ?? "");
    setError(null);
    setPending(true);

    try {
      await api.auth.resetPassword({ token, password });
      await clearAccountCache(queryClient);
      setDone(true);
    } catch (err) {
      if (err instanceof ApiError && err.code === "password_reset_invalid") {
        setLinkInvalid(true);
      } else {
        setError(messageFromUnknown(err));
      }
    } finally {
      setPending(false);
    }
  }

  if (done) {
    return (
      <AuthShell>
        <h1 className="text-2xl font-semibold tracking-tight">{t("reset.doneTitle")}</h1>
        <p className="mt-4 text-sm text-muted-foreground">{t("reset.doneBody")}</p>
        <Button
          size="lg"
          className="mt-8 w-full"
          nativeButton={false}
          role="link"
          render={<Link to="/sign-in" />}
        >
          {t("signIn.submit")}
        </Button>
      </AuthShell>
    );
  }

  if (linkInvalid) {
    return (
      <AuthShell>
        <h1 className="text-2xl font-semibold tracking-tight">{t("reset.invalidTitle")}</h1>
        <p className="mt-4 text-sm text-muted-foreground">{t("reset.invalidBody")}</p>
        <Button
          size="lg"
          className="mt-8 w-full"
          nativeButton={false}
          role="link"
          render={<Link to="/forgot-password" />}
        >
          {t("reset.requestNew")}
        </Button>
      </AuthShell>
    );
  }

  return (
    <AuthShell>
      <h1 className="text-2xl font-semibold tracking-tight">{t("reset.title")}</h1>
      <form className="mt-8" onSubmit={handleSubmit}>
        <FieldGroup>
          <Field data-invalid={error ? true : undefined}>
            <FieldLabel htmlFor="password">{t("reset.password")}</FieldLabel>
            <Input
              id="password"
              name="password"
              type="password"
              autoComplete="new-password"
              autoFocus
              required
              minLength={12}
              maxLength={256}
              aria-invalid={error ? true : undefined}
            />
          </Field>
          <FieldError>{error}</FieldError>
          <Button type="submit" size="lg" className="w-full" disabled={pending}>
            {pending ? t("working", { ns: "common" }) : t("reset.submit")}
          </Button>
        </FieldGroup>
      </form>
    </AuthShell>
  );
}
