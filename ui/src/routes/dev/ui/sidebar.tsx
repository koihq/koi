import {
  Analytics01Icon,
  File01Icon,
  QuoteIcon,
  UserMultipleIcon,
} from "@hugeicons/core-free-icons";
import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { KoiMark } from "@/components/blocks/koi-mark";
import { KoiSidebar, type KoiSidebarItem } from "@/components/blocks/koi-sidebar";
import { SidebarInset, SidebarProvider, SidebarTrigger, useSidebar } from "@/components/ui/sidebar";
import { TooltipProvider } from "@/components/ui/tooltip";
import { CatalogCode, CatalogPage, CatalogSection, validateUiSearch } from "./-catalog";
import "./-sidebar.css";

const pages = [
  { id: "mentions", icon: Analytics01Icon },
  { id: "competitors", icon: UserMultipleIcon },
  { id: "citations", icon: QuoteIcon },
  { id: "brief", icon: File01Icon },
] as const;

type Page = (typeof pages)[number]["id"];

export const Route = createFileRoute("/dev/ui/sidebar")({
  validateSearch: validateUiSearch,
  component: SidebarCatalog,
});

function SidebarCatalog() {
  const { t } = useTranslation("dev");

  return (
    <CatalogPage
      title="Koi sidebar"
      lead="The shipped navigation block inside the same provider and inset used by the app. On a narrow viewport, the toggle opens its drawer."
    >
      <CatalogSection
        title="Preview"
        description="Select a page, collapse the desktop rail, or open the mobile drawer. The page content reflects the selected item."
      >
        <TooltipProvider delay={150}>
          <SidebarProvider className="sidebar-catalog-stage">
            <SidebarCatalogPreview />
          </SidebarProvider>
        </TooltipProvider>
      </CatalogSection>
      <CatalogSection
        title="States"
        description="Expanded: 232px sidebar. Collapsed: 48px icon rail. Mobile: 288px drawer. The same KoiSidebar renders all three."
      >
        <p className="text-sm text-muted-foreground">{t("sidebarStates")}</p>
      </CatalogSection>
      <CatalogSection title="Use it">
        <div className="grid gap-4">
          <CatalogCode
            title="Import"
            code={
              'import { KoiSidebar } from "@/components/blocks/koi-sidebar";\nimport { SidebarInset, SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";'
            }
          />
          <CatalogCode
            title="Usage"
            code={
              '<SidebarProvider>\n  <KoiSidebar\n    header={<WorkspaceMenu name="Koi" />}\n    workspace={workspaceItems}\n    workspaceLabel="Workspace"\n    navigationLabel="Main navigation"\n    action={actionItems}\n    actionLabel="Act"\n  />\n  <SidebarInset>\n    <header><SidebarTrigger aria-label="Toggle sidebar" /></header>\n    <main>Page content</main>\n  </SidebarInset>\n</SidebarProvider>'
            }
          />
        </div>
      </CatalogSection>
    </CatalogPage>
  );
}

function SidebarCatalogPreview() {
  const { t } = useTranslation("home");
  const { t: dev } = useTranslation("dev");
  const { t: common } = useTranslation("common");
  const { setOpenMobile } = useSidebar();
  const [selected, setSelected] = useState<Page>("mentions");

  const items: KoiSidebarItem[] = pages.map((page) => ({
    id: page.id,
    label: t(`routes.${page.id}`),
    icon: page.icon,
    active: selected === page.id,
    render: (
      <button
        type="button"
        aria-current={selected === page.id ? "page" : undefined}
        onClick={() => {
          setSelected(page.id);
          setOpenMobile(false);
        }}
      />
    ),
  }));

  return (
    <>
      <KoiSidebar
        header={
          <div className="flex h-10 min-w-0 items-center gap-2 px-2 group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:px-0">
            <KoiMark className="size-5 shrink-0" />
            <span className="min-w-0 truncate text-[13px] font-medium group-data-[collapsible=icon]:hidden">
              {common("brand")}
            </span>
          </div>
        }
        workspace={items.slice(0, 3)}
        workspaceLabel={t("workspace")}
        navigationLabel={t("navigation")}
        action={items.slice(3)}
        actionLabel={t("action")}
      />
      <SidebarInset className="sidebar-catalog-inset min-w-0 md:border-[0.5px] md:border-border">
        <header className="flex h-14 shrink-0 items-center gap-3 px-4 md:px-6">
          <SidebarTrigger aria-label={t("toggleSidebar")} />
          <span className="text-sm font-medium">{t(`routes.${selected}`)}</span>
        </header>
        <div className="min-w-0 flex-1 px-6 py-8 md:px-10">
          <h2 className="max-w-lg text-2xl font-medium tracking-tight">{t(`pages.${selected}`)}</h2>
          <p className="mt-3 text-sm text-muted-foreground">{dev("previewContent")}</p>
        </div>
      </SidebarInset>
    </>
  );
}
