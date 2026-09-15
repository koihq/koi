import { createFileRoute, Link, useRouter } from "@tanstack/react-router";

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
        title="Sign in"
        submitLabel="Sign in"
        passwordComplete="current-password"
        alt={
          <>
            Need an account?{" "}
            <Link to="/sign-up" className="text-foreground underline-offset-4 hover:underline">
              Create one
            </Link>
          </>
        }
        onSubmit={async (body) => {
          await api.auth.login(body);
          await router.invalidate();
          await router.navigate({ to: "/" });
        }}
      />
    </AuthShell>
  );
}
