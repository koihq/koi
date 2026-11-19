import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute, useRouter } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { useApi } from "@/api/apiProvider";
import { authProvidersOptions, clearAccountCache } from "@/lib/queries";
import { AuthShell } from "@/routes/-auth-shell";
import { CredentialsForm } from "@/routes/-credentials-form";
import {
  type GoogleErrorCode,
  googleErrorMessage,
  parseGoogleError,
} from "@/routes/-google-sign-in";

export const Route = createFileRoute("/sign-in/")({
  validateSearch: (search: Record<string, unknown>): { error?: GoogleErrorCode } => ({
    error: parseGoogleError(search.error),
  }),
  // Knowing about Google before the first paint keeps the form from moving when the button appears.
  loader: ({ context }) =>
    context.queryClient.ensureQueryData(authProvidersOptions(context.api)).catch(() => undefined),
  component: SignIn,
});

function SignIn() {
  const api = useApi();
  const router = useRouter();
  const queryClient = useQueryClient();
  const { t } = useTranslation("auth");
  const { error } = Route.useSearch();

  return (
    <AuthShell>
      <CredentialsForm
        intent="sign-in"
        passwordComplete="current-password"
        notice={error && googleErrorMessage(t, error)}
        onSubmit={async (body) => {
          await api.auth.login(body);
          await clearAccountCache(queryClient);
          await router.invalidate();
          await router.navigate({ to: "/" });
        }}
      />
    </AuthShell>
  );
}
