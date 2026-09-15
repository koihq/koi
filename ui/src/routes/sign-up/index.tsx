import { createFileRoute, Link, useRouter } from "@tanstack/react-router";

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
        title="Create an account"
        submitLabel="Create account"
        passwordComplete="new-password"
        alt={
          <>
            Already have one?{" "}
            <Link to="/sign-in" className="text-foreground underline-offset-4 hover:underline">
              Sign in
            </Link>
          </>
        }
        onSubmit={async (body) => {
          await api.auth.signup(body);
          await router.invalidate();
          await router.navigate({ to: "/" });
        }}
      />
    </AuthShell>
  );
}
