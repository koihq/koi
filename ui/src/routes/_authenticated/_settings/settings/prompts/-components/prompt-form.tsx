import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";

import type { CreatePrompt } from "@/api/prompts.api";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { promptFormSchema } from "@/routes/_authenticated/_settings/settings/prompts/-utils/prompt-form-schema";

export function PromptForm({
  initialText = "",
  pending,
  onSave,
  onCancel,
}: {
  initialText?: string;
  pending: boolean;
  onSave: (request: CreatePrompt) => Promise<void>;
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
