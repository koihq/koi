import type { ReactNode } from "react";

import { KoiMark } from "@/components/blocks/koi-mark";

export function AuthShell({ children }: { children: ReactNode }) {
  return (
    <main className="flex min-h-svh flex-col items-center justify-center px-6">
      <KoiMark className="mb-8" />
      <div className="w-full max-w-sm">{children}</div>
    </main>
  );
}
