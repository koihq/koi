import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

export const Route = createFileRoute("/_authenticated/competitors")({
  component: Competitors,
});

function Competitors() {
  const { t } = useTranslation("home");
  return <p className="text-muted-foreground">{t("pages.competitors")}</p>;
}
