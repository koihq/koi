import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import {
  ButtonMatrix,
  CatalogCode,
  CatalogPage,
  CatalogPreview,
  CatalogSection,
  validateUiSearch,
} from "./-catalog";

export const Route = createFileRoute("/dev/ui/button")({
  validateSearch: validateUiSearch,
  component: DevUiButton,
});

function DevUiButton() {
  const { t } = useTranslation("dev");

  return (
    <CatalogPage
      title="Button"
      lead="Use a label that names the action. The primitive supplies six variants and eight sizes."
    >
      <CatalogSection
        title="Preview"
        description="Hover, focus, and press the buttons to inspect their feedback."
      >
        <CatalogPreview className="flex flex-wrap items-center gap-3">
          <Button type="button">{t("add")}</Button>
          <Button type="button" variant="outline">
            {t("edit")}
          </Button>
          <Button type="button" variant="destructive">
            {t("delete")}
          </Button>
          <Button type="button" disabled>
            {t("disabled")}
          </Button>
        </CatalogPreview>
      </CatalogSection>
      <CatalogSection
        title="Variants and sizes"
        description="Text and icon sizes are shown for each supported variant. Disabled and busy are native states."
      >
        <ButtonMatrix />
      </CatalogSection>
      <CatalogSection title="Use it">
        <div className="grid gap-4">
          <CatalogCode title="Import" code={'import { Button } from "@/components/ui/button";'} />
          <CatalogCode
            title="Usage"
            code={'<Button type="button" variant="outline" size="sm">Edit</Button>'}
          />
        </div>
      </CatalogSection>
    </CatalogPage>
  );
}
