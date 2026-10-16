import { createFileRoute, Link, Outlet, redirect, useLocation } from "@tanstack/react-router";
import { type ReactNode, useLayoutEffect, useMemo, useRef, useState } from "react";
import type { Competitor } from "@/api/competitors.api";
import type { Prompt } from "@/api/prompts.api";
import { DeleteItemDialog } from "@/components/blocks/delete-item-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { AccountMenu } from "./-components/account-menu";
import { BrandSwitcher } from "./-components/brand-switcher";
import { Icon } from "./-components/common";
import { CompetitorEditor } from "./-components/competitor-editor";
import { CompetitorList } from "./-components/competitor-list";
import { HomeSignal } from "./-components/home-signal";
import { WorkspaceContext } from "./-components/home-workspace-context";
import { copy, ListPagination, LocalChoice } from "./-components/list-controls";
import { PromptDetail } from "./-components/prompt-detail";
import { PromptEditor } from "./-components/prompt-editor";
import { PromptList } from "./-components/prompt-list";
import { SettingsForm } from "./-components/settings-form";
import { useExplorationState } from "./-hooks/use-exploration-state";
import {
  type ExplorationSearch,
  type ExplorationState,
  type ExplorationView,
  type PreviewBrand,
  pageOf,
  parseExplorationSearch,
} from "./-lib/exploration-fixtures";
import { HOME_BRANDS, HOME_LOGOS, type HomeBrand } from "./-lib/home-evidence";
import {
  buildEvidence,
  type EvidenceRecord,
  filterPromptRows,
  type PromptRow,
  promptRows,
} from "./-lib/prompt-visibility";
import C from "./-locales/mvp.en.json";
import V from "./-locales/prompt-visibility.en.json";
import "./-styles.css";
import "./-home-context.css";

export const Route = createFileRoute("/dev/explorations/home-context")({
  validateSearch: parseExplorationSearch,
  beforeLoad: ({ location, search }) => {
    if (location.pathname.replace(/\/$/, "") === HOME_PATH && search.view === "competitors") {
      throw redirect({
        to: COMPETITORS_PATH,
        search: { ...search, view: undefined, promptId: undefined, answerId: undefined },
        replace: true,
      });
    }
  },
  component: HomeContext,
});

type Theme = "light" | "dark";

const HOME_PATH = "/dev/explorations/home-context";
const COMPETITORS_PATH = "/dev/explorations/home-context/competitors";
const THEME_STORAGE_KEY = "koi:home-context:theme:v1";

function HomeContext() {
  const { state } = Route.useSearch();
  const [theme, setTheme] = useState<Theme>(() => {
    try {
      return localStorage.getItem(THEME_STORAGE_KEY) === "light" ? "light" : "dark";
    } catch {
      return "dark";
    }
  });

  function changeTheme(next: Theme) {
    setTheme(next);

    try {
      localStorage.setItem(THEME_STORAGE_KEY, next);
    } catch {
      // The preference still works for this visit when browser storage is unavailable.
    }
  }

  return <Workspace key={state ?? "populated"} theme={theme} onThemeChange={changeTheme} />;
}

