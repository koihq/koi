import { ArrowRight01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";

import { BrandIcon } from "../brand-icon";
import { copy } from "../copy";
import { alternatives, openSourcePage } from "./list";
import { Closing } from "./parts";
import { tools } from "./tools";

const words = copy.alternatives.index;

const entries = [
  ...alternatives.map((page) => ({ ...page, logo: tools[page.rival] })),
  { ...openSourcePage, logo: tools.koi },
];

export function ComparisonsPage() {
  return (
    <>
      <article className="mx-auto max-w-340 px-4 sm:px-7">
        <header className="pt-32 pb-20 sm:pt-40">
          <h1 className="site-display max-w-240">{words.heading}</h1>
          <p className="site-lead mt-6 max-w-180 text-muted-foreground">{words.lead}</p>
        </header>
        <ul className="max-w-220 border-t border-border pb-28">
          {entries.map((entry) => (
            <li key={entry.path} className="border-b border-border">
              <a href={entry.path} className="group flex items-start gap-4 rounded-control py-6">
                <BrandIcon name={entry.logo.name} domain={entry.logo.domain} className="size-8" />
                <span className="min-w-0 flex-1">
                  <span className="block text-section font-medium">{entry.name}</span>
                  <span className="mt-1.5 block text-body text-pretty text-muted-foreground">
                    {entry.summary}
                  </span>
                </span>
                <HugeiconsIcon
                  icon={ArrowRight01Icon}
                  aria-hidden="true"
                  className="mt-1 size-4 shrink-0 text-muted-foreground transition-transform group-hover:translate-x-0.5 motion-reduce:transition-none"
                />
              </a>
            </li>
          ))}
        </ul>
      </article>
      <Closing />
    </>
  );
}
