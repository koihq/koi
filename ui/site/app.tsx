import {
  ArrowRight01Icon,
  ArrowUpRight01Icon,
  Copy01Icon,
  PlusSignIcon,
  Tick02Icon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import NumberFlow from "@number-flow/react";
import { Fragment, useEffect, useRef, useState } from "react";
import { BrandLabel } from "@/components/blocks/brand-label";
import { TickPicker } from "@/components/blocks/tick-picker";
import { Button, buttonVariants } from "@/components/ui/button";
import { formatRate } from "@/lib/format";
import catalog from "../../pkg/pricing/catalog.json";
import mark from "./assets/koi-mark-64.png";
import { agentPrompt, copy, engineNames, engineShortNames, links } from "./copy";
import { FixDemo } from "./demos/fix-demo";
import { OutcomeDot } from "./demos/parts";
import { SetupDemo } from "./demos/setup-demo";
import { AgentStack, EngineChip, EngineLogo, engineList, GithubLogo } from "./logos";
import { Pond } from "./pond/ponds";
import recorded from "./summary/summary-data.json";
import { SummaryMock } from "./summary/summary-mock";

/** The summary is laid out for this width and scaled to fit, so the hero shows its wide layout. */
const APP_WIDTH = 1320;
const APP_HEIGHT = 1000;

function SiteMark({ size }: { size: number }) {
  return <img src={mark} alt="" width={size} height={size} className="block" />;
}

function ExternalIcon() {
  return <HugeiconsIcon icon={ArrowUpRight01Icon} className="size-3.5" aria-hidden="true" />;
}

function LinkButton({
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

function PrimaryLink({ children }: { children: string }) {
  return <LinkButton href={links.app}>{children}</LinkButton>;
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
function AgentPrompt() {
  const [failed, setFailed] = useState(false);

  return (
    <>
      <AgentPill onFailure={() => setFailed(true)} />
      {failed && <PromptFallback />}
    </>
  );
}

function Actions({ align = "center" }: { align?: "center" | "start" }) {
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

/**
 * The lead, with each engine as its mark and short name. Google's two share the G, so the name says
 * which one. Each chip's label is the full name, and the separators are there for screen readers.
 */
function HeroLead() {
  const { lead } = copy.hero;

  return (
    <p className="site-lead mt-6 max-w-160 text-muted-foreground">
      {`${lead.before} `}
      <span className="site-lead-engines">
        {engineList.map((engine, index) => (
          <Fragment key={engine}>
            <EngineChip engine={engine} />
            {index < engineList.length - 1 && " "}
            {index < engineList.length - 2 && <span className="sr-only">{", "}</span>}
            {index === engineList.length - 2 && <span className="sr-only">{` ${lead.and} `}</span>}
          </Fragment>
        ))}
      </span>
      {` ${lead.after}`}
    </p>
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
          href="#pricing"
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

/**
 * The summary page in one frame, drawn at desktop width and scaled to the column. On a phone it is
 * drawn at the phone's width instead, the way the app itself lays out there.
 */
function ProductFrame() {
  const frame = useRef<HTMLDivElement>(null);
  const [scale, setScale] = useState(1);
  const [wide, setWide] = useState(true);

  useEffect(() => {
    const element = frame.current;

    if (!element) {
      return;
    }

    const observer = new ResizeObserver(([entry]) => {
      const width = entry.contentRect.width;

      setWide(width >= 720);
      setScale(Math.min(1, width / APP_WIDTH));
    });

    observer.observe(element);

    return () => observer.disconnect();
  }, []);

  return (
    <figure className="mt-14 sm:mt-16">
      <div
        ref={frame}
        role="img"
        aria-label={copy.hero.demoTitle}
        className="site-app-frame site-product"
        style={{ height: wide ? APP_HEIGHT * scale : 640 }}
      >
        <div
          className="site-app-canvas"
          style={wide ? { width: APP_WIDTH, transform: `scale(${scale})` } : undefined}
        >
          <SummaryMock />
        </div>
      </div>
    </figure>
  );
}

function Hero() {
  return (
    <section className="mx-auto max-w-340 px-4 pt-32 pb-28 text-center sm:px-7 sm:pt-36">
      <div className="flex flex-col items-center">
        <h1
          className="site-hero site-rise max-w-220"
          style={{ "--rise": 1 } as React.CSSProperties}
        >
          {copy.hero.title}
        </h1>
        <div
          className="site-rise flex flex-col items-center"
          style={{ "--rise": 2 } as React.CSSProperties}
        >
          <HeroLead />
        </div>
        <div className="site-rise mt-8" style={{ "--rise": 3 } as React.CSSProperties}>
          <Actions />
        </div>
      </div>
      <div className="site-fade text-left" style={{ "--rise": 5 } as React.CSSProperties}>
        <ProductFrame />
      </div>
    </section>
  );
}

/** Recorded prompts the brand is named in least, each with the competitor named most instead. */
const passedOver = [...recorded.prompts]
  .sort((left, right) => (left.visibility.metric.rate ?? 0) - (right.visibility.metric.rate ?? 0))
  .slice(0, 3)
  .map((prompt) => ({
    question: prompt.text,
    rival: [...prompt.visibility.competitors].sort(
      (left, right) => (right.metric.rate ?? 0) - (left.metric.rate ?? 0),
    )[0],
  }));

const tileSpans = {
  one: "",
  wide: "md:col-span-2",
  row: "md:col-span-2 xl:col-span-1",
};

/** A bento tile: run-in copy at the top, its picture below. Every tile shares radius, gap and padding. */
function Tile({
  title,
  body,
  span = "one",
  aside,
  children,
}: {
  title: string;
  body: string;
  /** wide: two thirds from xl and the full row below it. row: one third from xl, the full row below. */
  span?: "one" | "wide" | "row";
  aside?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <article className={`site-tile ${tileSpans[span]}`}>
      <div className="flex flex-wrap items-center justify-between gap-x-8 gap-y-5 p-6 sm:p-7">
        <h3 className="site-tile-copy max-w-110">
          {title} <span className="text-muted-foreground">{body}</span>
        </h3>
        {aside}
      </div>
      {children}
    </article>
  );
}

/**
 * ChatGPT and Gemini as tiles, and Google's two answer surfaces in one wide card, because they
 * share the same mark.
 */
function EnginesPicture() {
  return (
    <ul
      aria-label={engineList.map((engine) => engineNames[engine]).join(", ")}
      className="mx-auto my-auto grid w-fit grid-cols-2 gap-3 px-6 pb-8"
    >
      <li className="site-engine-mark">
        <EngineLogo engine="chatgpt" size={36} decorative />
      </li>
      <li className="site-engine-mark">
        <EngineLogo engine="gemini" size={36} decorative />
      </li>
      <li className="site-engine-mark site-engine-family col-span-2">
        <EngineLogo engine="google" size={36} decorative />
        <span aria-hidden="true" className="grid gap-0.5">
          <span>{engineShortNames.googleAiMode}</span>
          <span>{engineShortNames.google}</span>
        </span>
      </li>
    </ul>
  );
}

/** Answers that share the brand's name: only the one about the product counts as a mention. */
function NamesakePicture() {
  const { namesake } = copy.bento;

  return (
    <ul aria-label={namesake.label} className="site-product mt-auto grid gap-4 px-6 pb-7 sm:px-7">
      {namesake.answers.map((answer) => (
        <li key={answer.text} className="grid gap-1.5">
          <p
            className={`text-sm/relaxed font-normal ${answer.counted ? "text-foreground" : "text-muted-foreground"}`}
          >
            {answer.text}
          </p>
          <span className={`text-xs ${answer.counted ? "text-positive" : "text-muted-foreground"}`}>
            {answer.verdict}
          </span>
        </li>
      ))}
    </ul>
  );
}

/** For the questions where engines pass the brand over, the competitor they name most. */
function RivalsPicture() {
  return (
    <ul className="site-product mt-auto grid gap-5 px-6 pb-7 sm:px-7">
      {passedOver.map(({ question, rival }) => (
        <li key={question} className="grid gap-2">
          <p className="line-clamp-2 text-body font-normal text-muted-foreground">{question}</p>
          <BrandLabel
            size="sm"
            name={rival.name}
            domain={rival.domain}
            detail={formatRate(rival.metric.rate ?? 0, "en")}
          />
        </li>
      ))}
    </ul>
  );
}

/** One saved answer, as the app shows it: the engine, the question, the answer and its sources. */
function EvidencePicture() {
  const { evidence } = copy.bento;

  return (
    <div className="site-product koi-demo flex-1 ps-6 sm:ps-7">
      <div className="site-evidence">
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <EngineLogo engine="chatgpt" size={16} decorative />
          <span>{evidence.engine}</span>
        </div>
        <p className="mt-2 text-sm font-semibold">{evidence.question}</p>
        <p className="mt-3 max-w-150 text-sm/relaxed font-normal">
          {`${evidence.before} `}
          <mark className="site-evidence-brand">{evidence.brand}</mark>
          {` ${evidence.after}`}
        </p>
        <div className="mt-4 flex flex-wrap items-center gap-2">
          <span className="me-2 inline-flex items-center gap-2 text-xs">
            <OutcomeDot outcome="cited" size={10} />
            {evidence.outcome}
          </span>
          {evidence.sources.map((source) => (
            <span key={source} className="site-source">
              {source}
            </span>
          ))}
        </div>
      </div>
    </div>
  );
}

function Bento() {
  const { bento } = copy;

  return (
    <section className="mx-auto max-w-340 px-4 pb-32 sm:px-7">
      <h2 className="site-heading max-w-180">{bento.title}</h2>
      <p className="site-lead mt-4 max-w-160 text-muted-foreground">{bento.lead}</p>

      <div className="mt-10 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <Tile
          span="wide"
          title={bento.setup.title}
          body={bento.setup.body}
          aside={
            <div className="flex flex-wrap items-center gap-3">
              <AgentPrompt />
              <a
                href={links.skill}
                className="inline-flex items-center gap-1 rounded-control px-2 py-2 text-body text-muted-foreground transition-colors hover:text-foreground"
              >
                {bento.setup.readSkill}
                <ExternalIcon />
              </a>
            </div>
          }
        >
          <div
            role="img"
            aria-label={bento.setup.demo}
            className="site-product mt-auto min-w-0 ps-6 max-sm:pe-6 sm:ps-7"
          >
            <SetupDemo />
          </div>
        </Tile>

        <Tile title={bento.daily.title} body={bento.daily.body}>
          <EnginesPicture />
        </Tile>

        <Tile title={bento.namesake.title} body={bento.namesake.body}>
          <NamesakePicture />
        </Tile>

        <Tile span="wide" title={bento.fix.title} body={bento.fix.body}>
          <div
            role="img"
            aria-label={bento.fix.demo}
            className="site-product mt-auto min-w-0 px-6 pb-6 sm:px-7 sm:pb-7"
          >
            <FixDemo />
          </div>
        </Tile>

        <Tile span="wide" title={bento.evidence.title} body={bento.evidence.body}>
          <EvidencePicture />
        </Tile>

        <Tile span="row" title={bento.rivals.title} body={bento.rivals.body}>
          <RivalsPicture />
        </Tile>
      </div>
    </section>
  );
}

const prompts = catalog.options.map((option) => option.prompts);
const priceOf = (amount: number) =>
  catalog.options.find((option) => option.prompts === amount)?.euros ?? 0;

function Pricing() {
  const [amount, setAmount] = useState(50);
  const euros = priceOf(amount);
  const panel = useRef<HTMLDivElement>(null);

  return (
    <section id="pricing" className="scroll-mt-8 pb-32">
      <div className="mx-auto max-w-340 px-4 sm:px-7">
        <h2 className="site-display max-w-240">
          {`${copy.pricing.title} `}
          <span className="text-muted-foreground">{copy.pricing.titleMuted}</span>
        </h2>
      </div>

      <div className="relative mt-12 overflow-hidden max-lg:pb-140">
        <Pond count={amount} avoid={panel} />
        <div className="relative mx-auto max-w-340 px-4 py-12 sm:px-7 lg:py-20">
          <div
            ref={panel}
            className="w-full max-w-110 rounded-panel border border-border bg-background p-6 sm:p-7"
          >
            <p className="site-heading tabular-nums">
              <NumberFlow value={amount} locales="en" />
              {` ${copy.pricing.unit}. `}
              <NumberFlow value={amount} locales="en" />
              {` ${copy.pricing.koiKoi}`}
            </p>
            <div className="mt-6">
              <TickPicker
                compact
                options={prompts}
                value={amount}
                onChange={setAmount}
                unit={copy.pricing.unit}
                label={copy.pricing.pickerLabel}
                locale="en"
              />
            </div>
            <div className="mt-6 flex flex-wrap items-end justify-between gap-x-6 gap-y-2">
              <p className="text-metric tabular-nums">
                <NumberFlow
                  value={euros}
                  locales="en"
                  format={{ style: "currency", currency: "EUR", maximumFractionDigits: 0 }}
                />
                <span className="ms-2 text-section font-normal tracking-normal text-muted-foreground">
                  {copy.pricing.perMonth}
                </span>
              </p>
              <p className="text-right text-xs text-muted-foreground tabular-nums">
                <NumberFlow
                  value={euros / amount}
                  locales="en"
                  format={{
                    style: "currency",
                    currency: "EUR",
                    minimumFractionDigits: 2,
                    maximumFractionDigits: 2,
                  }}
                />{" "}
                {copy.pricing.perPrompt}
                <br />
                {copy.pricing.tax}
              </p>
            </div>
            <div className="mt-6 grid justify-items-start gap-3">
              <PrimaryLink>{copy.pricing.cta}</PrimaryLink>
              <p className="text-xs text-muted-foreground">{copy.pricing.note}</p>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

/** The repository and the way to run it yourself. Prices live in pricing, just above. */
function OpenSource() {
  const { openSource } = copy;

  return (
    <section className="mx-auto max-w-340 px-4 pb-32 sm:px-7">
      <GithubLogo size={28} />
      <h2 className="site-heading mt-6">{openSource.title}</h2>
      <p className="site-lead mt-4 max-w-150 text-muted-foreground">{openSource.body}</p>
      <div className="mt-8 flex flex-wrap items-center gap-3">
        <a
          href={links.github}
          data-slot="button"
          className={buttonVariants({ variant: "outline", size: "cta", className: "gap-2" })}
        >
          <GithubLogo size={18} />
          {openSource.repo}
          <span className="text-muted-foreground">{openSource.license}</span>
        </a>
        <a
          href={links.selfHost}
          className="inline-flex items-center gap-1 rounded-control px-2 py-2 text-body text-muted-foreground transition-colors hover:text-foreground"
        >
          {openSource.selfCta}
          <ExternalIcon />
        </a>
      </div>
    </section>
  );
}

/** The questions as FAQPage structured data, so search and answer engines can read them as such. */
const faqSchema = JSON.stringify({
  "@context": "https://schema.org",
  "@type": "FAQPage",
  mainEntity: copy.faq.items.map((item) => ({
    "@type": "Question",
    name: item.q,
    acceptedAnswer: { "@type": "Answer", text: item.a },
  })),
}).replaceAll("<", "\\u003c");

function Faq() {
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
        {copy.faq.items.map((item, index) => (
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
      <script
        type="application/ld+json"
        // biome-ignore lint/security/noDangerouslySetInnerHtml: static JSON from copy.ts, with "<" escaped
        dangerouslySetInnerHTML={{ __html: faqSchema }}
      />
    </section>
  );
}

/** The last ask, over a fixed shoal. Orange still means an engine named you. */
function Closing() {
  const panel = useRef<HTMLDivElement>(null);

  return (
    <section className="relative mb-16 overflow-hidden max-lg:pb-100">
      <Pond count={220} avoid={panel} />
      <div className="relative mx-auto max-w-340 px-4 py-20 sm:px-7 lg:py-28">
        <div
          ref={panel}
          className="flex w-full max-w-170 flex-col items-start gap-8 rounded-panel bg-background p-6 sm:p-8"
        >
          <h2 className="site-display">{copy.closing.title}</h2>
          <Actions align="start" />
        </div>
      </div>
    </section>
  );
}

function Footer() {
  return (
    <footer className="mx-auto flex max-w-340 flex-wrap items-center justify-between gap-4 px-4 pb-10 text-xs text-muted-foreground sm:px-7">
      <span className="flex items-center gap-2">
        <SiteMark size={20} />
        {copy.footer.rights}
      </span>
      <span className="flex flex-wrap gap-4">
        <a href={links.github} className="transition-colors hover:text-foreground">
          {copy.footer.source}
        </a>
        <a href={links.skill} className="transition-colors hover:text-foreground">
          {copy.footer.skill}
        </a>
      </span>
    </footer>
  );
}

export function App() {
  return (
    <>
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:absolute focus:start-4 focus:top-4 focus:z-30 focus:rounded-control focus:bg-surface focus:px-3 focus:py-2 focus:text-body"
      >
        {copy.skipLink}
      </a>
      <Header />
      <main id="main">
        <Hero />
        <Bento />
        <Pricing />
        <OpenSource />
        <Faq />
        <Closing />
      </main>
      <Footer />
    </>
  );
}
