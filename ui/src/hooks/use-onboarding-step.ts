import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";

import { useApi } from "@/api/apiProvider";
import type { OnboardingEventStepEnum } from "@/api/generated";
import { capabilitiesOptions } from "@/lib/queries";

/** Reports a step on Koi Cloud. The funnel must never get in the way of the step itself. */
export function useOnboardingTracker() {
  const api = useApi();
  const capabilities = useQuery(capabilitiesOptions(api));
  const cloud = capabilities.data?.cloudBilling;

  return (step: OnboardingEventStepEnum) => {
    if (cloud) void api.brands.recordStep(step).catch(() => undefined);
  };
}

export function useOnboardingStep(step: OnboardingEventStepEnum | null) {
  const api = useApi();
  const capabilities = useQuery(capabilitiesOptions(api));
  const cloud = capabilities.data?.cloudBilling;

  useEffect(() => {
    if (cloud && step) void api.brands.recordStep(step).catch(() => undefined);
  }, [api, cloud, step]);
}
