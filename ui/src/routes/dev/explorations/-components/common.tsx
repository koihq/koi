import {
  Add01Icon,
  Analytics01Icon,
  ArrowRight01Icon,
  Copy01Icon,
  Download01Icon,
  File01Icon,
  Home01Icon,
  Link01Icon,
  Menu01Icon,
  MessageQuestionIcon,
  MoreHorizontalIcon,
  QuoteIcon,
  Search01Icon,
  Settings01Icon,
  UserMultipleIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Link, useNavigate } from "@tanstack/react-router";
import { type ReactNode, useState } from "react";
import { KoiMark } from "@/components/blocks/koi-mark";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import {
  type DemoState,
  DIRECTIONS,
  dateLabel,
  ENGINES,
  engineName,
  metrics,
  type Observation,
  percent,
  SECTIONS,
  type Section,
  trend,
} from "../-lib/data";
import { useMock } from "../-lib/state";
import C from "../-locales/en.json";

export { C };
const ICONS = {
  overview: Home01Icon,
  mentions: Analytics01Icon,
  prompts: MessageQuestionIcon,
  competitors: UserMultipleIcon,
  citations: Link01Icon,
  brief: File01Icon,
  search: Search01Icon,
  more: MoreHorizontalIcon,
  menu: Menu01Icon,
  settings: Settings01Icon,
  add: Add01Icon,
  next: ArrowRight01Icon,
  quote: QuoteIcon,
  copy: Copy01Icon,
  download: Download01Icon,
};
export function Icon({ name, size = 18 }: { name: keyof typeof ICONS; size?: number }) {
  return <HugeiconsIcon icon={ICONS[name]} size={size} strokeWidth={1.6} aria-hidden={true} />;
}

export function ExploreLink({
  section,
  children,
  className,
  onClick,
}: {
  section: Section;
  children?: ReactNode;
  className?: string;
  onClick?: () => void;
}) {
  const { direction, demoState } = useMock();
  return (
    <Link
      to="/dev/explorations/$direction/$section"
      params={{ direction, section }}
      search={{ state: demoState }}
      className={className}
      onClick={onClick}
    >
      {children ?? C[section]}
    </Link>
  );
}

