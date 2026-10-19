import { useQuery } from "@tanstack/react-query";
import { type FormEvent, useId, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import { useApi } from "@/api/apiProvider";
import type { Competitor, CompetitorSearchResult, CreateCompetitor } from "@/api/competitors.api";
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
import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { useDebouncedValue } from "@/hooks/use-debounced-value";

export function CompetitorDialog({
  brandID,
  competitor,
  label,
  pending,
  onSave,
  open: controlledOpen,
  onOpenChange,
  hideTrigger = false,
  triggerVariant = "default",
}: {
  brandID: string;
  competitor?: Competitor;
  label: string;
  pending: boolean;
  onSave: (request: CreateCompetitor) => Promise<void>;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  hideTrigger?: boolean;
  triggerVariant?: "default" | "ghost";
}) {
  const { t } = useTranslation("management");
  const api = useApi();
  const fieldID = useId();
  const allowNameOnly = competitor?.source === "observed" && !competitor.confirmed;
  const [open, setOpen] = useState(false);
  const visible = controlledOpen ?? open;
  const changeOpen = onOpenChange ?? setOpen;
  const [manual, setManual] = useState(false);
  const [inputValue, setInputValue] = useState("");
  const [query, setQuery] = useState("");
  const [name, setName] = useState(competitor?.name ?? "");
  const [domain, setDomain] = useState(competitor?.domain ?? "");
  const [githubRepo, setGithubRepo] = useState(competitor?.githubRepo ?? "");
  const [aliases, setAliases] = useState(competitor?.aliases.join(", ") ?? "");
  const [additionalURLs, setAdditionalURLs] = useState(competitor?.additionalUrls.join(", ") ?? "");
  const [invalid, setInvalid] = useState(false);
  const [saveError, setSaveError] = useState<"duplicate" | "invalidDomain" | "other">();
  const saving = useRef(false);
  const debouncedQuery = useDebouncedValue(query.trim(), 400);

  const capabilities = useQuery({
    queryKey: ["capabilities"],
    queryFn: api.capabilities.get,
    enabled: visible && !competitor,
    staleTime: Number.POSITIVE_INFINITY,
    retry: false,
  });

  const searchEnabled = capabilities.data?.competitorSearch === true;

  const showManual =
    Boolean(competitor) ||
    manual ||
    capabilities.isError ||
    capabilities.data?.competitorSearch === false;

  const search = useQuery({
    queryKey: ["competitor-search", brandID, debouncedQuery],
    queryFn: ({ signal }) => api.competitors.search(brandID, debouncedQuery, { signal }),
    enabled: visible && searchEnabled && !showManual && debouncedQuery.length >= 3,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    retry: false,
  });

  const results = query.trim() === debouncedQuery ? (search.data ?? []) : [];

  function reset() {
    setManual(false);
    setInputValue("");
    setQuery("");
    setName(competitor?.name ?? "");
    setDomain(competitor?.domain ?? "");
    setGithubRepo(competitor?.githubRepo ?? "");
    setAliases(competitor?.aliases.join(", ") ?? "");
    setAdditionalURLs(competitor?.additionalUrls.join(", ") ?? "");
    setInvalid(false);
    setSaveError(undefined);
  }

  async function save(request: CreateCompetitor) {
    if (pending || saving.current) return;

    saving.current = true;
    setSaveError(undefined);

    try {
      await onSave(request);
      changeOpen(false);
      reset();
    } catch (error) {
      const domainField =
        error instanceof ApiError
          ? error.fields?.find((field) =>
              ["domain", "github_repo", "additional_urls", "aliases"].includes(field.field),
            )
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
    if (!name.trim() || (!allowNameOnly && !domain.trim() && !githubRepo.trim())) {
      setInvalid(true);

      return;
    }

    setInvalid(false);
    await save({
      name: name.trim(),
      domain: domain.trim(),
      githubRepo: githubRepo.trim(),
      aliases: aliases
        .split(",")
        .map((value) => value.trim())
        .filter(Boolean),
      additionalUrls: additionalURLs
        .split(",")
        .map((value) => value.trim())
        .filter(Boolean),
    });
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
      open={visible}
      onOpenChange={(nextOpen) => {
        changeOpen(nextOpen);
        reset();
      }}
    >
      {!hideTrigger && (
        <DialogTrigger
          render={
            <Button
              type="button"
              size={competitor ? "xs" : "sm"}
              variant={competitor ? "ghost" : triggerVariant}
              disabled={pending}
            />
          }
        >
          {label}
        </DialogTrigger>
      )}
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t(competitor ? "editCompetitor" : "addCompetitor")}</DialogTitle>
          <DialogDescription>{t("competitorDescription")}</DialogDescription>
        </DialogHeader>
        {showManual ? (
          <form onSubmit={(event) => void submitManual(event)} className="space-y-4">
            <Field data-invalid={invalid && !name.trim() ? true : undefined}>
              <FieldLabel htmlFor={`${fieldID}-name`}>{t("name")}</FieldLabel>
              <Input
                id={`${fieldID}-name`}
                autoFocus
                value={name}
                onChange={(event) => setName(event.target.value)}
                aria-invalid={invalid && !name.trim() ? true : undefined}
                placeholder={t("competitorNamePlaceholder")}
              />
            </Field>
            <Field
              data-invalid={
                invalid && !allowNameOnly && !domain.trim() && !githubRepo.trim() ? true : undefined
              }
            >
              <FieldLabel htmlFor={`${fieldID}-domain`}>{t("domain")}</FieldLabel>
              <Input
                id={`${fieldID}-domain`}
                value={domain}
                onChange={(event) => setDomain(event.target.value)}
                aria-invalid={
                  invalid && !allowNameOnly && !domain.trim() && !githubRepo.trim()
                    ? true
                    : undefined
                }
                placeholder={t("competitorDomainPlaceholder")}
              />
            </Field>
            <Field>
              <FieldLabel htmlFor={`${fieldID}-github`}>{t("competitorGithubLabel")}</FieldLabel>
              <Input
                id={`${fieldID}-github`}
                type="url"
                value={githubRepo}
                onChange={(event) => setGithubRepo(event.target.value)}
                placeholder="https://github.com/owner/repo"
              />
            </Field>
            <Field>
              <FieldLabel htmlFor={`${fieldID}-aliases`}>{t("competitorAliases")}</FieldLabel>
              <Input
                id={`${fieldID}-aliases`}
                value={aliases}
                onChange={(event) => setAliases(event.target.value)}
                aria-describedby={`${fieldID}-aliases-hint`}
              />
              <FieldDescription id={`${fieldID}-aliases-hint`}>
                {t("competitorAliasesHint")}
              </FieldDescription>
            </Field>
            <Field>
              <FieldLabel htmlFor={`${fieldID}-urls`}>{t("competitorAdditionalURLs")}</FieldLabel>
              <Input
                id={`${fieldID}-urls`}
                value={additionalURLs}
                onChange={(event) => setAdditionalURLs(event.target.value)}
                aria-describedby={`${fieldID}-urls-hint`}
              />
              <FieldDescription id={`${fieldID}-urls-hint`}>
                {t("competitorAdditionalURLsHint")}
              </FieldDescription>
            </Field>
            {competitor?.source === "observed" && !competitor.confirmed ? (
              <p className="text-sm text-muted-foreground">{t("observedEditNotice")}</p>
            ) : null}
            {invalid ? (
              <FieldError>
                {t(allowNameOnly ? "competitorNameInvalid" : "competitorInvalid")}
              </FieldError>
            ) : null}
            {saveError ? <FieldError>{saveErrorMessage}</FieldError> : null}
            <div className="flex items-center justify-between gap-2">
              {searchEnabled && !competitor ? (
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
              <FieldLabel htmlFor={`${fieldID}-search`}>{t("competitorSearchLabel")}</FieldLabel>
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
                  id={`${fieldID}-search`}
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
