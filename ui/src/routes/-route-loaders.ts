import { redirect } from "@tanstack/react-router";
import type { Api } from "@/api/apiProvider";
import { ApiError } from "@/api/errors";

export async function requireAccount(api: Api) {
  try {
    return await api.auth.me();
  } catch (error) {
    if (error instanceof ApiError && error.code === "unauthorized") {
      throw redirect({ to: "/sign-in" });
    }

    throw error;
  }
}

export async function loadAccountBrand(api: Api) {
  const me = await requireAccount(api);
  const page = await api.brands.list({ page: 1, pageSize: 1 });

  return { me, brand: page.items[0] };
}
