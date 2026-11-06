import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { ApiError, messageOr } from "@/api/errors";

/**
 * What the server said went wrong when it has something specific to say, else `fallback`.
 * A full prompt allowance also gets the way out. Render it inside an alert or field error.
 */
export function ApiErrorMessage({ error, fallback }: { error: unknown; fallback: string }) {
  const { t } = useTranslation("billing");
  const limitReached = error instanceof ApiError && error.code === "prompt_limit_reached";

  return (
    <>
      {messageOr(error, fallback)}
      {limitReached && (
        <>
          {" "}
          <Link to="/settings/billing" className="underline underline-offset-[3px]">
            {t("changeAllowance")}
          </Link>
        </>
      )}
    </>
  );
}
