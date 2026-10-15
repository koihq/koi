import { createFileRoute, Link } from "@tanstack/react-router";
import { type ReactNode, useLayoutEffect, useRef, useState } from "react";
import type { Competitor } from "@/api/competitors.api";
import type { Prompt } from "@/api/prompts.api";
import { DeleteItemDialog } from "@/components/blocks/delete-item-dialog";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { AccountMenu } from "./-components/account-menu";
import { AnswerFeed } from "./-components/answer-feed";
import { BrandSwitcher } from "./-components/brand-switcher";
import { Icon } from "./-components/common";
import { CompetitorEditor } from "./-components/competitor-editor";
import { CompetitorList } from "./-components/competitor-list";
import { HomeSignal } from "./-components/home-signal";
import { copy, ListPagination, LocalChoice } from "./-components/list-controls";
import { PromptEditor } from "./-components/prompt-editor";
import { PromptList } from "./-components/prompt-list";
import { SettingsForm } from "./-components/settings-form";
import { useExplorationState } from "./-hooks/use-exploration-state";
import { ENGINES } from "./-lib/data";
import {
  collectionAnswers,
  type ExplorationState,
  type ExplorationView,
  type PreviewBrand,
  pageOf,
  parseExplorationSearch,
  resultForState,
} from "./-lib/exploration-fixtures";
import { HOME_BRANDS, HOME_LOGOS, type HomeBrand } from "./-lib/home-evidence";
import C from "./-locales/mvp.en.json";
import "./-styles.css";
import "./-home-context.css";

export const Route = createFileRoute("/dev/explorations/home-context")({
  validateSearch: parseExplorationSearch,
  component: HomeContext,
});

const DESTINATIONS = [
  { view: "visibility", icon: "overview" },
  { view: "mentions", icon: "quote" },
  { view: "prompts", icon: "prompts" },
  { view: "competitors", icon: "competitors" },
] as const;
const MAX_VISIBLE_BRANDS = 3;

function HomeContext() {
  const { state } = Route.useSearch();
  return <Workspace key={state ?? "populated"} />;
}

