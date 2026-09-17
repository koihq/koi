import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { KoiMark } from "@/components/blocks/koi-mark";
import { Button } from "@/components/ui/button";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";

import { CatalogPage, CatalogSection, validateUiSearch } from "./-catalog";

export const Route = createFileRoute("/dev/ui/patterns")({
  validateSearch: validateUiSearch,
  component: DevUiPatterns,
});

function DevUiPatterns() {
  const { t } = useTranslation("dev");
  const { t: auth } = useTranslation("auth");
  const { t: home } = useTranslation("home");

  return (
    <CatalogPage title={t("patterns")} lead={t("patternsLead")}>
      <CatalogSection title={t("definitionList")}>
        <div className="flex min-w-0 flex-col gap-16 min-[640px]:flex-row">
          <dl className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-2">
            <dt className="text-muted-foreground">{t("mention")}</dt>
            <dd className="m-0 tabular-nums">{t("mentionValue")}</dd>
            <dt className="text-muted-foreground">{t("substitute")}</dt>
            <dd className="m-0">{t("substituteValue")}</dd>
            <dt className="text-muted-foreground">{t("citation")}</dt>
            <dd className="m-0 tabular-nums">{t("citationValue")}</dd>
          </dl>
          <div className="flex flex-col gap-2 text-muted-foreground">
            <p className="tabular-nums">{t("run")}</p>
            <p>{t("chatgpt")}</p>
            <p>{t("gemini")}</p>
          </div>
        </div>
      </CatalogSection>
      <CatalogSection title={t("auth")}>
        <div className="flex flex-col items-center py-6">
          <KoiMark className="mb-8" />
          <div className="w-full max-w-sm">
            <p className="text-2xl font-medium tracking-tight">{auth("signIn.title")}</p>
            <div className="mt-8">
              <FieldGroup>
                <Field>
                  <FieldLabel htmlFor="pattern-email">{auth("fields.email")}</FieldLabel>
                  <Input id="pattern-email" type="email" autoComplete="email" />
                </Field>
                <Field>
                  <FieldLabel htmlFor="pattern-password">{auth("fields.password")}</FieldLabel>
                  <Input id="pattern-password" type="password" autoComplete="current-password" />
                </Field>
                <Button type="button" size="lg">
                  {auth("signIn.submit")}
                </Button>
              </FieldGroup>
            </div>
          </div>
        </div>
      </CatalogSection>
      <CatalogSection title={t("home")}>
        <div>
          <div className="flex items-center justify-between gap-4">
            <KoiMark />
            <Button type="button" variant="ghost" size="sm">
              {home("signOut")}
            </Button>
          </div>
          <div className="mt-24 max-w-lg">
            <p className="text-2xl font-medium tracking-tight">{t("foundersEmail")}</p>
            <p className="mt-2 text-muted-foreground">{t("foundersOrg")}</p>
          </div>
        </div>
      </CatalogSection>
    </CatalogPage>
  );
}
