import { type FormEvent, useState } from "react";
import type { Competitor, CreateCompetitor } from "@/api/competitors.api";
import { brandFormSchema } from "@/components/blocks/brand-form-schema";
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
} from "@/components/ui/dialog";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { SUGGESTIONS } from "../-lib/exploration-fixtures";
import { HOME_LOGOS } from "../-lib/home-evidence";
import C from "../-locales/mvp.en.json";

const directory = [
  ...Object.entries(HOME_LOGOS).flatMap(([name, logo]) =>
    logo.domain ? [{ name, domain: logo.domain }] : [],
  ),
  ...Object.values(SUGGESTIONS),
];
const fieldNames = ["name", "domain", "githubRepo", "aliases", "additionalUrls"] as const;
type FieldName = (typeof fieldNames)[number];

export function CompetitorEditor({
  competitor,
  lookup,
  onClose,
  onSave,
  finalFocus,
}: {
  competitor?: Competitor;
  lookup: boolean;
  onClose: () => void;
  onSave: (request: CreateCompetitor) => void;
  finalFocus: () => HTMLElement | null;
}) {
  const [values, setValues] = useState({
    name: competitor?.name ?? "",
    domain: competitor?.domain ?? "",
    githubRepo: competitor?.githubRepo ?? "",
    aliases: competitor?.aliases.join(", ") ?? "",
    additionalUrls: competitor?.additionalUrls.join(", ") ?? "",
  });
  const [errors, setErrors] = useState<Partial<Record<FieldName, string>>>({});
  const [error, setError] = useState("");
  const [details, setDetails] = useState(
    Boolean(
      competitor?.githubRepo || competitor?.aliases.length || competitor?.additionalUrls.length,
    ),
  );
  const split = (value: string) =>
    value
      .split(",")
      .map((item) => item.trim())
      .filter(Boolean);

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    const form = event.currentTarget;
    const parsed = brandFormSchema.safeParse({ ...values, country: "US", description: "" });
    const next: Partial<Record<FieldName, string>> = {};
    const allowNameOnly =
      competitor?.source === "observed" &&
      !competitor.confirmed &&
      !values.domain.trim() &&
      !values.githubRepo.trim();
    if (!parsed.success) {
      for (const issue of parsed.error.issues) {
        const field = issue.path[0] as FieldName;
        if (allowNameOnly && (field === "domain" || field === "githubRepo")) continue;
        next[field] =
          field === "domain"
            ? C.invalidDomain
            : field === "githubRepo"
              ? C.invalidGithub
              : C.required;
      }
    }

    const additionalUrls = split(values.additionalUrls);
    for (const value of additionalUrls) {
      try {
        const url = new URL(value);
        const repo =
          url.hostname === "github.com" &&
          brandFormSchema.safeParse({
            name: "Repository",
            domain: "",
            githubRepo: value,
            country: "US",
            description: "",
          }).success;
        if (
          !["https:", "http:"].includes(url.protocol) ||
          url.username ||
          url.password ||
          url.search ||
          url.hash ||
          (url.pathname !== "/" && !repo)
        )
          next.additionalUrls = C.invalidURLs;
      } catch {
        next.additionalUrls = C.invalidURLs;
      }
    }

    setErrors(next);
    if (Object.keys(next).length) {
      if (next.githubRepo || next.additionalUrls) setDetails(true);
      requestAnimationFrame(() => form.querySelector<HTMLElement>("[aria-invalid=true]")?.focus());
      return;
    }

    try {
      onSave({
        name: values.name.trim(),
        domain: values.domain.trim().toLowerCase(),
        githubRepo: values.githubRepo.trim(),
        aliases: split(values.aliases),
        additionalUrls,
      });
      onClose();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : C.saveError);
    }
  }

  function field(name: FieldName) {
    const id = `hc-competitor-${name}`;
    return (
      <Field key={name} data-invalid={Boolean(errors[name]) || undefined}>
        <FieldLabel htmlFor={id}>{C[name]}</FieldLabel>
        <Input
          id={id}
          value={values[name]}
          onChange={(event) => setValues((current) => ({ ...current, [name]: event.target.value }))}
          aria-invalid={Boolean(errors[name]) || undefined}
          aria-describedby={errors[name] ? `${id}-error` : undefined}
          autoCapitalize={name === "name" ? "words" : "none"}
          spellCheck={name === "name"}
        />
        <FieldError id={`${id}-error`}>{errors[name]}</FieldError>
      </Field>
    );
  }

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <DialogContent className="hc-form-dialog" finalFocus={finalFocus}>
        <DialogHeader>
          <DialogTitle>{competitor ? C.editCompetitor : C.addCompetitor}</DialogTitle>
          <DialogDescription>{C.identityHelp}</DialogDescription>
        </DialogHeader>
        {!competitor && lookup && (
          <Field>
            <FieldLabel htmlFor="hc-competitor-lookup">{C.lookup}</FieldLabel>
            <Combobox<(typeof directory)[number]>
              items={directory}
              itemToStringLabel={(item) => item.name}
              onValueChange={(item) => {
                if (item)
                  setValues((current) => ({ ...current, name: item.name, domain: item.domain }));
              }}
            >
              <ComboboxInput id="hc-competitor-lookup" placeholder={C.lookupHint} />
              <ComboboxContent className="hc-brand-menu">
                <ComboboxEmpty>{C.lookupEmpty}</ComboboxEmpty>
                <ComboboxList>
                  {(item: (typeof directory)[number]) => (
                    <ComboboxItem key={item.domain} value={item}>
                      {item.name}
                    </ComboboxItem>
                  )}
                </ComboboxList>
              </ComboboxContent>
            </Combobox>
          </Field>
        )}
        <form onSubmit={submit} noValidate>
          <FieldGroup>
            {field("name")}
            {field("domain")}
            <details open={details} onToggle={(event) => setDetails(event.currentTarget.open)}>
              <summary>{C.moreDetails}</summary>
              <FieldGroup>
                {field("githubRepo")}
                {field("aliases")}
                {field("additionalUrls")}
              </FieldGroup>
            </details>
          </FieldGroup>
          {error && (
            <p className="hc-form-error" role="alert">
              {error}
            </p>
          )}
          <div className="hc-form-actions">
            <Button type="button" variant="ghost" onClick={onClose}>
              {C.cancel}
            </Button>
            <Button type="submit">{competitor ? C.save : C.addCompetitor}</Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
