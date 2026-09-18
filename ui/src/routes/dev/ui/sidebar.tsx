import {
  Analytics01Icon,
  File01Icon,
  Home01Icon,
  QuoteIcon,
  Settings01Icon,
  UserMultipleIcon,
} from "@hugeicons/core-free-icons";
import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { KoiSidebar, type KoiSidebarItem } from "@/components/blocks/koi-sidebar";
import { SidebarProvider, SidebarTrigger, useSidebar } from "@/components/ui/sidebar";
import { TooltipProvider } from "@/components/ui/tooltip";
import { CatalogPage, validateUiSearch } from "./-catalog";
import "./-sidebar.css";

const samplePages = [
  { id: "home", icon: Home01Icon },
  { id: "mentions", icon: Analytics01Icon },
  { id: "competitors", icon: UserMultipleIcon },
  { id: "citations", icon: QuoteIcon },
  { id: "brief", icon: File01Icon },
] as const;
type Page = (typeof samplePages)[number]["id"] | "settings";

export const Route = createFileRoute("/dev/ui/sidebar")({
  validateSearch: validateUiSearch,
  component: SidebarCatalog,
});

function SidebarCatalog() {
  const { t } = useTranslation("dev");
  return (
    <CatalogPage title={t("sidebar")} lead={t("sidebarStudy.lead")}>
      <TooltipProvider delay={150}>
        <SidebarProvider className="sidebar-catalog-stage">
          <SidebarCatalogPreview />
        </SidebarProvider>
      </TooltipProvider>
    </CatalogPage>
  );
}

function SidebarCatalogPreview() {
  const { t } = useTranslation("dev");
  const { open, setOpenMobile } = useSidebar();
  const [selected, setSelected] = useState<Page>("mentions");
  const [brand, setBrand] = useState("Koi");
  const items: KoiSidebarItem[] = samplePages.map((page) => ({
    id: page.id,
    label: t(`sidebarStudy.pages.${page.id}`),
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
        brand={
          <select
            aria-label={t("sidebarStudy.chooseBrand")}
            value={brand}
            onChange={(event) => setBrand(event.target.value)}
          >
            <option value="Koi">{t("sidebarStudy.koi")}</option>
            <option value="Acme">{t("sidebarStudy.acme")}</option>
          </select>
        }
        home={items[0]}
        workspace={items.slice(1, 4)}
        workspaceLabel={t("sidebarStudy.workspace")}
        navigationLabel={t("sidebarStudy.navigation")}
        action={items.slice(4)}
        actionLabel={t("sidebarStudy.action")}
        footer={{
          id: "settings",
          label: t("sidebarStudy.pages.settings"),
          icon: Settings01Icon,
          active: selected === "settings",
          render: (
            <button
              type="button"
              aria-current={selected === "settings" ? "page" : undefined}
              onClick={() => {
                setSelected("settings");
                setOpenMobile(false);
              }}
            />
          ),
        }}
      />
      <section className="sidebar-catalog-page" aria-label={t("sidebarStudy.preview")}>
        <header className="sidebar-catalog-header">
          <SidebarTrigger aria-label={t(open ? "sidebarStudy.collapse" : "sidebarStudy.expand")} />
          <span>{t(`sidebarStudy.pages.${selected}`)}</span>
          <span className="sidebar-catalog-demo">{t("sidebarStudy.sample")}</span>
        </header>
        <div className="sidebar-catalog-body">
          <p className="sidebar-catalog-eyebrow">{t("sidebarStudy.brandReport", { brand })}</p>
          <h2>{t(`sidebarStudy.content.${selected}.heading`, { brand })}</h2>
          <p className="sidebar-catalog-summary">
            {t(`sidebarStudy.content.${selected}.body`, { brand })}
          </p>
          <dl className="sidebar-catalog-facts">
            {(["first", "second", "third"] as const).map((fact) => (
              <div key={fact}>
                <dt>{t(`sidebarStudy.content.${selected}.${fact}Label`)}</dt>
                <dd>{t(`sidebarStudy.content.${selected}.${fact}Value`, { brand })}</dd>
              </div>
            ))}
          </dl>
          <div className="sidebar-catalog-note">
            <span>{t("sidebarStudy.latestRun")}</span>
            <p>{t("sidebarStudy.runNote")}</p>
          </div>
        </div>
      </section>
    </>
  );
}
