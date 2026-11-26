import {
  ArrowRight01Icon,
  ArrowUpRight01Icon,
  Copy01Icon,
  PlusSignIcon,
  Tick02Icon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { type ReactNode, useEffect, useRef, useState } from "react";
import { Button, buttonVariants } from "@/components/ui/button";
import { comparePages } from "./alternatives/list";
import mark from "./assets/koi-mark-64.png";
import { agentPrompt, copy, links } from "./copy";
import { analyticsEnabled, CookieSettingsButton } from "./legal/cookie-settings";
import { AgentStack, GithubLogo } from "./logos";
import type { Question } from "./schema";

/** What every page of the site shares. Nothing here may import the demos, charts or pond. */

function SiteMark({ size }: { size: number }) {
  return <img src={mark} alt="" width={size} height={size} className="block" />;
}

export function ExternalIcon() {
  return <HugeiconsIcon icon={ArrowUpRight01Icon} className="size-3.5" aria-hidden="true" />;
}

export function LinkButton({
  href,
  size = "cta",
  children,
}: {
  href: string;
  size?: "cta" | "lg";
  children: string;
}) {
  return (
    <a href={href} data-slot="button" className={buttonVariants({ size })}>
      {children}
    </a>
  );
}

/**
 * The agents the prompt works with, and the button that copies it. If copying fails, the caller shows
 * the prompt itself.
 */
function AgentPill({ onFailure }: { onFailure?: () => void }) {
  const [state, setState] = useState<"idle" | "copied" | "failed">("idle");
  const timer = useRef<number | undefined>(undefined);

  useEffect(() => () => window.clearTimeout(timer.current), []);

  async function copyPrompt() {
    window.clearTimeout(timer.current);

    try {
      await navigator.clipboard.writeText(agentPrompt);
      setState("copied");
    } catch {
      setState("failed");
      onFailure?.();
    }

    timer.current = window.setTimeout(() => setState("idle"), 2600);
  }

  return (
    <div className="site-agent-pill">
      <AgentStack />
      <Button
        size="cta"
        className="h-9 rounded-full px-4"
        onClick={copyPrompt}
        aria-label={state === "idle" ? copy.hero.agentLabel : undefined}
      >
        <span className="site-swap">
          <span data-on={state === "idle"} aria-hidden={state !== "idle"}>
            <HugeiconsIcon icon={Copy01Icon} className="size-4" aria-hidden="true" />
            {copy.hero.agent}
          </span>
          <span data-on={state === "copied"} aria-hidden={state !== "copied"}>
            <HugeiconsIcon icon={Tick02Icon} className="size-4" aria-hidden="true" />
            {copy.hero.copied}
          </span>
          <span data-on={state === "failed"} aria-hidden={state !== "failed"}>
            {copy.hero.copyFailed}
          </span>
        </span>
      </Button>
      <output className="sr-only">{state === "copied" ? copy.hero.copiedStatus : ""}</output>
    </div>
  );
}

function PromptFallback() {
  return (
    <p className="max-w-140 basis-full rounded-control border border-border px-3 py-2 text-left text-xs/relaxed select-all">
      {agentPrompt}
    </p>
  );
}

/** The pill on its own, for a row that has its own secondary link. */
export function AgentPrompt() {
  const [failed, setFailed] = useState(false);

  return (
    <>
      <AgentPill onFailure={() => setFailed(true)} />
      {failed && <PromptFallback />}
    </>
  );
}

export function Actions({ align = "center" }: { align?: "center" | "start" }) {
  const [failed, setFailed] = useState(false);

  return (
    <div className={`flex flex-col gap-3 ${align === "center" ? "items-center" : "items-start"}`}>
      <AgentPill onFailure={() => setFailed(true)} />
      <a
        href={links.app}
        className="inline-flex items-center gap-1 rounded-control px-2 py-2 text-body text-muted-foreground transition-colors hover:text-foreground"
      >
        {copy.hero.browser}
        <HugeiconsIcon icon={ArrowRight01Icon} className="size-4" aria-hidden="true" />
      </a>
      {failed && <PromptFallback />}
    </div>
  );
}

function Header() {
  return (
    <header className="site-rise absolute inset-x-0 top-0 z-20 mx-auto flex h-16 w-full max-w-340 items-center justify-between gap-4 px-4 sm:px-7">
      <a href="/" aria-label={copy.nav.home} className="flex items-center rounded-control">
        <SiteMark size={28} />
      </a>

      <nav className="flex items-center gap-1">
        <a
          href="/#pricing"
          className="hidden rounded-control px-3 py-2 text-body text-muted-foreground transition-colors hover:text-foreground sm:block"
        >
          {copy.nav.pricing}
        </a>
        <a
          href={links.docs}
          className="hidden rounded-control px-3 py-2 text-body text-muted-foreground transition-colors hover:text-foreground sm:block"
        >
          {copy.nav.docs}
        </a>
        <a
          href={links.github}
          className="hidden items-center gap-2 rounded-control px-3 py-2 text-body text-muted-foreground transition-colors hover:text-foreground sm:flex"
        >
          <GithubLogo />
          {copy.nav.github}
        </a>
        <a
          href={links.signIn}
          className="rounded-control px-3 py-2 text-body text-muted-foreground transition-colors hover:text-foreground"
        >
          {copy.nav.signIn}
        </a>
        <LinkButton href={links.app} size="lg">
          {copy.nav.start}
        </LinkButton>
      </nav>
    </header>
  );
}

/** The questions a page answers, and where to ask another one. The page ships them as FAQPage data. */
export function Faq({ items }: { items: Question[] }) {
  return (
    <section className="mx-auto grid max-w-340 gap-10 px-4 pb-32 sm:px-7 md:grid-cols-[minmax(0,4fr)_minmax(0,8fr)]">
      <div>
        <h2 className="site-heading">{copy.faq.title}</h2>
        <p className="mt-3 text-body text-muted-foreground">
          {`${copy.faq.more} `}
          <a
            href={links.issues}
            className="text-foreground underline underline-offset-4 transition-colors hover:text-muted-foreground"
          >
            {copy.faq.ask}
          </a>
        </p>
      </div>
      <div className="border-t border-border">
        {items.map((item, index) => (
          <details
            key={item.q}
            name="faq"
            open={index === 0}
            className="site-question border-b border-border"
          >
            <summary className="flex cursor-pointer list-none items-center justify-between gap-6 rounded-control py-5 text-section [&::-webkit-details-marker]:hidden">
              {item.q}
              <HugeiconsIcon
                icon={PlusSignIcon}
                className="site-question-icon size-4 shrink-0 text-muted-foreground"
                aria-hidden="true"
              />
            </summary>
            <p className="max-w-150 pe-10 pb-6 text-body font-normal text-muted-foreground">
              {item.a}
            </p>
          </details>
        ))}
      </div>
    </section>
  );
}

function FooterColumn({ title, children }: { title: string; children: ReactNode }) {
  return (
    <nav aria-label={title}>
      <p aria-hidden="true">{title}</p>
      <ul className="mt-3 grid justify-items-start gap-0.5 text-muted-foreground">{children}</ul>
    </nav>
  );
}

const footerLinkClass = "inline-block py-1 transition-colors hover:text-foreground";

function FooterLink({ href, children }: { href: string; children: string }) {
  return (
    <li>
      <a href={href} className={footerLinkClass}>
        {children}
      </a>
    </li>
  );
}

/** The comparisons are reached from here: the page header keeps to the product. */
function Footer() {
  const { footer } = copy;

  return (
    <footer className="mx-auto grid max-w-340 gap-x-10 gap-y-12 px-4 pt-12 pb-12 text-body sm:px-7 md:grid-cols-[minmax(0,4fr)_minmax(0,8fr)]">
      <div className="flex flex-col items-start justify-between gap-6 max-md:order-last">
        <a href="/" aria-label={copy.nav.home} className="rounded-control">
          <SiteMark size={28} />
        </a>
        <p className="text-xs text-muted-foreground">{footer.owner}</p>
      </div>

      <div className="grid grid-cols-2 gap-x-6 gap-y-10 lg:grid-cols-4">
        <FooterColumn title={footer.product}>
          <FooterLink href="/#pricing">{copy.nav.pricing}</FooterLink>
          <FooterLink href={links.app}>{copy.nav.start}</FooterLink>
          <FooterLink href={links.signIn}>{copy.nav.signIn}</FooterLink>
          <FooterLink href={links.skill}>{footer.skill}</FooterLink>
        </FooterColumn>

        <FooterColumn title={footer.compare}>
          {comparePages.map((page) => (
            <FooterLink key={page.path} href={page.path}>
              {page.name}
            </FooterLink>
          ))}
        </FooterColumn>

        <FooterColumn title={footer.openSource}>
          <FooterLink href={links.github}>{footer.source}</FooterLink>
          <FooterLink href={links.docs}>{copy.nav.docs}</FooterLink>
          <FooterLink href={links.selfHost}>{footer.selfHost}</FooterLink>
          <FooterLink href={links.mcp}>{footer.mcp}</FooterLink>
        </FooterColumn>

        <FooterColumn title={footer.legal}>
          <FooterLink href={links.privacy}>{footer.privacy}</FooterLink>
          <FooterLink href={links.terms}>{footer.terms}</FooterLink>
          {analyticsEnabled && (
            <li>
              <CookieSettingsButton className={footerLinkClass} />
            </li>
          )}
        </FooterColumn>
      </div>
    </footer>
  );
}

/** The skip link, header and footer every page of the site shares. */
export function SiteLayout({ children }: { children: ReactNode }) {
  return (
    <>
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:absolute focus:start-4 focus:top-4 focus:z-30 focus:rounded-control focus:bg-surface focus:px-3 focus:py-2 focus:text-body"
      >
        {copy.skipLink}
      </a>
      <Header />
      <main id="main">{children}</main>
      <Footer />
    </>
  );
}
