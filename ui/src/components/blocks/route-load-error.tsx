import type { ErrorComponentProps } from "@tanstack/react-router";
import { useRouter } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";

export function RouteLoadError({ reset }: ErrorComponentProps) {
  const { t } = useTranslation("brand");
  const router = useRouter();

  return (
    <main className="flex min-h-svh items-center justify-center px-6">
      <div className="max-w-sm text-center">
        <h1 className="text-2xl font-medium tracking-tight">{t("loadError.title")}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{t("loadError.description")}</p>
        <Button
          className="mt-4"
          onClick={() => {
            reset();
            void router.invalidate();
          }}
        >
          {t("retry")}
        </Button>
      </div>
    </main>
  );
}
