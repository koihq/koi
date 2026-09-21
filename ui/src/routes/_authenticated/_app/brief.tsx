import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

export const Route = createFileRoute("/_authenticated/_app/brief")({
  component: Brief,
});

function Brief() {
  const { t } = useTranslation("home");
  return <p className="text-muted-foreground">{t("pages.brief")}</p>;
}
