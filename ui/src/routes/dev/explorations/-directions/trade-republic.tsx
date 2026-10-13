import { Link, useNavigate } from "@tanstack/react-router";
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import GridLayout, { type Layout, noCompactor, useContainerWidth } from "react-grid-layout";
import { gridBounds, type LayoutConstraint, minMaxSize } from "react-grid-layout/core";
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
import { Account, C, Choice, Filters, Icon, StateGate } from "../-components/common";
import type { Section } from "../-lib/data";
import { useMock } from "../-lib/state";
import { TradeRepublicPanel } from "./-panel";
import {
  addPanel,
  availableSlot,
  defaultWorkspaces,
  GRID_COLS,
  GRID_ROWS,
  layoutFits,
  makeWorkspace,
  PANELS,
  type Panel,
  readWorkspaces,
  reorderPanel,
  resizePanel,
  STORAGE_KEY,
  type Template,
  type Workspace,
} from "./-workspace";

const BOUNDED_COMPACTOR = { ...noCompactor, preventCollision: true };
const CANVAS_SIZE: LayoutConstraint = {
  name: "finite-canvas",
  constrainSize: (item, w, h) => ({
    w: Math.min(w, GRID_COLS - item.x),
    h: Math.min(h, GRID_ROWS - item.y),
  }),
};
const CONSTRAINTS = [gridBounds, minMaxSize, CANVAS_SIZE];

