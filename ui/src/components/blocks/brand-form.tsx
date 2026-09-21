import { useQuery } from "@tanstack/react-query";
import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";

import { useApi } from "@/api/apiProvider";
import { ApiError, messageFromUnknown } from "@/api/errors";
import type { BrandRequest, BrandRes, CountryOption } from "@/api/generated";
import {
  type BrandField,
  brandFormFields,
  brandFormSchema,
} from "@/components/blocks/brand-form-schema";
import { Button } from "@/components/ui/button";
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { fieldErrorsFromZod } from "@/lib/utils";

export function BrandForm({
  brand,
  submitLabel,
  onSubmit,
}: {
  brand?: BrandRes;
  submitLabel: string;
  onSubmit: (body: BrandRequest) => Promise<void>;
}) {
  const { t } = useTranslation("brand");
  const [country, setCountry] = useState(brand?.country ?? "");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Partial<Record<BrandField, string>>>({});

  function validationMessage(field: BrandField): string {
    return t(`validation.${field}.invalid`);
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    setError(null);
    setFieldErrors({});

    const parsed = brandFormSchema.safeParse({
      name: String(data.get("name") ?? ""),
      domain: String(data.get("domain") ?? ""),
      githubRepo: String(data.get("github_repo") ?? ""),
      country,
      description: String(data.get("description") ?? ""),
    });
    if (!parsed.success) {
      setFieldErrors(fieldErrorsFromZod(parsed.error.issues, brandFormFields, validationMessage));
      requestAnimationFrame(() => form.querySelector<HTMLElement>("[aria-invalid=true]")?.focus());
      return;
    }

    setPending(true);

    try {
      await onSubmit(parsed.data);
    } catch (err) {
      if (err instanceof ApiError && err.fields?.length) {
        const next: Partial<Record<BrandField, string>> = {};
        for (const item of err.fields) {
          if (
            item.field === "name" ||
            item.field === "domain" ||
            item.field === "github_repo" ||
            item.field === "country" ||
            item.field === "description"
          ) {
            next[item.field] = validationMessage(item.field);
          }
        }
        if (Object.keys(next).length > 0) {
          setFieldErrors(next);
          requestAnimationFrame(() =>
            form.querySelector<HTMLElement>("[aria-invalid=true]")?.focus(),
          );
        } else {
          setError(messageFromUnknown(err));
        }
      } else {
        setError(messageFromUnknown(err));
      }
    } finally {
      setPending(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} noValidate>
      <FieldGroup>
        <Field data-invalid={fieldErrors.name ? true : undefined}>
          <FieldLabel htmlFor="brand-name">{t("fields.name.label")}</FieldLabel>
          <Input
            id="brand-name"
            name="name"
            defaultValue={brand?.name}
            autoComplete="organization"
            required
            aria-invalid={fieldErrors.name ? true : undefined}
            aria-describedby={fieldErrors.name ? "brand-name-error" : undefined}
          />
          <FieldError id="brand-name-error">{fieldErrors.name}</FieldError>
        </Field>

        <Field data-invalid={fieldErrors.domain ? true : undefined}>
          <FieldLabel htmlFor="brand-domain">{t("fields.domain.label")}</FieldLabel>
          <Input
            id="brand-domain"
            name="domain"
            defaultValue={brand?.domain}
            placeholder={t("fields.domain.placeholder")}
            autoCapitalize="none"
            spellCheck={false}
            aria-invalid={fieldErrors.domain ? true : undefined}
            aria-describedby={fieldErrors.domain ? "brand-domain-error" : "brand-identifier-help"}
          />
          <FieldError id="brand-domain-error">{fieldErrors.domain}</FieldError>
        </Field>

        <Field data-invalid={fieldErrors.github_repo ? true : undefined}>
          <FieldLabel htmlFor="brand-github">{t("fields.githubRepo.label")}</FieldLabel>
          <Input
            id="brand-github"
            name="github_repo"
            type="url"
            defaultValue={brand?.githubRepo}
            placeholder={t("fields.githubRepo.placeholder")}
            autoCapitalize="none"
            spellCheck={false}
            aria-invalid={fieldErrors.github_repo ? true : undefined}
            aria-describedby={
              fieldErrors.github_repo ? "brand-github-error" : "brand-identifier-help"
            }
          />
          <FieldDescription id="brand-identifier-help">
            {t("fields.identifierHelp")}
          </FieldDescription>
          <FieldError id="brand-github-error">{fieldErrors.github_repo}</FieldError>
        </Field>

        <Field data-invalid={fieldErrors.country ? true : undefined}>
          <FieldLabel htmlFor="brand-country">{t("fields.country.label")}</FieldLabel>
          <CountryCombobox
            initialCode={brand?.country}
            invalid={Boolean(fieldErrors.country)}
            describedBy={
              fieldErrors.country ? "brand-country-help brand-country-error" : "brand-country-help"
            }
            onValueChange={setCountry}
          />
          <FieldDescription id="brand-country-help">
            {t("fields.country.description")}
          </FieldDescription>
          <FieldError id="brand-country-error">{fieldErrors.country}</FieldError>
        </Field>

        <Field data-invalid={fieldErrors.description ? true : undefined}>
          <FieldLabel htmlFor="brand-description">{t("fields.description.label")}</FieldLabel>
          <Input
            id="brand-description"
            name="description"
            defaultValue={brand?.description}
            placeholder={t("fields.description.placeholder")}
            aria-invalid={fieldErrors.description ? true : undefined}
            aria-describedby={
              fieldErrors.description
                ? "brand-description-help brand-description-error"
                : "brand-description-help"
            }
          />
          <FieldDescription id="brand-description-help">
            {t("fields.description.description")}
          </FieldDescription>
          <FieldError id="brand-description-error">{fieldErrors.description}</FieldError>
        </Field>

        <FieldError>{error}</FieldError>
        <div>
          <Button type="submit" disabled={pending}>
            {pending ? t("working", { ns: "common" }) : submitLabel}
          </Button>
        </div>
      </FieldGroup>
    </form>
  );
}

function countryLabel(country: CountryOption): string {
  return `${country.code} (${country.name})`;
}

const countryNames = new Intl.DisplayNames(["en"], { type: "region" });

function countryFromCode(code: string): CountryOption | null {
  const normalized = code.trim().toUpperCase();
  if (!/^[A-Z]{2}$/.test(normalized)) return null;
  return { code: normalized, name: countryNames.of(normalized) ?? normalized };
}

function CountryCombobox({
  initialCode = "",
  invalid,
  describedBy,
  onValueChange,
}: {
  initialCode?: string;
  invalid: boolean;
  describedBy: string;
  onValueChange: (code: string) => void;
}) {
  const { t } = useTranslation("brand");
  const api = useApi();
  const initialCountry = countryFromCode(initialCode);
  const [query, setQuery] = useState(initialCountry?.code ?? "");
  const [inputValue, setInputValue] = useState(() =>
    initialCountry ? countryLabel(initialCountry) : "",
  );
  const [selected, setSelected] = useState<CountryOption | null>(initialCountry);
  const trimmed = query.trim();
  const debounced = useDebouncedValue(trimmed);
  const search = trimmed ? debounced : "";
  const countries = useQuery({
    queryKey: ["brands", "countries", search],
    queryFn: ({ signal }) => api.brands.listBrandCountries(search || undefined, { signal }),
    staleTime: Number.POSITIVE_INFINITY,
    gcTime: Number.POSITIVE_INFINITY,
  });
  const options = countries.data ?? [];

  let emptyMessage = t("fields.country.noResults");
  if (countries.isPending || countries.isFetching || trimmed !== search) {
    emptyMessage = t("fields.country.searching");
  }
  if (countries.isError) emptyMessage = t("fields.country.searchError");

  return (
    <Combobox
      name="country"
      items={options}
      filter={null}
      value={selected}
      inputValue={inputValue}
      itemToStringLabel={countryLabel}
      itemToStringValue={(country) => country.code}
      isItemEqualToValue={(country, value) => country.code === value.code}
      onInputValueChange={(next, details) => {
        setInputValue(next);
        if (details.reason === "input-change" || details.reason === "input-clear") {
          setQuery(next);
          setSelected(null);
          onValueChange("");
        }
      }}
      onValueChange={(country) => {
        setSelected(country);
        onValueChange(country?.code ?? "");
        if (country) {
          setInputValue(countryLabel(country));
          setQuery(country.code);
        }
      }}
    >
      <ComboboxInput
        id="brand-country"
        className="w-full"
        placeholder={t("fields.country.placeholder")}
        aria-invalid={invalid || undefined}
        aria-describedby={describedBy}
        aria-busy={countries.isFetching}
        showClear={Boolean(inputValue)}
      />
      <ComboboxContent>
        <ComboboxEmpty>{emptyMessage}</ComboboxEmpty>
        <ComboboxList>
          {options.map((country) => (
            <ComboboxItem key={country.code} value={country}>
              {countryLabel(country)}
            </ComboboxItem>
          ))}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  );
}
