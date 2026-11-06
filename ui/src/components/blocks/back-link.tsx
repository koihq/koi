import { ArrowLeft02Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Link } from "@tanstack/react-router";
import { Button } from "@/components/ui/button";

/** The way back to the visibility page. Its arrow lines up with the page edge. */
export function BackLink({ label, hash }: { label: string; hash?: string }) {
  return (
    <Button
      variant="ghost"
      size="circle-sm"
      nativeButton={false}
      role="link"
      aria-label={label}
      title={label}
      className="-ms-2 justify-self-start text-muted-foreground"
      render={<Link to="/" hash={hash} />}
    >
      <HugeiconsIcon icon={ArrowLeft02Icon} className="size-[18px]" aria-hidden="true" />
    </Button>
  );
}
