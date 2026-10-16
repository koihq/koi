import { createContext, useContext } from "react";
import type { useExplorationState } from "../-hooks/use-exploration-state";
import type { ExplorationSearch } from "../-lib/exploration-fixtures";
import type { HomeBrand } from "../-lib/home-evidence";
import type { PromptRow } from "../-lib/prompt-visibility";

type WorkspaceContextValue = {
  brand: HomeBrand;
  state: ReturnType<typeof useExplorationState>;
  search: ExplorationSearch;
  rows: PromptRow[];
  update: (patch: Partial<ExplorationSearch>) => void;
};

export const WorkspaceContext = createContext<WorkspaceContextValue | null>(null);

export function useHomeWorkspace() {
  const workspace = useContext(WorkspaceContext);
  if (!workspace) throw new Error("Competitors requires the Home context workspace");
  return workspace;
}
