import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { CatalogPage, FieldColumn, IMPORTS, SpecList, SpecRow, validateUiSearch } from "./-catalog";

export const Route = createFileRoute("/dev/ui/field")({
  validateSearch: validateUiSearch,
  component: DevUiField,
});

function DevUiField() {
  const { t } = useTranslation("dev");

  return (
    <CatalogPage title={t("field")} lead={t("fieldLead")}>
      <SpecList>
        <SpecRow label={t("import")}>{IMPORTS.field}</SpecRow>
        <SpecRow label={t("input")}>{IMPORTS.input}</SpecRow>
      </SpecList>
      <div className="flex w-full min-w-0 flex-wrap gap-10">
        <div className="w-[320px] max-w-full">
          <FieldColumn id="narrow" />
        </div>
        <div className="min-w-0 flex-1 basis-[min(100%,320px)]">
          <FieldColumn id="wide" />
        </div>
      </div>
    </CatalogPage>
  );
}
