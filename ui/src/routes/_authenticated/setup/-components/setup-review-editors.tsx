import { useTranslation } from "react-i18next";

import { LoadState } from "@/components/blocks/load-state";
import { Button } from "@/components/ui/button";
import { CompetitorEditor } from "@/routes/_authenticated/_settings/settings/competitors/-components/competitor-editor";
import { useCompetitorEditor } from "@/routes/_authenticated/_settings/settings/competitors/-hooks/use-competitor-editor";
import { PromptEditor } from "@/routes/_authenticated/_settings/settings/prompts/-components/prompt-editor";
import { usePromptEditor } from "@/routes/_authenticated/_settings/settings/prompts/-hooks/use-prompt-editor";
import { listsReady } from "@/routes/_authenticated/setup/-utils/lists-ready";

export function SetupReviewEditors({
  brandID,
  onFinish,
}: {
  brandID: string;
  onFinish: () => void;
}) {
  const { t } = useTranslation("brand");
  const prompts = usePromptEditor(brandID);
  const competitors = useCompetitorEditor(brandID);

  if (
    !prompts.prompts.data ||
    !prompts.unconfirmed.data ||
    !competitors.competitors.data ||
    !competitors.unconfirmed.data
  ) {
    return (
      <div className="mt-12">
        <LoadState
          failed={
            prompts.prompts.isError ||
            prompts.unconfirmed.isError ||
            competitors.competitors.isError ||
            competitors.unconfirmed.isError
          }
        />
      </div>
    );
  }

  const ready = listsReady(
    prompts.prompts.data,
    competitors.competitors.data,
    prompts.unconfirmed.data,
    competitors.unconfirmed.data,
  );

  return (
    <div className="mt-12 space-y-12">
      <PromptEditor
        prompts={prompts.prompts.data}
        unconfirmedItems={prompts.unconfirmed.data.totalItems}
        onPageChange={prompts.setPage}
        onCreate={prompts.create.mutateAsync}
        onUpdate={(id, request) => prompts.update.mutateAsync({ id, request })}
        onDelete={(id) => prompts.remove.mutate(id)}
        onConfirm={(id) => prompts.confirm.mutate(id)}
        onConfirmAll={() => prompts.confirmAll.mutate()}
        pending={{
          create: prompts.create.isPending,
          update: prompts.update.isPending,
          delete: prompts.remove.isPending,
          confirm: prompts.confirm.isPending,
          confirmAll: prompts.confirmAll.isPending,
        }}
      />
      <CompetitorEditor
        competitors={competitors.competitors.data}
        unconfirmedItems={competitors.unconfirmed.data.totalItems}
        onPageChange={competitors.setPage}
        onCreate={competitors.create.mutateAsync}
        onUpdate={(id, request) => competitors.update.mutateAsync({ id, request })}
        onDelete={(id) => competitors.remove.mutate(id)}
        onConfirm={(id) => competitors.confirm.mutate(id)}
        onConfirmAll={() => competitors.confirmAll.mutate()}
        pending={{
          create: competitors.create.isPending,
          update: competitors.update.isPending,
          delete: competitors.remove.isPending,
          confirm: competitors.confirm.isPending,
          confirmAll: competitors.confirmAll.isPending,
        }}
      />
      <div>
        <Button type="button" size="lg" disabled={!ready} onClick={onFinish}>
          {t("lists.finish")}
        </Button>
        {!ready ? (
          <p className="mt-2 text-xs text-muted-foreground">{t("lists.notReady")}</p>
        ) : null}
      </div>
    </div>
  );
}
