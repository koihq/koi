import { createFileRoute } from "@tanstack/react-router";

import { CompetitorEditor } from "@/components/blocks/visibility-list-editors";

export const Route = createFileRoute("/_authenticated/_settings/settings/competitors")({
  component: CompetitorSettings,
});

function CompetitorSettings() {
  const { brand } = Route.useRouteContext();
  return (
    <div className="mx-auto max-w-2xl">
      <CompetitorEditor brandID={brand.id} />
    </div>
  );
}
