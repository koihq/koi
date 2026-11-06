import { type FormEvent, useId, useState } from "react";
import { useTranslation } from "react-i18next";

import { validBrandSite } from "@/components/blocks/brand-form-schema";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";

export function WebsiteStep({
  initial,
  onSubmit,
}: {
  initial: string;
  onSubmit: (site: string) => void;
}) {
  const { t } = useTranslation("brand");
  const id = useId();
  const [invalid, setInvalid] = useState(false);

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const site = String(new FormData(event.currentTarget).get("site") ?? "").trim();
    if (!validBrandSite(site)) {
      setInvalid(true);

      return;
    }

    onSubmit(site);
  }

  return (
    <form onSubmit={handleSubmit} noValidate className="grid justify-items-start gap-6">
      <Field data-invalid={invalid ? true : undefined}>
        <FieldLabel htmlFor={`${id}-site`} className="sr-only">
          {t("setup.site")}
        </FieldLabel>
        <Input
          id={`${id}-site`}
          name="site"
          defaultValue={initial}
          placeholder={t("setup.sitePlaceholder")}
          autoCapitalize="none"
          spellCheck={false}
          autoFocus
          aria-invalid={invalid ? true : undefined}
          aria-describedby={invalid ? `${id}-site-error` : undefined}
          onChange={() => setInvalid(false)}
        />
        <FieldError id={`${id}-site-error`}>{invalid ? t("setup.siteInvalid") : null}</FieldError>
      </Field>
      <Button type="submit" size="lg">
        {t("setup.websiteContinue")}
      </Button>
    </form>
  );
}
