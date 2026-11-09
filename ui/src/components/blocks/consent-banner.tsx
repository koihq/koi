import { useState } from "react";

import type { ConsentChoice } from "@/analytics/analytics";
import { readConsent } from "@/analytics/consent";
import { Button } from "@/components/ui/button";

/**
 * Asks once whether analytics may use a cookie. The landing and the app share the answer, so a
 * visitor who chose on getkoi.co is not asked again after signing up.
 */
export function ConsentBanner({
  message,
  accept,
  reject,
  onChoose,
}: {
  message: string;
  accept: string;
  reject: string;
  onChoose: (choice: ConsentChoice) => void;
}) {
  const [open, setOpen] = useState(() => readConsent() === null);

  if (!open) return null;

  const choose = (choice: ConsentChoice) => {
    onChoose(choice);
    setOpen(false);
  };

  return (
    <section
      aria-label={message}
      className="fixed inset-x-4 bottom-4 z-40 mx-auto flex max-w-md flex-col gap-3 surface-float p-4 text-sm sm:flex-row sm:items-center sm:gap-4"
    >
      <p className="text-muted-foreground">{message}</p>
      <div className="flex shrink-0 gap-2">
        <Button variant="outline" size="lg" onClick={() => choose("denied")}>
          {reject}
        </Button>
        <Button size="lg" onClick={() => choose("granted")}>
          {accept}
        </Button>
      </div>
    </section>
  );
}
