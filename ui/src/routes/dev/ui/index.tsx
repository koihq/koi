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
  { label: "token.quiet", name: "token.quietName" },
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
      <div className="grid min-w-0 gap-10 min-[1000px]:grid-cols-2">
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
              <span className="text-2xl font-medium tracking-tight">
                {t("typeRole.titleSample")}
              </span>
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
      </div>
      <CatalogSection title={t("primitives")}>
        <div className="grid min-w-0 gap-6 min-[760px]:grid-cols-3">
          <div className="flex min-w-0 flex-col gap-4">
            <Link
              to="/dev/ui/mark"
              search={{ variant }}
              className="w-fit rounded-sm text-xs/relaxed font-medium text-foreground underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground"
            >
              {t("mark")}
            </Link>
            <KoiMark />
          </div>
          <div className="flex min-w-0 flex-col gap-4">
            <Link
              to="/dev/ui/button"
              search={{ variant }}
              className="w-fit rounded-sm text-xs/relaxed font-medium text-foreground underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground"
            >
              {t("button")}
            </Link>
            <div className="flex flex-wrap items-center gap-3">
              <Button type="button">{t("add")}</Button>
              <Button type="button" variant="outline">
                {t("add")}
              </Button>
              <Button type="button" variant="ghost">
                {t("add")}
              </Button>
            </div>
          </div>
          <div className="flex min-w-0 flex-col gap-4">
            <Link
              to="/dev/ui/field"
              search={{ variant }}
              className="w-fit rounded-sm text-xs/relaxed font-medium text-foreground underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground"
            >
              {t("field")}
            </Link>
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="overview-field">{t("placeholder")}</FieldLabel>
                <Input id="overview-field" placeholder="koi.dev" />
              </Field>
            </FieldGroup>
          </div>
        </div>
      </CatalogSection>
    </CatalogPage>
  );
}
