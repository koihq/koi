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
import { callApi } from "@/api/shared/apiClient";
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
    tracking: (brandID: string, initOverrides?: RequestInit) =>
      callApi(() => api.getVisibilityTracking(brandID, initOverrides)),
    list: (
      brandID: string,
      page = 1,
      pageSize = 10,
      initOverrides?: RequestInit,
    ): Promise<VisibilityRunPage> =>
      callApi(() => api.listVisibilityRuns(brandID, page, pageSize, initOverrides)),
    getById: (
      brandID: string,
      runID: string,
      initOverrides?: RequestInit,
    ): Promise<VisibilityRun> => callApi(() => api.getVisibilityRun(brandID, runID, initOverrides)),
    listResults: (
      brandID: string,
      runID: string,
      page = 1,
      pageSize = 40,
      initOverrides?: RequestInit,
    ): Promise<VisibilityResultPage> =>
      callApi(() => api.listVisibilityResults(brandID, runID, page, pageSize, initOverrides)),
    historicalPrompts: (
      brandID: string,
      params: HistoricalPromptListParams,
      initOverrides?: RequestInit,
    ): Promise<HistoricalPromptPage> =>
      callApi(() =>
        api.listHistoricalPrompts(
          brandID,
          params.page,
          params.pageSize,
          params.q,
          params.sort,
          params.descending,
          params.scope.days,
          params.scope.engines.join(",") || undefined,
          initOverrides,
        ),
      ),
    summary: (
      brandID: string,
      scope: VisibilityScope,
      initOverrides?: RequestInit,
    ): Promise<VisibilityReport> =>
      callApi(() =>
        api.getVisibilitySummary(
          brandID,
          scope.days,
          scope.engines.join(",") || undefined,
          initOverrides,
        ),
      ),
    promptVisibility: (
      brandID: string,
      promptID: string,
      scope: VisibilityScope,
      initOverrides?: RequestInit,
    ): Promise<VisibilityReport> =>
      callApi(() =>
        api.getPromptVisibility(
          brandID,
          promptID,
          scope.days,
          scope.engines.join(",") || undefined,
          initOverrides,
        ),
      ),
    promptHistory: (
      brandID: string,
      promptID: string,
      page = 1,
      pageSize = 10,
      initOverrides?: RequestInit,
    ): Promise<PromptHistoryRunPage> =>
      callApi(() => api.listPromptHistoryRuns(brandID, promptID, page, pageSize, initOverrides)),
  };
}
