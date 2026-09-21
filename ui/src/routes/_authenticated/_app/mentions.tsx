import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

export const Route = createFileRoute("/_authenticated/_app/mentions")({
  component: Mentions,
});

function Mentions() {
  const { t } = useTranslation("home");
  return <p className="text-muted-foreground">{t("pages.mentions")}</p>;
}
