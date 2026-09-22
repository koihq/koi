import {
  Analytics01Icon,
  File01Icon,
  QuoteIcon,
  UserMultipleIcon,
} from "@hugeicons/core-free-icons";
import {
  createFileRoute,
  Link,
  Outlet,
  redirect,
  useLocation,
  useRouteContext,
} from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { KoiSidebar, type KoiSidebarItem } from "@/components/blocks/koi-sidebar";
import { WorkspaceMenu } from "@/components/blocks/workspace-menu";
import { SidebarInset, SidebarProvider, SidebarTrigger, useSidebar } from "@/components/ui/sidebar";
import { TooltipProvider } from "@/components/ui/tooltip";

const navigation = [
  { to: "/mentions", key: "mentions", icon: Analytics01Icon },
  { to: "/competitors", key: "competitors", icon: UserMultipleIcon },
  { to: "/citations", key: "citations", icon: QuoteIcon },
  { to: "/brief", key: "brief", icon: File01Icon },
] as const;

export const Route = createFileRoute("/_authenticated/_app")({
  beforeLoad: ({ context }) => {
    if (!context.brand) {
      throw redirect({ to: "/setup" });
    }
    return { brand: context.brand };
  },
  component: AppLayout,
});

function AppLayout() {
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
          <main id="main-content" tabIndex={-1} className="min-w-0 flex-1 px-6 py-8 md:px-10">
            <Outlet />
          </main>
        </SidebarInset>
      </SidebarProvider>
    </TooltipProvider>
  );
}

function AppSidebar({ pathname }: { pathname: string }) {
  const { me } = useRouteContext({ from: "/_authenticated" });
  const { t } = useTranslation("home");
  const { setOpenMobile } = useSidebar();

  const items: KoiSidebarItem[] = navigation.map((item) => {
    const active = pathname === item.to;
    return {
      id: item.key,
      label: t(`routes.${item.key}`),
      icon: item.icon,
      active,
      render: (
        <Link
          to={item.to}
          aria-current={active ? "page" : undefined}
          onClick={() => setOpenMobile(false)}
        />
      ),
    };
  });
  return (
    <KoiSidebar
      header={<WorkspaceMenu name={me.organization.name} />}
      workspace={items.slice(0, 3)}
      workspaceLabel={t("workspace")}
      navigationLabel={t("navigation")}
      action={items.slice(3)}
      actionLabel={t("action")}
    />
  );
}
