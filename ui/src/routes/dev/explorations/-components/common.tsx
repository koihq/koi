import { Add01Icon, MoreHorizontalIcon, UserMultipleIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";

const ICONS = {
  add: Add01Icon,
  competitors: UserMultipleIcon,
  more: MoreHorizontalIcon,
};

export function Icon({ name, size = 18 }: { name: keyof typeof ICONS; size?: number }) {
  return <HugeiconsIcon icon={ICONS[name]} size={size} strokeWidth={1.6} aria-hidden={true} />;
}
