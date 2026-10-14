import { createFileRoute, Link } from "@tanstack/react-router";
import { useLayoutEffect, useState } from "react";
import { BrandSwitcher } from "./-components/brand-switcher";
import { Icon } from "./-components/common";
import { HomeSignal } from "./-components/home-signal";
import { ENGINES } from "./-lib/data";
import { HOME_BRANDS, HOME_LOGOS, type HomeBrand } from "./-lib/home-evidence";
import C from "./-locales/home-context.en.json";
import "./-styles.css";
import "./-home-context.css";

export const Route = createFileRoute("/dev/explorations/home-context")({ component: HomeContext });

const MAX_VISIBLE_BRANDS = 3;
const HOME_BRAND_OPTIONS = (Object.keys(HOME_BRANDS) as HomeBrand[]).map((id) => ({
  id,
  name: C[id].name,
  ...HOME_LOGOS[HOME_BRANDS[id].name],
}));

function HomeContext() {
  const [brand, setBrand] = useState<HomeBrand>("deja");
  const [periods, setPeriods] = useState<Partial<Record<HomeBrand, string>>>({});
  const [searchOpen, setSearchOpen] = useState(false);

  useLayoutEffect(() => {
    const root = document.documentElement;
    const previous = root.getAttribute("data-exploration-theme");
    const dark = root.classList.contains("dark");
    const scheme = root.style.colorScheme;
    const title = document.title;

    root.setAttribute("data-exploration-theme", "home-context");
    root.classList.add("dark");
    root.style.colorScheme = "dark";
    document.title = `${C.home} · Koi`;

    return () => {
      if (previous) root.setAttribute("data-exploration-theme", previous);
      else root.removeAttribute("data-exploration-theme");
      root.classList.toggle("dark", dark);
      root.style.colorScheme = scheme;
      document.title = title;
    };
  }, []);

  return (
    <div className="exploration home-context-study">
      <a href="#home-context-main" className="exp-skip">
        {C.skip}
      </a>
      <header className="hc-header">
        <BrandSwitcher
          brands={HOME_BRAND_OPTIONS}
          value={brand}
          onChange={setBrand}
          maxVisible={MAX_VISIBLE_BRANDS}
        />
        <span className="hc-demo-label">{C.demo}</span>
      </header>
      <main id="home-context-main" className="hc-main">
        <h1 className="sr-only">{`${C[brand].name} · ${C.home}`}</h1>
        <HomeSignal
          key={brand}
          brand={brand}
          topic={brand === "deja" ? "alternatives" : "discovery"}
          engines={ENGINES.map((engine) => engine.id)}
          days={periods[brand] ?? "30"}
          goal={C[brand].goal}
          onDaysChange={(days) => setPeriods((current) => ({ ...current, [brand]: days }))}
          searchOpen={searchOpen}
          onSearchOpenChange={setSearchOpen}
        />
      </main>
      <div className="hc-dock-group">
        <nav className="hc-dock" aria-label={C.navigation}>
          <a href="#home-context-main" aria-label={C.home} title={C.home}>
            <Icon name="overview" />
            <span className="hc-dock-label">{C.home}</span>
          </a>
          <a href="#hc-comparison-title" aria-label={C.comparison} title={C.comparison}>
            <Icon name="competitors" />
            <span className="hc-dock-label">{C.comparison}</span>
          </a>
          <a href="#hc-answers-title" aria-label={C.answers} title={C.answers}>
            <Icon name="quote" />
            <span className="hc-dock-label">{C.answers}</span>
          </a>
          <a href="#hc-agent-title" aria-label={C.brief} title={C.brief}>
            <Icon name="brief" />
            <span className="hc-dock-label">{C.brief}</span>
          </a>
          <Link to="/dev/explorations" aria-label={C.back} title={C.back}>
            <Icon name="menu" />
            <span className="hc-dock-label">{C.back}</span>
          </Link>
        </nav>
        <button
          type="button"
          className="hc-search-trigger"
          aria-label={C.search}
          onClick={() => setSearchOpen(true)}
        >
          <Icon name="search" size={21} />
        </button>
      </div>
    </div>
  );
}
