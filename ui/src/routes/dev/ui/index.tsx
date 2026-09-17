import { createFileRoute, Link, useSearch } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { KoiMark } from "@/components/blocks/koi-mark";
import { Button } from "@/components/ui/button";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

import { CatalogPage, CatalogSection, SpecList, SpecRow, validateUiSearch } from "./-catalog";

const TOKEN_ROWS = [
  { label: "token.canvas", name: "token.canvasName", swatch: "bg-background" },
  { label: "token.quiet", name: "token.quietName", swatch: "bg-muted" },
  { label: "token.controls", name: "token.controlsName", swatch: "bg-primary" },
  { label: "token.brand", name: "token.brandName", swatch: "bg-brand" },
  { label: "token.danger", name: "token.dangerName", swatch: "bg-destructive" },
  { label: "token.edges", name: "token.edgesName", swatch: "bg-border" },
  { label: "token.radius", name: "token.radiusName" },
  { label: "token.easeOut", name: "token.easeOutName" },
  { label: "token.easeInOut", name: "token.easeInOutName" },
] as const;

export const Route = createFileRoute("/dev/ui/")({
  validateSearch: validateUiSearch,
  component: DevUiOverview,
});

function DevUiOverview() {
  const { t } = useTranslation("dev");
  const { variant } = useSearch({ from: "/dev/ui" });

  return (
    <CatalogPage title={t("title")} lead={t("overviewLead")}>
      <CatalogSection title={t("tokens")}>
        <SpecList>
          {TOKEN_ROWS.map((row) => (
            <SpecRow key={row.label} label={t(row.label)} className="flex items-center">
              {"swatch" in row ? (
                <span
                  className={cn("size-6 shrink-0 rounded-sm border border-border", row.swatch)}
                  aria-hidden={true}
                />
              ) : null}
              <span className={"swatch" in row ? "ms-3" : undefined}>{t(row.name)}</span>
            </SpecRow>
          ))}
        </SpecList>
      </CatalogSection>
      <CatalogSection title={t("type")}>
        <SpecList className="gap-y-4">
          <SpecRow label={t("typeRole.title")}>
            <span className="text-2xl font-medium tracking-tight">{t("typeRole.titleSample")}</span>
          </SpecRow>
          <SpecRow label={t("typeRole.body")}>{t("typeRole.bodySample")}</SpecRow>
          <SpecRow label={t("typeRole.supporting")}>
            <span className="text-muted-foreground">{t("typeRole.supportingSample")}</span>
          </SpecRow>
          <SpecRow label={t("typeRole.control")}>
            <span className="text-xs/relaxed font-medium">{t("typeRole.controlSample")}</span>
          </SpecRow>
          <SpecRow label={t("typeRole.digits")}>
            <span className="tabular-nums">{t("typeRole.digitsSample")}</span>
          </SpecRow>
        </SpecList>
      </CatalogSection>
      <CatalogSection title={t("primitives")}>
        <SpecList className="items-center gap-y-8">
          <SpecRow
            label={
              <Link
                to="/dev/ui/mark"
                search={{ variant }}
                className="text-foreground underline-offset-4 hover:underline"
              >
                {t("mark")}
              </Link>
            }
          >
            <KoiMark />
          </SpecRow>
          <SpecRow
            label={
              <Link
                to="/dev/ui/button"
                search={{ variant }}
                className="text-foreground underline-offset-4 hover:underline"
              >
                {t("button")}
              </Link>
            }
          >
            <div className="flex flex-wrap items-center gap-3">
              <Button type="button">{t("default")}</Button>
              <Button type="button" variant="outline">
                {t("outline")}
              </Button>
              <Button type="button" variant="ghost">
                {t("ghost")}
              </Button>
            </div>
          </SpecRow>
          <SpecRow
            label={
              <Link
                to="/dev/ui/field"
                search={{ variant }}
                className="text-foreground underline-offset-4 hover:underline"
              >
                {t("field")}
              </Link>
            }
          >
            <div className="w-[320px] max-w-full">
              <FieldGroup>
                <Field>
                  <FieldLabel htmlFor="overview-field">{t("empty")}</FieldLabel>
                  <Input id="overview-field" placeholder={t("placeholder")} />
                </Field>
              </FieldGroup>
            </div>
          </SpecRow>
        </SpecList>
      </CatalogSection>
    </CatalogPage>
  );
}
