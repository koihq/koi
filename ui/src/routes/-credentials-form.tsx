import { type FormEvent, type ReactNode, useState } from "react";
import { messageFromUnknown } from "@/api/errors";
import type { CredentialsReq } from "@/api/generated";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";

export function CredentialsForm({
  title,
  submitLabel,
  alt,
  passwordComplete,
  onSubmit,
}: {
  title: string;
  submitLabel: string;
  alt: ReactNode;
  passwordComplete: "current-password" | "new-password";
  onSubmit: (body: CredentialsReq) => Promise<void>;
}) {
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
      <h1 className="text-2xl font-medium tracking-tight">{title}</h1>
      <form className="mt-8" onSubmit={handleSubmit}>
        <FieldGroup>
          <Field data-invalid={error ? true : undefined}>
            <FieldLabel htmlFor="email">Email</FieldLabel>
            <Input
              id="email"
              name="email"
              type="email"
              autoComplete="email"
              required
              className="h-9 text-sm"
              aria-invalid={error ? true : undefined}
            />
          </Field>
          <Field data-invalid={error ? true : undefined}>
            <FieldLabel htmlFor="password">Password</FieldLabel>
            <Input
              id="password"
              name="password"
              type="password"
              autoComplete={passwordComplete}
              required
              minLength={8}
              className="h-9 text-sm"
              aria-invalid={error ? true : undefined}
            />
          </Field>
          <FieldError>{error}</FieldError>
          <Button type="submit" size="lg" className="w-full" disabled={pending}>
            {pending ? "Working…" : submitLabel}
          </Button>
        </FieldGroup>
      </form>
      <p className="mt-6 text-sm text-muted-foreground">{alt}</p>
    </>
  );
}
