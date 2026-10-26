import { useTranslation } from "react-i18next";
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

export function DeleteItemDialog({
  itemID,
  title,
  description,
  actionLabel,
  onClose,
  pending,
  failed,
  onDelete,
  finalFocus,
}: {
  itemID?: string;
  title: string;
  description: string;
  actionLabel: string;
  onClose: () => void;
  pending: boolean;
  failed: boolean;
  onDelete: (id: string) => Promise<unknown>;
  finalFocus: () => HTMLElement | null;
}) {
  const { t } = useTranslation("management");

  return (
    <AlertDialog
      open={Boolean(itemID)}
      onOpenChange={(open) => {
        if (!open && !pending) onClose();
      }}
    >
      <AlertDialogContent finalFocus={finalFocus}>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
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
              if (!itemID) return;

              try {
                await onDelete(itemID);
                onClose();
              } catch {
                // The mutation error remains visible for retry.
              }
            }}
          >
            {actionLabel}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
