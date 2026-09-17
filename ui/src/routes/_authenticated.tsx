import {
  Analytics01Icon,
  File01Icon,
  Home01Icon,
  Logout01Icon,
  QuoteIcon,
  UserMultipleIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import {
  createFileRoute,
  Link,
  Outlet,
  redirect,
  useLocation,
  useRouteContext,
  useRouter,
} from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { useApi } from "@/api/apiProvider";
import { KoiMark } from "@/components/blocks/koi-mark";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
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

const navigation = [
  { to: "/", key: "home", icon: Home01Icon },
  { to: "/mentions", key: "mentions", icon: Analytics01Icon },
  { to: "/competitors", key: "competitors", icon: UserMultipleIcon },
  { to: "/citations", key: "citations", icon: QuoteIcon },
  { to: "/brief", key: "brief", icon: File01Icon },
] as const;

export const Route = createFileRoute("/_authenticated")({
  beforeLoad: async ({ context }) => {
    try {
      const me = await context.api.auth.me();
      return { me };
    } catch {
      throw redirect({ to: "/sign-in" });
    }
  },
  component: AuthenticatedLayout,
});

function AuthenticatedLayout() {
  const { t } = useTranslation("home");
  const pathname = useLocation({ select: (location) => location.pathname });
  const currentPage = navigation.find((item) => item.to === pathname) ?? navigation[0];

  return (
    <TooltipProvider>
      <SidebarProvider>
        <a
          href="#main-content"
          className="sr-only focus:not-sr-only focus:absolute focus:start-4 focus:top-4 focus:z-50 focus:rounded-sm focus:bg-background focus:px-3 focus:py-2 focus:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground"
        >
          {t("skip")}
        </a>
        <AppSidebar pathname={pathname} />
        <SidebarInset className="min-w-0 md:border-[0.5px] md:border-border">
          <header className="flex h-14 shrink-0 items-center gap-3 px-4 md:px-6">
            <SidebarTrigger aria-label={t("toggleSidebar")} />
            <h1 className="text-sm font-medium">{t(`routes.${currentPage.key}`)}</h1>
          </header>
          <div id="main-content" tabIndex={-1} className="min-w-0 flex-1 px-6 py-8 md:px-10">
            <Outlet />
          </div>
        </SidebarInset>
      </SidebarProvider>
    </TooltipProvider>
  );
}

function AppSidebar({ pathname }: { pathname: string }) {
  const { me } = useRouteContext({ from: "/_authenticated" });
  const { t } = useTranslation("home");
  const { setOpenMobile } = useSidebar();
  const api = useApi();
  const router = useRouter();

  return (
    <Sidebar variant="inset" collapsible="icon">
      <SidebarHeader>
        <div className="flex h-9 items-center gap-2 px-2">
          <KoiMark className="size-5 shrink-0" />
          <span className="truncate text-sm font-medium group-data-[collapsible=icon]:hidden">
            {me.organization.name}
          </span>
          <SidebarTrigger className="ml-auto md:hidden" aria-label={t("toggleSidebar")} />
        </div>
      </SidebarHeader>
      <SidebarContent>
        <nav aria-label={t("navigation")}>
          <SidebarGroup>
            <SidebarMenu>
              {navigation.map((item) => {
                const label = t(`routes.${item.key}`);
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
                      tooltip={label}
                    >
                      <HugeiconsIcon icon={item.icon} strokeWidth={2} />
                      <span>{label}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                );
              })}
            </SidebarMenu>
          </SidebarGroup>
        </nav>
      </SidebarContent>
      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              tooltip={t("signOut")}
              onClick={async () => {
                await api.auth.logout();
                await router.invalidate();
                await router.navigate({ to: "/sign-in" });
              }}
            >
              <HugeiconsIcon icon={Logout01Icon} strokeWidth={2} />
              <span>{t("signOut")}</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  );
}
