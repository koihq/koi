import type { ReactNode } from "react";
import type { Layout } from "../kit/looks";
import { Window } from "../kit/window";

export type Outcome = "cited" | "mentioned" | "miss" | "pending";

/** Filled is cited, tinted is mentioned, outlined is not in the answer. Same language as the app. */
export function OutcomeDot({ outcome, size = 22 }: { outcome: Outcome; size?: number }) {
  return <span className="kd-dot" data-outcome={outcome} style={{ width: size, height: size }} />;
}

/** A desktop window on wide stages. On a phone the app is responsive, so its surface stands alone. */
export function Frame({
  layout,
  title,
  url,
  className,
  children,
}: {
  layout: Layout;
  title?: string;
  url?: string;
  className: string;
  children: ReactNode;
}) {
  if (layout === "compact") {
    return <div className={className}>{children}</div>;
  }

  return (
    <Window
      title={title}
      url={url}
      urlShape={url ? "pill" : undefined}
      lights="grey"
      className={className}
    >
      {children}
    </Window>
  );
}
