import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { Separator } from "@/components/ui/separator";
import {
  CatalogCode,
  CatalogPage,
  CatalogPreview,
  CatalogSection,
  validateUiSearch,
} from "./-catalog";

export const Route = createFileRoute("/dev/ui/separator")({
  validateSearch: validateUiSearch,
  component: DevUiSeparator,
});

function DevUiSeparator() {
  const { t } = useTranslation("dev");

  return (
    <CatalogPage
      title="Separator"
      lead="Use one line for a structural split. Space groups fields, headings, and rows."
    >
      <CatalogSection
        title="Preview"
        description="Horizontal and vertical orientations are supported."
      >
        <CatalogPreview>
          <div className="flex items-center gap-5 text-sm">
            <span>{t("navigation")}</span>
            <Separator orientation="vertical" className="h-6" />
            <span>{t("content")}</span>
          </div>
          <div className="mt-6 max-w-sm">
            <p className="mb-4 text-sm">{t("header")}</p>
            <Separator />
            <p className="mt-4 text-sm">{t("body")}</p>
          </div>
        </CatalogPreview>
      </CatalogSection>
      <CatalogSection title="Use it">
        <div className="grid gap-4">
          <CatalogCode
            title="Import"
            code={'import { Separator } from "@/components/ui/separator";'}
          />
          <CatalogCode title="Usage" code={'<Separator orientation="vertical" />'} />
        </div>
      </CatalogSection>
    </CatalogPage>
  );
}
