import { useQuery } from "@tanstack/react-query";
import { type FormEvent, useId, useState } from "react";
import { useTranslation } from "react-i18next";

import { useApi } from "@/api/apiProvider";
import type { CreateBrand } from "@/api/brands.api";
import { ApiError, messageFromUnknown } from "@/api/errors";
import { CountryCombobox } from "@/components/blocks/brand-form";
import { BRAND_NAME_MAX, brandFormSchema } from "@/components/blocks/brand-form-schema";
import { BrandIcon } from "@/components/blocks/brand-icon";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { brandNameFromSite, countryFromLocale, parseBrandSite } from "./first-brand";

type ConfirmField = "name" | "country";

function fieldFor(name: PropertyKey | undefined): ConfirmField | null {
  return name === "name" || name === "country" ? name : null;
}

/**
 * Shows what the site or repository really says. While the request runs the name shimmers, and it
 * appears when the answer arrives, so the motion reports real work and never adds a delay.
 */
export function FirstBrandForm({
  site,
  onSubmit,
}: {
  site: string;
  onSubmit: (body: CreateBrand) => Promise<void>;
}) {
  const { t } = useTranslation("brand");
  const api = useApi();
  const id = useId();
  const identifiers = parseBrandSite(site);
  const [edited, setEdited] = useState<string | null>(null);
  const [initialCountry] = useState(() => countryFromLocale(navigator.language));
  const [country, setCountry] = useState(initialCountry);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Partial<Record<ConfirmField, string>>>({});

  const lookup = useQuery({
    queryKey: ["brands", "lookup", identifiers.domain, identifiers.githubRepo],
    queryFn: ({ signal }) => api.brands.lookup(identifiers, { signal }),
    retry: false,
    staleTime: Number.POSITIVE_INFINITY,
  });

  const looking = lookup.isPending;
  const found = lookup.data?.found ? lookup.data : null;
  const name = edited ?? found?.name ?? brandNameFromSite(site);

  const messages: Record<ConfirmField, string> = {
    name: t("validation.name.invalid"),
    country: t("validation.country.invalid"),
  };

  function showFieldErrors(form: HTMLFormElement, fields: ConfirmField[]) {
    setFieldErrors(Object.fromEntries(fields.map((field) => [field, messages[field]])));
    requestAnimationFrame(() => form.querySelector<HTMLElement>("[aria-invalid=true]")?.focus());
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;

    setError(null);
    setFieldErrors({});

    const parsed = brandFormSchema.safeParse({
      ...identifiers,
      name,
      country,
      description: lookup.data?.description ?? "",
    });
    if (!parsed.success) {
      const fields = parsed.error.issues.map((issue) => fieldFor(issue.path[0]));
      showFieldErrors(
        form,
        fields.filter((field) => field !== null),
      );

      return;
    }

    setPending(true);

    try {
      await onSubmit(parsed.data);
    } catch (err) {
      const fields = err instanceof ApiError ? (err.fields ?? []) : [];
      const known = fields.map((item) => fieldFor(item.field)).filter((field) => field !== null);

      if (known.length > 0) showFieldErrors(form, known);
      else setError(messageFromUnknown(err));
    } finally {
      setPending(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} noValidate>
      <FieldGroup>
        <Field data-invalid={fieldErrors.name ? true : undefined}>
          <FieldLabel htmlFor={`${id}-name`}>{t("fields.name.label")}</FieldLabel>
          {looking ? (
            <output className="block">
              <span className="sr-only">{t("setup.looking")}</span>
              <Skeleton className="h-9 w-full" />
            </output>
          ) : (
            <div className="flex animate-in items-center gap-3 fade-in duration-200">
              <BrandIcon name={name} domain={identifiers.domain} className="size-9" />
              <Input
                id={`${id}-name`}
                name="name"
                value={name}
                autoComplete="organization"
                maxLength={BRAND_NAME_MAX}
                aria-invalid={fieldErrors.name ? true : undefined}
                aria-describedby={fieldErrors.name ? `${id}-name-error` : `${id}-name-source`}
                onChange={(event) => setEdited(event.currentTarget.value)}
              />
            </div>
          )}
          {found && edited === null && (
            <p
              id={`${id}-name-source`}
              className="animate-in text-xs text-muted-foreground fade-in duration-200"
            >
              {found.source === "github"
                ? t("setup.sourceGithub")
                : t("setup.sourceWebsite", { site: identifiers.domain })}
            </p>
          )}
          <FieldError id={`${id}-name-error`}>{fieldErrors.name}</FieldError>
        </Field>

        <Field data-invalid={fieldErrors.country ? true : undefined}>
          <FieldLabel htmlFor="brand-country">{t("setup.country")}</FieldLabel>
          <CountryCombobox
            initialCode={initialCountry}
            invalid={Boolean(fieldErrors.country)}
            describedBy={fieldErrors.country ? `${id}-country-error` : ""}
            onValueChange={setCountry}
          />
          <FieldError id={`${id}-country-error`}>{fieldErrors.country}</FieldError>
        </Field>

        <FieldError>{error}</FieldError>
        <div>
          <Button type="submit" size="lg" disabled={pending || looking}>
            {pending ? t("working", { ns: "common" }) : t("setup.submit")}
          </Button>
        </div>
      </FieldGroup>
    </form>
  );
}