function Workspace() {
  const search = Route.useSearch();
  const navigate = Route.useNavigate();
  const brand = search.brand ?? "deja";
  const view = search.view ?? "visibility";
  const state = useExplorationState(search.state);
  const [periods, setPeriods] = useState<Partial<Record<HomeBrand, string>>>({});
  const [signedOut, setSignedOut] = useState(false);
  const [recovered, setRecovered] = useState(false);
  const data = state.brands[brand];
  const previousPage = useRef(`${brand}:${view}:false`);
  const options = (Object.keys(HOME_BRANDS) as HomeBrand[]).map((id) => ({
    id,
    name: state.brands[id].settings.name,
    ...HOME_LOGOS[HOME_BRANDS[id].name],
  }));

  useLayoutEffect(() => {
    const root = document.documentElement;
    const previous = root.getAttribute("data-exploration-theme");
    const dark = root.classList.contains("dark");
    const scheme = root.style.colorScheme;
    const title = document.title;

    root.setAttribute("data-exploration-theme", "home-context");
    root.classList.add("dark");
    root.style.colorScheme = "dark";

    return () => {
      if (previous) root.setAttribute("data-exploration-theme", previous);
      else root.removeAttribute("data-exploration-theme");
      root.classList.toggle("dark", dark);
      root.style.colorScheme = scheme;
      document.title = title;
    };
  }, []);

  useLayoutEffect(() => {
    document.title = `${C[view]} · Koi`;
    const page = `${brand}:${view}:${signedOut}`;
    if (previousPage.current !== page) {
      document.getElementById("hc-page-heading")?.focus({ preventScroll: true });
      window.scrollTo({ top: 0, behavior: "instant" });
      previousPage.current = page;
    }
  }, [brand, view, signedOut]);

  if (signedOut)
    return (
      <div className="exploration home-context-study hc-signed-out">
        <main>
          <h1 id="hc-page-heading" tabIndex={-1}>
            {C.signedOut}
          </h1>
          <Button variant="ghost" onClick={() => setSignedOut(false)}>
            {C.return}
          </Button>
        </main>
      </div>
    );

  return (
    <div className="exploration home-context-study">
      <a href="#home-context-main" className="exp-skip">
        {C.skip}
      </a>
      <header className="hc-header">
        <BrandSwitcher
          brands={options}
          value={brand}
          onChange={(next) => void navigate({ search: { ...search, brand: next } })}
          maxVisible={MAX_VISIBLE_BRANDS}
        />
        <AccountMenu brand={brand} onLogout={() => setSignedOut(true)} />
      </header>
      <main id="home-context-main" className="hc-main">
        {!recovered && (search.state === "loading" || search.state === "error") ? (
          <section className="hc-evidence-main hc-management">
            <Heading view={view} />
            <p role={search.state === "error" ? "alert" : "status"}>
              {search.state === "error" ? C.loadError : C.loading}
            </p>
            {search.state === "error" && (
              <Button variant="ghost" onClick={() => setRecovered(true)}>
                {C.retry}
              </Button>
            )}
          </section>
        ) : view === "visibility" ? (
          <>
            <h1 id="hc-page-heading" tabIndex={-1} className="sr-only">
              {C.visibility}
            </h1>
            {search.state === "empty" || search.state === "review" ? (
              <div className="hc-evidence">
                <section className="hc-evidence-main hc-management">
                  <p>{C.emptyMentions}</p>
                  <TrackingStatus data={data} brand={brand} />
                </section>
              </div>
            ) : (
              <HomeSignal
                key={brand}
                brand={brand}
                topic={brand === "deja" ? "alternatives" : "discovery"}
                engines={ENGINES.map((engine) => engine.id)}
                days={periods[brand] ?? "30"}
                onDaysChange={(days) => setPeriods((current) => ({ ...current, [brand]: days }))}
              />
            )}
          </>
        ) : view === "mentions" ? (
          <Mentions key={`${brand}:${view}`} brand={brand} data={data} scenario={search.state} />
        ) : view === "settings" ? (
          <section className="hc-settings">
            <Heading view={view} />
            <SettingsForm
              key={brand}
              brand={data.settings}
              onSave={(request) => state.saveBrand(brand, request)}
            />
          </section>
        ) : (
          <Management
            key={`${brand}:${view}`}
            kind={view}
            brand={brand}
            state={state}
            scenario={search.state}
          />
        )}
      </main>
      <div className="hc-dock-group">
        <nav className="hc-dock" aria-label={C.navigation}>
          {DESTINATIONS.map((destination) => (
            <Link
              key={destination.view}
              from="/dev/explorations/home-context"
              to="/dev/explorations/home-context"
              search={{ ...search, view: destination.view, brand }}
              aria-current={view === destination.view ? "page" : undefined}
              aria-label={C[destination.view]}
              title={C[destination.view]}
            >
              <Icon name={destination.icon} />
              <span className="hc-dock-label">{C[destination.view]}</span>
            </Link>
          ))}
        </nav>
      </div>
    </div>
  );
}

function Heading({ view, children }: { view: ExplorationView; children?: ReactNode }) {
  return (
    <div className="hc-section-heading hc-view-heading">
      <h1 id="hc-page-heading" tabIndex={-1}>
        {C[view]}
      </h1>
      <div className="hc-actions">{children}</div>
    </div>
  );
}

