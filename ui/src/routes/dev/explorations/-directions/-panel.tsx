import { EngineList } from "../-components/common";
import { AnswerInspector, ManagementScreen, MentionsScreen } from "../-components/screens";
import { VisibilityChart } from "./-visibility";
import type { Panel } from "./-workspace";

export function TradeRepublicPanel({
  panel,
  selectedPrompt,
  onSelectPrompt,
}: {
  panel: Panel;
  selectedPrompt: string | null;
  onSelectPrompt: (id: string | null) => void;
}) {
  if (panel === "inspector") return <AnswerInspector promptId={selectedPrompt} />;
  if (panel === "trend")
    return <VisibilityChart selectedPrompt={selectedPrompt} onSelectPrompt={onSelectPrompt} />;
  if (panel === "engines") return <EngineList logos />;
  if (panel === "mentions") return <MentionsScreen onSelectPrompt={onSelectPrompt} />;
  return (
    <ManagementScreen
      kind={panel}
      workspace
      selectedPrompt={selectedPrompt}
      onSelectPrompt={onSelectPrompt}
    />
  );
}
