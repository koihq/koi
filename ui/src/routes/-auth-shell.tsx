import type { ReactNode } from "react";

export function AuthShell({ children }: { children: ReactNode }) {
  return (
    <main className="flex min-h-svh flex-col items-center justify-center px-6">
      <p className="mb-8 text-sm tracking-tight">Koi</p>
      <div className="w-full max-w-sm">{children}</div>
    </main>
  );
}
