import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";

export function ListPagination({
  currentPage,
  totalPages,
  onPageChange,
}: {
  currentPage: number;
  totalPages: number;
  onPageChange: (page: number) => void;
}) {
  const { t } = useTranslation("brand");
  if (totalPages <= 1) return null;

  return (
    <nav
      aria-label={t("lists.pagination.label")}
      className="mt-6 flex items-center justify-between gap-4"
    >
      <Button
        type="button"
        size="sm"
        variant="outline"
        disabled={currentPage <= 1}
        onClick={() => onPageChange(currentPage - 1)}
      >
        {t("lists.pagination.previous")}
      </Button>
      <p className="text-xs text-muted-foreground tabular-nums">
        {t("lists.pagination.page", { current: currentPage, total: totalPages })}
      </p>
      <Button
        type="button"
        size="sm"
        variant="outline"
        disabled={currentPage >= totalPages}
        onClick={() => onPageChange(currentPage + 1)}
      >
        {t("lists.pagination.next")}
      </Button>
    </nav>
  );
}
