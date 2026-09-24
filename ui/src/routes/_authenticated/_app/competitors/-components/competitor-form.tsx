import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";

import type { Competitor, CreateCompetitor } from "@/api/competitors.api";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { competitorFormSchema } from "@/routes/_authenticated/_app/competitors/-types/competitor-form-schema";

export function CompetitorForm({
  initial,
  pending,
  onSave,
  onCancel,
}: {
  initial?: Competitor;
  pending: boolean;
  onSave: (request: CreateCompetitor) => Promise<void>;
  onCancel: () => void;
}) {
  const { t } = useTranslation("management");
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
    } catch {
      setFailed(true);
    }
  }

  return (
    <form onSubmit={submit} className="space-y-4">
      <FieldGroup className="sm:grid sm:grid-cols-2">
        <Field data-invalid={invalidFields.has("name") || undefined}>
          <FieldLabel htmlFor="competitor-name">{t("name")}</FieldLabel>
          <Input
            id="competitor-name"
            autoFocus
            value={name}
            onChange={(event) => setName(event.target.value)}
            aria-invalid={invalidFields.has("name") || undefined}
          />
          {invalidFields.has("name") ? <FieldError>{t("nameInvalid")}</FieldError> : null}
        </Field>
        <Field data-invalid={invalidFields.has("domain") || undefined}>
          <FieldLabel htmlFor="competitor-domain">{t("domain")}</FieldLabel>
          <Input
            id="competitor-domain"
            value={domain}
            onChange={(event) => setDomain(event.target.value)}
            aria-invalid={invalidFields.has("domain") || undefined}
            placeholder={t("domainPlaceholder")}
          />
          {invalidFields.has("domain") ? <FieldError>{t("domainInvalid")}</FieldError> : null}
        </Field>
        <Field className="sm:col-span-2">
          <FieldLabel htmlFor="competitor-aliases">{t("aliases")}</FieldLabel>
          <Input
            id="competitor-aliases"
            value={aliases}
            onChange={(event) => setAliases(event.target.value)}
            placeholder={t("aliasesPlaceholder")}
          />
        </Field>
      </FieldGroup>
      {failed ? <FieldError>{t("competitorSaveError")}</FieldError> : null}
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          {t("cancel")}
        </Button>
        <Button type="submit" disabled={pending}>
          {t("saveCompetitor")}
        </Button>
      </div>
    </form>
  );
}