export default function TradeRepublic({ workspaceId }: { section: Section; workspaceId?: string }) {
  const { setNotice, summary } = useMock();
  const navigate = useNavigate({ from: "/dev/explorations/$direction/$section" });
  const [workspaces, setWorkspaces] = useState(readWorkspaces);
  const [creator, setCreator] = useState(false);
  const [rename, setRename] = useState(false);
  const [name, setName] = useState("");
  const [focusedPanel, setFocusedPanel] = useState<Panel | null>(null);
  const [height, setHeight] = useState(0);
  const [revision, setRevision] = useState(0);
  const sequence = useRef(0);
  const active = workspaces.find((view) => view.id === workspaceId) ?? workspaces[0];
  const { width, containerRef, mounted } = useContainerWidth();
  const mobile = width < 768;
  const focused =
    focusedPanel && active.panels.includes(focusedPanel) ? focusedPanel : active.panels[0];
  const slot = availableSlot(active.layout);
  const selectedPrompt = active.selectedPrompt;
  const setSelectedPrompt = (id: string | null) => update({ ...active, selectedPrompt: id });

  useLayoutEffect(() => {
    document.documentElement.setAttribute("data-tr-workspace", "");
    const node = containerRef.current;
    const observer = new ResizeObserver(([entry]) => setHeight(entry.contentRect.height));
    if (node) observer.observe(node);
    return () => {
      observer.disconnect();
      document.documentElement.removeAttribute("data-tr-workspace");
    };
  }, [containerRef]);

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
      params: { direction: "trade-republic", section: "overview" },
      search: (previous) => ({ ...previous, workspace: workspace.id }),
    });
  }

  function create(template: Template) {
    let id = `workspace-${++sequence.current}`;
    while (workspaces.some((workspace) => workspace.id === id))
      id = `workspace-${++sequence.current}`;
    const title =
      template === "research"
        ? C.templateResearch
        : template === "compare"
          ? C.templateCompare
          : C.emptyView;
    const next = makeWorkspace(id, title, template);
    setWorkspaces((current) => [...current, next]);
    setCreator(false);
    select(next);
    setNotice(C.workspaceAdded);
  }

  function changeLayout(layout: Layout) {
    if (!layoutFits(layout)) {
      setRevision((value) => value + 1);
      return;
    }
    update({ ...active, layout: layout.map((item) => ({ ...item })) });
  }

  function move(panel: Panel, delta: number) {
    update(reorderPanel(active, panel, delta));
    setRevision((value) => value + 1);
    setNotice(C.panelMoved);
  }

  function resize(panel: Panel, dimension: "w" | "h", delta: number) {
    const next = resizePanel(active, panel, dimension, delta);
    update(next);
    setRevision((value) => value + 1);
    setNotice(next === active ? C.panelDoesNotFit : C.panelResized);
  }

  function panelNode(panel: Panel) {
    return (
      <section key={panel} className="tr-panel" data-panel={panel}>
        <header className="tr-panel-header">
          <span className="tr-drag-handle" title={C.dragPanel}>
            <i />
            {C.tally}
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
              <DropdownMenuItem onClick={() => resize(panel, "w", 1)}>{C.wider}</DropdownMenuItem>
              <DropdownMenuItem onClick={() => resize(panel, "w", -1)}>
                {C.narrower}
              </DropdownMenuItem>
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
                    layout: active.layout.filter((item) => item.i !== panel),
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
          <TradeRepublicPanel
            panel={panel}
            selectedPrompt={selectedPrompt}
            onSelectPrompt={setSelectedPrompt}
          />
        </div>
      </section>
    );
  }

  return (
    <div className="exp-tr tr-workspace">
      <header className="tr-header">
        <nav className="tr-tabs" aria-label={C.views}>
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
            aria-label={C.newView}
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
            <DropdownMenu>
              <DropdownMenuTrigger
                render={<Button variant="outline" />}
                aria-label={C.addPanel}
                disabled={!slot}
                title={!slot ? C.noPanelSpace : undefined}
              >
                <Icon name="add" size={14} />
                {C.addPanel}
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                {(Object.keys(PANELS) as Panel[]).map((panel) => (
                  <DropdownMenuItem
                    key={panel}
                    disabled={active.panels.includes(panel)}
                    onClick={() => {
                      update(addPanel(active, panel));
                      setFocusedPanel(panel);
                    }}
                  >
                    {PANELS[panel]}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
            <DropdownMenu>
              <DropdownMenuTrigger
                render={<Button variant="ghost" size="icon-lg" />}
                aria-label={C.viewActions}
              >
                <Icon name="more" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem
                  render={
                    <Link
                      to="/dev/explorations/trade-republic/panels"
                      search={{ state: "populated" }}
                    />
                  }
                >
                  {C.panelCatalog}
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => {
                    setName(active.name);
                    setRename(true);
                  }}
                >
                  {C.renameView}
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => {
                    update(makeWorkspace(active.id, active.name, active.template));
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
                  {C.closeView}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
        {!slot ? <p className="tr-capacity-hint">{C.noPanelSpace}</p> : null}
        {mobile && active.panels.length ? (
          <Choice
            label={C.activePanel}
            value={focused}
            options={active.panels.map((panel) => ({ value: panel, label: PANELS[panel] }))}
            onChange={(panel) => setFocusedPanel(panel as Panel)}
          />
        ) : null}
        <div ref={containerRef} className="tr-grid-container">
          <StateGate>
            {!active.panels.length ? (
              <div className="tr-empty-workspace">
                <Icon name="add" size={32} />
                <h2>{C.workspaceEmpty}</h2>
                <p>{C.viewEmptyLead}</p>
              </div>
            ) : mobile ? (
              panelNode(focused)
            ) : mounted && height > 0 ? (
              <GridLayout
                key={`${active.id}:${revision}`}
                width={width}
                layout={active.layout}
                autoSize={false}
                style={{ height }}
                gridConfig={{
                  cols: GRID_COLS,
                  maxRows: GRID_ROWS,
                  rowHeight: Math.max(1, (height - 8 * (GRID_ROWS - 1)) / GRID_ROWS),
                  margin: [8, 8],
                  containerPadding: [0, 0],
                }}
                dragConfig={{
                  enabled: true,
                  bounded: true,
                  handle: ".tr-drag-handle",
                  cancel: "button,a,input,[role=button]",
                }}
                resizeConfig={{ enabled: true, handles: ["se"] }}
                compactor={BOUNDED_COMPACTOR}
                constraints={CONSTRAINTS}
                onLayoutChange={changeLayout}
                onDragStop={() => setNotice(C.layoutSaved)}
                onResizeStop={() => setNotice(C.layoutSaved)}
              >
                {active.panels.map(panelNode)}
              </GridLayout>
            ) : null}
          </StateGate>
        </div>
      </main>
      <Dialog open={creator} onOpenChange={setCreator}>
        <DialogContent className="tr-workspace-dialog">
          <DialogHeader>
            <DialogTitle>{C.chooseView}</DialogTitle>
            <DialogDescription>{C.viewLead}</DialogDescription>
          </DialogHeader>
          <div className="tr-templates">
            {(
              [
                { id: "compare", title: C.templateCompare, lead: C.templateCompareLead },
                { id: "research", title: C.templateResearch, lead: C.templateResearchLead },
                { id: "empty", title: C.emptyView, lead: C.templateEmptyLead },
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
            <DialogTitle>{C.renameView}</DialogTitle>
            <DialogDescription>{C.viewName}</DialogDescription>
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
              <FieldLabel htmlFor="tr-workspace-name">{C.viewName}</FieldLabel>
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
