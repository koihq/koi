import { useState } from "react";
import { useTranslation } from "react-i18next";

import type { CreatePrompt, Prompt } from "@/api/prompts.api";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { PromptForm } from "@/routes/_authenticated/_app/prompts/-components/prompt-form";

export function PromptDialog({
  prompt,
  label,
  pending,
  onSave,
  variant = "default",
}: {
  prompt?: Prompt;
  label: string;
  pending: boolean;
  onSave: (request: CreatePrompt) => Promise<void>;
  variant?: "default" | "ghost";
}) {
  const { t } = useTranslation("management");
  const [open, setOpen] = useState(false);
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button type="button" variant={variant} size="sm" />}>
        {label}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{prompt ? t("editPrompt") : t("addPrompt")}</DialogTitle>
          <DialogDescription>{t("promptDescription")}</DialogDescription>
        </DialogHeader>
        <PromptForm
          key={prompt?.id ?? "new"}
          initialText={prompt?.text}
          pending={pending}
          onCancel={() => setOpen(false)}
          onSave={async (request) => {
            await onSave(request);
            setOpen(false);
          }}
        />
      </DialogContent>
    </Dialog>
  );
}
