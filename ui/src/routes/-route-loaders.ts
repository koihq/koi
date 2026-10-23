import type { QueryClient } from "@tanstack/react-query";
import { redirect } from "@tanstack/react-router";
import type { Api } from "@/api/apiProvider";
import { ApiError } from "@/api/errors";
import { accountOptions, brandsOptions, capabilitiesOptions } from "@/lib/queries";

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
  const [brands, capabilities] = await Promise.all([
    queryClient.fetchQuery(brandsOptions(api, me.organization.id)),
    queryClient.ensureQueryData(capabilitiesOptions(api)),
  ]);
  const brand = brands.find((candidate) => candidate.id === me.activeBrandId) ?? brands[0];

  return { me, brand, brands, capabilities };
}
