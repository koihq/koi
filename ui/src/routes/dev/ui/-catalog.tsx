import { Add01Icon, Delete01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { type ReactNode, useLayoutEffect, useState } from "react";
import { useTranslation } from "react-i18next";

import { KoiMark } from "@/components/blocks/koi-mark";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

export const VARIANTS = ["light", "dark"] as const;

export type Variant = (typeof VARIANTS)[number];

export const BUTTON_VARIANTS = [
  "default",
  "outline",
  "secondary",
  "ghost",
  "destructive",
  "link",
] as const;

export const TEXT_SIZES = ["xs", "sm", "default", "lg"] as const;
export const ICON_SIZES = ["icon-xs", "icon-sm", "icon", "icon-lg"] as const;
export const MARK_SIZES = ["size-6", "size-8", "size-16"] as const;

const CATALOG_TO = {
  "/dev/ui": true,
  "/dev/ui/button": true,
  "/dev/ui/field": true,
  "/dev/ui/mark": true,
  "/dev/ui/separator": true,
  "/dev/ui/sidebar": true,
} as const;

export type CatalogTo = keyof typeof CATALOG_TO;

export function catalogTo(pathname: string): CatalogTo {
  const path = pathname.endsWith("/") && pathname !== "/" ? pathname.slice(0, -1) : pathname;

  return path in CATALOG_TO ? (path as CatalogTo) : "/dev/ui";
}

const LONG_VALUE =
  "https://answerengines.example/prompts/best-aeo-tool-for-founders-without-spaces";

export function parseVariant(value: unknown): Variant {
  return value === "dark" ? "dark" : "light";
}

export function validateUiSearch(search: Record<string, unknown>): { variant: Variant } {
  return { variant: parseVariant(search.variant) };
}

function applyTheme(dark: boolean) {
  const style = document.createElement("style");
  style.append(document.createTextNode("*,*::before,*::after{transition:none !important}"));
  document.head.append(style);
  document.documentElement.classList.toggle("dark", dark);
  document.documentElement.style.colorScheme = dark ? "dark" : "light";
  void document.body.offsetHeight;
  requestAnimationFrame(() => {
    requestAnimationFrame(() => style.remove());
  });
}

export function useCatalogTheme(variant: Variant) {
  useLayoutEffect(() => {
    applyTheme(variant === "dark");

    return () => {
      document.documentElement.classList.remove("dark");
      document.documentElement.style.colorScheme = "";
    };
  }, [variant]);
}

export function CatalogPage({
  title,
  lead,
  children,
}: {
  title: string;
  lead: string;
  children?: ReactNode;
}) {
  return (
    <>
      <h1 className="text-2xl font-medium tracking-tight">{title}</h1>
      <p className="mt-2 max-w-2xl text-sm leading-relaxed text-muted-foreground text-pretty">
        {lead}
      </p>
      {children ? <div className="mt-10 flex min-w-0 flex-col gap-12">{children}</div> : null}
    </>
  );
}

export function CatalogSection({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children: ReactNode;
}) {
  return (
    <section className="min-w-0">
      <h2 className="text-base font-medium text-foreground">{title}</h2>
      {description ? (
        <p className="mt-1 max-w-2xl text-sm text-muted-foreground">{description}</p>
      ) : null}
      <div className="mt-5 min-w-0">{children}</div>
    </section>
  );
}

export function CatalogPreview({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn("min-w-0 rounded-xl border border-border bg-background p-6 sm:p-8", className)}
    >
      {children}
    </div>
  );
}

export function CatalogCode({ title, code }: { title: string; code: string }) {
  const { t } = useTranslation("dev");
  const [copied, setCopied] = useState(false);

  return (
    <div className="min-w-0 overflow-hidden rounded-lg border border-border">
      <div className="flex items-center justify-between gap-4 border-b border-border px-4 py-2">
        <span className="text-xs/relaxed font-medium">{title}</span>
        <Button
          type="button"
          variant="ghost"
          size="xs"
          onClick={async () => {
            try {
              await navigator.clipboard.writeText(code);
              setCopied(true);
            } catch {
              setCopied(false);
            }
          }}
          aria-label={`Copy ${title.toLowerCase()}`}
        >
          {copied ? t("copied") : t("copy")}
        </Button>
      </div>
      <pre className="overflow-x-auto bg-muted/40 p-4 text-xs/relaxed text-foreground">
        <code>{code}</code>
      </pre>
    </div>
  );
}

