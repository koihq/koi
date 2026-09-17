import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

export const Route = createFileRoute("/_authenticated/")({
  component: Home,
});

function Home() {
  const { t } = useTranslation("home");
  return <p className="text-muted-foreground">{t("pages.home")}</p>;
}
