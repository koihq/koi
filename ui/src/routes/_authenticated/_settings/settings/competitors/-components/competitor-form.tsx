import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";

import type { Competitor, CreateCompetitor } from "@/api/competitors.api";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { competitorFormSchema } from "@/routes/_authenticated/_settings/settings/competitors/-utils/competitor-form-schema";

export function CompetitorForm({
  initial,
  pending,
  onSave,
  onCancel,
}: {
  initial?: Competitor;
  pending: boolean;
  onSave: (request: CreateCompetitor) => Promise<void>;
  onCancel?: () => void;
}) {
  const { t } = useTranslation("brand");
  const [name, setName] = useState(initial?.name ?? "");
  const [domain, setDomain] = useState(initial?.domain ?? "");
  const [aliases, setAliases] = useState(initial?.aliases.join(", ") ?? "");
  const [invalidFields, setInvalidFields] = useState<Set<string>>(new Set());
  const [failed, setFailed] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const result = competitorFormSchema.safeParse({ name, domain, aliases });
    if (!result.success) {
      setInvalidFields(new Set(result.error.issues.map((issue) => String(issue.path[0]))));
      return;
    }
    setInvalidFields(new Set());
    setFailed(false);
    try {
      await onSave(result.data);
      if (!initial) {
        setName("");
        setDomain("");
        setAliases("");
      }
    } catch {
      setFailed(true);
    }
  }

  return (
    <form onSubmit={submit}>
      <FieldGroup className="sm:grid sm:grid-cols-2">
        <Field data-invalid={invalidFields.has("name") || undefined}>
          <FieldLabel htmlFor={`competitor-name-${initial?.id ?? "add"}`}>
            {t("lists.competitors.name")}
          </FieldLabel>
          <Input
            id={`competitor-name-${initial?.id ?? "add"}`}
            value={name}
            onChange={(event) => setName(event.target.value)}
            aria-invalid={invalidFields.has("name") || undefined}
          />
          {invalidFields.has("name") ? (
            <FieldError>{t("lists.competitors.nameInvalid")}</FieldError>
          ) : null}
        </Field>
        <Field data-invalid={invalidFields.has("domain") || undefined}>
          <FieldLabel htmlFor={`competitor-domain-${initial?.id ?? "add"}`}>
            {t("lists.competitors.domain")}
          </FieldLabel>
          <Input
            id={`competitor-domain-${initial?.id ?? "add"}`}
            value={domain}
            onChange={(event) => setDomain(event.target.value)}
            aria-invalid={invalidFields.has("domain") || undefined}
            placeholder={t("lists.competitors.domainPlaceholder")}
          />
          {invalidFields.has("domain") ? (
            <FieldError>{t("lists.competitors.domainInvalid")}</FieldError>
          ) : null}
        </Field>
        <Field className="sm:col-span-2">
          <FieldLabel htmlFor={`competitor-aliases-${initial?.id ?? "add"}`}>
            {t("lists.competitors.aliases")}
          </FieldLabel>
          <Input
            id={`competitor-aliases-${initial?.id ?? "add"}`}
            value={aliases}
            onChange={(event) => setAliases(event.target.value)}
            placeholder={t("lists.competitors.aliasesPlaceholder")}
          />
        </Field>
      </FieldGroup>
      <div className="mt-3 flex gap-2">
        {onCancel ? (
          <Button type="button" variant="ghost" onClick={onCancel}>
            {t("lists.cancel")}
          </Button>
        ) : null}
        <Button type="submit" disabled={pending}>
          {initial ? t("lists.save") : t("lists.competitors.add")}
        </Button>
      </div>
      {failed ? <FieldError className="mt-2">{t("lists.saveError")}</FieldError> : null}
    </form>
  );
}
