import { type FormEvent, useState } from "react";
import type { Prompt } from "@/api/prompts.api";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { promptFormSchema } from "@/lib/prompt-form-schema";
import C from "../-locales/mvp.en.json";

export function PromptEditor({
  prompt,
  onClose,
  onSave,
  finalFocus,
}: {
  prompt?: Prompt;
  onClose: () => void;
  onSave: (text: string) => void;
  finalFocus: () => HTMLElement | null;
}) {
  const [text, setText] = useState(prompt?.text ?? "");
  const [error, setError] = useState("");
  const [invalid, setInvalid] = useState(false);

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    const parsed = promptFormSchema.safeParse({ text });
    setInvalid(!parsed.success);
    if (!parsed.success) {
      event.currentTarget.querySelector<HTMLInputElement>("input")?.focus();
      return;
    }

    try {
      onSave(parsed.data.text);
      onClose();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : C.saveError);
    }
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
          <DialogTitle>{prompt ? C.editPrompt : C.addPrompt}</DialogTitle>
          <DialogDescription>{C.previewNote}</DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} noValidate>
          <Field data-invalid={invalid || undefined}>
            <FieldLabel htmlFor="hc-prompt-text">{C.prompt}</FieldLabel>
            <Input
              id="hc-prompt-text"
              value={text}
              onChange={(event) => setText(event.target.value)}
              aria-invalid={invalid || undefined}
              aria-describedby={invalid ? "hc-prompt-error" : undefined}
            />
            <FieldError id="hc-prompt-error">{invalid ? C.required : null}</FieldError>
          </Field>
          {error && (
            <p className="hc-form-error" role="alert">
              {error}
            </p>
          )}
          <div className="hc-form-actions">
            <Button variant="ghost" type="button" onClick={onClose}>
              {C.cancel}
            </Button>
            <Button type="submit">{prompt ? C.save : C.addPrompt}</Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
