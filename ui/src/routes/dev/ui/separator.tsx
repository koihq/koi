import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { CatalogPage, IMPORTS, SpecList, SpecRow, validateUiSearch } from "./-catalog";

export const Route = createFileRoute("/dev/ui/separator")({
  validateSearch: validateUiSearch,
  component: DevUiSeparator,
});

function DevUiSeparator() {
  const { t } = useTranslation("dev");

  return (
    <CatalogPage title={t("separator")} lead={t("separatorLead")}>
      <SpecList>
        <SpecRow label={t("import")}>{IMPORTS.separator}</SpecRow>
      </SpecList>
      <p className="max-w-lg text-muted-foreground">{t("separatorNote")}</p>
    </CatalogPage>
  );
}
