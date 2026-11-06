import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate, useRouter } from "@tanstack/react-router";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";

import { useApi } from "@/api/apiProvider";
import { ClientLogo, clients } from "@/components/blocks/mcp-clients";
import { Button } from "@/components/ui/button";
import { useCopy } from "@/hooks/use-copy";
import { accountOptions } from "@/lib/queries";
import { accountKey, credentialsKey } from "@/lib/query-keys";

const POLL_MS = 5_000;
const SKILLS_COMMAND = "npx skills add koihq/koi --skill koi-setup --skill koi-brief";

function agentClient(name: string | undefined) {
  return (
    clients.find((entry) => entry.name === name) ??
    clients.find((entry) => entry.command) ??
    clients[0]
  );
}

function CopyBlock({ value, prominent = false }: { value: string; prominent?: boolean }) {
  const { t } = useTranslation("brand");
  const { copy, copied, failed } = useCopy();

  return (
    <div className="grid gap-2">
      <div className="flex min-w-0 items-start gap-3 rounded-control bg-muted/40 py-2 ps-3 pe-2">
        <span className="min-w-0 flex-1 break-words whitespace-pre-wrap text-sm font-medium select-all">
          {value}
        </span>
        {!prominent && (
          <Button variant="secondary" onClick={() => void copy(value)}>
            {copied ? t("setup.copied") : t("setup.copy")}
          </Button>
        )}
      </div>
      {prominent && (
        <Button size="lg" className="justify-self-start" onClick={() => void copy(value)}>
          {copied ? t("setup.copied") : t("setup.copyPrompt")}
        </Button>
      )}
      {failed && (
        <p role="alert" className="text-sm text-destructive">
          {t("setup.copyError")}
        </p>
      )}
    </div>
  );
}

/** Watches for the agent's connection and for the brand it creates, which ends setup. */
function useAgentProgress() {
  const api = useApi();
  const router = useRouter();
  const queryClient = useQueryClient();

  const credentials = useQuery({
    queryKey: [...credentialsKey, "setup"],
    queryFn: () => api.credentials.list(),
    refetchInterval: POLL_MS,
  });
  const account = useQuery({ ...accountOptions(api), refetchInterval: POLL_MS, staleTime: 0 });
  const hasBrand = Boolean(account.data?.activeBrand);

  useEffect(() => {
    if (!hasBrand) return;

    void queryClient.invalidateQueries({ queryKey: accountKey }).then(() => router.invalidate());
  }, [hasBrand, queryClient, router]);

  return credentials.data?.items[0];
}

export function AgentConnect({ mcpUrl, clientName }: { mcpUrl: string; clientName?: string }) {
  const { t } = useTranslation("brand");
  const connections = useTranslation("connections");
  const navigate = useNavigate();
  const connected = useAgentProgress();
  const client = agentClient(clientName);
  const install = client.command ? `${client.command(mcpUrl)}\n${SKILLS_COMMAND}` : mcpUrl;

  // Signing in is the proof this step is done, so the next one opens by itself.
  useEffect(() => {
    if (!connected) return;

    void navigate({
      to: "/setup",
      search: { path: "agent", stage: "prompt", client: client.name },
      replace: true,
    });
  }, [client.name, connected, navigate]);

  return (
    <div className="grid gap-6">
      <fieldset className="-ms-1 flex flex-wrap gap-0.5">
        <legend className="sr-only">{t("setup.agentClient")}</legend>
        {clients.map((entry) => (
          <Button
            key={entry.name}
            type="button"
            variant="ghost"
            size="pill"
            aria-pressed={client.name === entry.name}
            className="gap-2 pe-3 text-muted-foreground"
            onClick={() =>
              void navigate({
                to: "/setup",
                search: { path: "agent", client: entry.name },
                replace: true,
              })
            }
          >
            <ClientLogo logo={entry.logo} className="size-4" />
            {entry.name}
          </Button>
        ))}
      </fieldset>

      <div className="grid gap-3 text-sm">
        <p>{client.command ? t("setup.agentInstall") : connections.t(client.instructions)}</p>
        <CopyBlock key={client.name} value={install} />
        <p className="text-muted-foreground">{connections.t(client.signIn)}</p>
      </div>

      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <Button
          size="lg"
          role="link"
          render={
            <Link to="/setup" search={{ path: "agent", stage: "prompt", client: client.name }} />
          }
          nativeButton={false}
        >
          {t("setup.agentConnectedContinue")}
        </Button>
        <Button
          variant="link"
          role="link"
          render={<Link to="/setup" search={{ path: "koi" }} />}
          nativeButton={false}
        >
          {t("setup.useKoi")}
        </Button>
      </div>
    </div>
  );
}

export function AgentPrompt({ clientName }: { clientName?: string }) {
  const { t } = useTranslation("brand");
  const connected = useAgentProgress();
  const client = agentClient(clientName);

  return (
    <div className="grid gap-6">
      <CopyBlock
        prominent
        value={t(client.command ? "setup.agentPromptText" : "setup.agentPromptTextWeb")}
      />
      <output className="block text-sm text-muted-foreground">
        {connected
          ? t("setup.agentConnected", { client: connected.name || connected.clientId })
          : t("setup.agentWaiting")}
      </output>
    </div>
  );
}
