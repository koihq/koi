import { createFileRoute } from "@tanstack/react-router";

import { LoadState } from "@/components/blocks/load-state";
import { CompetitorEditor } from "@/routes/_authenticated/_settings/settings/competitors/-components/competitor-editor";
import { useCompetitorEditor } from "@/routes/_authenticated/_settings/settings/competitors/-hooks/use-competitor-editor";

export const Route = createFileRoute("/_authenticated/_settings/settings/competitors/")({
  component: CompetitorSettings,
});

function CompetitorSettings() {
  const { brand } = Route.useRouteContext();
  const editor = useCompetitorEditor(brand.id);
  if (!editor.competitors.data || !editor.unconfirmed.data) {
    return <LoadState failed={editor.competitors.isError || editor.unconfirmed.isError} />;
  }

  return (
    <div className="mx-auto max-w-2xl">
      <CompetitorEditor
        competitors={editor.competitors.data}
        unconfirmedItems={editor.unconfirmed.data.totalItems}
        onPageChange={editor.setPage}
        onCreate={editor.create.mutateAsync}
        onUpdate={(id, request) => editor.update.mutateAsync({ id, request })}
        onDelete={(id) => editor.remove.mutate(id)}
        onConfirm={(id) => editor.confirm.mutate(id)}
        onConfirmAll={() => editor.confirmAll.mutate()}
        pending={{
          create: editor.create.isPending,
          update: editor.update.isPending,
          delete: editor.remove.isPending,
          confirm: editor.confirm.isPending,
          confirmAll: editor.confirmAll.isPending,
        }}
      />
    </div>
  );
}