export function MarkOnCanvases() {
  return (
    <div className="flex w-full min-w-0 flex-wrap gap-4">
      <div className="flex shrink-0 items-end gap-4 bg-[oklch(1_0_0)] p-6">
        {MARK_SIZES.map((size) => (
          <KoiMark key={`light-${size}`} className={size} />
        ))}
      </div>
      <div className="flex shrink-0 items-end gap-4 bg-[oklch(0.145_0_0)] p-6">
        {MARK_SIZES.map((size) => (
          <KoiMark key={`dark-${size}`} className={size} />
        ))}
      </div>
    </div>
  );
}

export function ButtonMatrix() {
  const { t } = useTranslation("dev");

  return (
    <div className="flex w-full min-w-0 flex-col gap-6">
      {BUTTON_VARIANTS.map((buttonVariant) => (
        <section
          key={buttonVariant}
          className="grid min-w-0 gap-3 min-[640px]:grid-cols-[6rem_minmax(0,1fr)]"
        >
          <h2 className="pt-1 text-xs/relaxed font-medium">{t(buttonVariant)}</h2>
          <div className="grid min-w-0 gap-4 min-[980px]:grid-cols-[max-content_max-content] min-[980px]:gap-10">
            <div className="min-w-0">
              <p className="mb-2 text-xs/relaxed text-muted-foreground">{t("textSizes")}</p>
              <div className="flex flex-wrap items-end gap-3">
                {TEXT_SIZES.map((size) => (
                  <div key={size} className="flex flex-col items-start gap-1">
                    <span className="text-[0.625rem] text-muted-foreground">{size}</span>
                    <Button type="button" variant={buttonVariant} size={size}>
                      {t(buttonVariant === "destructive" ? "delete" : "add")}
                    </Button>
                  </div>
                ))}
              </div>
            </div>
            <div className="min-w-0">
              <p className="mb-2 text-xs/relaxed text-muted-foreground">{t("iconSizes")}</p>
              <div className="flex flex-wrap items-end gap-3">
                {ICON_SIZES.map((size) => (
                  <div key={size} className="flex flex-col items-start gap-1">
                    <span className="text-[0.625rem] text-muted-foreground">{size}</span>
                    <Button
                      type="button"
                      variant={buttonVariant}
                      size={size}
                      aria-label={t(buttonVariant === "destructive" ? "delete" : "add")}
                    >
                      <HugeiconsIcon
                        icon={buttonVariant === "destructive" ? Delete01Icon : Add01Icon}
                        aria-hidden={true}
                      />
                    </Button>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>
      ))}
      <div className="flex flex-wrap items-center gap-3 pt-2">
        <Button type="button" disabled>
          {t("disabled")}
        </Button>
        <Button type="button" size="lg" disabled aria-busy>
          {t("working", { ns: "common" })}
        </Button>
      </div>
    </div>
  );
}

export function FieldColumn({ id }: { id: string }) {
  const { t } = useTranslation("dev");
  const emptyId = `${id}-empty`;
  const filledId = `${id}-filled`;
  const invalidId = `${id}-invalid`;
  const invalidErrorId = `${id}-invalid-error`;
  const disabledId = `${id}-disabled`;
  const longId = `${id}-long`;

  return (
    <FieldGroup>
      <Field>
        <FieldLabel htmlFor={emptyId}>{t("empty")}</FieldLabel>
        <Input id={emptyId} placeholder={t("placeholder")} />
      </Field>
      <Field>
        <FieldLabel htmlFor={filledId}>{t("filled")}</FieldLabel>
        <Input id={filledId} defaultValue="koi.dev" />
      </Field>
      <Field data-invalid={true}>
        <FieldLabel htmlFor={invalidId}>{t("invalid")}</FieldLabel>
        <Input
          id={invalidId}
          defaultValue="not-an-email"
          aria-invalid
          aria-describedby={invalidErrorId}
        />
        <FieldError id={invalidErrorId}>{t("invalidError")}</FieldError>
      </Field>
      <Field data-disabled={true}>
        <FieldLabel htmlFor={disabledId}>{t("disabled")}</FieldLabel>
        <Input id={disabledId} defaultValue="koi.dev" disabled />
      </Field>
      <Field>
        <FieldLabel htmlFor={longId}>{t("long")}</FieldLabel>
        <Input id={longId} defaultValue={LONG_VALUE} />
      </Field>
    </FieldGroup>
  );
}
