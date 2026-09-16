import { Add01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { createFileRoute, Link, redirect, useNavigate } from "@tanstack/react-router";
import { useEffect, useLayoutEffect } from "react";
import { useTranslation } from "react-i18next";

import { KoiMark } from "@/components/blocks/koi-mark";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";

const VARIANTS = ["light", "dark"] as const;
type Variant = (typeof VARIANTS)[number];

const BUTTON_VARIANTS = [
  "default",
  "outline",
  "secondary",
  "ghost",
  "destructive",
  "link",
] as const;
const TEXT_SIZES = ["xs", "sm", "default", "lg"] as const;
const ICON_SIZES = ["icon-xs", "icon-sm", "icon", "icon-lg"] as const;
const MARK_SIZES = ["size-6", "size-8", "size-16"] as const;

const LONG_VALUE =
  "https://answerengines.example/prompts/best-aeo-tool-for-founders-without-spaces";

const PICKER_CSS = `.variant-picker {
  position: fixed;
  bottom: 24px;
  left: 50%;
  translate: -50% 0;
  z-index: 2147483647;
  display: flex;
  gap: 2px;
  padding: 4px;
  border-radius: 999px;
  background: rgb(20 20 20 / 0.9);
  box-shadow: inset 0 0 0 1px rgb(255 255 255 / 0.1), 0 8px 24px rgb(0 0 0 / 0.25);
  font: 13px/1 -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  user-select: none;
}

.variant-picker a {
  padding: 7px 14px;
  border: 0;
  border-radius: 999px;
  background: none;
  color: rgb(255 255 255 / 0.6);
  cursor: pointer;
  text-decoration: none;
}

.variant-picker a:visited {
  color: rgb(255 255 255 / 0.6);
}

.variant-picker a:hover {
  color: rgb(255 255 255 / 0.85);
}

.variant-picker a[aria-current="true"],
.variant-picker a[aria-current="true"]:visited {
  background: rgb(255 255 255 / 0.14);
  color: rgb(255 255 255);
}

.variant-picker a:focus-visible {
  outline: 2px solid rgb(255 255 255 / 0.7);
  outline-offset: 2px;
}`;

function parseVariant(value: unknown): Variant {
  return value === "dark" ? "dark" : "light";
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

export const Route = createFileRoute("/dev/ui")({
  validateSearch: (search: Record<string, unknown>): { variant: Variant } => ({
    variant: parseVariant(search.variant),
  }),
  beforeLoad: () => {
    if (!import.meta.env.DEV) {
      throw redirect({ to: "/" });
    }
  },
  component: DevUi,
});

function DevUi() {
  const { t } = useTranslation("dev");
  const { variant } = Route.useSearch();
  const navigate = useNavigate({ from: "/dev/ui" });

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

      void navigate({ search: { variant: next }, replace: true });
    }

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [navigate, variant]);

  return (
    <>
      <style>{PICKER_CSS}</style>
      <main className="min-h-svh w-full min-w-0 max-w-[100vw] overflow-x-clip px-6 pt-10 pb-24">
        <h1 className="text-2xl font-medium tracking-tight">{t("title")}</h1>
        <div className="mt-16 flex w-full min-w-0 flex-col gap-16">
          <section className="min-w-0">
            <h2 className="text-xs/relaxed font-medium text-muted-foreground">{t("mark")}</h2>
            <div className="mt-6 flex w-full min-w-0 flex-wrap gap-4">
              <div
                className="flex shrink-0 items-end gap-4 p-6"
                style={{ background: "oklch(1 0 0)" }}
              >
                {MARK_SIZES.map((size) => (
                  <KoiMark key={`light-${size}`} className={size} />
                ))}
              </div>
              <div
                className="flex shrink-0 items-end gap-4 p-6"
                style={{ background: "oklch(0.145 0 0)" }}
              >
                {MARK_SIZES.map((size) => (
                  <KoiMark key={`dark-${size}`} className={size} />
                ))}
              </div>
            </div>
          </section>
          <section className="min-w-0">
            <h2 className="text-xs/relaxed font-medium text-muted-foreground">{t("button")}</h2>
            <div className="mt-6 flex w-full min-w-0 flex-col gap-4">
              {BUTTON_VARIANTS.map((buttonVariant) => (
                <div
                  key={buttonVariant}
                  className="flex w-full min-w-0 flex-wrap items-center gap-3"
                >
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
                      <HugeiconsIcon icon={Add01Icon} aria-hidden />
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
          </section>
          <section className="min-w-0">
            <h2 className="text-xs/relaxed font-medium text-muted-foreground">{t("field")}</h2>
            <div className="mt-6 flex w-full min-w-0 flex-wrap gap-10">
              <div className="w-[320px] max-w-full">
                <FieldColumn id="narrow" />
              </div>
              <div className="min-w-0 flex-1 basis-[min(100%,320px)]">
                <FieldColumn id="wide" />
              </div>
            </div>
          </section>
          <section className="min-w-0">
            <h2 className="text-xs/relaxed font-medium text-muted-foreground">
              {t("composition")}
            </h2>
            <div className="mt-6 flex min-w-0 flex-col gap-16 min-[640px]:flex-row">
              <dl className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-2">
                <dt className="text-muted-foreground">{t("mention")}</dt>
                <dd className="m-0 tabular-nums">{t("mentionValue")}</dd>
                <dt className="text-muted-foreground">{t("substitute")}</dt>
                <dd className="m-0">{t("substituteValue")}</dd>
                <dt className="text-muted-foreground">{t("citation")}</dt>
                <dd className="m-0 tabular-nums">{t("citationValue")}</dd>
              </dl>
              <div className="flex flex-col gap-2 text-muted-foreground">
                <p className="tabular-nums">{t("run")}</p>
                <p>{t("chatgpt")}</p>
                <p>{t("gemini")}</p>
              </div>
            </div>
          </section>
        </div>
      </main>
      <nav className="variant-picker" aria-label={t("variants")}>
        {VARIANTS.map((item) => (
          <Link
            key={item}
            to="/dev/ui"
            search={{ variant: item }}
            replace
            aria-current={variant === item ? "true" : undefined}
          >
            {t(item)}
          </Link>
        ))}
      </nav>
    </>
  );
}

function FieldColumn({ id }: { id: string }) {
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
