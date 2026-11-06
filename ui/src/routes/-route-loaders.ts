import type { QueryClient } from "@tanstack/react-query";
import { redirect } from "@tanstack/react-router";
import type { Api } from "@/api/apiProvider";
import { ApiError } from "@/api/errors";
import { accountOptions, capabilitiesOptions } from "@/lib/queries";

/** A hover preload reads the cached account; only a real navigation asks the server again. */
export async function requireAccount(api: Api, queryClient: QueryClient, preload = false) {
  try {
    const options = accountOptions(api);

    return await (preload ? queryClient.ensureQueryData(options) : queryClient.fetchQuery(options));
  } catch (error) {
    if (error instanceof ApiError && error.code === "unauthorized") {
      throw redirect({ to: "/sign-in" });
    }

    throw error;
  }
}

export async function loadAccountBrand(api: Api, queryClient: QueryClient, preload = false) {
  const me = await requireAccount(api, queryClient, preload);
  const capabilities = await queryClient.ensureQueryData(capabilitiesOptions(api));

  return { me, brand: me.activeBrand, capabilities };
}
