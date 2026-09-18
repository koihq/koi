import { createFileRoute, Link, useSearch } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import styles from "@/styles.css?raw";
import { CatalogPage, CatalogSection, validateUiSearch } from "./-catalog";

const colorTokens = [
  "background",
  "foreground",
  "primary",
  "primary-foreground",
  "secondary",
  "muted",
  "muted-foreground",
  "accent",
  "destructive",
  "border",
  "input",
  "ring",
  "sidebar",
  "sidebar-accent",
  "sidebar-selected",
  "brand",
] as const;

const spacingSteps = [
  { step: 1, className: "w-1" },
  { step: 2, className: "w-2" },
  { step: 3, className: "w-3" },
  { step: 4, className: "w-4" },
  { step: 6, className: "w-6" },
  { step: 8, className: "w-8" },
] as const;
const radiusClasses = [
  "rounded-sm",
  "rounded-md",
  "rounded-lg",
  "rounded-xl",
  "rounded-2xl",
  "rounded-3xl",
  "rounded-4xl",
] as const;

function declarations(selector: string) {
  const start = styles.indexOf(`${selector} {`);
  if (start < 0) return {} as Record<string, string>;
  const block = styles.slice(start, styles.indexOf("}", start));
  return Object.fromEntries(
    [...block.matchAll(/--([\w-]+):\s*([^;]+);/g)].map((match) => [match[1], match[2].trim()]),
  );
}

const light = declarations(":root");
const dark = declarations(".dark");

export const Route = createFileRoute("/dev/ui/")({
  validateSearch: validateUiSearch,
  component: DevUiOverview,
});

function StyleMeasurement({
  className,
  property,
}: {
  className: string;
  property: "width" | "borderTopLeftRadius" | "boxShadow";
}) {
  const { variant } = useSearch({ from: "/dev/ui" });
  const ref = useRef<HTMLDivElement>(null);
  const [value, setValue] = useState("");
  useEffect(() => {
    if (ref.current && document.documentElement.classList.contains("dark") === (variant === "dark")) {
      setValue(getComputedStyle(ref.current)[property]);
    }
  }, [property, variant]);
  return (
    <div className="flex min-w-0 items-center gap-4">
      <div ref={ref} className={className} aria-hidden="true" />
      <span className="min-w-0 wrap-anywhere text-xs/relaxed tabular-nums text-muted-foreground">
        {value}
      </span>
    </div>
  );
}

