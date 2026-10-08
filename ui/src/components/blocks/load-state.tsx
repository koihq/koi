import { useTranslation } from "react-i18next";

export function LoadState({ failed }: { failed: boolean }) {
  const { t } = useTranslation("brand");

  return (
    <p role={failed ? "alert" : undefined} className="text-sm text-muted-foreground">
      {failed ? t("lists.loadError") : t("lists.loading")}
    </p>
  );
}
