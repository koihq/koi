// The app's summary page, drawn with the app's own blocks on data recorded from a real instance.
// It is a picture of the product, not the product: nothing fetches, routes or saves, and the whole
// thing is inert, so the landing scrolls past it as cheaply as past an image.
import "@/config/i18n";

import {
  Add01Icon,
  ArrowDown01Icon,
  RankingIcon,
  Search01Icon,
  UserIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { useTranslation } from "react-i18next";
import {
  type PromptRes,
  PromptResFromJSON,
  type VisibilityReport,
  VisibilityReportFromJSON,
} from "@/api/generated";
import { BrandIcon } from "@/components/blocks/brand-icon";
import { BrandLabel } from "@/components/blocks/brand-label";
import { EngineFilter, RangeControl } from "@/components/blocks/chart-controls";
import { DataTable, type DataTableColumn } from "@/components/blocks/data-table";
import { StatusTabs } from "@/components/blocks/status-tabs";
import { VisibilityChart } from "@/components/blocks/visibility-chart";
import { VisibilityHeadline } from "@/components/blocks/visibility-headline";
import { Button } from "@/components/ui/button";
import { TooltipProvider } from "@/components/ui/tooltip";
import { formatDayRange, formatPoints, formatRate } from "@/lib/format";
import { change, rate, series } from "@/lib/visibility";
import { EngineMeterCell } from "@/routes/_authenticated/_app/-components/engine-meter";
import { PromptSelectCheckbox } from "@/routes/_authenticated/_app/-components/prompt-select-checkbox";
import recorded from "./summary-data.json";

const brand = recorded.brand;
const summary: VisibilityReport = VisibilityReportFromJSON(recorded.summary);
const prompts: PromptRes[] = recorded.prompts.map(PromptResFromJSON);
const counts = { tracking: recorded.counts.tracking, drafts: 3, stopped: 3 };
const noop = () => {};

const headClass = "h-auto px-3 py-1.5 text-start text-xs text-muted-foreground whitespace-normal";
const cellClass =
  "px-3 py-4 align-middle text-start text-body whitespace-normal [overflow-wrap:anywhere]";

/** The app's change chip, as the prompt list shows it beside a rate. */
function DeltaChip({ value, locale }: { value: number | null; locale: string }) {
  if (value === null) return null;

  return (
    <span
      className={`rounded-chip px-1.5 py-0.5 text-xs tabular-nums ${value > 0 ? "bg-positive/12 text-positive" : value < 0 ? "bg-negative/12 text-negative" : "bg-muted text-muted-foreground"}`}
    >
      {formatPoints(value, locale)}
    </span>
  );
}

function Visibility({ locale }: { locale: string }) {
  const { t } = useTranslation("home");
  const points = series(summary);
  const delta = change(summary);

  return (
    <section className="relative min-w-0 rounded-panel bg-surface p-4 sm:px-11 sm:py-9">
      <div className="flex min-h-8 flex-wrap items-center justify-between gap-x-4 gap-y-2">
        <h2 className="text-section font-semibold">{t("dashboard.aiVisibility")}</h2>
        <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
          <EngineFilter selected={[]} onToggle={noop} />
          <RangeControl days={30} onChange={noop} />
        </div>
      </div>
      <div className="mt-4 flex flex-wrap items-center gap-x-3 gap-y-2">
        <VisibilityHeadline
          brand={brand}
          rate={rate(summary)}
          comparisonRate={null}
          paired={false}
          locale={locale}
          animated={false}
          variant="home"
        />
        {delta !== null && <DeltaChip value={Number(delta.toFixed(1))} locale={locale} />}
      </div>
      <p className="mt-1 min-h-[18px] text-xs text-muted-foreground tabular-nums">
        {t("dashboard.periodAverage", {
          range: formatDayRange(points[0].date, points[points.length - 1].date, locale),
        })}
      </p>
      <VisibilityChart
        series={points}
        comparison={[]}
        brand={brand}
        inspected={null}
        inspectionInput="keyboard"
        onInspect={noop}
        locale={locale}
        scopeKey="landing"
      />
    </section>
  );
}

function RecommendedInstead({ locale }: { locale: string }) {
  const { t } = useTranslation("home");
  const own = rate(summary);
  const rivals = summary.competitors
    .filter((competitor) => competitor.metric.rate)
    .sort((left, right) => (right.metric.rate ?? 0) - (left.metric.rate ?? 0));

  return (
    <aside className="min-w-0 rounded-panel bg-surface p-4 sm:px-11 sm:py-9 @min-[80rem]:px-5">
      <div className="mb-3 flex min-h-8 items-center justify-between gap-3">
        <h2 className="text-section font-semibold">{t("dashboard.recommendedInstead")}</h2>
        <Button variant="ghost" size="circle-sm" className="-me-2 text-muted-foreground">
          <HugeiconsIcon icon={RankingIcon} aria-hidden="true" />
        </Button>
      </div>
      <div className="-mx-2 grid gap-0.5">
        {rivals.map((competitor) => {
          const rivalRate = competitor.metric.rate ?? null;
          const gap = rivalRate !== null && own !== null ? rivalRate - own : null;

          return (
            <div
              key={competitor.id}
              className="grid min-h-11 w-full grid-cols-[minmax(0,1fr)_auto_auto] items-center gap-3 rounded-control px-2 py-2 text-body"
            >
              <span className="flex min-w-0">
                <BrandLabel name={competitor.name} domain={competitor.domain} />
              </span>
              <span className="text-end font-semibold tabular-nums">
                {rivalRate === null ? "—" : formatRate(rivalRate, locale)}
              </span>
              <span
                className={`w-16 text-end text-xs tabular-nums ${gap === null ? "text-muted-foreground" : gap > 0 ? "text-negative" : "text-positive"}`}
              >
                {gap === null ? "—" : formatPoints(gap, locale)}
              </span>
            </div>
          );
        })}
      </div>
    </aside>
  );
}

function Prompts({ locale }: { locale: string }) {
  const { t } = useTranslation(["management", "home"]);

  const renderRival = (row: PromptRes) => {
    const first = [...(row.visibility?.competitors ?? [])]
      .filter((competitor) => (competitor.metric.rate ?? 0) > 0)
      .sort((left, right) => (right.metric.rate ?? -1) - (left.metric.rate ?? -1))[0];

    if (!first) {
      return <span className="text-xs text-muted-foreground">{t("home:dashboard.noRival")}</span>;
    }

    return (
      <BrandLabel
        size="sm"
        name={first.name}
        domain={first.domain}
        detail={first.metric.rate == null ? "—" : formatRate(first.metric.rate, locale)}
      />
    );
  };

  const columns: DataTableColumn<PromptRes>[] = [
    {
      id: "select",
      header: <PromptSelectCheckbox label="" checked={false} onChange={noop} />,
      headClassName: `${headClass} w-10 px-0! text-center`,
      cellClassName: `${cellClass} px-0! text-center`,
      cell: () => <PromptSelectCheckbox label="" checked={false} onChange={noop} />,
    },
    {
      id: "prompt",
      header: t("prompt"),
      headClassName: `${headClass} w-[38%] ps-0!`,
      cellClassName: `${cellClass} ps-0!`,
      cell: (row) => (
        <span className="block max-w-2xl text-sm/normal font-semibold text-foreground">
          {row.text}
        </span>
      ),
    },
    {
      id: "rate",
      header: t("home:dashboard.visibility"),
      headClassName: `${headClass} w-[32%]`,
      cellClassName: cellClass,
      cell: (row) => (
        <span className="inline-flex items-center gap-3">
          <EngineMeterCell
            rate={rate(row.visibility) ?? 0}
            rateFor={(engine) => rate(row.visibility, engine)}
            locale={locale}
            change={<DeltaChip value={change(row.visibility)} locale={locale} />}
          />
        </span>
      ),
    },
    {
      id: "rival",
      header: t("home:dashboard.topRival"),
      headClassName: headClass,
      cellClassName: cellClass,
      cell: renderRival,
    },
    {
      id: "actions",
      header: null,
      headClassName: `${headClass} w-36`,
      cellClassName: `${cellClass} ps-0! pe-8!`,
      cell: () => null,
    },
  ];

  return (
    <section className="relative min-w-0 rounded-panel bg-surface p-4 sm:px-11 sm:py-9">
      <div className="mb-5 flex flex-wrap items-center justify-between gap-x-6 gap-y-3">
        <div className="flex flex-wrap items-center gap-x-5 gap-y-2">
          <h2 className="text-section font-semibold">{t("prompts")}</h2>
          <StatusTabs
            label={t("statusLabel")}
            value="tracking"
            onChange={noop}
            options={[
              { value: "tracking", label: t("statusTracking"), count: counts.tracking },
              { value: "drafts", label: t("statusDrafts"), count: counts.drafts },
              { value: "stopped", label: t("statusStopped"), count: counts.stopped },
            ]}
          />
        </div>
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="circle-sm" className="text-muted-foreground">
            <HugeiconsIcon icon={Search01Icon} aria-hidden="true" />
          </Button>
          <Button variant="icon-outline" size="circle-sm" className="text-muted-foreground">
            <HugeiconsIcon icon={Add01Icon} aria-hidden="true" />
          </Button>
        </div>
      </div>
      <TooltipProvider delay={0}>
        <div className="@container">
          <div className="-mx-10 hidden @5xl:block">
            <DataTable
              interactive
              className="table-fixed"
              data={prompts}
              columns={columns}
              getRowId={(row) => row.id}
            />
          </div>
          <ul className="-mx-2 grid gap-0.5 @5xl:hidden">
            {prompts.map((row) => (
              <li key={row.id} className="flex min-w-0 items-start gap-2 rounded-control px-2 py-3">
                <PromptSelectCheckbox label="" checked={false} onChange={noop} />
                <div className="grid min-w-0 flex-1 gap-2">
                  <span className="block text-sm/normal font-semibold">{row.text}</span>
                  <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
                    <span className="inline-flex items-center gap-3">
                      <EngineMeterCell
                        rate={rate(row.visibility) ?? 0}
                        rateFor={(engine) => rate(row.visibility, engine)}
                        locale={locale}
                        change={null}
                      />
                    </span>
                    {renderRival(row)}
                  </div>
                </div>
              </li>
            ))}
          </ul>
        </div>
      </TooltipProvider>
    </section>
  );
}

/** The summary page as a signed-in founder sees it: header, visibility, rivals and prompts. */
export function SummaryMock() {
  const { i18n } = useTranslation();
  const locale = i18n.language;

  return (
    <div
      inert
      className="@container font-sans text-sm leading-normal bg-background text-foreground"
    >
      <header className="relative mx-auto flex min-h-8 w-full max-w-[1376px] items-center justify-between gap-4 px-4 pt-5 pb-6 sm:px-7 sm:pt-7 sm:pb-10">
        <span className="inline-flex max-w-full items-center gap-2 rounded-full p-1 pe-3">
          <BrandIcon
            name={brand.name}
            domain={brand.domain}
            lookupLogo={false}
            className="size-6"
          />
          <span className="max-w-40 truncate text-sm">{brand.name}</span>
          <HugeiconsIcon
            icon={ArrowDown01Icon}
            strokeWidth={2}
            className="size-3.5 text-muted-foreground"
          />
        </span>
        <Button variant="icon-outline" size="circle-sm" className="text-muted-foreground">
          <HugeiconsIcon icon={UserIcon} className="size-[18px]" aria-hidden="true" />
        </Button>
      </header>
      <div className="mx-auto w-full max-w-[1376px] px-4 pb-10 sm:px-7">
        <div className="grid gap-6">
          <div className="grid gap-6 @min-[80rem]:grid-cols-[minmax(0,1fr)_22rem]">
            <Visibility locale={locale} />
            <RecommendedInstead locale={locale} />
          </div>
          <Prompts locale={locale} />
        </div>
      </div>
    </div>
  );
}
