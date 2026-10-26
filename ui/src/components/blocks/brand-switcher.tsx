import { useMutation } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "@/api/apiProvider";
import type { Brand } from "@/api/brands.api";
import { BrandIcon } from "@/components/blocks/brand-icon";
import { Button } from "@/components/ui/button";
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
  ComboboxTrigger,
} from "@/components/ui/combobox";
import { useBrandSearch } from "@/hooks/use-brand-search";

export function BrandSwitcher({ brand }: { brand: Brand }) {
  const api = useApi();
  const router = useRouter();
  const { t } = useTranslation("home");
  const [open, setOpen] = useState(false);
  const search = useBrandSearch(brand.organizationId, open);
  const activation = useMutation({
    mutationFn: (id: string) => api.brands.activate(id),
    onSuccess: async () => {
      await router.invalidate();
      await router.navigate({ to: "/" });
    },
  });

  const busy = search.searching || search.result.isFetching;

  return (
    <div className="min-w-0">
      <Combobox
        items={search.items}
        filter={null}
        value={brand}
        inputValue={search.input}
        open={open}
        disabled={activation.isPending}
        itemToStringLabel={(item) => item.name}
        itemToStringValue={(item) => item.id}
        isItemEqualToValue={(item, value) => item.id === value.id}
        onOpenChange={(next) => {
          if (next) {
            search.changeSearch("");
            activation.reset();
          }
          setOpen(next);
        }}
        onInputValueChange={(value, details) => {
          if (details.reason === "input-change" || details.reason === "input-clear") {
            search.changeSearch(value);
          }
        }}
        onValueChange={(value) => {
          if (value && value.id !== brand.id && !activation.isPending) {
            activation.mutate(value.id);
          }
        }}
      >
        <ComboboxTrigger
          aria-label={t("header.activeBrand", { name: brand.name })}
          title={brand.name}
          className="inline-flex max-w-full items-center gap-2 rounded-full p-1 pe-3 outline-none hover:bg-muted focus-visible:ring-2 focus-visible:ring-ring"
        >
          <BrandIcon name={brand.name} domain={brand.domain} className="size-6" />
          <span className="max-w-40 truncate text-sm">{brand.name}</span>
        </ComboboxTrigger>
        <ComboboxContent className="w-80 max-w-[calc(100vw-32px)]">
          <ComboboxInput
            aria-label={t("header.searchBrands")}
            placeholder={t("header.searchBrands")}
            showTrigger={false}
            showClear={Boolean(search.input)}
            aria-busy={busy}
          />
          <ComboboxEmpty>
            {search.searching
              ? t("header.searchingBrands")
              : search.result.isError
                ? t("header.brandSearchError")
                : t("header.noBrands")}
          </ComboboxEmpty>
          <ComboboxList>
            {search.items.map((item) => (
              <ComboboxItem key={item.id} value={item}>
                <BrandIcon name={item.name} domain={item.domain} />
                <span className="min-w-0 flex-1 pe-4">
                  <span className="block truncate" title={item.name}>
                    {item.name}
                  </span>
                  <span
                    className="block truncate text-muted-foreground"
                    title={item.domain || item.githubRepo}
                  >
                    {item.domain || item.githubRepo}
                  </span>
                </span>
              </ComboboxItem>
            ))}
          </ComboboxList>
          {search.result.isError && !search.searching ? (
            <div className="p-2">
              <Button variant="link" onClick={() => void search.result.refetch()}>
                {t("dashboard.retry")}
              </Button>
            </div>
          ) : search.result.data && search.result.data.totalPages > 1 ? (
            <div className="flex items-center justify-between gap-2 p-2">
              <Button
                variant="ghost"
                size="sm"
                disabled={busy || !search.result.data.hasPrev}
                onClick={() => search.setPage(search.page - 1)}
              >
                {t("dashboard.previous")}
              </Button>
              <output className="text-xs text-muted-foreground">
                {t("dashboard.page", {
                  current: search.page,
                  total: search.result.data.totalPages,
                })}
              </output>
              <Button
                variant="ghost"
                size="sm"
                disabled={busy || !search.result.data.hasNext}
                onClick={() => search.setPage(search.page + 1)}
              >
                {t("dashboard.next")}
              </Button>
            </div>
          ) : null}
        </ComboboxContent>
      </Combobox>
      {activation.isError && (
        <p role="alert" className="mt-1 text-xs text-destructive">
          {t("header.brandActivationError")}
        </p>
      )}
    </div>
  );
}
