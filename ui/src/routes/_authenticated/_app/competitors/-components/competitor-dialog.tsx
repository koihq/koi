import { useState } from "react";
import { useTranslation } from "react-i18next";

import type { Competitor, CreateCompetitor } from "@/api/competitors.api";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { CompetitorForm } from "@/routes/_authenticated/_app/competitors/-components/competitor-form";

export function CompetitorDialog({
  competitor,
  label,
  pending,
  onSave,
  variant = "default",
}: {
  competitor?: Competitor;
  label: string;
  pending: boolean;
  onSave: (request: CreateCompetitor) => Promise<void>;
  variant?: "default" | "ghost";
}) {
  const { t } = useTranslation("management");
  const [open, setOpen] = useState(false);
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button type="button" variant={variant} size="sm" />}>
        {label}
      </DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{competitor ? t("editCompetitor") : t("addCompetitor")}</DialogTitle>
          <DialogDescription>{t("competitorDescription")}</DialogDescription>
        </DialogHeader>
        <CompetitorForm
          key={competitor?.id ?? "new"}
          initial={competitor}
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
