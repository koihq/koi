import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import {
  CatalogCode,
  CatalogPage,
  CatalogPreview,
  CatalogSection,
  FieldColumn,
  validateUiSearch,
} from "./-catalog";

export const Route = createFileRoute("/dev/ui/field")({
  validateSearch: validateUiSearch,
  component: DevUiField,
});

function DevUiField() {
  const { t } = useTranslation("dev");

  return (
    <CatalogPage
      title="Field and input"
      lead="Keep the label, input, and error together. FieldGroup supplies spacing between fields."
    >
      <CatalogSection
        title="Preview"
        description="The first field accepts input and shows the normal focus treatment."
      >
        <CatalogPreview>
          <div className="max-w-sm">
            <FieldColumn id="preview" />
          </div>
        </CatalogPreview>
      </CatalogSection>
      <CatalogSection
        title="States"
        description="Empty, filled, invalid, disabled, and long values use the shipped Field and Input components."
      >
        <p className="text-sm text-muted-foreground">{t("fieldStates")}</p>
      </CatalogSection>
      <CatalogSection title="Use it">
        <div className="grid gap-4">
          <CatalogCode
            title="Import"
            code={
              'import { Field, FieldLabel } from "@/components/ui/field";\nimport { Input } from "@/components/ui/input";'
            }
          />
          <CatalogCode
            title="Usage"
            code={
              '<Field>\n  <FieldLabel htmlFor="brand">Brand or domain</FieldLabel>\n  <Input id="brand" placeholder="koi.dev" />\n</Field>'
            }
          />
        </div>
      </CatalogSection>
    </CatalogPage>
  );
}
