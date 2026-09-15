import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";

export function AuthShell({ children }: { children: ReactNode }) {
  const { t } = useTranslation();

  return (
    <main className="flex min-h-svh flex-col items-center justify-center px-6">
      <p className="mb-8 text-sm tracking-tight">{t("brand")}</p>
      <div className="w-full max-w-sm">{children}</div>
    </main>
  );
}
