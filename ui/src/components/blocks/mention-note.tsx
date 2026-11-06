import { useTranslation } from "react-i18next";

import type { VisibilityResult } from "@/api/generated";

/** Explains an answer that names the brand but doesn't count as a mention. */
export function MentionNote({
  verdict,
  brand,
}: {
  verdict: VisibilityResult["mentionVerdict"] | undefined;
  brand: string;
}) {
  const { t } = useTranslation("visibility");

  if (!verdict || verdict === "mention") return null;

  return <p className="text-sm text-muted-foreground">{t(`mentionNotes.${verdict}`, { brand })}</p>;
}
