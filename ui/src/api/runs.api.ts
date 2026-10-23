import type {
  HistoricalPromptPage,
  PromptHistoryRunPage,
  VisibilityApi,
  VisibilityReport,
  VisibilityResult,
  VisibilityResultPage,
  VisibilityRunPage,
  VisibilityRunStatus,
} from "@/api/generated";
import type { PromptSort } from "@/api/prompts.api";
import type { VisibilityScope } from "@/lib/visibility";

export type VisibilityRun = VisibilityRunStatus;

export type { VisibilityResult };
export type HistoricalPromptWithVisibility = HistoricalPromptPage["items"][number];

export type HistoricalPromptListParams = {
  page: number;
  pageSize: 10 | 20 | 50;
  q?: string;
  sort?: PromptSort;
  descending?: boolean;
  scope: VisibilityScope;
};

export function createRunsApi(api: VisibilityApi) {
  return {
    tracking: (brandId: string, init?: RequestInit) => api.getVisibilityTracking({ brandId }, init),
    list: (
      brandId: string,
      page = 1,
      pageSize = 10,
      init?: RequestInit,
    ): Promise<VisibilityRunPage> => api.listVisibilityRuns({ brandId, page, pageSize }, init),
    getById: (runID: string, init?: RequestInit): Promise<VisibilityRun> =>
      api.getVisibilityRun({ runID }, init),
    listResults: (
      runID: string,
      page = 1,
      pageSize = 40,
      init?: RequestInit,
    ): Promise<VisibilityResultPage> => api.listVisibilityResults({ runID, page, pageSize }, init),
    historicalPrompts: (
      brandId: string,
      { scope, ...params }: HistoricalPromptListParams,
      init?: RequestInit,
    ): Promise<HistoricalPromptPage> =>
      api.listHistoricalPrompts(
        { ...params, brandId, days: scope.days, engines: scope.engines.join(",") || undefined },
        init,
      ),
    summary: (
      brandId: string,
      scope: VisibilityScope,
      init?: RequestInit,
    ): Promise<VisibilityReport> =>
      api.getVisibilitySummary(
        { brandId, days: scope.days, engines: scope.engines.join(",") || undefined },
        init,
      ),
    promptVisibility: (
      brandId: string,
      promptID: string,
      scope: VisibilityScope,
      init?: RequestInit,
    ): Promise<VisibilityReport> =>
      api.getPromptVisibility(
        { brandId, promptID, days: scope.days, engines: scope.engines.join(",") || undefined },
        init,
      ),
    promptHistory: (
      brandId: string,
      promptID: string,
      page = 1,
      pageSize = 10,
      init?: RequestInit,
    ): Promise<PromptHistoryRunPage> =>
      api.listPromptHistoryRuns({ brandId, promptID, page, pageSize }, init),
  };
}
