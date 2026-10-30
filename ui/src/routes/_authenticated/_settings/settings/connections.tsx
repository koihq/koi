import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "@/api/apiProvider";
import type { Credential } from "@/api/generated";
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
import { Button } from "@/components/ui/button";
import { credentialsKey } from "@/lib/query-keys";

export const Route = createFileRoute("/_authenticated/_settings/settings/connections")({
  component: ConnectionsSettings,
});

function ConnectionsSettings() {
  const { t, i18n } = useTranslation("connections");
  const api = useApi();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [toRevoke, setToRevoke] = useState<Credential | null>(null);
  const credentials = useQuery({
    queryKey: [...credentialsKey, page],
    queryFn: () => api.credentials.list(page),
  });
  const revoke = useMutation({
    mutationFn: (id: string) => api.credentials.revoke(id),
    onSuccess: async () => {
      setToRevoke(null);
      setPage(1);
      await queryClient.invalidateQueries({ queryKey: credentialsKey });
    },
  });
  const items = credentials.data?.items ?? [];

  function credentialRow(item: Credential) {
    return (
      <li key={item.id} className="flex flex-wrap items-center justify-between gap-4 py-3">
        <div className="min-w-0">
          <p className="break-words font-medium">{item.name || item.clientId}</p>
          {item.expiresAt && (
            <p className="text-xs text-muted-foreground">
              {t("expires", {
                date: new Intl.DateTimeFormat(i18n.language).format(item.expiresAt),
              })}
            </p>
          )}
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => {
            revoke.reset();
            setToRevoke(item);
          }}
        >
          {t("revoke")}
        </Button>
      </li>
    );
  }

  return (
    <div className="max-w-2xl pb-24">
      <h1 className="text-[30px] font-semibold tracking-[-0.035em]">{t("title")}</h1>
      <p className="mt-2 text-sm text-muted-foreground">{t("description")}</p>

      {credentials.isPending ? (
        <p className="mt-8 text-sm text-muted-foreground">{t("loading")}</p>
      ) : credentials.isError ? (
        <p role="alert" className="mt-8 text-sm text-destructive">
          {t("loadError")}{" "}
          <Button variant="link" onClick={() => void credentials.refetch()}>
            {t("retry")}
          </Button>
        </p>
      ) : items.length ? (
        <ul className="mt-8 space-y-1">{items.map(credentialRow)}</ul>
      ) : (
        <p className="mt-8 text-sm text-muted-foreground">{t("emptyConnections")}</p>
      )}

      {credentials.data && credentials.data.totalPages > 1 && (
        <nav
          aria-label={t("pagination")}
          className="mt-8 flex items-center justify-between gap-4 text-xs text-muted-foreground"
        >
          <span>{t("page", { current: page, total: credentials.data.totalPages })}</span>
          <div className="flex gap-2">
            <Button
              variant="ghost"
              size="sm"
              disabled={!credentials.data.hasPrev}
              onClick={() => setPage(page - 1)}
            >
              {t("previous")}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              disabled={!credentials.data.hasNext}
              onClick={() => setPage(page + 1)}
            >
              {t("next")}
            </Button>
          </div>
        </nav>
      )}

      <AlertDialog
        open={toRevoke !== null}
        onOpenChange={(open) => {
          if (!open) setToRevoke(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("revokeTitle")}</AlertDialogTitle>
            <AlertDialogDescription>{t("revokeDescription")}</AlertDialogDescription>
          </AlertDialogHeader>
          {revoke.isError && (
            <p role="alert" className="text-sm text-destructive">
              {t("revokeError")}
            </p>
          )}
          <AlertDialogFooter>
            <AlertDialogCancel>{t("cancel")}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={revoke.isPending}
              onClick={() => {
                if (toRevoke) revoke.mutate(toRevoke.id);
              }}
            >
              {t("revoke")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
