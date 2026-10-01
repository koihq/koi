import { useQuery } from "@tanstack/react-query";
import { type FormEvent, useRef, useState } from "react";
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
import { Input } from "@/components/ui/input";
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
  const [manual, setManual] = useState(false);
  const [inputValue, setInputValue] = useState("");
  const [query, setQuery] = useState("");
  const [name, setName] = useState("");
  const [domain, setDomain] = useState("");
	const [githubRepo, setGithubRepo] = useState("");
  const [invalid, setInvalid] = useState(false);
  const [saveError, setSaveError] = useState<"duplicate" | "invalidDomain" | "other">();
  const saving = useRef(false);
  const debouncedQuery = useDebouncedValue(query.trim(), 400);

  const capabilities = useQuery({
    queryKey: ["capabilities"],
    queryFn: api.capabilities.get,
    enabled: open,
    staleTime: Number.POSITIVE_INFINITY,
    retry: false,
  });
  const searchEnabled = capabilities.data?.competitorSearch === true;
  const showManual =
    manual || capabilities.isError || capabilities.data?.competitorSearch === false;
  const search = useQuery({
    queryKey: ["competitor-search", brandID, debouncedQuery],
    queryFn: ({ signal }) => api.competitors.search(brandID, debouncedQuery, { signal }),
    enabled: open && searchEnabled && !showManual && debouncedQuery.length >= 3,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    retry: false,
  });
  const results = query.trim() === debouncedQuery ? (search.data ?? []) : [];

  function reset() {
    setManual(false);
    setInputValue("");
    setQuery("");
    setName("");
    setDomain("");
	setGithubRepo("");
    setInvalid(false);
    setSaveError(undefined);
  }

  async function save(request: CreateCompetitor) {
    if (pending || saving.current) return;
    saving.current = true;
    setSaveError(undefined);
    try {
      await onSave(request);
      setOpen(false);
      reset();
    } catch (error) {
      const domainField =
        error instanceof ApiError
          ? error.fields?.find((field) => field.field === "domain" || field.field === "github_repo")
          : undefined;
      setSaveError(
        domainField?.code === "taken"
          ? "duplicate"
          : domainField?.code === "invalid"
            ? "invalidDomain"
            : "other",
      );
    } finally {
      saving.current = false;
    }
  }

  async function submitManual(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!name.trim() || (!domain.trim() && !githubRepo.trim())) {
      setInvalid(true);
      return;
    }
    setInvalid(false);
    await save({ name: name.trim(), domain: domain.trim(), githubRepo: githubRepo.trim() });
  }

  const emptyMessage =
    inputValue.trim().length < 3
      ? t("competitorSearchHint")
      : search.isError
        ? t("competitorSearchError")
        : search.isFetching
          ? t("loading")
          : t("competitorSearchEmpty");
  const saveErrorMessage =
    saveError === "duplicate"
      ? t("competitorAlreadyAdded")
      : saveError === "invalidDomain"
        ? t("competitorDomainInvalid")
        : t("competitorSaveError");

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        setOpen(nextOpen);
        if (!nextOpen) reset();
      }}
    >
      <DialogTrigger render={<Button type="button" size="sm" />}>{label}</DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("addCompetitor")}</DialogTitle>
          <DialogDescription>{t("competitorDescription")}</DialogDescription>
        </DialogHeader>
        {showManual ? (
          <form onSubmit={(event) => void submitManual(event)} className="space-y-4">
            <Field data-invalid={invalid && !name.trim() ? true : undefined}>
              <FieldLabel htmlFor="competitor-name">{t("name")}</FieldLabel>
              <Input
                id="competitor-name"
                autoFocus
                value={name}
                onChange={(event) => setName(event.target.value)}
                aria-invalid={invalid && !name.trim() ? true : undefined}
                placeholder={t("competitorNamePlaceholder")}
              />
            </Field>
            <Field data-invalid={invalid && !domain.trim() && !githubRepo.trim() ? true : undefined}>
              <FieldLabel htmlFor="competitor-domain">{t("domain")}</FieldLabel>
              <Input
                id="competitor-domain"
                value={domain}
                onChange={(event) => setDomain(event.target.value)}
                aria-invalid={invalid && !domain.trim() && !githubRepo.trim() ? true : undefined}
                placeholder={t("competitorDomainPlaceholder")}
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="competitor-github">{t("competitorGithubLabel")}</FieldLabel>
              <Input id="competitor-github" type="url" value={githubRepo} onChange={(event) => setGithubRepo(event.target.value)} placeholder="https://github.com/owner/repo" />
            </Field>
            {invalid ? <FieldError>{t("competitorInvalid")}</FieldError> : null}
            {saveError ? <FieldError>{saveErrorMessage}</FieldError> : null}
            <div className="flex items-center justify-between gap-2">
              {searchEnabled ? (
                <Button type="button" variant="link" size="sm" onClick={() => setManual(false)}>
                  {t("backToSearch")}
                </Button>
              ) : (
                <span />
              )}
              <Button type="submit" disabled={pending || saving.current}>
                {t("saveCompetitor")}
              </Button>
            </div>
          </form>
        ) : searchEnabled ? (
          <>
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
                onValueChange={(value) => {
                  if (value) void save({ name: value.name, domain: value.domain });
                }}
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
                          <span className="block truncate text-muted-foreground">
                            {result.domain}
                          </span>
                        </span>
                      </ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </Field>
            {search.isError ? <FieldError>{t("competitorSearchError")}</FieldError> : null}
            {saveError ? <FieldError>{saveErrorMessage}</FieldError> : null}
            <Button
              type="button"
              variant="link"
              size="sm"
              onClick={() => {
                setName(inputValue);
                setManual(true);
              }}
            >
              {t("enterManually")}
            </Button>
          </>
        ) : (
          <div className="space-y-2">
            <p className="text-sm text-muted-foreground">{t("loading")}</p>
            <Button type="button" variant="link" size="sm" onClick={() => setManual(true)}>
              {t("enterManually")}
            </Button>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
