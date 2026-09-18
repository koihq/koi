import { createFileRoute, Link, Outlet, redirect, useLocation } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { KoiMark } from "@/components/blocks/koi-mark";
import { Separator } from "@/components/ui/separator";

import { catalogTo, useCatalogTheme, VARIANTS, type Variant, validateUiSearch } from "./-catalog";

const NAV = [
  { to: "/dev/ui", key: "title" },
  { to: "/dev/ui/mark", key: "mark" },
  { to: "/dev/ui/button", key: "button" },
  { to: "/dev/ui/field", key: "field" },
  { to: "/dev/ui/separator", key: "separator" },
  { to: "/dev/ui/sidebar", key: "sidebar" },
] as const;

export const Route = createFileRoute("/dev/ui")({
  validateSearch: validateUiSearch,
  beforeLoad: () => {
    if (!import.meta.env.DEV) {
      throw redirect({ to: "/" });
    }
  },
  component: DevUiLayout,
});

function DevUiLayout() {
  const { t } = useTranslation("dev");
  const { variant } = Route.useSearch();

  useCatalogTheme(variant);

  return (
    <div className="flex min-h-svh min-w-0 flex-col min-[800px]:flex-row">
      <a
        href="#dev-ui-main"
        className="sr-only focus:not-sr-only focus:absolute focus:start-6 focus:top-4 focus:z-10 focus:bg-background focus:px-2 focus:py-1 focus:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground"
      >
        {t("skip")}
      </a>
      <nav
        className="flex shrink-0 flex-col border-b border-border px-5 pt-5 pb-4 min-[800px]:sticky min-[800px]:top-0 min-[800px]:h-svh min-[800px]:w-56 min-[800px]:border-b-0 min-[800px]:px-6 min-[800px]:py-8"
        aria-label={t("catalog")}
      >
        <div className="flex items-center gap-2 text-sm font-medium">
          <KoiMark className="size-6" />
          <span>{t("koiUi")}</span>
        </div>
        <p className="mt-8 hidden text-xs/relaxed font-medium text-muted-foreground min-[800px]:block">
          {t("catalog")}
        </p>
        <div className="mt-4 flex gap-x-5 overflow-x-auto min-[800px]:mt-2 min-[800px]:flex-col min-[800px]:gap-x-0">
          {NAV.map((item) => (
            <Link
              key={item.to}
              to={item.to}
              search={{ variant }}
              activeOptions={{ exact: true, includeSearch: false }}
              activeProps={{ "aria-current": "page" }}
              className="block shrink-0 rounded-sm py-2 text-sm text-muted-foreground data-[status=active]:font-medium data-[status=active]:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground"
            >
              {t(item.key)}
            </Link>
          ))}
        </div>
        <nav className="mt-3 flex gap-4 min-[800px]:mt-auto" aria-label={t("variants")}>
          {VARIANTS.map((item) => (
            <ThemeLink key={item} item={item} variant={variant} />
          ))}
        </nav>
      </nav>
      <Separator orientation="vertical" className="hidden min-[800px]:block" />
      <main
        id="dev-ui-main"
        className="min-w-0 flex-1 scroll-mt-10 px-5 pt-8 pb-16 min-[800px]:px-10 min-[800px]:py-10"
      >
        <div className="mx-auto w-full max-w-4xl">
          <Outlet />
        </div>
      </main>
    </div>
  );
}

function ThemeLink({ item, variant }: { item: Variant; variant: Variant }) {
  const { t } = useTranslation("dev");
  const pathname = useLocation({ select: (location) => location.pathname });

  return (
    <Link
      to={catalogTo(pathname)}
      search={(previous) => ({ ...previous, variant: item })}
      replace
      aria-current={variant === item ? "true" : undefined}
      className="block rounded-sm py-1.5 text-xs/relaxed text-muted-foreground underline-offset-4 aria-[current=true]:font-medium aria-[current=true]:text-foreground aria-[current=true]:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground"
    >
      {t(item)}
    </Link>
  );
}
