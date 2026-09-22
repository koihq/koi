import {
  ArrowLeft02Icon,
  Building02Icon,
  MessageQuestionIcon,
  UserMultipleIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { createFileRoute, Link, Outlet, redirect, useLocation } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { RouteLoadError } from "@/components/blocks/route-load-error";
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarTrigger,
  useSidebar,
} from "@/components/ui/sidebar";
import { TooltipProvider } from "@/components/ui/tooltip";

export const Route = createFileRoute("/_authenticated/_settings")({
  beforeLoad: ({ context }) => {
    if (!context.brand) {
      throw redirect({ to: "/setup" });
    }
    return { brand: context.brand };
  },
  errorComponent: RouteLoadError,
  component: SettingsLayout,
});

function SettingsLayout() {
  const { t } = useTranslation("brand");

  return (
    <TooltipProvider>
      <SidebarProvider>
        <a
          href="#settings-content"
          className="sr-only focus:not-sr-only focus:absolute focus:start-4 focus:top-4 focus:z-50 focus:rounded-sm focus:bg-background focus:px-3 focus:py-2 focus:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground"
        >
          {t("skip", { ns: "home" })}
        </a>
        <SettingsSidebar />
        <SidebarInset className="min-w-0 md:border-[0.5px] md:border-border">
          <header className="flex h-14 shrink-0 items-center gap-3 px-4 md:px-6">
            <SidebarTrigger aria-label={t("toggleSidebar", { ns: "home" })} />
            <h1 className="text-sm font-medium">{t("settings.section")}</h1>
          </header>
          <main id="settings-content" tabIndex={-1} className="min-w-0 flex-1 px-6 py-8 md:px-10">
            <Outlet />
          </main>
        </SidebarInset>
      </SidebarProvider>
    </TooltipProvider>
  );
}

function SettingsSidebar() {
  const { t } = useTranslation("brand");
  const { setOpenMobile } = useSidebar();
  const pathname = useLocation({ select: (location) => location.pathname });
  const items = [
    { to: "/settings/brand", label: t("settings.brand"), icon: Building02Icon },
    { to: "/settings/prompts", label: t("settings.prompts"), icon: MessageQuestionIcon },
    {
      to: "/settings/competitors",
      label: t("settings.competitors"),
      icon: UserMultipleIcon,
    },
  ] as const;

  return (
    <Sidebar variant="inset" collapsible="icon">
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton render={<Link to="/mentions" />} tooltip={t("settings.back")}>
              <HugeiconsIcon icon={ArrowLeft02Icon} strokeWidth={2} />
              <span>{t("settings.back")}</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <nav aria-label={t("settings.navigation")}>
          <SidebarGroup>
            <SidebarGroupLabel>{t("settings.section")}</SidebarGroupLabel>
            <SidebarMenu>
              {items.map((item) => {
                const active = pathname === item.to;
                return (
                  <SidebarMenuItem key={item.to}>
                    <SidebarMenuButton
                      render={
                        <Link
                          to={item.to}
                          aria-current={active ? "page" : undefined}
                          onClick={() => setOpenMobile(false)}
                        />
                      }
                      isActive={active}
                      tooltip={item.label}
                    >
                      <HugeiconsIcon icon={item.icon} strokeWidth={2} />
                      <span>{item.label}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                );
              })}
            </SidebarMenu>
          </SidebarGroup>
        </nav>
      </SidebarContent>
    </Sidebar>
  );
}
