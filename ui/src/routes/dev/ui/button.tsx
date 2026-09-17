import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import {
  ButtonMatrix,
  CatalogPage,
  IMPORTS,
  SpecList,
  SpecRow,
  validateUiSearch,
} from "./-catalog";

export const Route = createFileRoute("/dev/ui/button")({
  validateSearch: validateUiSearch,
  component: DevUiButton,
});

function DevUiButton() {
  const { t } = useTranslation("dev");

  return (
    <CatalogPage title={t("button")} lead={t("buttonLead")}>
      <SpecList>
        <SpecRow label={t("import")}>{IMPORTS.button}</SpecRow>
        <SpecRow label={t("variant")}>{t("buttonApi.variant")}</SpecRow>
        <SpecRow label={t("size")}>{t("buttonApi.size")}</SpecRow>
      </SpecList>
      <ButtonMatrix />
    </CatalogPage>
  );
}
