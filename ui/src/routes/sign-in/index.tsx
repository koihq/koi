import { createFileRoute, useRouter } from "@tanstack/react-router";

import { useApi } from "@/api/apiProvider";
import { AuthShell } from "@/routes/-auth-shell";
import { CredentialsForm } from "@/routes/-credentials-form";

export const Route = createFileRoute("/sign-in/")({
  component: SignIn,
});

function SignIn() {
  const api = useApi();
  const router = useRouter();

  return (
    <AuthShell>
      <CredentialsForm
        intent="sign-in"
        passwordComplete="current-password"
        onSubmit={async (body) => {
          await api.auth.login(body);
          await router.invalidate();
          await router.navigate({ to: "/" });
        }}
      />
    </AuthShell>
  );
}
