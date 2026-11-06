import { Copy01Icon, Tick02Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import type { VisibilityEngine, VisibilityResult, VisibilityRunStatus } from "@/api/generated";
import { BrandIcon } from "@/components/blocks/brand-icon";
import { OutcomeMark } from "@/components/blocks/outcome-mark";
import { Button } from "@/components/ui/button";
import { agentBrief } from "@/lib/agent-brief";
import { markAnswer } from "@/lib/answer-marks";
import { answerMessageKey, getAnswerState } from "@/lib/answer-state";
import { formatDateTime } from "@/lib/format";
import { safeSourceURL } from "@/lib/source-url";
import { engines } from "@/lib/visibility";

type Brand = { name: string; domain?: string | null };

function hostname(href: string) {
  return new URL(href).hostname.replace(/^www\./, "");
}

function sameSite(host: string, domain?: string | null) {
  const site = domain?.toLowerCase().replace(/^www\./, "");

  return Boolean(site) && (host === site || host.endsWith(`.${site}`));
}

/** Copies the selected answer as a markdown brief for the founder's own agent. */
export function CopyForAgent({
  brand,
  prompt,
  run,
  result,
}: {
  brand: Brand;
  prompt: string;
  run: VisibilityRunStatus;
  result?: VisibilityResult;
}) {
  const { t, i18n } = useTranslation(["home", "visibility"]);
  const [status, setStatus] = useState<"idle" | "copied" | "failed">("idle");

  useEffect(() => {
    if (status !== "copied") return;

    const timer = window.setTimeout(() => setStatus("idle"), 2000);
    return () => window.clearTimeout(timer);
  }, [status]);

  if (!result) return null;

  async function copy(answer: VisibilityResult) {
    const engine = engines.find((item) => item.id === answer.engine)?.name ?? answer.engine;
    const brief = agentBrief({
      brand,
      prompt,
      engine,
      collectedAt: formatDateTime(run.createdAt, i18n.language),
      outcome: t(`visibility:outcomes.${getAnswerState(run, answer.engine, answer)}`),
      result: answer,
    });

    try {
      await navigator.clipboard.writeText(brief);
      setStatus("copied");
    } catch {
      setStatus("failed");
    }
  }

  return (
    <div className="flex items-center gap-3">
      <output aria-live="polite" className="text-xs text-destructive empty:hidden">
        {status === "failed" ? t("home:detail.copyFailed") : ""}
      </output>
      <Button variant="outline" size="lg" onClick={() => void copy(result)}>
        <HugeiconsIcon
          icon={status === "copied" ? Tick02Icon : Copy01Icon}
          data-icon="inline-start"
          aria-hidden="true"
        />
        {t(status === "copied" ? "home:detail.copied" : "home:detail.copyForAgent")}
      </Button>
    </div>
  );
}

/** Engine switch inside a run: each engine with its outcome; one is pressed. */
export function EngineSwitch({
  run,
  results,
  selected,
  onSelect,
  emphasize = false,
}: {
  run: VisibilityRunStatus;
  results: VisibilityResult[];
  selected: VisibilityEngine;
  onSelect: (engine: VisibilityEngine) => void;
  emphasize?: boolean;
}) {
  const { t } = useTranslation("home");

  return (
    <fieldset className="flex flex-wrap gap-1" aria-label={t("detail.answersByEngine")}>
      {engines.map((engine) => (
        <Button
          key={engine.id}
          type="button"
          variant="ghost"
          size="pill"
          aria-pressed={selected === engine.id}
          className="gap-2 pe-3 text-muted-foreground"
          onClick={() => onSelect(engine.id)}
        >
          <OutcomeMark
            engine={engine.id}
            emphasize={emphasize}
            state={getAnswerState(
              run,
              engine.id,
              results.find((entry) => entry.engine === engine.id),
            )}
          />
          {engine.name}
        </Button>
      ))}
    </fieldset>
  );
}

/**
 * One engine answer as evidence: whether the brand made it, the answer with every brand
 * marked where it appears, who else appeared and how, and the sources by site.
 */
export function RunAnswer({
  brand,
  run,
  result,
  engine,
  rivalDomain,
}: {
  brand: Brand;
  run: VisibilityRunStatus;
  result?: VisibilityResult;
  engine: VisibilityEngine;
  rivalDomain: (name: string) => string | undefined;
}) {
  const { t } = useTranslation(["home", "visibility"]);
  const state = getAnswerState(run, engine, result);

  if (!result) {
    return (
      <p className="py-8 text-sm text-muted-foreground">
        {t(`visibility:answerMessages.${answerMessageKey(state)}`)}
      </p>
    );
  }

  const rivals = result.appearances.map((appearance) => appearance.name);
  const segments = markAnswer(result.rawAnswer, brand.name, rivals);
  const sources = [...new Set(result.sources)]
    .map((source) => {
      const href = safeSourceURL(source);
      const host = href ? hostname(href) : source;

      return { source, href, host, own: sameSite(host, brand.domain) };
    })
    .sort((left, right) => Number(right.own) - Number(left.own));

  return (
    <div className="grid min-w-0 gap-10 lg:grid-cols-[minmax(0,1fr)_16rem]">
      <article className="min-w-0">
        <p className="flex items-center gap-2 text-section font-semibold">
          <OutcomeMark engine={engine} state={state} />
          {t("home:detail.brandOutcome", {
            brand: brand.name,
            outcome: t(`visibility:brandOutcomes.${state}`),
          })}
        </p>
        {result.errorCode && (
          <p role="alert" className="mt-3 text-sm text-destructive">
            {t("visibility:error", { code: result.errorCode })}
          </p>
        )}
        <p className="mt-6 max-w-[68ch] whitespace-pre-wrap break-words text-sm font-normal leading-7 text-foreground">
          {result.rawAnswer
            ? segments.map((segment, index) =>
                segment.mark ? (
                  <mark
                    // biome-ignore lint/suspicious/noArrayIndexKey: segments are positional and never reorder.
                    key={index}
                    data-mark={segment.mark}
                    className={
                      segment.mark === "brand"
                        ? "bg-transparent font-semibold text-foreground underline decoration-foreground decoration-2 underline-offset-4"
                        : "bg-transparent text-foreground underline decoration-muted-foreground decoration-dotted decoration-2 underline-offset-4"
                    }
                  >
                    {segment.text}
                  </mark>
                ) : (
                  segment.text
                ),
              )
            : t(`visibility:answerMessages.${answerMessageKey(state)}`)}
        </p>
      </article>

      <aside className="grid min-w-0 content-start gap-8">
        {result.appearances.length > 0 && (
          <section aria-labelledby="run-rivals-heading">
            <h3 id="run-rivals-heading" className="text-xs text-muted-foreground">
              {t("home:detail.appearances")}
            </h3>
            <ul className="mt-3 grid gap-3">
              {result.appearances.map((appearance) => {
                const kinds = [
                  ...new Set(
                    appearance.evidence.map((item) => t(`visibility:evidenceKinds.${item.kind}`)),
                  ),
                ];

                return (
                  <li
                    key={appearance.competitorId ?? appearance.name}
                    className="flex items-center gap-2 text-sm"
                  >
                    <BrandIcon
                      name={appearance.name}
                      domain={rivalDomain(appearance.name)}
                      className="size-5"
                    />
                    <span className="min-w-0 flex-1 truncate">{appearance.name}</span>
                    <span className="text-xs text-muted-foreground">{kinds.join(" · ")}</span>
                  </li>
                );
              })}
            </ul>
          </section>
        )}

        <section aria-labelledby="run-sources-heading">
          <h3 id="run-sources-heading" className="text-xs text-muted-foreground">
            {t("home:detail.sources")}
          </h3>
          {sources.length ? (
            <ul className="mt-3 grid gap-3">
              {sources.map(({ source, href, host, own }) => (
                <li key={source} className="flex min-w-0 items-center gap-2 text-sm">
                  <BrandIcon name={host} domain={host} className="size-5" />
                  {href ? (
                    <a
                      href={href}
                      target="_blank"
                      rel="noopener noreferrer"
                      title={href}
                      className="min-w-0 flex-1 truncate underline-offset-4 hover:underline"
                    >
                      {host}
                    </a>
                  ) : (
                    <span className="min-w-0 flex-1 truncate">{source}</span>
                  )}
                  {own && (
                    <span className="text-xs text-muted-foreground">
                      {t("home:detail.yourSite")}
                    </span>
                  )}
                </li>
              ))}
            </ul>
          ) : (
            <p className="mt-3 text-sm text-muted-foreground">{t("home:detail.noSources")}</p>
          )}
        </section>
      </aside>
    </div>
  );
}