function TrackingStatus({ data, brand }: { data: PreviewBrand; brand: HomeBrand }) {
  return (
    <div className="hc-tracking-status">
      <p>
        {!data.prompts.some((item) => item.confirmed)
          ? C.promptsRequired
          : data.reviewed.prompts && data.reviewed.competitors
            ? C.trackingReady
            : C.reviewRequired}
      </p>
      <div className="hc-actions">
        {!data.reviewed.prompts && (
          <Link
            from="/dev/explorations/home-context"
            to="/dev/explorations/home-context"
            search={(previous) => ({ ...previous, view: "prompts", brand })}
            className="hc-text-link"
          >
            {C.reviewPromptsLink}
          </Link>
        )}
        {!data.reviewed.competitors && (
          <Link
            from="/dev/explorations/home-context"
            to="/dev/explorations/home-context"
            search={(previous) => ({ ...previous, view: "competitors", brand })}
            className="hc-text-link"
          >
            {C.reviewCompetitorsLink}
          </Link>
        )}
      </div>
    </div>
  );
}

function Mentions({
  brand,
  data,
  scenario,
}: {
  brand: HomeBrand;
  data: PreviewBrand;
  scenario?: ExplorationState;
}) {
  const [day, setDay] = useState(89);
  const [engine, setEngine] = useState("all");
  const [outcome, setOutcome] = useState("all");
  const [page, setPage] = useState(1);
  const answers = collectionAnswers(brand, day, scenario);
  const collectionState = day === 89 ? scenario : undefined;
  const pairs = answers.map((answer, index) => ({
    answer,
    result: resultForState(answer, brand, index, collectionState),
  }));
  const filtered = pairs.filter(
    ({ result }) =>
      (engine === "all" || result.engine === engine) &&
      (outcome === "all" || (result.outcome ?? "pending") === outcome),
  );
  const response = pageOf(filtered, page, 40);
  const collections = Array.from({ length: 90 }, (_, index) => ({
    value: String(89 - index),
    label: new Date(Date.UTC(2026, 6, 16 + 89 - index)).toISOString().slice(0, 10),
  }));

  function clearFilters() {
    setEngine("all");
    setOutcome("all");
    setPage(1);
  }

  return (
    <section className="hc-mentions">
      <Heading view="mentions">
        {answers.length > 0 && (
          <LocalChoice
            label={C.collection}
            value={String(day)}
            options={collections}
            onChange={(value) => {
              setDay(Number(value));
              setPage(1);
            }}
          />
        )}
        <DropdownMenu>
          <DropdownMenuTrigger render={<Button variant="ghost" size="sm" />}>
            {C.filters}
            {engine !== "all" || outcome !== "all"
              ? ` (${Number(engine !== "all") + Number(outcome !== "all")})`
              : ""}
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="hc-brand-menu">
            <DropdownMenuLabel>{C.engine}</DropdownMenuLabel>
            <DropdownMenuRadioGroup
              value={engine}
              onValueChange={(value) => {
                setEngine(value);
                setPage(1);
              }}
            >
              <DropdownMenuRadioItem value="all">{C.all}</DropdownMenuRadioItem>
              {ENGINES.map((item) => (
                <DropdownMenuRadioItem key={item.id} value={item.id}>
                  {item.name}
                </DropdownMenuRadioItem>
              ))}
            </DropdownMenuRadioGroup>
            <DropdownMenuLabel>{C.outcome}</DropdownMenuLabel>
            <DropdownMenuRadioGroup
              value={outcome}
              onValueChange={(value) => {
                setOutcome(value);
                setPage(1);
              }}
            >
              <DropdownMenuRadioItem value="all">{C.all}</DropdownMenuRadioItem>
              {(["mentioned", "cited", "miss", "error", "pending"] as const).map((value) => (
                <DropdownMenuRadioItem key={value} value={value}>
                  {C[value]}
                </DropdownMenuRadioItem>
              ))}
            </DropdownMenuRadioGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </Heading>
      {collectionState === "running" || collectionState === "partial" ? (
        <output className="hc-collection-status">
          {copy(C.runProgress, {
            completed: pairs.filter(({ result }) => result.outcome !== null).length,
            total: pairs.length,
          })}
          {collectionState === "partial" && <span>{C.runFailed}</span>}
        </output>
      ) : null}
      {answers.length === 0 ? (
        <div className="hc-evidence-main hc-management">
          <p>{C.emptyMentions}</p>
          <TrackingStatus data={data} brand={brand} />
        </div>
      ) : response.items.length ? (
        <AnswerFeed
          key={`${day}:${engine}:${outcome}:${response.currentPage}`}
          brand={brand}
          answers={response.items.map((item) => item.answer)}
          results={response.items.map((item) => item.result)}
          stopped={collectionState === "partial"}
        />
      ) : (
        <div className="hc-empty">
          <p>{C.noMatches}</p>
          <Button variant="ghost" onClick={clearFilters}>
            {C.clearFilters}
          </Button>
        </div>
      )}
      <ListPagination page={response} pageSize={40} onPage={setPage} />
    </section>
  );
}

