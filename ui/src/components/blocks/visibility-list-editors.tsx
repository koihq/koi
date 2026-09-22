import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";

import { useApi } from "@/api/apiProvider";
import type { CompetitorRequest, CompetitorRes, PromptRequest, PromptRes } from "@/api/generated";
import { BrandIcon } from "@/components/blocks/brand-icon";
import {
  competitorFormSchema,
  listsReady,
  promptFormSchema,
} from "@/components/blocks/visibility-lists";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";

const promptKey = (brandID: string) => ["brands", brandID, "prompts"] as const;
const competitorKey = (brandID: string) => ["brands", brandID, "competitors"] as const;

function PromptForm({
  initialText = "",
  pending,
  onSave,
  onCancel,
}: {
  initialText?: string;
  pending: boolean;
  onSave: (request: PromptRequest) => Promise<void>;
  onCancel?: () => void;
}) {
  const { t } = useTranslation("brand");
  const [text, setText] = useState(initialText);
  const [invalid, setInvalid] = useState(false);
  const [failed, setFailed] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const result = promptFormSchema.safeParse({ text });
    if (!result.success) {
      setInvalid(true);
      return;
    }
    setInvalid(false);
    setFailed(false);
    try {
      await onSave(result.data);
      setText("");
    } catch {
      setFailed(true);
    }
  }

  return (
    <form onSubmit={submit} className="flex flex-col gap-3 sm:flex-row sm:items-end">
      <Field data-invalid={invalid || undefined} className="flex-1">
        <FieldLabel htmlFor={`prompt-${initialText ? "edit" : "add"}`}>
          {t("lists.prompts.field")}
        </FieldLabel>
        <Input
          id={`prompt-${initialText ? "edit" : "add"}`}
          value={text}
          onChange={(event) => setText(event.target.value)}
          aria-invalid={invalid || undefined}
          placeholder={t("lists.prompts.placeholder")}
        />
        {invalid ? <FieldError>{t("lists.prompts.invalid")}</FieldError> : null}
      </Field>
      <div className="flex gap-2">
        {onCancel ? (
          <Button type="button" variant="ghost" onClick={onCancel}>
            {t("lists.cancel")}
          </Button>
        ) : null}
        <Button type="submit" disabled={pending}>
          {initialText ? t("lists.save") : t("lists.prompts.add")}
        </Button>
      </div>
      {failed ? <FieldError>{t("lists.saveError")}</FieldError> : null}
    </form>
  );
}

