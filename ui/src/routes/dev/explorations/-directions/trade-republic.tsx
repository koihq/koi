import { useNavigate } from "@tanstack/react-router";
import { type ReactNode, useEffect, useRef, useState } from "react";
import GridLayout, { type Layout, useContainerWidth, verticalCompactor } from "react-grid-layout";
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
import { Field, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Account,
  BrandHeader,
  C,
  CompetitorList,
  Directory,
  EngineList,
  Filters,
  Icon,
  StateGate,
  TrendChart,
} from "../-components/common";
import {
  AnswerInspector,
  BriefScreen,
  CitationsScreen,
  Heatmap,
  ManagementScreen,
  MentionsScreen,
  PromptScreener,
} from "../-components/screens";
import type { Section } from "../-lib/data";
import { useMock } from "../-lib/state";
import {
  addPanel,
  defaultWorkspaces,
  makeWorkspace,
  PANELS,
  type Panel,
  readWorkspaces,
  reorderPanel,
  STORAGE_KEY,
  type Workspace,
} from "./-workspace";

export default function TradeRepublic({
  section,
  workspaceId,
}: {
  section: Section;
  workspaceId?: string;
}) {
  const { setNotice, filters, summary } = useMock();
  const navigate = useNavigate({ from: "/dev/explorations/$direction/$section" });
  const [workspaces, setWorkspaces] = useState(readWorkspaces);
  const [creator, setCreator] = useState(false);
  const [rename, setRename] = useState(false);
  const [name, setName] = useState("");
  const [directory, setDirectory] = useState(false);
  const [selectedPrompt, setSelectedPrompt] = useState<string | null>("prompt-9");
  const [revision, setRevision] = useState(0);
  const sequence = useRef(0);
  const active =
    workspaces.find((workspace) => workspace.id === (workspaceId ?? section)) ??
    workspaces.find((workspace) => workspace.section === section) ??
    makeWorkspace(section, C[section], section);
  const { width, containerRef, mounted } = useContainerWidth();
  const mobile = width < 768;
  const size = width < 1200 ? "tablet" : "desktop";
  const cols = size === "desktop" ? 12 : 6;

  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(workspaces));
    } catch {
      setNotice(C.unsavedLayout);
    }
  }, [workspaces, setNotice]);

  function update(next: Workspace) {
    setWorkspaces((current) => {
      const existing = current.find((workspace) => workspace.id === next.id);
      if (existing && JSON.stringify(existing) === JSON.stringify(next)) return current;
      return existing
        ? current.map((workspace) => (workspace.id === next.id ? next : workspace))
        : [...current, next];
    });
  }

  function select(workspace: Workspace) {
    void navigate({
      to: "/dev/explorations/$direction/$section",
      params: { direction: "trade-republic", section: workspace.section },
      search: (previous) => ({ ...previous, workspace: workspace.id }),
    });
  }

  function create(template: "overview" | "research" | "improve" | "empty") {
    let id = `workspace-${++sequence.current}`;
    while (workspaces.some((workspace) => workspace.id === id))
      id = `workspace-${++sequence.current}`;
    const title =
      template === "overview"
        ? C.templateOverview
        : template === "research"
          ? C.templateResearch
          : template === "improve"
            ? C.templateImprove
            : C.templateEmpty;
    const next = makeWorkspace(id, title, template === "improve" ? "brief" : "overview", template);
    setWorkspaces((current) => [...current, next]);
    setCreator(false);
    select(next);
    setNotice(C.workspaceAdded);
  }

  function changeLayout(layout: Layout) {
    update({ ...active, [size]: layout.map((item) => ({ ...item })) });
  }

  function move(panel: Panel, delta: number) {
    update(reorderPanel(active, panel, delta));
    setRevision((value) => value + 1);
    setNotice(C.panelMoved);
  }

  function resize(panel: Panel, dimension: "w" | "h", delta: number) {
    if (mobile) {
      update({
        ...active,
        mobileSizes: {
          ...active.mobileSizes,
          [panel]: Math.max(260, Math.min(900, (active.mobileSizes[panel] ?? 420) + delta * 60)),
        },
      });
    } else {
      update({
        ...active,
        [size]: active[size].map((item) =>
          item.i === panel
            ? {
                ...item,
                [dimension]: Math.max(
                  dimension === "w" ? (item.minW ?? 2) : (item.minH ?? 5),
                  Math.min(dimension === "w" ? cols - item.x : 30, item[dimension] + delta),
                ),
              }
            : item,
        ),
      });
    }
    setRevision((value) => value + 1);
    setNotice(C.panelResized);
  }

  function panelContent(panel: Panel): ReactNode {
    if (panel === "heatmap") return <Heatmap onSelect={setSelectedPrompt} />;
    if (panel === "screener")
      return <PromptScreener selected={selectedPrompt} onSelect={setSelectedPrompt} />;
    if (panel === "inspector") return <AnswerInspector promptId={selectedPrompt} />;
    if (panel === "trend") return <TrendChart />;
    if (panel === "watchlist") return <CompetitorList />;
    if (panel === "engines") return <EngineList />;
    if (panel === "brief") return <BriefScreen compact />;
    if (panel === "mentions") return <MentionsScreen onSelectPrompt={setSelectedPrompt} />;
    if (panel === "prompts" || panel === "competitors") return <ManagementScreen kind={panel} />;
    return <CitationsScreen />;
  }

  function panelNode(panel: Panel) {
    return (
      <section
        key={panel}
        className="tr-panel"
        data-panel={panel}
        style={
          mobile
            ? {
                height:
                  active.mobileSizes[panel] ??
                  (panel === "brief" || panel === "prompts" || panel === "mentions" ? 680 : 420),
              }
            : undefined
        }
      >
        <header className="tr-panel-header">
          <span className="tr-drag-handle" title={C.dragPanel}>
            <i />
            {C.tally}
            <span className="tr-panel-title">{PANELS[panel]}</span>
          </span>
          <DropdownMenu>
            <DropdownMenuTrigger
              render={<Button variant="ghost" size="icon-sm" />}
              aria-label={`${C.panelActions}: ${PANELS[panel]}`}
            >
              <Icon name="more" size={14} />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem
                disabled={active.panels.indexOf(panel) === 0}
                onClick={() => move(panel, -1)}
              >
                {C.moveUp}
              </DropdownMenuItem>
              <DropdownMenuItem
                disabled={active.panels.indexOf(panel) === active.panels.length - 1}
                onClick={() => move(panel, 1)}
              >
                {C.moveDown}
              </DropdownMenuItem>
              {!mobile ? (
                <>
                  <DropdownMenuItem onClick={() => resize(panel, "w", 1)}>
                    {C.wider}
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => resize(panel, "w", -1)}>
                    {C.narrower}
                  </DropdownMenuItem>
                </>
              ) : null}
              <DropdownMenuItem onClick={() => resize(panel, "h", 1)}>{C.taller}</DropdownMenuItem>
              <DropdownMenuItem onClick={() => resize(panel, "h", -1)}>
                {C.shorter}
              </DropdownMenuItem>
              <DropdownMenuItem
                variant="destructive"
                onClick={() => {
                  update({
                    ...active,
                    panels: active.panels.filter((item) => item !== panel),
                    desktop: active.desktop.filter((item) => item.i !== panel),
                    tablet: active.tablet.filter((item) => item.i !== panel),
                  });
                  setNotice(C.panelRemoved);
                }}
              >
                {C.removePanel}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </header>
        <div className="tr-panel-body">
          <h2>{PANELS[panel]}</h2>
          {panelContent(panel)}
        </div>
      </section>
    );
  }

  return (
    <div className="exp-tr">
      <header className="tr-header">
        <div className="tr-logo">
          <BrandHeader />
        </div>
        <nav className="tr-tabs" aria-label={C.navigation}>
          {workspaces.map((workspace) => (
            <button
              key={workspace.id}
              type="button"
              aria-current={active.id === workspace.id ? "page" : undefined}
              onClick={() => select(workspace)}
            >
              {workspace.name}
            </button>
          ))}
          <Button
            variant="ghost"
            size="icon-lg"
            aria-label={C.newWorkspace}
            onClick={() => setCreator(true)}
          >
            <Icon name="add" size={16} />
          </Button>
        </nav>
        <div className="tr-account">
          <span>
            {`${summary.presence.toFixed(1)}%`}
            <small>{C.presence}</small>
          </span>
          <Account />
        </div>
      </header>
      <main id="exploration-main" className="tr-main">
        <h1 className="sr-only">{active.name}</h1>
        <div className="tr-toolbar">
          <Filters scan />
          <div>
            <Button
              variant="ghost"
              size="icon-lg"
              aria-label={C.searchNavigation}
              onClick={() => setDirectory(true)}
            >
              <Icon name="search" size={16} />
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger render={<Button variant="outline" />} aria-label={C.addPanel}>
                <Icon name="add" size={14} />
                {C.addPanel}
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                {(Object.keys(PANELS) as Panel[]).map((panel) => (
                  <DropdownMenuItem
                    key={panel}
                    disabled={active.panels.includes(panel)}
                    onClick={() => update(addPanel(active, panel))}
                  >
                    {PANELS[panel]}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
            <DropdownMenu>
              <DropdownMenuTrigger
                render={<Button variant="ghost" size="icon-lg" />}
                aria-label={C.workspaceMenu}
              >
                <Icon name="more" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem
                  onClick={() => {
                    setName(active.name);
                    setRename(true);
                  }}
                >
                  {C.renameWorkspace}
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => {
                    update(makeWorkspace(active.id, active.name, active.section));
                    setRevision((value) => value + 1);
                    setNotice(C.layoutReset);
                  }}
                >
                  {C.resetLayout}
                </DropdownMenuItem>
                <DropdownMenuItem
                  disabled={workspaces.length <= 1}
                  variant="destructive"
                  onClick={() => {
                    const remaining = workspaces.filter((workspace) => workspace.id !== active.id);
                    setWorkspaces(remaining);
                    select(remaining[0]);
                    setNotice(C.workspaceClosed);
                  }}
                >
                  {C.closeWorkspace}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
        <StateGate>
          <div ref={containerRef} className="tr-grid-container">
            {!active.panels.length ? (
              <div className="tr-empty-workspace">
                <Icon name="add" size={32} />
                <h1>{C.workspaceEmpty}</h1>
                <p>{C.workspaceEmptyLead}</p>
              </div>
            ) : mounted ? (
              mobile ? (
                <div className="tr-mobile-panels">{active.panels.map(panelNode)}</div>
              ) : (
                <GridLayout
                  key={`${active.id}:${size}:${revision}`}
                  width={width}
                  layout={active[size]}
                  gridConfig={{ cols, rowHeight: 38, margin: [8, 8], containerPadding: [0, 0] }}
                  dragConfig={{
                    enabled: true,
                    handle: ".tr-drag-handle",
                    cancel: "button,a,input,[role=button]",
                  }}
                  resizeConfig={{ enabled: true, handles: ["se"] }}
                  compactor={verticalCompactor}
                  onLayoutChange={changeLayout}
                  onDragStop={() => setNotice(C.layoutSaved)}
                  onResizeStop={() => setNotice(C.layoutSaved)}
                >
                  {active.panels.map(panelNode)}
                </GridLayout>
              )
            ) : null}
          </div>
        </StateGate>
        <footer className="tr-footer">
          <span>{C.lastUpdated}</span>
          <span>{`${C.sample} · ${filters.days} days`}</span>
        </footer>
      </main>
      <Directory open={directory} onClose={() => setDirectory(false)} />
      <Dialog open={creator} onOpenChange={setCreator}>
        <DialogContent className="tr-workspace-dialog">
          <DialogHeader>
            <DialogTitle>{C.workspaceTemplates}</DialogTitle>
            <DialogDescription>{C.workspaceEmptyLead}</DialogDescription>
          </DialogHeader>
          <div className="tr-templates">
            {(
              [
                { id: "overview", title: C.templateOverview, lead: C.templateOverviewLead },
                { id: "research", title: C.templateResearch, lead: C.templateResearchLead },
                { id: "improve", title: C.templateImprove, lead: C.templateImproveLead },
                { id: "empty", title: C.templateEmpty, lead: C.templateEmptyLead },
              ] as const
            ).map((template) => (
              <button key={template.id} type="button" onClick={() => create(template.id)}>
                <div className={`tr-template-preview ${template.id}`} aria-hidden={true}>
                  {Array.from({ length: template.id === "empty" ? 1 : 5 }, (_, index) => (
                    <span key={String(index)}>
                      <i />
                      <i />
                      <i />
                    </span>
                  ))}
                </div>
                <strong>{template.title}</strong>
                <p>{template.lead}</p>
              </button>
            ))}
          </div>
        </DialogContent>
      </Dialog>
      <Dialog open={rename} onOpenChange={setRename}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{C.renameWorkspace}</DialogTitle>
            <DialogDescription>{C.workspaceName}</DialogDescription>
          </DialogHeader>
          <form
            onSubmit={(event) => {
              event.preventDefault();
              if (!name.trim()) return;
              update({ ...active, name: name.trim() });
              setRename(false);
              setNotice(C.workspaceRenamed);
            }}
          >
            <Field>
              <FieldLabel htmlFor="tr-workspace-name">{C.workspaceName}</FieldLabel>
              <Input
                id="tr-workspace-name"
                value={name}
                maxLength={60}
                required
                onChange={(event) => setName(event.target.value)}
              />
            </Field>
            <div className="exp-form-actions">
              <Button type="submit">{C.save}</Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}

export { defaultWorkspaces };
