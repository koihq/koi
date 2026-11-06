import { ArrowLeft01Icon, ArrowRight01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";

/** "1–20 of 32" and two arrows. Hidden when the list is empty. */
export function PageNav({
  page,
  pageSize,
  total,
  hasPrev,
  hasNext,
  onPage,
}: {
  page: number;
  pageSize: number;
  total: number;
  hasPrev: boolean;
  hasNext: boolean;
  onPage: (page: number) => void;
}) {
  const { t } = useTranslation("home");

  if (total === 0) return null;

  return (
    <nav
      aria-label={t("dashboard.pagination")}
      className="-me-2 mt-6 flex items-center justify-end gap-3 text-xs text-muted-foreground"
    >
      <span className="tabular-nums">
        {t("dashboard.pageRange", {
          start: (page - 1) * pageSize + 1,
          end: Math.min(total, page * pageSize),
          total,
        })}
      </span>
      <div className="flex gap-1">
        <Button
          variant="ghost"
          size="circle-sm"
          aria-label={t("dashboard.previous")}
          disabled={!hasPrev}
          onClick={() => onPage(page - 1)}
        >
          <HugeiconsIcon icon={ArrowLeft01Icon} aria-hidden="true" />
        </Button>
        <Button
          variant="ghost"
          size="circle-sm"
          aria-label={t("dashboard.next")}
          disabled={!hasNext}
          onClick={() => onPage(page + 1)}
        >
          <HugeiconsIcon icon={ArrowRight01Icon} aria-hidden="true" />
        </Button>
      </div>
    </nav>
  );
}
