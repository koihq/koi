import { UserIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Link } from "@tanstack/react-router";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { HomeBrand } from "../-lib/home-evidence";
import C from "../-locales/mvp.en.json";

export function AccountMenu({ brand, onLogout }: { brand: HomeBrand; onLogout: () => void }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="hc-account-trigger" aria-label={C.account}>
        <span className="hc-account-pill">
          <HugeiconsIcon icon={UserIcon} size={16} strokeWidth={1.6} aria-hidden="true" />
        </span>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" sideOffset={8} className="hc-brand-menu">
        <DropdownMenuItem
          render={
            <Link
              from="/dev/explorations/home-context"
              to="/dev/explorations/home-context"
              search={(previous) => ({ ...previous, view: "settings", brand })}
            />
          }
        >
          {C.settings}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={onLogout}>{C.logout}</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
