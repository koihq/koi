import type { Layout, LayoutItem } from "react-grid-layout";
import C from "../-locales/en.json";

export const GRID_COLS = 12;
export const GRID_ROWS = 12;
export const MIN_PANEL = 2;
export const PANELS = {
  trend: C.visibility,
  prompts: C.prompts,
  competitors: C.competitors,
  engines: C.enginePerformance,
  mentions: C.mentions,
  inspector: C.context,
};
export type Panel = keyof typeof PANELS;
export type Template = "research" | "compare" | "empty";
export type Workspace = {
  id: string;
  name: string;
  template: Template;
  selectedPrompt: string | null;
  panels: Panel[];
  layout: LayoutItem[];
};
export const STORAGE_KEY = "koi-exploration:trade-republic:views:v3";

function overlaps(a: LayoutItem, b: LayoutItem) {
  return a.x < b.x + b.w && a.x + a.w > b.x && a.y < b.y + b.h && a.y + a.h > b.y;
}

export function layoutFits(layout: Layout): boolean {
  return layout.every(
    (item, index) =>
      item != null &&
      [item.x, item.y, item.w, item.h].every(Number.isInteger) &&
      item.x >= 0 &&
      item.y >= 0 &&
      item.w >= MIN_PANEL &&
      item.h >= MIN_PANEL &&
      item.x + item.w <= GRID_COLS &&
      item.y + item.h <= GRID_ROWS &&
      !layout.slice(index + 1).some((other) => other != null && overlaps(item, other)),
  );
}

export function availableSlot(layout: Layout, w = MIN_PANEL, h = MIN_PANEL) {
  for (let y = 0; y <= GRID_ROWS - h; y++) {
    for (let x = 0; x <= GRID_COLS - w; x++) {
      const candidate = { i: "candidate", x, y, w, h };
      if (!layout.some((item) => overlaps(candidate, item))) return { x, y, w, h };
    }
  }
  return null;
}

export function makeWorkspace(
  id: string,
  name: string,
  template: Template = "research",
): Workspace {
  const panels: Panel[] =
    template === "empty"
      ? []
      : template === "research"
        ? ["trend", "prompts", "inspector"]
        : ["trend", "competitors", "engines"];
  const layout =
    template === "empty"
      ? []
      : [
          { i: panels[0], x: 0, y: 0, w: 8, h: 7, minW: 2, minH: 2 },
          { i: panels[1], x: 0, y: 7, w: 8, h: 5, minW: 2, minH: 2 },
          { i: panels[2], x: 8, y: 0, w: 4, h: 12, minW: 2, minH: 2 },
        ];
  return { id, name, template, selectedPrompt: null, panels, layout };
}

export function defaultWorkspaces() {
  return [
    makeWorkspace("research", C.templateResearch),
    makeWorkspace("compare", C.templateCompare, "compare"),
  ];
}

export function validWorkspaces(value: unknown): value is Workspace[] {
  if (!Array.isArray(value) || value.length === 0 || value.length > 30) return false;
  const ids = new Set<string>();
  return value.every((view) => {
    if (
      !view ||
      typeof view.id !== "string" ||
      ids.has(view.id) ||
      typeof view.name !== "string" ||
      !view.name.trim() ||
      view.name.length > 60 ||
      !["research", "compare", "empty"].includes(view.template) ||
      (view.selectedPrompt !== null && typeof view.selectedPrompt !== "string") ||
      !Array.isArray(view.panels) ||
      view.panels.some((panel: string) => !Object.hasOwn(PANELS, panel)) ||
      new Set(view.panels).size !== view.panels.length ||
      !Array.isArray(view.layout) ||
      view.layout.length !== view.panels.length ||
      new Set(view.layout.map((item: LayoutItem) => item?.i)).size !== view.panels.length ||
      !view.layout.every((item: LayoutItem) => item != null && view.panels.includes(item.i)) ||
      !layoutFits(view.layout)
    )
      return false;
    ids.add(view.id);
    return true;
  });
}

export function readWorkspaces(): Workspace[] {
  try {
    const value: unknown = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? "null");
    if (validWorkspaces(value)) return value;
  } catch {
    /* Unavailable storage uses the preset views. */
  }
  return defaultWorkspaces();
}

export function addPanel(workspace: Workspace, panel: Panel): Workspace {
  if (workspace.panels.includes(panel)) return workspace;
  const slot = availableSlot(workspace.layout);
  if (!slot) return workspace;
  return {
    ...workspace,
    panels: [...workspace.panels, panel],
    layout: [...workspace.layout, { i: panel, ...slot, minW: MIN_PANEL, minH: MIN_PANEL }],
  };
}

export function resizePanel(
  workspace: Workspace,
  panel: Panel,
  dimension: "w" | "h",
  delta: number,
) {
  const layout = workspace.layout.map((item) =>
    item.i === panel
      ? { ...item, [dimension]: Math.max(MIN_PANEL, item[dimension] + delta) }
      : item,
  );
  return layoutFits(layout) ? { ...workspace, layout } : workspace;
}

export function reorderPanel(workspace: Workspace, panel: Panel, delta: number): Workspace {
  const panels = [...workspace.panels];
  const index = panels.indexOf(panel);
  const target = Math.max(0, Math.min(panels.length - 1, index + delta));
  [panels[index], panels[target]] = [panels[target], panels[index]];
  const slots = workspace.layout;
  return { ...workspace, panels, layout: panels.map((id, index) => ({ ...slots[index], i: id })) };
}
