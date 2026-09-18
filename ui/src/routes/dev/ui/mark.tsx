import { createFileRoute } from "@tanstack/react-router";

import {
  CatalogCode,
  CatalogPage,
  CatalogPreview,
  CatalogSection,
  MarkOnCanvases,
  validateUiSearch,
} from "./-catalog";

export const Route = createFileRoute("/dev/ui/mark")({
  validateSearch: validateUiSearch,
  component: DevUiMark,
});

function DevUiMark() {
  return (
    <CatalogPage
      title="Koi mark"
      lead="The same transparent PNG appears on light and dark surfaces. Orange is the brand accent."
    >
      <CatalogSection
        title="Preview"
        description="The shipped asset at 24, 32, and 64 pixels on both canvases."
      >
        <CatalogPreview>
          <MarkOnCanvases />
        </CatalogPreview>
      </CatalogSection>
      <CatalogSection title="Use it">
        <div className="grid gap-4">
          <CatalogCode
            title="Import"
            code={'import { KoiMark } from "@/components/blocks/koi-mark";'}
          />
          <CatalogCode title="Usage" code={'<KoiMark className="size-8" />'} />
        </div>
      </CatalogSection>
    </CatalogPage>
  );
}
