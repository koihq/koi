import { cn } from "cn";
import { useTranslation } from "react-i18next";

import mark from "@/assets/koi-mark.png";

export function KoiMark({ className }: { className?: string }) {
  const { t } = useTranslation();

  return (
    <img
      src={mark}
      alt={t("brand")}
      width={32}
      height={32}
      className={cn("block size-8", className)}
    />
  );
}