function DevUiOverview() {
  const { t } = useTranslation("dev");
  const { variant } = useSearch({ from: "/dev/ui" });
  const components = [
    { to: "/dev/ui/button", name: "Button", detail: "Actions and their variants" },
    { to: "/dev/ui/field", name: "Field and input", detail: "Labels, values, and errors" },
    { to: "/dev/ui/separator", name: "Separator", detail: "A structural split" },
    { to: "/dev/ui/mark", name: "Koi mark", detail: "The shipped PNG asset" },
    { to: "/dev/ui/sidebar", name: "Sidebar", detail: "The product navigation block" },
  ] as const;

  return (
    <CatalogPage
      title="Design system"
      lead="The tokens and components used in Koi. Values below come from the current stylesheet and rendered utilities."
    >
      <CatalogSection
        title="Components"
        description="Open a component for live states and copyable usage."
      >
        <div className="grid gap-x-8 gap-y-4 sm:grid-cols-2">
          {components.map((item) => (
            <Link
              key={item.to}
              to={item.to}
              search={{ variant }}
              className="group rounded-sm py-1 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground"
            >
              <span className="text-sm font-medium group-hover:underline">{item.name}</span>
              <span className="mt-1 block text-xs/relaxed text-muted-foreground">
                {item.detail}
              </span>
            </Link>
          ))}
        </div>
      </CatalogSection>
      <CatalogSection
        title="Color"
        description="Semantic tokens in both themes. The mark keeps the same brand color in each."
      >
        <div className="grid gap-4 lg:grid-cols-2">
          {(["light", "dark"] as const).map((mode) => {
            const values = mode === "light" ? light : { ...light, ...dark };
            return (
              <div
                key={mode}
                className="min-w-0 overflow-hidden rounded-xl border"
                style={{
                  backgroundColor: values.background,
                  color: values.foreground,
                  borderColor: values.border,
                }}
              >
                <div
                  className="border-b px-4 py-3 text-sm font-medium capitalize"
                  style={{ borderColor: values.border }}
                >
                  {mode}
                </div>
                <div>
                  {colorTokens.map((token) => (
                    <div
                      key={token}
                      className="flex min-w-0 items-center gap-3 border-b px-4 py-2.5 last:border-0"
                      style={{ borderColor: values.border }}
                    >
                      <span
                        className="size-8 shrink-0 rounded-md border"
                        style={{ backgroundColor: values[token], borderColor: values.border }}
                        aria-hidden="true"
                      />
                      <div className="min-w-0">
                        <div className="text-xs/relaxed font-medium">{`--${token}`}</div>
                        <div
                          className="wrap-anywhere text-xs/relaxed"
                          style={{ color: values["muted-foreground"] }}
                        >
                          {values[token]}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            );
          })}
        </div>
      </CatalogSection>
      <CatalogSection
        title="Spacing"
        description="Rendered from Tailwind's current spacing unit. These steps appear in controls, fields, and page layout."
      >
        <div className="grid gap-4 sm:grid-cols-2">
          {spacingSteps.map(({ step, className }) => (
            <div
              key={step}
              className="grid grid-cols-[4.5rem_minmax(0,1fr)] items-center gap-4 text-sm"
            >
              <span>{`space-${step}`}</span>
              <StyleMeasurement
                className={`h-5 shrink-0 bg-primary ${className}`}
                property="width"
              />
            </div>
          ))}
        </div>
      </CatalogSection>
      <CatalogSection
        title="Radius"
        description="The scale derives from --radius in the stylesheet; components use these utility classes."
      >
        <div className="grid gap-5 sm:grid-cols-2">
          {radiusClasses.map((radius) => (
            <div
              key={radius}
              className="grid grid-cols-[6rem_minmax(0,1fr)] items-center gap-4 text-sm"
            >
              <span>{radius}</span>
              <StyleMeasurement
                className={`h-10 w-20 shrink-0 border border-border bg-muted ${radius}`}
                property="borderTopLeftRadius"
              />
            </div>
          ))}
        </div>
      </CatalogSection>
      <CatalogSection
        title="Typography"
        description="Inter Variable is the only typeface used in the product."
      >
        <div className="grid gap-6">
          <div>
            <div className="mb-2 text-xs/relaxed text-muted-foreground">{t("typeTitle")}</div>
            <p className="text-2xl font-medium tracking-tight">{t("typeTitleSample")}</p>
          </div>
          <div>
            <div className="mb-2 text-xs/relaxed text-muted-foreground">{t("typeBody")}</div>
            <p>{t("typeBodySample")}</p>
          </div>
          <div>
            <div className="mb-2 text-xs/relaxed text-muted-foreground">{t("typeControl")}</div>
            <p className="text-xs/relaxed font-medium">{t("typeControlSample")}</p>
          </div>
          <div>
            <div className="mb-2 text-xs/relaxed text-muted-foreground">{t("typeSupporting")}</div>
            <p className="text-sm text-muted-foreground">{t("typeSupportingSample")}</p>
          </div>
        </div>
      </CatalogSection>
      <CatalogSection
        title="Elevation"
        description="Shadows appear only where a surface overlaps another surface."
      >
        <div className="grid gap-8 sm:grid-cols-2">
          <div>
            <p className="mb-4 text-sm font-medium">{t("shadowSidebar")}</p>
            <StyleMeasurement
              className="h-16 w-24 shrink-0 rounded-lg bg-popover shadow-sm"
              property="boxShadow"
            />
          </div>
          <div>
            <p className="mb-4 text-sm font-medium">{t("shadowSheet")}</p>
            <StyleMeasurement
              className="h-16 w-24 shrink-0 rounded-lg bg-popover shadow-lg"
              property="boxShadow"
            />
          </div>
        </div>
      </CatalogSection>
    </CatalogPage>
  );
}
