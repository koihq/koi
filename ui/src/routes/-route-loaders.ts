import { redirect } from "@tanstack/react-router";
import type { Api } from "@/api/apiProvider";
import { ApiError } from "@/api/errors";
import { collectPages } from "@/lib/collect-pages";

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
  const brands = await collectPages((page) => api.brands.list({ page, pageSize: 100 }));
  const brand = brands.find((candidate) => candidate.id === me.activeBrandId) ?? brands[0];

  return { me, brand, brands };
}
