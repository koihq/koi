import { useTranslation } from "react-i18next";
import type { Competitor } from "@/api/competitors.api";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";

export function DeleteCompetitorDialog({
  item,
  onClose,
  pending,
  failed,
  onDelete,
  finalFocus,
}: {
  item?: Competitor;
  onClose: () => void;
  pending: boolean;
  failed: boolean;
  onDelete: (id: string) => Promise<unknown>;
  finalFocus: () => HTMLElement | null;
}) {
  const { t } = useTranslation("management");

  return (
    <AlertDialog
      open={Boolean(item)}
      onOpenChange={(open) => {
        if (!open && !pending) onClose();
      }}
    >
      <AlertDialogContent finalFocus={finalFocus}>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("deleteCompetitorTitle")}</AlertDialogTitle>
          <AlertDialogDescription>{t("competitorDeleteDescription")}</AlertDialogDescription>
        </AlertDialogHeader>
        {failed ? (
          <p role="alert" className="text-sm text-destructive">
            {t("deleteError")}
          </p>
        ) : null}
        <AlertDialogFooter>
          <AlertDialogCancel disabled={pending}>{t("cancel")}</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            disabled={pending}
            onClick={async () => {
              if (!item) return;

              try {
                await onDelete(item.id);
                onClose();
              } catch {
                // The mutation error remains visible for retry.
              }
            }}
          >
            {t("deleteCompetitor")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
