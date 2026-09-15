import { createFileRoute, useRouter } from "@tanstack/react-router";

import { useApi } from "@/api/apiProvider";
import { AuthShell } from "@/routes/-auth-shell";
import { CredentialsForm } from "@/routes/-credentials-form";

export const Route = createFileRoute("/sign-up/")({
  component: SignUp,
});

function SignUp() {
  const api = useApi();
  const router = useRouter();

  return (
    <AuthShell>
      <CredentialsForm
        intent="sign-up"
        passwordComplete="new-password"
        onSubmit={async (body) => {
          await api.auth.signup(body);
          await router.invalidate();
          await router.navigate({ to: "/" });
        }}
      />
    </AuthShell>
  );
}
