import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import {
  CatalogPage,
  IMPORTS,
  MarkOnCanvases,
  SpecList,
  SpecRow,
  validateUiSearch,
} from "./-catalog";

export const Route = createFileRoute("/dev/ui/mark")({
  validateSearch: validateUiSearch,
  component: DevUiMark,
});

function DevUiMark() {
  const { t } = useTranslation("dev");

  return (
    <CatalogPage title={t("mark")} lead={t("markLead")}>
      <SpecList>
        <SpecRow label={t("import")}>{IMPORTS.mark}</SpecRow>
      </SpecList>
      <MarkOnCanvases />
    </CatalogPage>
  );
}