function Management({
  kind,
  brand,
  state,
  scenario,
}: {
  kind: "prompts" | "competitors";
  brand: HomeBrand;
  state: ReturnType<typeof useExplorationState>;
  scenario?: ExplorationState;
}) {
  const data = state.brands[brand];
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("all");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [editor, setEditor] = useState<{ item?: Prompt | Competitor }>();
  const [deleting, setDeleting] = useState<Prompt | Competitor>();
  const [deleteFailed, setDeleteFailed] = useState(false);
  const [draftMessage, setDraftMessage] = useState("");
  const [draftRetried, setDraftRetried] = useState(false);
  const trigger = useRef<HTMLElement | null>(null);
  const text = query.trim().toLowerCase();
  const match = (item: Prompt | Competitor) =>
    (status === "all" || item.confirmed === (status === "confirmed")) &&
    ("text" in item ? item.text : `${item.name} ${item.domain} ${item.githubRepo}`)
      .toLowerCase()
      .includes(text);
  const promptPage = pageOf(data.prompts.filter(match), page, pageSize);
  const competitorPage = pageOf(data.competitors.filter(match), page, pageSize);
  const response = kind === "prompts" ? promptPage : competitorPage;
  const hasItems = data[kind].length > 0;
  const addLabel = kind === "prompts" ? C.addPrompt : C.addCompetitor;
  const initial = !data.reviewed.prompts && !data.reviewed.competitors;
  const draftBusy = scenario === "draft-queued" || scenario === "draft-running";
  const draftFailed = scenario === "draft-failed" && !draftRetried;
  const dismissing =
    deleting && "source" in deleting && deleting.source === "observed" && !deleting.confirmed;

  function remember() {
    trigger.current = document.activeElement as HTMLElement;
  }
  function edit(item?: Prompt | Competitor) {
    remember();
    setEditor({ item });
  }
  function remove(item: Prompt | Competitor) {
    remember();
    setDeleteFailed(false);
    setDeleting(item);
  }
  function finalFocus() {
    return trigger.current?.isConnected
      ? trigger.current
      : document.getElementById("hc-page-heading");
  }
  function draft() {
    setDraftRetried(true);
    const count = initial
      ? state.draft(brand, "prompts") + state.draft(brand, "competitors")
      : state.draft(brand, kind);
    setDraftMessage(count ? copy(C.draftDone, { count }) : C.draftEmpty);
  }

  return (
    <section className="hc-evidence-main hc-management">
      <Heading view={kind}>
        <Button size="sm" variant="ghost" onClick={() => edit()}>
          <Icon name="add" size={16} />
          {addLabel}
        </Button>
      </Heading>
      {!data.reviewed[kind] && (
        <div className="hc-review">
          <p>{kind === "prompts" ? C.reviewPrompts : C.reviewCompetitors}</p>
          <Button
            size="sm"
            variant="outline"
            disabled={kind === "prompts" && data.prompts.length === 0}
            onClick={() => {
              state.review(brand, kind);
              setDraftMessage(C.reviewComplete);
            }}
          >
            {C.finishReview}
          </Button>
        </div>
      )}
      <div className="hc-list-toolbar">
        <Input
          aria-label={kind === "prompts" ? C.searchPrompts : C.searchCompetitors}
          placeholder={kind === "prompts" ? C.searchPrompts : C.searchCompetitors}
          value={query}
          onChange={(event) => {
            setQuery(event.target.value);
            setPage(1);
          }}
        />
        <LocalChoice
          label={C.status}
          value={status}
          options={["all", "confirmed", "unconfirmed"].map((value) => ({
            value,
            label: C[value as "all" | "confirmed" | "unconfirmed"],
          }))}
          onChange={(value) => {
            setStatus(value);
            setPage(1);
          }}
        />
        {scenario !== "optional-off" && (
          <Button size="sm" variant="ghost" disabled={draftBusy} onClick={draft}>
            {draftFailed
              ? C.retry
              : initial
                ? C.draftInitial
                : kind === "prompts"
                  ? C.draftPrompts
                  : C.draftCompetitors}
          </Button>
        )}
      </div>
      <output className="hc-list-status">
        {scenario === "optional-off"
          ? C.optionalOff
          : draftBusy
            ? scenario === "draft-queued"
              ? C.draftQueued
              : C.draftRunning
            : draftFailed
              ? C.draftFailed
              : draftMessage}
      </output>
      {response.items.length ? (
        kind === "prompts" ? (
          <PromptList
            items={promptPage.items}
            onEdit={edit}
            onConfirm={(id) => state.confirm(brand, kind, id)}
            onDelete={remove}
          />
        ) : (
          <CompetitorList
            items={competitorPage.items}
            onEdit={edit}
            onConfirm={(id) => state.confirm(brand, kind, id)}
            onDelete={remove}
          />
        )
      ) : (
        <div className="hc-empty">
          <p>{hasItems ? C.noMatches : kind === "prompts" ? C.emptyPrompts : C.emptyCompetitors}</p>
          {hasItems ? (
            <Button
              variant="ghost"
              onClick={() => {
                setQuery("");
                setStatus("all");
                setPage(1);
              }}
            >
              {C.clearFilters}
            </Button>
          ) : (
            <Button variant="ghost" onClick={() => edit()}>
              {addLabel}
            </Button>
          )}
        </div>
      )}
      {hasItems && (
        <ListPagination
          page={response}
          pageSize={pageSize}
          onPage={setPage}
          onPageSize={(value) => {
            setPageSize(value);
            setPage(1);
          }}
        />
      )}
      {editor &&
        (kind === "prompts" ? (
          <PromptEditor
            prompt={editor.item as Prompt | undefined}
            onClose={() => setEditor(undefined)}
            onSave={(value) => state.savePrompt(brand, value, editor.item?.id)}
            finalFocus={finalFocus}
          />
        ) : (
          <CompetitorEditor
            competitor={editor.item as Competitor | undefined}
            lookup={scenario !== "optional-off"}
            onClose={() => setEditor(undefined)}
            onSave={(value) => state.saveCompetitor(brand, value, editor.item?.id)}
            finalFocus={finalFocus}
          />
        ))}
      <DeleteItemDialog
        itemID={deleting?.id}
        title={
          dismissing
            ? C.dismissTitle
            : kind === "prompts"
              ? C.deletePromptTitle
              : C.deleteCompetitorTitle
        }
        description={dismissing ? C.dismissDescription : C.historyPreserved}
        actionLabel={
          dismissing ? C.dismiss : kind === "prompts" ? C.deletePrompt : C.deleteCompetitor
        }
        onClose={() => setDeleting(undefined)}
        pending={false}
        failed={deleteFailed}
        onDelete={async (id) => {
          try {
            state.remove(brand, kind, id);
          } catch (error) {
            setDeleteFailed(true);
            throw error;
          }
        }}
        finalFocus={finalFocus}
      />
    </section>
  );
}
