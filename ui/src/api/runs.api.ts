import type {
  HistoricalPromptPage,
  PromptHistoryRunPage,
  VisibilityApi,
  VisibilityResult,
  VisibilityResultPage,
  VisibilityRunPage,
  VisibilityRunStatus,
} from "@/api/generated";
import { callApi } from "@/api/shared/apiClient";

export type VisibilityRun = VisibilityRunStatus;

export type { VisibilityResult };

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
      page = 1,
      pageSize = 20,
      initOverrides?: RequestInit,
    ): Promise<HistoricalPromptPage> =>
      callApi(() => api.listHistoricalPrompts(brandID, page, pageSize, initOverrides)),
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
