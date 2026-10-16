import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "@/api/apiProvider";
import { Button } from "@/components/ui/button";
import { runKey } from "@/lib/query-keys";

export function HistoricalPrompts({ brandID }: { brandID: string }) {
  const { t, i18n } = useTranslation("home");
  const api = useApi();
  const [page, setPage] = useState(1);
  const history = useQuery({
    queryKey: [...runKey(brandID), "past-prompts", page],
    queryFn: ({ signal }) => api.runs.historicalPrompts(brandID, page, 20, { signal }),
  });

  if (history.isPending) return null;

  return (
    <section
      aria-labelledby="past-prompts-heading"
      className="mt-10 border-t border-border/70 pt-8"
    >
      <h3 id="past-prompts-heading" className="text-base font-medium">
        {t("history.title")}
      </h3>
      {history.isError ? (
        <p role="alert" className="mt-4 text-sm text-destructive">
          {t("history.loadError")}{" "}
          <Button variant="link" onClick={() => void history.refetch()}>
            {t("dashboard.retry")}
          </Button>
        </p>
      ) : history.data.items.length === 0 ? (
        <p className="mt-3 text-sm text-muted-foreground">{t("history.empty")}</p>
      ) : (
        <>
          <ul className="mt-4 divide-y divide-border/50">
            {history.data.items.map((prompt) => (
              <li
                key={prompt.promptId}
                className="flex flex-wrap items-center justify-between gap-3 py-3"
              >
                <Link
                  to="/prompts/$promptId"
                  params={{ promptId: prompt.promptId }}
                  className="max-w-2xl text-sm underline-offset-4 hover:underline"
                >
                  {prompt.promptText}
                </Link>
                <time
                  className="text-xs text-muted-foreground"
                  dateTime={prompt.lastSeenAt.toISOString()}
                >
                  {prompt.lastSeenAt.toLocaleDateString(i18n.language)}
                </time>
              </li>
            ))}
          </ul>
          <div className="mt-4 flex items-center justify-between text-xs text-muted-foreground">
            <span>
              {t("dashboard.page", {
                current: history.data.currentPage,
                total: history.data.totalPages,
              })}
            </span>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={!history.data.hasPrev}
                onClick={() => setPage(page - 1)}
              >
                {t("dashboard.previous")}
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={!history.data.hasNext}
                onClick={() => setPage(page + 1)}
              >
                {t("dashboard.next")}
              </Button>
            </div>
          </div>
        </>
      )}
    </section>
  );
}
