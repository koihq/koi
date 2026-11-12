import type { ReactNode } from "react";

import { App, SiteLayout } from "./app";
import { Privacy } from "./legal/privacy";
import { Terms } from "./legal/terms";

export type Page = {
  /** The path the page is served at, and the folder its prerendered HTML is written to. */
  path: string;
  title: string;
  description: string;
  render: () => ReactNode;
};

/** The home page keeps the title and description written in index.html. */
export const pages: Page[] = [
  { path: "/", title: "", description: "", render: () => <App /> },
  {
    path: "/privacy",
    title: "Privacy Policy · Koi",
    description:
      "What personal data Koi Cloud collects, why, who receives it, how long it is kept and the rights you have over it.",
    render: () => (
      <SiteLayout>
        <Privacy />
      </SiteLayout>
    ),
  },
  {
    path: "/terms",
    title: "Terms of Service · Koi",
    description:
      "The terms for using Koi Cloud: business use, plans and trial, cancellation and refunds, acceptable use and liability.",
    render: () => (
      <SiteLayout>
        <Terms />
      </SiteLayout>
    ),
  },
];

/** `/privacy/` and `/privacy` are the same page; anything unknown falls back to the home page. */
export function pageFor(pathname: string) {
  const path = pathname.replace(/\/+$/, "") || "/";

  return pages.find((page) => page.path === path) ?? pages[0];
}
