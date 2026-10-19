type PageItems<T> = { items: T[]; hasNext: boolean };

export async function collectPages<T>(fetchPage: (page: number) => Promise<PageItems<T>>) {
  const items: T[] = [];

  for (let page = 1; ; page += 1) {
    const response = await fetchPage(page);
    items.push(...response.items);

    if (!response.hasNext) return items;
  }
}
