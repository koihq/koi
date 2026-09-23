import { createFileRoute } from "@tanstack/react-router";

import { LoadState } from "@/components/blocks/load-state";
import { PromptEditor } from "@/routes/_authenticated/_settings/settings/prompts/-components/prompt-editor";
import { usePromptEditor } from "@/routes/_authenticated/_settings/settings/prompts/-hooks/use-prompt-editor";

export const Route = createFileRoute("/_authenticated/_settings/settings/prompts/")({
  component: PromptSettings,
});

function PromptSettings() {
  const { brand } = Route.useRouteContext();
  const editor = usePromptEditor(brand.id);
  if (!editor.prompts.data || !editor.unconfirmed.data) {
    return <LoadState failed={editor.prompts.isError || editor.unconfirmed.isError} />;
  }

  return (
    <div className="mx-auto max-w-2xl">
      <PromptEditor
        prompts={editor.prompts.data}
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
