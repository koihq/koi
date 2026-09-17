import { Add01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { useLocation, useNavigate } from "@tanstack/react-router";
import { type ReactNode, useEffect, useLayoutEffect } from "react";
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

export const IMPORTS = {
  mark: "@/components/blocks/koi-mark",
  button: "@/components/ui/button",
  field: "@/components/ui/field",
  input: "@/components/ui/input",
  separator: "@/components/ui/separator",
} as const;

const CATALOG_TO = {
  "/dev/ui": true,
  "/dev/ui/button": true,
  "/dev/ui/field": true,
  "/dev/ui/mark": true,
  "/dev/ui/patterns": true,
  "/dev/ui/separator": true,
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

function isTypingTarget(target: EventTarget | null) {
  if (!(target instanceof HTMLElement)) {
    return false;
  }
  return Boolean(target.closest("input, textarea, select, [contenteditable=true]"));
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
  const navigate = useNavigate();
  const pathname = useLocation({ select: (location) => location.pathname });

  useLayoutEffect(() => {
    applyTheme(variant === "dark");
    return () => {
      document.documentElement.classList.remove("dark");
      document.documentElement.style.colorScheme = "";
    };
  }, [variant]);

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.metaKey || event.ctrlKey || event.altKey || isTypingTarget(event.target)) {
        return;
      }

      let next: Variant | undefined;
      if (event.key === "ArrowRight" || event.key === "ArrowLeft") {
        event.preventDefault();
        const delta = event.key === "ArrowRight" ? 1 : -1;
        next = VARIANTS[(VARIANTS.indexOf(variant) + delta + VARIANTS.length) % VARIANTS.length];
      } else if (event.key === "1") {
        next = "light";
      } else if (event.key === "2") {
        next = "dark";
      }

      if (!next || next === variant) {
        return;
      }

      void navigate({ to: catalogTo(pathname), search: { variant: next }, replace: true });
    }

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [navigate, pathname, variant]);
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
      <p className="mt-2 max-w-lg text-muted-foreground">{lead}</p>
      {children ? <div className="mt-16 flex min-w-0 flex-col gap-16">{children}</div> : null}
    </>
  );
}

export function CatalogSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="min-w-0">
      <h2 className="text-xs/relaxed font-medium text-muted-foreground">{title}</h2>
      <div className="mt-6 min-w-0">{children}</div>
    </section>
  );
}

export function SpecList({ className, children }: { className?: string; children: ReactNode }) {
  return (
    <dl className={cn("grid grid-cols-[auto_1fr] items-baseline gap-x-6 gap-y-2", className)}>
      {children}
    </dl>
  );
}

export function SpecRow({
  label,
  className,
  children,
}: {
  label: ReactNode;
  className?: string;
  children: ReactNode;
}) {
  return (
    <>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className={cn("m-0 min-w-0", className)}>{children}</dd>
    </>
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
    <div className="flex w-full min-w-0 flex-col gap-4">
      {BUTTON_VARIANTS.map((buttonVariant) => (
        <div key={buttonVariant} className="flex w-full min-w-0 flex-wrap items-center gap-3">
          {TEXT_SIZES.map((size) => (
            <Button key={size} type="button" variant={buttonVariant} size={size}>
              {t(buttonVariant === "destructive" ? "delete" : buttonVariant)}
            </Button>
          ))}
          {ICON_SIZES.map((size) => (
            <Button
              key={size}
              type="button"
              variant={buttonVariant}
              size={size}
              aria-label={t("add")}
            >
              <HugeiconsIcon icon={Add01Icon} aria-hidden={true} />
            </Button>
          ))}
        </div>
      ))}
      <div className="flex flex-wrap items-center gap-3">
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
