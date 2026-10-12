import type { LayoutItem } from "react-grid-layout";
import { SECTIONS, type Section } from "../-lib/data";
import C from "../-locales/en.json";

export const PANELS = {
  heatmap: C.heatmap,
  screener: C.screener,
  inspector: C.context,
  trend: C.panelTrend,
  watchlist: C.watchlist,
  engines: C.enginePerformance,
  brief: C.brief,
  mentions: C.mentions,
  prompts: C.prompts,
  competitors: C.competitors,
  citations: C.citations,
};
export type Panel = keyof typeof PANELS;
export type Workspace = {
  id: string;
  name: string;
  section: Section;
  panels: Panel[];
  desktop: LayoutItem[];
  tablet: LayoutItem[];
  mobileSizes: Partial<Record<Panel, number>>;
};
export const STORAGE_KEY = "koi-exploration:trade-republic:workspaces:v1";

function grid(panels: Panel[], cols: number): LayoutItem[] {
  const overview = panels.join() === "heatmap,screener,inspector,trend,watchlist";
  if (overview && cols === 12)
    return [
      { i: "heatmap", x: 0, y: 0, w: 4, h: 8, minW: 3, minH: 5 },
      { i: "screener", x: 4, y: 0, w: 5, h: 8, minW: 3, minH: 5 },
      { i: "inspector", x: 9, y: 0, w: 3, h: 16, minW: 2, minH: 5 },
      { i: "trend", x: 0, y: 8, w: 4, h: 8, minW: 3, minH: 5 },
      { i: "watchlist", x: 4, y: 8, w: 5, h: 8, minW: 3, minH: 5 },
    ];
  return panels.map((panel, index) => {
    const broad = ["mentions", "prompts", "competitors", "citations", "brief", "screener"].includes(
      panel,
    );
    const w = cols === 12 ? (broad ? 8 : 4) : broad ? 6 : 3;
    return {
      i: panel,
      x: cols === 12 && index === 1 ? 8 : 0,
      y: cols === 12 && index < 2 ? 0 : index * 8,
      w,
      h: broad ? 14 : 8,
      minW: broad ? Math.min(cols, 4) : 2,
      minH: 5,
    };
  });
}

export function makeWorkspace(
  id: string,
  name: string,
  section: Section,
  template: "overview" | "research" | "improve" | "empty" = "overview",
): Workspace {
  const panels: Panel[] =
    id === "overview" || !SECTIONS.includes(id as Section)
      ? template === "empty"
        ? []
        : template === "research"
          ? ["screener", "inspector", "heatmap", "engines"]
          : template === "improve"
            ? ["brief", "inspector", "watchlist"]
            : ["heatmap", "screener", "inspector", "trend", "watchlist"]
      : section === "mentions"
        ? ["mentions", "inspector"]
        : section === "brief"
          ? ["brief", "inspector"]
          : section === "citations"
            ? ["citations", "inspector"]
            : section === "prompts" || section === "competitors"
              ? [section, "inspector", "trend"]
              : ["heatmap", "screener", "inspector", "trend", "watchlist"];
  return {
    id,
    name,
    section,
    panels,
    desktop: grid(panels, 12),
    tablet: grid(panels, 6),
    mobileSizes: {},
  };
}

export function defaultWorkspaces() {
  return SECTIONS.map((section) => makeWorkspace(section, C[section], section));
}

export function validWorkspaces(value: unknown): value is Workspace[] {
  if (!Array.isArray(value) || value.length === 0 || value.length > 30) return false;
  const ids = new Set<string>();
  return value.every((workspace) => {
    if (
      !workspace ||
      typeof workspace.id !== "string" ||
      ids.has(workspace.id) ||
      typeof workspace.name !== "string" ||
      !workspace.name.trim() ||
      workspace.name.length > 60 ||
      !SECTIONS.includes(workspace.section) ||
      !Array.isArray(workspace.panels) ||
      workspace.panels.some((panel: string) => !Object.hasOwn(PANELS, panel)) ||
      new Set(workspace.panels).size !== workspace.panels.length ||
      (workspace.mobileSizes != null &&
        (typeof workspace.mobileSizes !== "object" ||
          Object.entries(workspace.mobileSizes).some(
            ([panel, height]) =>
              !Object.hasOwn(PANELS, panel) ||
              !Number.isInteger(height) ||
              Number(height) < 220 ||
              Number(height) > 1400,
          )))
    )
      return false;
    ids.add(workspace.id);
    return (["desktop", "tablet"] as const).every(
      (size) =>
        Array.isArray(workspace[size]) &&
        workspace[size].length === workspace.panels.length &&
        new Set(workspace[size].map((item: LayoutItem) => item?.i)).size ===
          workspace.panels.length &&
        workspace[size].every(
          (item: LayoutItem) =>
            item != null &&
            typeof item.i === "string" &&
            workspace.panels.includes(item.i) &&
            [item.x, item.y, item.w, item.h].every(Number.isInteger) &&
            item.x >= 0 &&
            item.y >= 0 &&
            item.w >= 2 &&
            item.h >= 5 &&
            item.h <= 30 &&
            item.x + item.w <= (size === "desktop" ? 12 : 6),
        ),
    );
  });
}

export function readWorkspaces(): Workspace[] {
  try {
    const value: unknown = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? "null");
    if (validWorkspaces(value))
      return value.map((workspace) => ({ ...workspace, mobileSizes: workspace.mobileSizes ?? {} }));
  } catch {
    /* Unavailable or outdated storage uses the preset workspace. */
  }
  return defaultWorkspaces();
}

export function addPanel(workspace: Workspace, panel: Panel): Workspace {
  if (workspace.panels.includes(panel)) return workspace;
  const panels = [...workspace.panels, panel];
  const append = (layout: LayoutItem[], cols: number) => [
    ...layout,
    {
      i: panel,
      x: 0,
      y: Math.max(0, ...layout.map((item) => item.y + item.h)),
      w: Math.min(cols, 4),
      h: 8,
      minW: 2,
      minH: 5,
    },
  ];
  return {
    ...workspace,
    panels,
    desktop: append(workspace.desktop, 12),
    tablet: append(workspace.tablet, 6),
  };
}

export function reorderPanel(workspace: Workspace, panel: Panel, delta: number): Workspace {
  const panels = [...workspace.panels];
  const index = panels.indexOf(panel);
  const target = Math.max(0, Math.min(panels.length - 1, index + delta));
  [panels[index], panels[target]] = [panels[target], panels[index]];
  const reorder = (layout: LayoutItem[]) => {
    const slots = [...layout].sort((a, b) => a.y - b.y || a.x - b.x);
    return panels.map((id, i) => ({ ...slots[i], i: id }));
  };
  return {
    ...workspace,
    panels,
    desktop: reorder(workspace.desktop),
    tablet: reorder(workspace.tablet),
  };
}
