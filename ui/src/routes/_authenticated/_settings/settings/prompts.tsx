import { createFileRoute } from "@tanstack/react-router";

import { PromptEditor } from "@/components/blocks/visibility-list-editors";

export const Route = createFileRoute("/_authenticated/_settings/settings/prompts")({
  component: PromptSettings,
});

function PromptSettings() {
  const { brand } = Route.useRouteContext();
  return (
    <div className="mx-auto max-w-2xl">
      <PromptEditor brandID={brand.id} />
    </div>
  );
}
