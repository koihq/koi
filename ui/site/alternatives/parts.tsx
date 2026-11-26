import { Fragment } from "react";

import { BrandIcon } from "../brand-icon";
import { copy, links } from "../copy";
import { Actions } from "../layout";
import { TextLink } from "../legal/legal-page";
import type { Tool } from "./tools";

const dayFormat = new Intl.DateTimeFormat("en-US", { dateStyle: "long", timeZone: "UTC" });

export function formatDay(day: string) {
  return dayFormat.format(new Date(`${day}T00:00:00Z`));
}

/** The newest date among the tools a page shows. */
export function lastChecked(shown: Tool[]) {
  return shown.map((tool) => tool.checked).sort()[shown.length - 1];
}

export function Name({ tool }: { tool: Tool }) {
  return (
    <span className="inline-flex items-center gap-2.5">
      <BrandIcon name={tool.name} domain={tool.domain} className="size-6" />
      {tool.name}
    </span>
  );
}

export function Closing() {
  const { alternatives: words } = copy;

  return (
    <div className="mx-auto max-w-340 px-4 sm:px-7">
      <section className="pb-28">
        <h2 className="site-display max-w-200">{words.closing}</h2>
        <p className="site-lead mt-4 max-w-160 text-muted-foreground">{words.closingBody}</p>
        <div className="mt-8">
          <Actions align="start" />
        </div>
      </section>
    </div>
  );
}

/** Where every fact comes from: the bias, the date, each page read, and how to correct us. */
export function Sources({ shown }: { shown: Tool[] }) {
  const { alternatives: words } = copy;
  const read = shown.flatMap((tool) => tool.sources.map((source) => ({ tool, source })));

  return (
    <p className="mt-6 max-w-180 text-body text-pretty text-muted-foreground">
      {`${words.disclosure} ${words.checked} ${formatDay(lastChecked(shown))}, ${words.sources} `}
      {read.map(({ tool, source }, index) => (
        <Fragment key={source.url}>
          <TextLink href={source.url}>{`${tool.name} ${source.label}`}</TextLink>
          {index < read.length - 1 ? ", " : ". "}
        </Fragment>
      ))}
      {`${words.note} `}
      <TextLink href={links.correction}>{words.correct}</TextLink>
    </p>
  );
}
