import { UserIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useRouter } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useAnalytics } from "@/analytics/analyticsProvider";
import { useApi } from "@/api/apiProvider";
import type { Brand } from "@/api/brands.api";
import { BrandSwitcher } from "@/components/blocks/brand-switcher";
import { KoiMark } from "@/components/blocks/koi-mark";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useTheme } from "@/hooks/use-theme";
import { capabilitiesOptions, clearAccountCache } from "@/lib/queries";

/** `focused` keeps someone on their first check: no way off the page but theme and sign out. */
export function AppHeader({ brand, focused = false }: { brand: Brand; focused?: boolean }) {
  const api = useApi();
  const analytics = useAnalytics();
  const router = useRouter();
  const queryClient = useQueryClient();
  const { theme, changeTheme } = useTheme();
  const { t } = useTranslation("home");
  const [signOutFailed, setSignOutFailed] = useState(false);
  const capabilities = useQuery(capabilitiesOptions(api));

  async function signOut() {
    setSignOutFailed(false);

    try {
      await api.auth.logout();
      analytics.reset();
      await clearAccountCache(queryClient);
      await router.invalidate();
      await router.navigate({ to: "/sign-in" });
    } catch {
      setSignOutFailed(true);
    }
  }

  return (
    <header className="relative mx-auto flex min-h-8 w-full max-w-[1376px] items-center justify-between gap-4 px-4 pt-5 pb-6 sm:px-7 sm:pt-7 sm:pb-10">
      {focused ? <KoiMark /> : <BrandSwitcher key={brand.id} brand={brand} />}

      <div className="flex shrink-0 items-center gap-3">
        <DropdownMenu>
          <DropdownMenuTrigger
            aria-label={t("header.account")}
            render={
              <Button variant="icon-outline" size="circle-sm" className="text-muted-foreground" />
            }
          >
            <HugeiconsIcon icon={UserIcon} className="size-[18px]" aria-hidden="true" />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" sideOffset={8}>
            {!focused && (
              <DropdownMenuItem render={<Link to="/settings/brand" />}>
                {t("header.brands")}
              </DropdownMenuItem>
            )}
            {!focused && (
              <DropdownMenuItem render={<Link to="/settings/connections" />}>
                {t("header.connections")}
              </DropdownMenuItem>
            )}
            {!focused && capabilities.data?.cloudBilling && (
              <DropdownMenuItem render={<Link to="/settings/billing" />}>
                {t("billing")}
              </DropdownMenuItem>
            )}
            {analytics.enabled && (
              <DropdownMenuItem onClick={analytics.openSupport}>
                {t("header.support")}
              </DropdownMenuItem>
            )}
            <DropdownMenuItem onClick={() => changeTheme(theme === "dark" ? "light" : "dark")}>
              {t(theme === "dark" ? "header.lightMode" : "header.darkMode")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => void signOut()}>{t("signOut")}</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        {signOutFailed && (
          <p
            role="alert"
            className="absolute end-4 top-full z-10 surface-float px-3 py-2 text-xs text-destructive sm:end-7"
          >
            {t("header.signOutError")}
          </p>
        )}
      </div>
    </header>
  );
}