export function Choice({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: string;
  options: { value: string; label: string }[];
  onChange: (value: string) => void;
}) {
  return (
    <Select
      value={value}
      items={options}
      onValueChange={(next) => {
        if (next) onChange(next);
      }}
    >
      <SelectTrigger aria-label={label}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {options.map((item) => (
          <SelectItem key={item.value} value={item.value}>
            {item.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

export function Filters({ scan = false }: { scan?: boolean }) {
  const { filters, setFilters, scan: runScan, demoState } = useMock();
  return (
    <div className="exp-filters">
      <Choice
        label={C.period}
        value={String(filters.days)}
        options={[7, 30, 90].map((days) => ({
          value: String(days),
          label: C[`${days}days` as "7days"],
        }))}
        onChange={(value) =>
          setFilters((current) => ({ ...current, days: Number(value) as 7 | 30 | 90 }))
        }
      />
      <Choice
        label={C.engine}
        value={filters.engine}
        options={[
          { value: "all", label: C.allEngines },
          ...ENGINES.map((engine) => ({ value: engine.id, label: engine.name })),
        ]}
        onChange={(value) =>
          setFilters((current) => ({ ...current, engine: value as typeof current.engine }))
        }
      />
      {scan ? (
        <Button size="lg" onClick={runScan} disabled={demoState === "loading"}>
          {C.scan}
        </Button>
      ) : null}
    </div>
  );
}

export function Hint({ label, children }: { label: string; children: ReactNode }) {
  return (
    <Tooltip>
      <TooltipTrigger render={<button type="button" className="exp-hint" />}>
        {children}
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}

export function BrandHeader({ greeting = false }: { greeting?: boolean }) {
  return (
    <div className="exp-brand">
      <KoiMark className="size-6" />
      <span>{greeting ? C.hello : C.tally}</span>
      {!greeting ? <span className="exp-brand-caption">{C.workspace}</span> : null}
    </div>
  );
}

export function Account() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button variant="ghost" size="icon-lg" aria-label={C.account} onClick={() => setOpen(true)}>
        <span className="exp-avatar">{C.account[0]}</span>
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{C.account}</DialogTitle>
            <DialogDescription>{C.accountRole}</DialogDescription>
          </DialogHeader>
          <p>{C.accountHint}</p>
        </DialogContent>
      </Dialog>
    </>
  );
}

export function PreviewControls() {
  const { direction, demoState, reset } = useMock();
  const navigate = useNavigate({ from: "/dev/explorations/$direction/$section" });
  return (
    <div className="exp-preview">
      <DropdownMenu>
        <DropdownMenuTrigger render={<Button variant="ghost" size="sm" />} aria-label={C.preview}>
          {C.explorations}
          <Icon name="more" size={14} />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem render={<Link to="/dev/explorations" />}>{C.back}</DropdownMenuItem>
          {DIRECTIONS.map((item) => (
            <DropdownMenuItem
              key={item}
              onClick={() =>
                void navigate({
                  to: "/dev/explorations/$direction/$section",
                  params: { direction: item, section: "overview" },
                  search: { state: demoState },
                })
              }
            >{`${direction === item ? "✓ " : ""}${C[item]}`}</DropdownMenuItem>
          ))}
          {(["populated", "loading", "empty"] as DemoState[]).map((state) => (
            <DropdownMenuItem
              key={state}
              onClick={() => void navigate({ search: (previous) => ({ ...previous, state }) })}
            >{`${demoState === state ? "✓ " : ""}${C[state]}`}</DropdownMenuItem>
          ))}
          <DropdownMenuItem onClick={reset}>{C.reset}</DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

export function Directory({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [query, setQuery] = useState("");
  const matches = SECTIONS.filter((section) =>
    C[section].toLowerCase().includes(query.toLowerCase()),
  );
  return (
    <Dialog
      open={open}
      onOpenChange={(value) => {
        if (!value) {
          setQuery("");
          onClose();
        }
      }}
    >
      <DialogContent className="exp-directory">
        <DialogHeader>
          <DialogTitle>{C.searchNavigation}</DialogTitle>
          <DialogDescription>{C.navigation}</DialogDescription>
        </DialogHeader>
        <Input
          aria-label={C.searchNavigation}
          placeholder={C.searchSections}
          value={query}
          onChange={(event) => setQuery(event.target.value)}
        />
        <nav aria-label={C.navigation}>
          {matches.map((section) => (
            <ExploreLink
              key={section}
              section={section}
              className="exp-directory-link"
              onClick={onClose}
            >
              <Icon name={section} />
              <span>{C[section]}</span>
              <Icon name="next" size={14} />
            </ExploreLink>
          ))}
        </nav>
        {!matches.length ? <p>{C.noSections}</p> : null}
      </DialogContent>
    </Dialog>
  );
}

export function Dock({ section, fey = false }: { section: Section; fey?: boolean }) {
  const [open, setOpen] = useState(false);
  const primary: Section[] = fey
    ? ["overview", "mentions", "competitors", "brief"]
    : ["overview", "mentions", "prompts", "brief"];
  return (
    <TooltipProvider delay={250}>
      <nav className={`exp-dock ${fey ? "exp-dock-fey" : ""}`} aria-label={C.navigation}>
        {primary.map((item) => (
          <Tooltip key={item}>
            <TooltipTrigger
              render={
                <ExploreLink
                  section={item}
                  className={`exp-dock-link ${section === item ? "is-active" : ""}`}
                />
              }
              aria-label={C[item]}
              aria-current={section === item ? "page" : undefined}
            >
              <Icon name={item} size={21} />
              <span className="sr-only">{C[item]}</span>
            </TooltipTrigger>
            <TooltipContent>{C[item]}</TooltipContent>
          </Tooltip>
        ))}
        <Tooltip>
          <TooltipTrigger
            render={<Button variant="ghost" size="icon-lg" />}
            aria-label={C.more}
            onClick={() => setOpen(true)}
          >
            <Icon name="more" size={21} />
          </TooltipTrigger>
          <TooltipContent>{C.more}</TooltipContent>
        </Tooltip>
        {fey ? (
          <Button
            variant="ghost"
            size="icon-lg"
            aria-label={C.searchNavigation}
            onClick={() => setOpen(true)}
          >
            <Icon name="search" size={21} />
          </Button>
        ) : null}
      </nav>
      <Directory open={open} onClose={() => setOpen(false)} />
    </TooltipProvider>
  );
}

export function StateGate({ children }: { children: ReactNode }) {
  const { demoState, scan } = useMock();
  if (demoState === "populated") return children;
  if (demoState === "loading")
    return (
      <section className="exp-loading" aria-busy={true} aria-label={C.collecting}>
        <div className="exp-loading-title">
          <Skeleton className="h-5 w-48" />
          <Skeleton className="mt-4 h-3 w-72 max-w-full" />
        </div>
        <div className="exp-loading-chart">
          <Skeleton className="h-full w-full" />
        </div>
        <div className="exp-loading-bottom">
          {[0, 1, 2].map((item) => (
            <Skeleton key={item} className="h-32 w-full" />
          ))}
        </div>
        <output>{C.collecting}</output>
      </section>
    );
  return (
    <div className="exp-empty">
      <div className="exp-empty-orbit" aria-hidden={true}>
        {ENGINES.map((engine) => (
          <span key={engine.id}>{engine.short}</span>
        ))}
      </div>
      <section className="exp-empty-content">
        <span className="exp-signal-dot" />
        <h1>{C.emptyTitle}</h1>
        <p>{C.emptyLead}</p>
        <Button size="lg" onClick={scan}>
          {C.scan}
        </Button>
        <span className="exp-sample-note">{C.sample}</span>
      </section>
    </div>
  );
}

export function MetricStrip({ compact = false }: { compact?: boolean }) {
  const { summary, previous, prompts } = useMock();
  const delta = summary.presence - previous.presence;
  return (
    <dl className={`exp-metrics ${compact ? "is-compact" : ""}`}>
      <div>
        <dt>
          <Hint label={C.presenceHint}>{C.presence}</Hint>
          <span className="exp-series-dot" />
        </dt>
        <dd>{percent(summary.presence)}</dd>
        <span
          className={delta >= 0 ? "exp-positive" : "exp-negative"}
        >{`${delta >= 0 ? "+" : ""}${delta.toFixed(1)} pp`}</span>
      </div>
      <div>
        <dt>
          <Hint label={C.citationHint}>{C.citationRate}</Hint>
        </dt>
        <dd>{percent(summary.cited)}</dd>
        <span className="exp-muted">{C.compared}</span>
      </div>
      <div>
        <dt>{C.answers}</dt>
        <dd>{summary.completed.toLocaleString("en-US")}</dd>
        <span className="exp-muted">{`${summary.failures} ${C.failures.toLowerCase()}`}</span>
      </div>
      <div>
        <dt>{C.tracking}</dt>
        <dd>{prompts.filter((prompt) => prompt.confirmed).length}</dd>
        <span className="exp-muted">{`${ENGINES.length} ${C.allEngines.toLowerCase().replace("all ", "")}`}</span>
      </div>
    </dl>
  );
}

export function TrendChart({
  large = false,
  compare = true,
}: {
  large?: boolean;
  compare?: boolean;
}) {
  const { observations, filters } = useMock();
  const points = trend(observations, filters);
  const competitor = trend(observations, filters, "typeform");
  const [active, setActive] = useState<number | null>(null);
  const index = active === null ? null : Math.min(active, points.length - 1);
  const width = 1000;
  const height = large ? 350 : 230;
  const plot = (values: typeof points) =>
    values
      .map(
        (point, i) =>
          `${i ? "L" : "M"}${((i / (values.length - 1)) * width).toFixed(1)},${(height - (point.value / 100) * (height - 12)).toFixed(1)}`,
      )
      .join(" ");
  const point = index === null ? null : points[index];
  const x = index === null ? 0 : (index / (points.length - 1)) * width;

  return (
    <figure className={`exp-chart ${large ? "is-large" : ""}`}>
      <div className="exp-chart-label">
        <span>{C.daily}</span>
        <span>
          {percent(
            metrics(
              observations.filter(
                (item) =>
                  item.day === 179 && (filters.engine === "all" || item.engine === filters.engine),
              ),
            ).presence,
          )}
        </span>
      </div>
      <svg
        viewBox={`0 0 ${width} ${height}`}
        preserveAspectRatio="none"
        role="img"
        aria-label={C.trendDescription}
        onPointerMove={(event) => {
          const box = event.currentTarget.getBoundingClientRect();
          setActive(
            Math.max(
              0,
              Math.min(
                points.length - 1,
                Math.round(((event.clientX - box.left) / box.width) * (points.length - 1)),
              ),
            ),
          );
        }}
        onPointerLeave={() => setActive(null)}
      >
        {[25, 50, 75].map((value) => (
          <line
            key={value}
            x1={0}
            x2={width}
            y1={height - (value / 100) * (height - 12)}
            y2={height - (value / 100) * (height - 12)}
            className="exp-gridline"
          />
        ))}
        {compare ? <path d={plot(competitor)} className="exp-comparison-line" /> : null}
        <path d={plot(points)} className="exp-chart-line" />
        {point ? (
          <>
            <line x1={x} x2={x} y1={0} y2={height} className="exp-chart-crosshair" />
            <circle
              cx={x}
              cy={height - (point.value / 100) * (height - 12)}
              r={4}
              className="exp-chart-point"
            />
          </>
        ) : null}
      </svg>
      {point ? (
        <output
          className="exp-chart-tooltip"
          style={{ left: `${Math.max(12, Math.min(85, (x / width) * 100))}%` }}
        >{`${dateLabel(point.day)} · ${percent(point.value)}`}</output>
      ) : null}
      <div className="exp-chart-dates">
        <span>{dateLabel(points[0].day)}</span>
        <span>{dateLabel(points[Math.floor(points.length / 2)].day)}</span>
        <span>{dateLabel(179)}</span>
      </div>
      <input
        className="exp-chart-slider"
        type="range"
        min={0}
        max={points.length - 1}
        value={index ?? points.length - 1}
        aria-label={C.inspectDate}
        aria-valuetext={point ? `${dateLabel(point.day)}: ${percent(point.value)}` : undefined}
        onChange={(event) => setActive(Number(event.target.value))}
        onBlur={() => setActive(null)}
      />
      <figcaption className="exp-chart-legend">
        <span>
          <i />
          {C.tallyLegend}
        </span>
        {compare ? (
          <span>
            <i />
            {C.competitorLegend}
          </span>
        ) : null}
        <span>{C.trendDescription.split(".")[0]}</span>
      </figcaption>
    </figure>
  );
}

export function EngineList() {
  const { results } = useMock();
  return (
    <div className="exp-engine-list">
      {ENGINES.map((engine) => {
        const selected = results.filter((item) => item.engine === engine.id);
        return (
          <div key={engine.id} className="exp-engine-row">
            <span className="exp-letter">{engine.short}</span>
            <span>{engine.name}</span>
            <div className="exp-bar">
              <span style={{ width: `${metrics(selected).presence}%` }} />
            </div>
            <strong>{selected.length ? percent(metrics(selected).presence) : "—"}</strong>
          </div>
        );
      })}
    </div>
  );
}

export function CompetitorList() {
  const { results, competitors } = useMock();
  const ranked = [
    { id: "tally", name: "Tally", domain: "tally.so", confirmed: true },
    ...competitors,
  ]
    .map((brand) => ({ ...brand, rate: metrics(results, brand.id).presence }))
    .sort((a, b) => b.rate - a.rate);
  return (
    <div className="exp-competitor-list">
      {ranked.map((brand, index) => (
        <ExploreLink
          key={brand.id}
          section="competitors"
          className={`exp-ranking-row ${brand.id === "tally" ? "is-brand" : ""}`}
        >
          <span className="exp-rank">{index + 1}</span>
          <span className="exp-letter">{brand.name[0]}</span>
          <span>
            {brand.name}
            <small>{brand.domain}</small>
          </span>
          <div className="exp-bar">
            <span style={{ width: `${brand.rate}%` }} />
          </div>
          <strong>{percent(brand.rate)}</strong>
        </ExploreLink>
      ))}
    </div>
  );
}

export function BriefTeaser() {
  return (
    <div className="exp-brief-teaser">
      <span className="exp-overline">
        <Icon name="brief" size={16} />
        {C.recommended}
      </span>
      <h2>{C.briefTitle}</h2>
      <p>{C.briefLead}</p>
      <ExploreLink section="brief" className="exp-text-link">
        {C.openBrief}
        <Icon name="next" size={16} />
      </ExploreLink>
    </div>
  );
}

export function EvidenceDialog() {
  const { selected, setSelected, competitors } = useMock();
  return (
    <Dialog
      open={Boolean(selected)}
      onOpenChange={(open) => {
        if (!open) setSelected(null);
      }}
    >
      <DialogContent className="exp-evidence-dialog">
        <DialogHeader>
          <DialogTitle>
            {selected ? `${engineName(selected.engine)} ${C.answer.toLowerCase()}` : C.answer}
          </DialogTitle>
          <DialogDescription>{selected?.promptText}</DialogDescription>
        </DialogHeader>
        {selected ? (
          <>
            <div className="exp-evidence-meta">
              <Outcome observation={selected} />
              <span>{dateLabel(selected.day)}</span>
              <span>{C.sample}</span>
            </div>
            <p className="exp-answer-text">{selected.rawAnswer}</p>
            <h3>{C.whoAppeared}</h3>
            <div className="exp-appeared">
              {selected.brands.map((id) => (
                <span key={id}>
                  {id === "tally"
                    ? C.tally
                    : (competitors.find((brand) => brand.id === id)?.name ?? id)}
                </span>
              ))}
            </div>
            <h3>{C.sources}</h3>
            {selected.sources.length ? (
              <ul className="exp-source-links">
                {selected.sources.map((source) => (
                  <li key={source}>
                    <a href={source} target="_blank" rel="noopener noreferrer">
                      {source}
                    </a>
                  </li>
                ))}
              </ul>
            ) : (
              <p>{C.noSources}</p>
            )}
            <p className="exp-muted">{C.sampleEvidence}</p>
          </>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

export function Outcome({ observation }: { observation?: Observation }) {
  const outcome = observation?.outcome;
  return (
    <span className={`exp-outcome ${outcome ?? "pending"}`}>
      <i />
      {outcome ? C[outcome] : C.notCollected}
    </span>
  );
}

export function Notice() {
  const { notice } = useMock();
  return (
    <output className="exp-notice" aria-live="polite">
      {notice ? <span>{notice}</span> : null}
    </output>
  );
}
