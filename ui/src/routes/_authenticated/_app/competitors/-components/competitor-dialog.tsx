import { useQuery } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "@/api/apiProvider";
import type { CompetitorSearchResult, CreateCompetitor } from "@/api/competitors.api";
import { ApiError } from "@/api/errors";
import { BrandIcon } from "@/components/blocks/brand-icon";
import { Button } from "@/components/ui/button";
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { useDebouncedValue } from "@/hooks/use-debounced-value";

export function CompetitorDialog({
  brandID,
  label,
  pending,
  onSave,
}: {
  brandID: string;
  label: string;
  pending: boolean;
  onSave: (request: CreateCompetitor) => Promise<void>;
}) {
  const { t } = useTranslation("management");
  const api = useApi();
  const [open, setOpen] = useState(false);
  const [inputValue, setInputValue] = useState("");
  const [query, setQuery] = useState("");
  const [saveError, setSaveError] = useState<"duplicate" | "other">();
  const saving = useRef(false);
  const debouncedQuery = useDebouncedValue(query.trim(), 400);

  const search = useQuery({
    queryKey: ["competitor-search", brandID, debouncedQuery],
    queryFn: ({ signal }) => api.competitors.search(brandID, debouncedQuery, { signal }),
    enabled: open && debouncedQuery.length >= 3,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    retry: false,
  });
  const results = query.trim() === debouncedQuery ? (search.data ?? []) : [];

  async function choose(result: CompetitorSearchResult | null) {
    if (!result || pending || saving.current) return;
    saving.current = true;
    setSaveError(undefined);
    try {
      await onSave({ name: result.name, domain: result.domain });
      setOpen(false);
      setInputValue("");
      setQuery("");
    } catch (error) {
      setSaveError(
        error instanceof ApiError &&
          error.fields?.some((field) => field.field === "domain" && field.code === "taken")
          ? "duplicate"
          : "other",
      );
    } finally {
      saving.current = false;
    }
  }

  const emptyMessage =
    inputValue.trim().length < 3
      ? t("competitorSearchHint")
      : search.isError
        ? t("competitorSearchError")
        : search.isFetching
          ? t("loading")
          : t("competitorSearchEmpty");

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        setOpen(nextOpen);
        if (!nextOpen) {
          setInputValue("");
          setQuery("");
          setSaveError(undefined);
        }
      }}
    >
      <DialogTrigger render={<Button type="button" size="sm" />}>{label}</DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("addCompetitor")}</DialogTitle>
          <DialogDescription>{t("competitorDescription")}</DialogDescription>
        </DialogHeader>
        <Field>
          <FieldLabel htmlFor="competitor-search">{t("competitorSearchLabel")}</FieldLabel>
          <Combobox<CompetitorSearchResult>
            items={results}
            filter={null}
            value={null}
            inputValue={inputValue}
            itemToStringLabel={(item) => item.name}
            itemToStringValue={(item) => item.domain}
            isItemEqualToValue={(item, value) => item.domain === value.domain}
            onInputValueChange={(value, details) => {
              setInputValue(value);
              if (details.reason === "input-change" || details.reason === "input-clear") {
                setQuery(value);
                setSaveError(undefined);
              }
            }}
            onValueChange={(value) => void choose(value)}
          >
            <ComboboxInput
              id="competitor-search"
              className="w-full"
              placeholder={t("competitorSearchPlaceholder")}
              autoFocus
              aria-busy={search.isFetching}
              showClear={Boolean(inputValue)}
            />
            <ComboboxContent>
              <ComboboxEmpty>{emptyMessage}</ComboboxEmpty>
              <ComboboxList>
                {results.map((result) => (
                  <ComboboxItem key={`${result.domain}:${result.name}`} value={result}>
                    <BrandIcon name={result.name} domain={result.domain} />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate font-medium">{result.name}</span>
                      <span className="block truncate text-muted-foreground">{result.domain}</span>
                    </span>
                  </ComboboxItem>
                ))}
              </ComboboxList>
            </ComboboxContent>
          </Combobox>
        </Field>
        {search.isError ? <FieldError>{t("competitorSearchError")}</FieldError> : null}
        {saveError ? (
          <FieldError>
            {t(saveError === "duplicate" ? "competitorAlreadyAdded" : "competitorSaveError")}
          </FieldError>
        ) : null}
        {pending ? <p className="text-sm text-muted-foreground">{t("loading")}</p> : null}
      </DialogContent>
    </Dialog>
  );
}
