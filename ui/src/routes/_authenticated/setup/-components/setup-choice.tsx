import { Link } from "@tanstack/react-router";
import type { ReactNode } from "react";
import { useId, useState } from "react";
import { useTranslation } from "react-i18next";

import { ClientLogo, clients } from "@/components/blocks/mcp-clients";
import { Button } from "@/components/ui/button";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";

type SetupPath = "koi" | "agent";

const agentLogos = [...new Set(clients.map((client) => client.logo))];

function ChoiceCard({
  value,
  preview,
  title,
  description,
}: {
  value: SetupPath;
  preview: ReactNode;
  title: string;
  description: string;
}) {
  const id = useId();

  return (
    <label
      htmlFor={id}
      className="relative grid cursor-pointer content-start gap-4 rounded-panel border border-border p-3 hover:bg-signal-pill/40 has-data-checked:border-foreground has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-ring"
    >
      <RadioGroupItem id={id} value={value} className="absolute end-6 top-6" />
      <span
        aria-hidden="true"
        className="flex h-28 flex-col justify-center gap-1 rounded-control bg-muted/40 py-3 ps-4 pe-10"
      >
        {preview}
      </span>
      <span className="grid gap-1 px-1 pb-1">
        <span className="text-section font-semibold">{title}</span>
        <span className="text-sm text-muted-foreground">{description}</span>
      </span>
    </label>
  );
}

export function SetupChoice() {
  const { t } = useTranslation("brand");
  const [path, setPath] = useState<SetupPath>("koi");

  return (
    <div className="grid justify-items-start gap-6">
      <RadioGroup
        aria-label={t("setup.description")}
        value={path}
        onValueChange={(value) => setPath(value as SetupPath)}
        className="gap-3 sm:grid-cols-2"
      >
        <ChoiceCard
          value="koi"
          title={t("setup.koiOption")}
          description={t("setup.koiOptionDescription")}
          preview={
            <>
              <span className="text-xs text-muted-foreground">{t("setup.koiPreviewLabel")}</span>
              <span className="text-section font-semibold">{t("setup.koiPreview")}</span>
            </>
          }
        />
        <ChoiceCard
          value="agent"
          title={t("setup.agentOption")}
          description={t("setup.agentOptionDescription")}
          preview={
            <span className="flex items-center gap-4">
              {agentLogos.map((logo) => (
                <ClientLogo key={logo} logo={logo} className="size-7" />
              ))}
            </span>
          }
        />
      </RadioGroup>

      <Button
        size="lg"
        role="link"
        render={<Link to="/setup" search={{ path }} />}
        nativeButton={false}
      >
        {t("setup.continue")}
      </Button>
    </div>
  );
}
