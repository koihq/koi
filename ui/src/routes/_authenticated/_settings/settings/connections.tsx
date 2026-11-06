import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
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
import { formatDay } from "@/lib/format";
import { capabilitiesOptions } from "@/lib/queries";
import { credentialsKey } from "@/lib/query-keys";

type McpClient = {
  name: string;
  logo: string;
  instructions: "chatgptStep" | "claudeStep" | "commandStep" | "cursorStep";
  signIn: "oauthStep" | "codexSignInStep" | "claudeCodeSignInStep";
  docs: string;
  /** Terminal clients take a whole command; the others take the URL in their settings. */
  command?: (url: string) => string;
};

const clients: McpClient[] = [
  {
    name: "ChatGPT",
    logo: openaiLogo,
    instructions: "chatgptStep",
    signIn: "oauthStep",
    docs: "https://developers.openai.com/api/docs/guides/custom-mcp-server",
  },
  {
    name: "Claude",
    logo: claudeLogo,
    instructions: "claudeStep",
    signIn: "oauthStep",
    docs: "https://support.claude.com/en/articles/11175166-get-started-with-custom-connectors-using-remote-mcp",
  },
  {
    name: "Codex",
    logo: openaiLogo,
    instructions: "commandStep",
    signIn: "codexSignInStep",
    docs: "https://developers.openai.com/codex/mcp",
    command: (url) => `codex mcp add koi --url ${url}`,
  },
  {
    name: "Claude Code",
    logo: claudeLogo,
    instructions: "commandStep",
    signIn: "claudeCodeSignInStep",
    docs: "https://code.claude.com/docs/en/mcp",
    command: (url) => `claude mcp add --transport http koi ${url}`,
  },
  {
    name: "Cursor",
    logo: cursorLogo,
    instructions: "cursorStep",
    signIn: "oauthStep",
    docs: "https://cursor.com/docs/context/mcp",
  },
];

const COPIED_FOR_MS = 2_000;

export const Route = createFileRoute("/_authenticated/_settings/settings/connections")({
  component: ConnectionsSettings,
});

function ClientLogo({ logo, className }: { logo: string; className: string }) {
  return (
    <img
      src={logo}
      alt=""
      className={logo === claudeLogo ? className : `${className} dark:invert`}
    />
  );
}

function Step({ number, children }: { number: number; children: React.ReactNode }) {
  return (
    <li className="grid grid-cols-[1.5rem_minmax(0,1fr)] gap-3">
      <span
        aria-hidden="true"
        className="grid size-6 place-items-center rounded-full bg-muted text-xs text-muted-foreground tabular-nums"
      >
        {number}
      </span>
      <div className="grid min-w-0 gap-3 pt-0.5 text-sm">{children}</div>
    </li>
  );
}

