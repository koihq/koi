import type { QueryClient } from "@tanstack/react-query";
import { redirect } from "@tanstack/react-router";
import type { Api } from "@/api/apiProvider";
import { ApiError } from "@/api/errors";
import { accountOptions, capabilitiesOptions } from "@/lib/queries";

export async function requireAccount(api: Api, queryClient: QueryClient) {
  try {
    return await queryClient.fetchQuery(accountOptions(api));
  } catch (error) {
    if (error instanceof ApiError && error.code === "unauthorized") {
      throw redirect({ to: "/sign-in" });
    }

    throw error;
  }
}

export async function loadAccountBrand(api: Api, queryClient: QueryClient) {
  const me = await requireAccount(api, queryClient);
  const capabilities = await queryClient.ensureQueryData(capabilitiesOptions(api));

  return { me, brand: me.activeBrand, capabilities };
}