function Workspace({
  theme,
  onThemeChange,
}: {
  theme: Theme;
  onThemeChange: (theme: Theme) => void;
}) {
  const search = Route.useSearch();
  const navigate = Route.useNavigate();
  const pathname = useLocation({ select: (location) => location.pathname });
  const competitorPage = pathname.replace(/\/$/, "") === COMPETITORS_PATH;
  const brand = search.brand ?? "deja";
  const view = competitorPage
    ? "competitors"
    : search.view === "settings"
      ? "settings"
      : "visibility";
  const state = useExplorationState(search.state);
  const [signedOut, setSignedOut] = useState(false);
  const [recovered, setRecovered] = useState(false);
  const preferences = useRef<
    Partial<Record<HomeBrand, Pick<ExplorationSearch, "days" | "engines">>>
  >({});
  const listEntry = useRef<{ search: ExplorationSearch; id: string; scroll: number } | null>(null);
  const previousPage = useRef({ brand, view, promptId: search.promptId, signedOut });
  const data = state.brands[brand];
  const days = search.days ?? 30;
  const records = useMemo(() => buildEvidence(brand, search.state), [brand, search.state]);
  const rows = useMemo(
    () => promptRows(records, brand, data, days, search.engines),
    [records, brand, data, days, search.engines],
  );
  const selected = search.promptId
    ? rows.find((row) => row.prompt.id === search.promptId)
    : undefined;
  const options = (Object.keys(HOME_BRANDS) as HomeBrand[]).map((id) => ({
    id,
    name: state.brands[id].settings.name,
    ...HOME_LOGOS[HOME_BRANDS[id].name],
  }));

  function update(patch: Partial<ExplorationSearch>) {
    void navigate({
      search: (previous) => ({ ...previous, ...patch }),
      replace: true,
      resetScroll: false,
    });
  }

  const listSearch = {
    ...(listEntry.current?.search ?? search),
    view: "visibility" as const,
    brand,
    promptId: undefined,
    answerId: undefined,
    responseEngine: undefined,
  };
  const back = (
    <Link
      from="/dev/explorations/home-context"
      to="/dev/explorations/home-context"
      search={listSearch}
      className="hc-text-link"
      onClick={(event) => {
        if (
          listEntry.current &&
          !event.metaKey &&
          !event.ctrlKey &&
          !event.shiftKey &&
          !event.altKey &&
          event.button === 0
        ) {
          event.preventDefault();
          window.history.back();
        }
      }}
    >
      {V.allPrompts}
    </Link>
  );

  useLayoutEffect(() => {
    const root = document.documentElement;
    const previous = root.getAttribute("data-exploration-theme");
    const dark = root.classList.contains("dark");
    const scheme = root.style.colorScheme;
    const title = document.title;

    root.setAttribute("data-exploration-theme", "home-context");

    return () => {
      if (previous) root.setAttribute("data-exploration-theme", previous);
      else root.removeAttribute("data-exploration-theme");
      root.classList.toggle("dark", dark);
      root.style.colorScheme = scheme;
      document.title = title;
    };
  }, []);

  useLayoutEffect(() => {
    const root = document.documentElement;

    root.setAttribute("data-home-context-theme-switching", "");
    root.classList.toggle("dark", theme === "dark");
    root.style.colorScheme = theme;
    root.getBoundingClientRect();

    const frame = requestAnimationFrame(() => {
      root.removeAttribute("data-home-context-theme-switching");
    });

    return () => {
      cancelAnimationFrame(frame);
      root.removeAttribute("data-home-context-theme-switching");
    };
  }, [theme]);

  useLayoutEffect(() => {
    document.title = `${view === "visibility" && selected ? selected.prompt.text : C[view]} · Koi`;
    const previous = previousPage.current;
    if (
      previous.brand !== brand ||
      previous.view !== view ||
      previous.promptId !== search.promptId ||
      previous.signedOut !== signedOut
    ) {
      const returning =
        previous.brand === brand && previous.promptId && !search.promptId && view === "visibility";
      const link = returning
        ? [...document.querySelectorAll<HTMLElement>("[data-prompt-id]")].find(
            (element) =>
              element.dataset.promptId === previous.promptId && element.offsetParent !== null,
          )
        : undefined;
      (link ?? document.getElementById("hc-page-heading"))?.focus({ preventScroll: true });
      if (returning && listEntry.current)
        window.scrollTo({ top: listEntry.current.scroll, behavior: "instant" });
      previousPage.current = { brand, view, promptId: search.promptId, signedOut };
    }
  }, [brand, view, search.promptId, signedOut, selected]);

  const signal = (
    <HomeSignal
      key={`${brand}:${selected?.prompt.id ?? "all"}`}
      brand={brand}
      data={data}
      records={records}
      scope={{
        days,
        engines: search.engines,
        promptIds: selected
          ? [selected.prompt.id]
          : data.prompts.filter((item) => item.confirmed).map((item) => item.id),
      }}
      compareId={search.compareId}
      onCompare={(compareId) => update({ compareId })}
      onDaysChange={(next) => update({ days: next, answerId: undefined })}
      onEnginesChange={(engines) =>
        update({
          engines: engines.length ? engines : undefined,
          responseEngine: undefined,
          answerId: undefined,
          page: undefined,
        })
      }
      onManage={() =>
        void navigate({
          to: COMPETITORS_PATH,
          search: { ...search, view: undefined, promptId: undefined, answerId: undefined },
        })
      }
    />
  );

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
          maxVisible={3}
          onChange={(next) => {
            preferences.current[brand] = { days: search.days, engines: search.engines };
            listEntry.current = null;
            void navigate({
              to: competitorPage ? COMPETITORS_PATH : HOME_PATH,
              search: { brand: next, state: search.state, ...preferences.current[next] },
            });
          }}
        />
        <div className="hc-header-actions">
          <TooltipProvider delay={350}>
            <Tooltip>
              <TooltipTrigger
                render={
                  <Link
                    to={COMPETITORS_PATH}
                    search={{
                      ...search,
                      view: undefined,
                      promptId: undefined,
                      answerId: undefined,
                    }}
                    className="hc-header-link"
                    aria-label={C.competitors}
                    aria-current={competitorPage ? "page" : undefined}
                  />
                }
              >
                <Icon name="competitors" size={16} />
              </TooltipTrigger>
              <TooltipContent side="bottom" sideOffset={8} className="hc-brand-tooltip">
                {C.competitors}
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
          <AccountMenu
            brand={brand}
            darkMode={theme === "dark"}
            onDarkModeChange={(checked) => onThemeChange(checked ? "dark" : "light")}
            onLogout={() => setSignedOut(true)}
          />
        </div>
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
        ) : view === "settings" ? (
          <section className="hc-settings">
            <Link
              from="/dev/explorations/home-context"
              to="/dev/explorations/home-context"
              search={{ ...search, view: "visibility", promptId: undefined }}
              className="hc-text-link"
            >
              {V.back}
            </Link>
            <Heading view="settings" />
            <SettingsForm
              key={brand}
              brand={data.settings}
              onSave={(request) => state.saveBrand(brand, request)}
            />
          </section>
        ) : competitorPage ? (
          <WorkspaceContext.Provider value={{ brand, state, search, rows, update }}>
            <Outlet />
          </WorkspaceContext.Provider>
        ) : search.promptId ? (
          selected ? (
            <PromptDetail
              key={`${brand}:${selected.prompt.id}`}
              row={selected}
              records={records}
              brand={brand}
              brandName={data.settings.name}
              search={search}
              onSearch={update}
              onSave={(text, id) => state.savePrompt(brand, text, id)}
              back={back}
            >
              {signal}
            </PromptDetail>
          ) : (
            <section className="hc-empty">
              <h1 id="hc-page-heading" tabIndex={-1}>
                {V.promptMissing}
              </h1>
              {back}
            </section>
          )
        ) : (
          <div className="hc-visibility-page">
            <h1 id="hc-page-heading" tabIndex={-1} className="sr-only">
              {C.visibility}
            </h1>
            {signal}
            {(search.state === "empty" || search.state === "review") && (
              <TrackingStatus data={data} brand={brand} />
            )}
            <Management
              key={`${brand}:prompts`}
              kind="prompts"
              brand={brand}
              state={state}
              scenario={search.state}
              rows={rows}
              records={records}
              search={search}
              onSearch={update}
              onOpen={(id) => {
                listEntry.current = { search, id, scroll: window.scrollY };
              }}
            />
          </div>
        )}
        <p className="hc-preview-note">{V.previewLabel}</p>
      </main>
    </div>
  );
}

