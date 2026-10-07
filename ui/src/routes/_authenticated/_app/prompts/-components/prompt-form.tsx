import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";

import type { CreatePrompt } from "@/api/prompts.api";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { promptFormSchema } from "@/routes/_authenticated/_app/prompts/-types/prompt-form-schema";

export function PromptForm({
  initialText = "",
  pending,
  onSave,
  onCancel,
}: {
  initialText?: string;
  pending: boolean;
  onSave: (request: CreatePrompt) => Promise<void>;
  onCancel: () => void;
}) {
  const { t } = useTranslation("management");
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
    } catch {
      setFailed(true);
    }
  }

  return (
    <form onSubmit={submit} className="space-y-4">
      <Field data-invalid={invalid || undefined}>
        <FieldLabel htmlFor="prompt-text">{t("prompt")}</FieldLabel>
        <Input
          id="prompt-text"
          autoFocus
          value={text}
          onChange={(event) => setText(event.target.value)}
          aria-invalid={invalid || undefined}
          placeholder={t("promptPlaceholder")}
        />
        {invalid ? <FieldError>{t("promptInvalid")}</FieldError> : null}
        {failed ? <FieldError>{t("promptSaveError")}</FieldError> : null}
      </Field>
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          {t("cancel")}
        </Button>
        <Button type="submit" disabled={pending}>
          {t("savePrompt")}
        </Button>
      </div>
    </form>
  );
}