function ConnectionsSettings() {
  const { t, i18n } = useTranslation("connections");
  const api = useApi();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [client, setClient] = useState(clients[0]);
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
  const copyValue = mcpUrl && (client.command ? client.command(mcpUrl) : mcpUrl);

  useEffect(() => {
    if (!copied) return;

    const timer = window.setTimeout(() => setCopied(false), COPIED_FOR_MS);

    return () => window.clearTimeout(timer);
  }, [copied]);

  async function copy(value: string) {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setCopyError(false);
    } catch {
      setCopyError(true);
    }
  }

  function credentialRow(item: Credential) {
    const knownClient = clients.find(
      (entry) => entry.name.toLowerCase() === item.name?.toLowerCase(),
    );

    return (
      <li
        key={item.id}
        className="flex min-w-0 items-center justify-between gap-3 rounded-control px-2 py-2 hover:bg-signal-pill"
      >
        <div className="flex min-w-0 items-center gap-2.5">
          {knownClient ? (
            <ClientLogo logo={knownClient.logo} className="size-5 shrink-0" />
          ) : (
            <BrandIcon
              name={item.name || item.clientId || "?"}
              lookupLogo={false}
              className="size-5 shrink-0"
            />
          )}
          <div className="min-w-0">
            <p className="truncate text-body font-medium" title={item.name || item.clientId}>
              {item.name || item.clientId}
            </p>
            {item.expiresAt && (
              <p className="text-xs text-muted-foreground tabular-nums">
                {t("expires", { date: formatDay(item.expiresAt, i18n.language) })}
              </p>
            )}
          </div>
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
    <div className="grid gap-6 pb-24 xl:grid-cols-[minmax(0,1fr)_22rem]">
      <section
        className="min-w-0 rounded-panel bg-surface p-4 sm:px-11 sm:py-9"
        aria-labelledby="connect-heading"
      >
        <h1 id="connect-heading" className="flex min-h-8 items-center text-section font-semibold">
          {t("connectTitle")}
        </h1>
        <fieldset className="-ms-1 mt-3 flex flex-wrap gap-0.5">
          <legend className="sr-only">{t("agent")}</legend>
          {clients.map((entry) => (
            <Button
              key={entry.name}
              type="button"
              variant="ghost"
              size="pill"
              aria-pressed={client.name === entry.name}
              className="gap-2 pe-3 text-muted-foreground"
              onClick={() => {
                setClient(entry);
                setCopied(false);
                setCopyError(false);
              }}
            >
              <ClientLogo logo={entry.logo} className="size-4" />
              {entry.name}
            </Button>
          ))}
        </fieldset>

        <ol className="mt-7 grid max-w-2xl gap-6">
          <Step number={1}>
            <p>{t(client.instructions)}</p>
            {capabilities.isPending ? (
              <output className="text-sm text-muted-foreground">{t("loadingUrl")}</output>
            ) : capabilities.isError ? (
              <p role="alert" className="text-sm text-destructive">
                {t("urlError")}{" "}
                <Button variant="link" onClick={() => void capabilities.refetch()}>
                  {t("retry")}
                </Button>
              </p>
            ) : !copyValue ? (
              <p role="alert" className="text-sm text-muted-foreground">
                {t("urlNotConfigured")}
              </p>
            ) : (
              <div className="flex min-w-0 items-center gap-3 rounded-control bg-muted/40 py-2 ps-3 pe-2">
                <span className="min-w-0 flex-1 break-all text-sm font-medium select-all">
                  {copyValue}
                </span>
                <Button variant="secondary" size="lg" onClick={() => void copy(copyValue)}>
                  {copied ? t("copied") : t("copy")}
                </Button>
              </div>
            )}
            {copyError && (
              <p role="alert" className="text-sm text-destructive">
                {t("copyError")}
              </p>
            )}
          </Step>
          <Step number={2}>
            <p>{t(client.signIn)}</p>
          </Step>
          <Step number={3}>
            <p>{t("tryStep")}</p>
          </Step>
        </ol>

        <a
          className="mt-7 inline-block text-sm underline underline-offset-4"
          href={client.docs}
          target="_blank"
          rel="noreferrer"
        >
          {t("clientDocs", { client: client.name })}
        </a>
      </section>

      <aside
        className="min-w-0 rounded-panel bg-surface p-4 sm:px-11 sm:py-9 xl:px-5"
        aria-labelledby="connected-heading"
      >
        <h2 id="connected-heading" className="flex min-h-8 items-center text-section font-semibold">
          {t("connectedTitle")}
        </h2>

        {credentials.isPending ? (
          <p className="mt-3 text-sm text-muted-foreground">{t("loading")}</p>
        ) : credentials.isError ? (
          <p role="alert" className="mt-3 text-sm text-destructive">
            {t("loadError")}{" "}
            <Button variant="link" onClick={() => void credentials.refetch()}>
              {t("retry")}
            </Button>
          </p>
        ) : items.length ? (
          <ul className="list-rows -mx-2 mt-3 grid gap-0.5">{items.map(credentialRow)}</ul>
        ) : (
          <p className="mt-3 text-xs text-muted-foreground">{t("emptyConnections")}</p>
        )}

        {credentials.data && credentials.data.totalPages > 1 && (
          <nav
            aria-label={t("pagination")}
            className="mt-4 flex items-center justify-between gap-4 text-xs text-muted-foreground"
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
      </aside>

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
