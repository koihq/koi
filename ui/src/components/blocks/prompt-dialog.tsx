import { useState } from "react";
import { useTranslation } from "react-i18next";

import type { CreatePrompt, Prompt } from "@/api/prompts.api";
import { PromptForm } from "@/components/blocks/prompt-form";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";

export function PromptDialog({
  prompt,
  label,
  pending,
  onSave,
  variant = "default",
  open: controlledOpen,
  onOpenChange,
  hideTrigger = false,
}: {
  prompt?: Prompt;
  label: string;
  pending: boolean;
  onSave: (request: CreatePrompt) => Promise<void>;
  variant?: "default" | "ghost";
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  hideTrigger?: boolean;
}) {
  const { t } = useTranslation("management");
  const [open, setOpen] = useState(false);
  const visible = controlledOpen ?? open;
  const changeOpen = onOpenChange ?? setOpen;

  return (
    <Dialog open={visible} onOpenChange={changeOpen}>
      {!hideTrigger && (
        <DialogTrigger render={<Button type="button" variant={variant} size="sm" />}>
          {label}
        </DialogTrigger>
      )}
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{prompt ? t("editPrompt") : t("addPrompt")}</DialogTitle>
          <DialogDescription>{t("promptDescription")}</DialogDescription>
        </DialogHeader>
        <PromptForm
          key={prompt?.id ?? "new"}
          initialText={prompt?.text}
          pending={pending}
          onCancel={() => changeOpen(false)}
          onSave={async (request) => {
            await onSave(request);
            changeOpen(false);
          }}
        />
      </DialogContent>
    </Dialog>
  );
}
