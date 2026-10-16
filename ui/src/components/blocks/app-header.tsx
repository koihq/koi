import { UserIcon, UserMultipleIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Link, useRouter } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";

import { useApi } from "@/api/apiProvider";
import type { Brand } from "@/api/brands.api";
import { BrandIcon } from "@/components/blocks/brand-icon";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useTheme } from "@/hooks/use-theme";

export function AppHeader({ brand }: { brand: Brand }) {
  const api = useApi();
  const router = useRouter();
  const { theme, changeTheme } = useTheme();
  const { t } = useTranslation("home");
  const [signOutFailed, setSignOutFailed] = useState(false);

  async function signOut() {
    setSignOutFailed(false);

    try {
      await api.auth.logout();
      await router.invalidate();
      await router.navigate({ to: "/sign-in" });
    } catch {
      setSignOutFailed(true);
    }
  }

  return (
    <header className="relative mx-auto flex min-h-8 w-full max-w-[1376px] items-center justify-between gap-4 px-4 pt-5 pb-6 sm:px-7 sm:pt-7 sm:pb-10">
      <Link
        to="/"
        aria-label={brand.name}
        title={brand.name}
        className="inline-grid size-8 place-items-center rounded-full outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        <BrandIcon name={brand.name} domain={brand.domain} className="size-6" />
      </Link>

      <div className="flex shrink-0 items-center gap-3">
        <Link
          to="/competitors"
          aria-label="Competitors"
          title="Competitors"
          className="grid size-8 place-items-center rounded-full text-muted-foreground outline-none hover:bg-muted hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
        >
          <HugeiconsIcon icon={UserMultipleIcon} size={18} aria-hidden="true" />
        </Link>

        <DropdownMenu>
          <DropdownMenuTrigger
            aria-label="Account"
            className="grid size-8 place-items-center rounded-full border border-border text-muted-foreground outline-none hover:bg-muted hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
          >
            <HugeiconsIcon icon={UserIcon} size={18} aria-hidden="true" />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" sideOffset={8}>
            <DropdownMenuItem render={<Link to="/settings/brand" />}>
              {t("header.settings")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => changeTheme(theme === "dark" ? "light" : "dark")}>
              {t(theme === "dark" ? "header.lightMode" : "header.darkMode")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => void signOut()}>{t("signOut")}</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        {signOutFailed && (
          <p
            role="alert"
            className="absolute end-4 top-full z-10 rounded-lg bg-popover px-3 py-2 text-xs text-destructive shadow-md sm:end-7"
          >
            {t("header.signOutError")}
          </p>
        )}
      </div>
    </header>
  );
}
