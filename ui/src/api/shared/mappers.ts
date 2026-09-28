import type { Page } from "@/api/generated";
import type { ApiListResponse } from "@/api/shared/types";

export function mapPage<T>(page: Page, mapItem: (item: unknown) => T): ApiListResponse<T> {
  return { ...page, items: page.items.map(mapItem) };
}