function Heading({
  view,
  children,
  secondary = false,
}: {
  view: ExplorationView;
  children?: ReactNode;
  secondary?: boolean;
}) {
  return (
    <div className="hc-section-heading hc-view-heading">
      {secondary ? (
        <h2>{C[view]}</h2>
      ) : (
        <h1 id="hc-page-heading" tabIndex={-1}>
          {C[view]}
        </h1>
      )}
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
          <a href="#hc-prompts" className="hc-text-link">
            {C.reviewPromptsLink}
          </a>
        )}
        {!data.reviewed.competitors && (
          <Link
            from="/dev/explorations/home-context"
            to={COMPETITORS_PATH}
            search={(previous) => ({
              ...previous,
              view: undefined,
              brand,
              promptId: undefined,
              answerId: undefined,
            })}
            className="hc-text-link"
          >
            {C.reviewCompetitorsLink}
          </Link>
        )}
      </div>
    </div>
  );
}

export function Management({
  kind,
  brand,
  state,
  scenario,
  rows,
  records,
  search,
  onSearch,
  onOpen,
}: {
  kind: "prompts" | "competitors";
  brand: HomeBrand;
  state: ReturnType<typeof useExplorationState>;
  scenario?: ExplorationState;
  rows: PromptRow[];
  records?: EvidenceRecord[];
  search: ExplorationSearch;
  onSearch: (patch: Partial<ExplorationSearch>) => void;
  onOpen: (id: string) => void;
}) {
  const data = state.brands[brand];
  const [localQuery, setLocalQuery] = useState("");
  const [localStatus, setLocalStatus] = useState("all");
  const [localPage, setLocalPage] = useState(1);
  const [localPageSize, setLocalPageSize] = useState(20);
  const query = kind === "prompts" ? (search.q ?? "") : localQuery;
  const status = kind === "prompts" ? (search.status ?? "all") : localStatus;
  const page = kind === "prompts" ? (search.page ?? 1) : localPage;
  const pageSize = kind === "prompts" ? (search.pageSize ?? 20) : localPageSize;
  const setQuery = (q: string) =>
    kind === "prompts" ? onSearch({ q, page: 1 }) : setLocalQuery(q);
  const setStatus = (value: string) =>
    kind === "prompts"
      ? onSearch({ status: value as ExplorationSearch["status"], page: 1 })
      : setLocalStatus(value);
  const setPage = (value: number) =>
    kind === "prompts" ? onSearch({ page: value }) : setLocalPage(value);
  const setPageSize = (value: number) =>
    kind === "prompts"
      ? onSearch({ pageSize: value as 10 | 20 | 50, page: 1 })
      : setLocalPageSize(value);
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
  const promptPage = pageOf(filterPromptRows(rows, search), page, pageSize);
  const competitorPage = pageOf(data.competitors.filter(match), page, pageSize);
  const response = kind === "prompts" ? promptPage : competitorPage;
  const hasItems = kind === "prompts" ? rows.length > 0 : data.competitors.length > 0;
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
    <section
      className="hc-evidence-main hc-management"
      id={kind === "prompts" ? "hc-prompts" : undefined}
    >
      <Heading view={kind} secondary={kind === "prompts"}>
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
            if (kind === "competitors") setPage(1);
          }}
        />
        <LocalChoice
          label={C.status}
          value={status}
          options={[
            ...["all", "confirmed", "unconfirmed"].map((value) => ({
              value,
              label: C[value as "all" | "confirmed" | "unconfirmed"],
            })),
            ...(kind === "prompts" ? [{ value: "past", label: V.pastPrompts }] : []),
          ]}
          onChange={(value) => {
            setStatus(value);
            if (kind === "competitors") setPage(1);
          }}
        />
        {kind === "prompts" && (
          <>
            <LocalChoice
              label={V.sort}
              value={search.sort ?? "original"}
              options={[
                { value: "original", label: V.original },
                { value: "prompt", label: V.sortPrompt },
                { value: "rate", label: V.sortRate },
                { value: "change", label: V.sortChange },
              ]}
              onChange={(value) => onSearch({ sort: value as ExplorationSearch["sort"], page: 1 })}
            />
            {search.sort && search.sort !== "original" && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onSearch({ descending: !search.descending, page: 1 })}
              >
                {search.descending ? V.descending : V.ascending}
              </Button>
            )}
          </>
        )}
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
            key={`${brand}:${search.days}:${search.engines?.join(",")}:${search.q}:${search.status}:${search.sort}:${search.descending}:${page}:${pageSize}`}
            rows={promptPage.items}
            records={records ?? []}
            brand={brand}
            search={search}
            onOpen={onOpen}
            onEdit={edit}
            onConfirm={(id) => state.confirm(brand, kind, id)}
            onDelete={remove}
            onConfirmMany={(ids) => state.confirmPrompts(brand, ids)}
            onDeleteMany={async (ids) => state.removePrompts(brand, ids)}
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
                if (kind === "prompts") onSearch({ q: undefined, status: undefined, page: 1 });
                else {
                  setQuery("");
                  setStatus("all");
                  setPage(1);
                }
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
            if (kind === "competitors") setPage(1);
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
