import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "@/api/apiProvider";
import type { Credential } from "@/api/generated";
import claudeLogo from "@/assets/mcp-clients/claude.svg";
import cursorLogo from "@/assets/mcp-clients/cursor.svg";
import openaiLogo from "@/assets/mcp-clients/openai.svg";
import { BrandIcon } from "@/components/blocks/brand-icon";
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
import { capabilitiesOptions } from "@/lib/queries";
import { credentialsKey } from "@/lib/query-keys";

const clients = [
  {
    name: "ChatGPT",
    logo: openaiLogo,
    instructions: "chatgptStep",
    docs: "https://developers.openai.com/api/docs/guides/custom-mcp-server",
  },
  {
    name: "Claude",
    logo: claudeLogo,
    instructions: "claudeStep",
    docs: "https://support.claude.com/en/articles/11175166-get-started-with-custom-connectors-using-remote-mcp",
  },
  {
    name: "Codex",
    logo: openaiLogo,
    instructions: "codexStep",
    docs: "https://developers.openai.com/resources/docs-mcp",
  },
  {
    name: "Claude Code",
    logo: claudeLogo,
    instructions: "claudeCodeStep",
    docs: "https://docs.claude.com/en/docs/claude-code/mcp",
  },
  {
    name: "Cursor",
    logo: cursorLogo,
    instructions: "cursorStep",
    docs: "https://docs.cursor.com/context/model-context-protocol",
  },
] as const;

export const Route = createFileRoute("/_authenticated/_settings/settings/connections")({
  component: ConnectionsSettings,
});

function ConnectionsSettings() {
  const { t, i18n } = useTranslation("connections");
  const api = useApi();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [client, setClient] = useState<(typeof clients)[number]>(clients[0]);
  const [copied, setCopied] = useState(false);
  const [copyError, setCopyError] = useState(false);
  const capabilities = useQuery(capabilitiesOptions(api));
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
  const mcpUrl = capabilities.data?.mcpUrl;

  function credentialRow(item: Credential) {
    const knownClient = clients.find(
      (entry) => entry.name.toLowerCase() === item.name?.toLowerCase(),
    );

    return (
      <li key={item.id} className="flex flex-wrap items-center justify-between gap-4 py-3">
        <div className="min-w-0">
          <p className="flex items-center gap-3 break-words font-medium">
            {knownClient ? (
              <img
                src={knownClient.logo}
                alt=""
                className={knownClient.logo === claudeLogo ? "size-5" : "size-5 dark:invert"}
              />
            ) : (
              <BrandIcon
                name={item.name || item.clientId || "?"}
                lookupLogo={false}
                className="size-5"
              />
            )}
            {item.name || item.clientId}
          </p>
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
      <section className="mt-10" aria-labelledby="connect-heading">
        <h2 id="connect-heading" className="text-lg font-medium tracking-tight">
          {t("connectTitle")}
        </h2>
        <fieldset className="mt-5 flex flex-wrap gap-2">
          <legend className="sr-only">{t("agent")}</legend>
          {clients.map((entry) => (
            <Button
              key={entry.name}
              size="lg"
              variant={client.name === entry.name ? "secondary" : "ghost"}
              aria-pressed={client.name === entry.name}
              onClick={() => {
                setClient(entry);
                setCopied(false);
              }}
            >
              <img
                src={entry.logo}
                alt=""
                className={entry.logo === claudeLogo ? "size-4" : "size-4 dark:invert"}
              />
              {entry.name}
            </Button>
          ))}
        </fieldset>
        <div className="mt-7 rounded-lg bg-muted/40 p-5">
          <p className="text-xs text-muted-foreground">{t("serverUrl")}</p>
          <div className="mt-3 flex min-w-0 flex-wrap items-center gap-3">
            {capabilities.isPending ? (
              <output className="text-sm text-muted-foreground">{t("loadingUrl")}</output>
            ) : capabilities.isError || !mcpUrl ? (
              <p role="alert" className="text-sm text-destructive">
                {t("urlError")}{" "}
                <Button variant="link" onClick={() => void capabilities.refetch()}>
                  {t("retry")}
                </Button>
              </p>
            ) : (
              <>
                <span className="min-w-0 flex-1 break-all text-sm font-medium select-all">
                  {mcpUrl}
                </span>
                <Button
                  size="sm"
                  disabled={!mcpUrl}
                  onClick={async () => {
                    try {
                      await navigator.clipboard.writeText(mcpUrl);
                      setCopied(true);
                      setCopyError(false);
                    } catch {
                      setCopyError(true);
                    }
                  }}
                >
                  {copied ? t("copied") : t("copyUrl")}
                </Button>
              </>
            )}
          </div>
          {copyError && (
            <p role="alert" className="mt-2 text-sm text-destructive">
              {t("copyError")}
            </p>
          )}
        </div>
        <p className="mt-6 text-sm">{t(client.instructions)}</p>
        <p className="mt-2 text-sm text-muted-foreground">{t("oauthStep")}</p>
        <a
          className="mt-4 inline-block text-sm underline underline-offset-4"
          href={client.docs}
          target="_blank"
          rel="noreferrer"
        >
          {t("clientDocs", { client: client.name })}
        </a>
      </section>

      <section className="mt-16" aria-labelledby="connected-heading">
        <h2 id="connected-heading" className="text-lg font-medium tracking-tight">
          {t("connectedTitle")}
        </h2>

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
      </section>

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
