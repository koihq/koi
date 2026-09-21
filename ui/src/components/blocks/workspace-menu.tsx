import { ArrowDown01Icon, Logout01Icon, Settings01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Link, useRouter } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { useApi } from "@/api/apiProvider";
import { KoiMark } from "@/components/blocks/koi-mark";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useSidebar } from "@/components/ui/sidebar";

export function WorkspaceMenu({ name }: { name: string }) {
  const { t } = useTranslation("brand");
  const api = useApi();
  const router = useRouter();
  const { setOpenMobile } = useSidebar();

  async function signOut() {
    await api.auth.logout();
    await router.invalidate();
    await router.navigate({ to: "/sign-in" });
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <button
            type="button"
            aria-label={t("menu.open", { name })}
            className="flex h-10 w-full min-w-0 items-center gap-2 rounded-md px-2 text-left text-[13px] font-medium text-sidebar-foreground outline-none hover:bg-sidebar-accent focus-visible:ring-2 focus-visible:ring-sidebar-ring group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:px-0"
          />
        }
      >
        <KoiMark className="size-5 shrink-0" />
        <span className="min-w-0 flex-1 truncate group-data-[collapsible=icon]:hidden">{name}</span>
        <HugeiconsIcon
          icon={ArrowDown01Icon}
          strokeWidth={2}
          className="size-3.5 shrink-0 text-muted-foreground group-data-[collapsible=icon]:hidden"
        />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" side="bottom" sideOffset={4}>
        <DropdownMenuItem
          render={<Link to="/settings/brand" />}
          onClick={() => setOpenMobile(false)}
        >
          <HugeiconsIcon icon={Settings01Icon} strokeWidth={2} />
          {t("menu.settings")}
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => void signOut()}>
          <HugeiconsIcon icon={Logout01Icon} strokeWidth={2} />
          {t("menu.signOut")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