function PromptEditorView({ brandID, prompts }: { brandID: string; prompts: PromptRes[] }) {
  const { t } = useTranslation("brand");
  const api = useApi();
  const queryClient = useQueryClient();
  const [editingID, setEditingID] = useState<string>();
  const invalidate = () => queryClient.invalidateQueries({ queryKey: promptKey(brandID) });
  const create = useMutation({
    mutationFn: (request: PromptRequest) => api.prompts.createPrompt(brandID, request),
    onSuccess: invalidate,
  });
  const update = useMutation({
    mutationFn: ({ id, request }: { id: string; request: PromptRequest }) =>
      api.prompts.updatePrompt(brandID, id, request),
    onSuccess: invalidate,
  });
  const remove = useMutation({
    mutationFn: (id: string) => api.prompts.deletePrompt(brandID, id),
    onSuccess: invalidate,
  });
  const confirm = useMutation({
    mutationFn: (id: string) => api.prompts.confirmPrompt(brandID, id),
    onSuccess: invalidate,
  });
  const confirmAll = useMutation({
    mutationFn: () => api.prompts.confirmAllPrompts(brandID),
    onSuccess: invalidate,
  });
  const hasUnconfirmed = prompts.some((prompt) => !prompt.confirmed);

  return (
    <section aria-labelledby="prompts-heading">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 id="prompts-heading" className="text-lg font-medium tracking-tight">
            {t("lists.prompts.title")}
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("lists.prompts.description")}</p>
        </div>
        {hasUnconfirmed ? (
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={confirmAll.isPending}
            onClick={() => confirmAll.mutate()}
          >
            {t("lists.confirmAll")}
          </Button>
        ) : null}
      </div>
      <div className="mt-6">
        <PromptForm
          pending={create.isPending}
          onSave={async (request) => {
            await create.mutateAsync(request);
          }}
        />
      </div>
      {prompts.length === 0 ? (
        <p className="mt-6 text-sm text-muted-foreground">{t("lists.prompts.empty")}</p>
      ) : (
        <ul className="mt-6 space-y-5">
          {prompts.map((prompt) => (
            <li key={prompt.id}>
              {editingID === prompt.id ? (
                <PromptForm
                  initialText={prompt.text}
                  pending={update.isPending}
                  onCancel={() => setEditingID(undefined)}
                  onSave={async (request) => {
                    await update.mutateAsync({ id: prompt.id, request });
                    setEditingID(undefined);
                  }}
                />
              ) : (
                <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                  <div className="min-w-0">
                    <p>{prompt.text}</p>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {prompt.source === "manual"
                        ? t("lists.source.manual")
                        : t("lists.source.drafted")}
                      {" · "}
                      {prompt.confirmed ? t("lists.confirmed") : t("lists.unconfirmed")}
                    </p>
                  </div>
                  <div className="flex shrink-0 flex-wrap gap-1">
                    {!prompt.confirmed ? (
                      <Button
                        type="button"
                        size="xs"
                        variant="outline"
                        disabled={confirm.isPending}
                        onClick={() => confirm.mutate(prompt.id)}
                      >
                        {t("lists.confirm")}
                      </Button>
                    ) : null}
                    <Button
                      type="button"
                      size="xs"
                      variant="ghost"
                      onClick={() => setEditingID(prompt.id)}
                    >
                      {t("lists.edit")}
                    </Button>
                    <Button
                      type="button"
                      size="xs"
                      variant="ghost"
                      disabled={remove.isPending}
                      onClick={() => {
                        if (window.confirm(t("lists.prompts.deleteConfirm")))
                          remove.mutate(prompt.id);
                      }}
                    >
                      {t("lists.delete")}
                    </Button>
                  </div>
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function CompetitorForm({
  initial,
  pending,
  onSave,
  onCancel,
}: {
  initial?: CompetitorRes;
  pending: boolean;
  onSave: (request: CompetitorRequest) => Promise<void>;
  onCancel?: () => void;
}) {
  const { t } = useTranslation("brand");
  const [name, setName] = useState(initial?.name ?? "");
  const [domain, setDomain] = useState(initial?.domain ?? "");
  const [aliases, setAliases] = useState(initial?.aliases.join(", ") ?? "");
  const [invalidFields, setInvalidFields] = useState<Set<string>>(new Set());
  const [failed, setFailed] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const result = competitorFormSchema.safeParse({ name, domain, aliases });
    if (!result.success) {
      setInvalidFields(new Set(result.error.issues.map((issue) => String(issue.path[0]))));
      return;
    }
    setInvalidFields(new Set());
    setFailed(false);
    try {
      await onSave(result.data);
      if (!initial) {
        setName("");
        setDomain("");
        setAliases("");
      }
    } catch {
      setFailed(true);
    }
  }

  return (
    <form onSubmit={submit}>
      <FieldGroup className="sm:grid sm:grid-cols-2">
        <Field data-invalid={invalidFields.has("name") || undefined}>
          <FieldLabel htmlFor={`competitor-name-${initial?.id ?? "add"}`}>
            {t("lists.competitors.name")}
          </FieldLabel>
          <Input
            id={`competitor-name-${initial?.id ?? "add"}`}
            value={name}
            onChange={(event) => setName(event.target.value)}
            aria-invalid={invalidFields.has("name") || undefined}
          />
          {invalidFields.has("name") ? (
            <FieldError>{t("lists.competitors.nameInvalid")}</FieldError>
          ) : null}
        </Field>
        <Field data-invalid={invalidFields.has("domain") || undefined}>
          <FieldLabel htmlFor={`competitor-domain-${initial?.id ?? "add"}`}>
            {t("lists.competitors.domain")}
          </FieldLabel>
          <Input
            id={`competitor-domain-${initial?.id ?? "add"}`}
            value={domain}
            onChange={(event) => setDomain(event.target.value)}
            aria-invalid={invalidFields.has("domain") || undefined}
            placeholder={t("lists.competitors.domainPlaceholder")}
          />
          {invalidFields.has("domain") ? (
            <FieldError>{t("lists.competitors.domainInvalid")}</FieldError>
          ) : null}
        </Field>
        <Field className="sm:col-span-2">
          <FieldLabel htmlFor={`competitor-aliases-${initial?.id ?? "add"}`}>
            {t("lists.competitors.aliases")}
          </FieldLabel>
          <Input
            id={`competitor-aliases-${initial?.id ?? "add"}`}
            value={aliases}
            onChange={(event) => setAliases(event.target.value)}
            placeholder={t("lists.competitors.aliasesPlaceholder")}
          />
        </Field>
      </FieldGroup>
      <div className="mt-3 flex gap-2">
        {onCancel ? (
          <Button type="button" variant="ghost" onClick={onCancel}>
            {t("lists.cancel")}
          </Button>
        ) : null}
        <Button type="submit" disabled={pending}>
          {initial ? t("lists.save") : t("lists.competitors.add")}
        </Button>
      </div>
      {failed ? <FieldError className="mt-2">{t("lists.saveError")}</FieldError> : null}
    </form>
  );
}

function CompetitorEditorView({
  brandID,
  competitors,
}: {
  brandID: string;
  competitors: CompetitorRes[];
}) {
  const { t } = useTranslation("brand");
  const api = useApi();
  const queryClient = useQueryClient();
  const [editingID, setEditingID] = useState<string>();
  const invalidate = () => queryClient.invalidateQueries({ queryKey: competitorKey(brandID) });
  const create = useMutation({
    mutationFn: (request: CompetitorRequest) => api.competitors.createCompetitor(brandID, request),
    onSuccess: invalidate,
  });
  const update = useMutation({
    mutationFn: ({ id, request }: { id: string; request: CompetitorRequest }) =>
      api.competitors.updateCompetitor(brandID, id, request),
    onSuccess: invalidate,
  });
  const remove = useMutation({
    mutationFn: (id: string) => api.competitors.deleteCompetitor(brandID, id),
    onSuccess: invalidate,
  });
  const confirm = useMutation({
    mutationFn: (id: string) => api.competitors.confirmCompetitor(brandID, id),
    onSuccess: invalidate,
  });
  const confirmAll = useMutation({
    mutationFn: () => api.competitors.confirmAllCompetitors(brandID),
    onSuccess: invalidate,
  });
  const hasUnconfirmed = competitors.some((competitor) => !competitor.confirmed);

  return (
    <section aria-labelledby="competitors-heading">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 id="competitors-heading" className="text-lg font-medium tracking-tight">
            {t("lists.competitors.title")}
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("lists.competitors.description")}</p>
        </div>
        {hasUnconfirmed ? (
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={confirmAll.isPending}
            onClick={() => confirmAll.mutate()}
          >
            {t("lists.confirmAll")}
          </Button>
        ) : null}
      </div>
      <div className="mt-6">
        <CompetitorForm
          pending={create.isPending}
          onSave={async (request) => {
            await create.mutateAsync(request);
          }}
        />
      </div>
      {competitors.length === 0 ? (
        <p className="mt-6 text-sm text-muted-foreground">{t("lists.competitors.empty")}</p>
      ) : (
        <ul className="mt-6 space-y-5">
          {competitors.map((competitor) => (
            <li key={competitor.id}>
              {editingID === competitor.id ? (
                <CompetitorForm
                  initial={competitor}
                  pending={update.isPending}
                  onCancel={() => setEditingID(undefined)}
                  onSave={async (request) => {
                    await update.mutateAsync({ id: competitor.id, request });
                    setEditingID(undefined);
                  }}
                />
              ) : (
                <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                  <div className="flex min-w-0 gap-3">
                    <BrandIcon name={competitor.name} domain={competitor.domain} />
                    <div className="min-w-0">
                      <p className="font-medium">{competitor.name}</p>
                      <p className="text-sm text-muted-foreground">{competitor.domain}</p>
                      {competitor.aliases.length > 0 ? (
                        <p className="mt-1 text-xs text-muted-foreground">
                          {t("lists.competitors.aliasesValue", {
                            aliases: competitor.aliases.join(", "),
                          })}
                        </p>
                      ) : null}
                      <p className="mt-1 text-xs text-muted-foreground">
                        {competitor.source === "manual"
                          ? t("lists.source.manual")
                          : competitor.source === "drafted"
                            ? t("lists.source.drafted")
                            : t("lists.source.observed")}
                        {" · "}
                        {competitor.confirmed ? t("lists.confirmed") : t("lists.unconfirmed")}
                      </p>
                    </div>
                  </div>
                  <div className="flex shrink-0 flex-wrap gap-1">
                    {!competitor.confirmed ? (
                      <Button
                        type="button"
                        size="xs"
                        variant="outline"
                        disabled={confirm.isPending}
                        onClick={() => confirm.mutate(competitor.id)}
                      >
                        {t("lists.confirm")}
                      </Button>
                    ) : null}
                    <Button
                      type="button"
                      size="xs"
                      variant="ghost"
                      onClick={() => setEditingID(competitor.id)}
                    >
                      {t("lists.edit")}
                    </Button>
                    <Button
                      type="button"
                      size="xs"
                      variant="ghost"
                      disabled={remove.isPending}
                      onClick={() => {
                        if (window.confirm(t("lists.competitors.deleteConfirm")))
                          remove.mutate(competitor.id);
                      }}
                    >
                      {t("lists.delete")}
                    </Button>
                  </div>
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function LoadState({ failed }: { failed: boolean }) {
  const { t } = useTranslation("brand");
  return (
    <p role={failed ? "alert" : undefined} className="text-sm text-muted-foreground">
      {failed ? t("lists.loadError") : t("lists.loading")}
    </p>
  );
}

export function PromptEditor({ brandID }: { brandID: string }) {
  const api = useApi();
  const query = useQuery({
    queryKey: promptKey(brandID),
    queryFn: () => api.prompts.listPrompts(brandID),
  });
  if (!query.data) return <LoadState failed={query.isError} />;
  return <PromptEditorView brandID={brandID} prompts={query.data} />;
}

export function CompetitorEditor({ brandID }: { brandID: string }) {
  const api = useApi();
  const query = useQuery({
    queryKey: competitorKey(brandID),
    queryFn: () => api.competitors.listCompetitors(brandID),
  });
  if (!query.data) return <LoadState failed={query.isError} />;
  return <CompetitorEditorView brandID={brandID} competitors={query.data} />;
}

export function VisibilityListEditors({
  brandID,
  onFinish,
}: {
  brandID: string;
  onFinish: () => void;
}) {
  const { t } = useTranslation("brand");
  const api = useApi();
  const prompts = useQuery({
    queryKey: promptKey(brandID),
    queryFn: () => api.prompts.listPrompts(brandID),
  });
  const competitors = useQuery({
    queryKey: competitorKey(brandID),
    queryFn: () => api.competitors.listCompetitors(brandID),
  });
  if (!prompts.data || !competitors.data) {
    return <LoadState failed={prompts.isError || competitors.isError} />;
  }
  const ready = listsReady(prompts.data, competitors.data);

  return (
    <div className="space-y-12">
      <PromptEditorView brandID={brandID} prompts={prompts.data} />
      <CompetitorEditorView brandID={brandID} competitors={competitors.data} />
      <div>
        <Button type="button" size="lg" disabled={!ready} onClick={onFinish}>
          {t("lists.finish")}
        </Button>
        {!ready ? (
          <p className="mt-2 text-xs text-muted-foreground">{t("lists.notReady")}</p>
        ) : null}
      </div>
    </div>
  );
}
