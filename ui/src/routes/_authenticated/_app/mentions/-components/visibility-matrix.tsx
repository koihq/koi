import { useTranslation } from "react-i18next";

import type { VisibilityResult, VisibilityRun } from "@/api/runs.api";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  groupVisibilityResults,
  safeSourceURL,
} from "@/routes/_authenticated/_app/mentions/-lib/visibility";

export function VisibilityMatrix({
  run,
  results,
}: {
  run: VisibilityRun;
  results: VisibilityResult[];
}) {
  const { t } = useTranslation("visibility");
  const rows = groupVisibilityResults(results);

  return (
    <div className="min-w-0 space-y-3">
      <p className="text-xs text-muted-foreground">{t("scrollHint")}</p>
      <Table aria-label={t("tableLabel")}>
        <TableHeader>
          <TableRow>
            <TableHead scope="col">{t("prompt")}</TableHead>
            {run.engines.map((engine) => (
              <TableHead key={engine} scope="col">
                {t(`engines.${engine}`)}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row) => (
            <TableRow key={row.promptID}>
              <TableHead
                scope="row"
                className="min-w-48 max-w-sm whitespace-normal break-words py-3"
              >
                {row.promptText}
              </TableHead>
              {run.engines.map((engine) => {
                const result = row.results.get(engine);

                return (
                  <TableCell key={engine} className="min-w-36 align-top">
                    {result?.outcome ? (
                      <div className="space-y-2">
                        <VisibilityAnswerDialog result={result} />
                        {result.appearances.length ? (
                          <ul className="max-w-56 space-y-1 whitespace-normal text-xs text-muted-foreground">
                            {result.appearances.map((appearance) => (
                              <li
                                key={appearance.competitorId ?? appearance.name}
                                className="break-words"
                              >
                                {appearance.name}
                                {!appearance.confirmed
                                  ? ` · ${t(appearance.competitorId ? "appearanceSuggestion" : "appearanceUnresolved")}`
                                  : ""}
                              </li>
                            ))}
                          </ul>
                        ) : null}
                      </div>
                    ) : (
                      <span className="text-muted-foreground">
                        {t(run.state === "failed" ? "notCollected" : "pending")}
                      </span>
                    )}
                  </TableCell>
                );
              })}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function VisibilityAnswerDialog({ result }: { result: VisibilityResult }) {
  const { t, i18n } = useTranslation("visibility");
  if (!result.outcome) return null;

  const engine = t(`engines.${result.engine}`);
  const outcome = t(`outcomes.${result.outcome}`);

  return (
    <Dialog>
      <DialogTrigger
        render={<Button type="button" variant="outline" size="sm" />}
        aria-label={t("cellLabel", { engine, prompt: result.promptText, outcome })}
      >
        {outcome}
      </DialogTrigger>
      <DialogContent className="overscroll-contain sm:max-w-2xl" showCloseButton={false}>
        <DialogHeader>
          <DialogTitle>{t("detailTitle", { engine })}</DialogTitle>
          <DialogDescription className="break-words">{result.promptText}</DialogDescription>
        </DialogHeader>
        <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-6 gap-y-2">
          <dt className="text-muted-foreground">{t("outcomeLabel")}</dt>
          <dd>{outcome}</dd>
          <dt className="text-muted-foreground">{t("collectorLabel")}</dt>
          <dd>{t(result.collector === "stub" ? "sample" : "live")}</dd>
          <dt className="text-muted-foreground">{t("fetchedAt")}</dt>
          <dd className="tabular-nums">{result.fetchedAt?.toLocaleString(i18n.language)}</dd>
        </dl>
        {result.collector === "stub" ? (
          <p className="text-muted-foreground">{t("sampleNotice")}</p>
        ) : null}
        {result.errorCode ? (
          <p className="text-destructive">{t("error", { code: result.errorCode })}</p>
        ) : null}
        <div className="space-y-3">
          <h3 className="font-medium">{t("whoAppeared")}</h3>
          {result.appearances.length ? (
            <ul className="space-y-4">
              {result.appearances.map((appearance) => (
                <li key={appearance.competitorId ?? appearance.name} className="min-w-0 space-y-1">
                  <p className="break-words font-medium">
                    {appearance.name}
                    <span className="ms-2 text-sm font-normal text-muted-foreground">
                      {t(
                        appearance.confirmed
                          ? "appearanceConfirmed"
                          : appearance.competitorId
                            ? "appearanceSuggestion"
                            : "appearanceUnresolved",
                      )}
                    </span>
                  </p>
                  <ul className="space-y-1 text-sm text-muted-foreground">
                    {appearance.evidence.map((evidence) => {
                      const href =
                        evidence.kind === "citation" || evidence.kind === "link"
                          ? safeSourceURL(evidence.value)
                          : undefined;

                      return (
                        <li key={`${evidence.kind}:${evidence.value}`} className="break-all">
                          {t(`evidenceKinds.${evidence.kind}`)}
                          {": "}
                          {href ? (
                            <a
                              href={href}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="underline underline-offset-4"
                            >
                              {evidence.value}
                            </a>
                          ) : (
                            evidence.value
                          )}
                        </li>
                      );
                    })}
                  </ul>
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-muted-foreground">{t("noAppearances")}</p>
          )}
        </div>
        <div className="space-y-2">
          <h3 className="font-medium">{t("answer")}</h3>
          <p className="whitespace-pre-wrap break-words">
            {result.rawAnswer ||
              t(
                result.engine === "google_ai_overviews" && result.outcome === "miss"
                  ? "noOverview"
                  : "noAnswer",
              )}
          </p>
        </div>
        <div className="space-y-2">
          <h3 className="font-medium">{t("sources")}</h3>
          {result.sources.length ? (
            <ul className="space-y-2">
              {[...new Set(result.sources)].map((source) => {
                const href = safeSourceURL(source);

                return (
                  <li key={source} className="break-all">
                    {href ? (
                      <a
                        href={href}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="underline underline-offset-4"
                      >
                        {source}
                      </a>
                    ) : (
                      source
                    )}
                  </li>
                );
              })}
            </ul>
          ) : (
            <p className="text-muted-foreground">{t("noSources")}</p>
          )}
        </div>
        <DialogFooter>
          <DialogClose render={<Button type="button" variant="outline" size="sm" />}>
            {t("close")}
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
