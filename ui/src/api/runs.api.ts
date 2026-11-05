import type {
  PromptHistoryRunPage,
  VisibilityApi,
  VisibilityReport,
  VisibilityResult,
  VisibilityResultPage,
  VisibilityRunPage,
  VisibilityRunStatus,
  VisibilitySummary,
} from "@/api/generated";
import type { VisibilityScope } from "@/lib/visibility";

export type VisibilityRun = VisibilityRunStatus;

export type { VisibilityResult };

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
    summary: (
      brandId: string,
      scope: VisibilityScope,
      init?: RequestInit,
    ): Promise<VisibilitySummary> =>
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
