import type { ReactNode } from "react";

import { copy } from "../copy";
import { legalUpdated } from "./business";

/** A long document in one readable column: the title, the date it last changed, then its sections. */
export function LegalPage({
  title,
  intro,
  children,
}: {
  title: string;
  intro: ReactNode;
  children: ReactNode;
}) {
  return (
    <article className="mx-auto max-w-2xl px-4 pt-32 pb-24 sm:px-7 sm:pt-40">
      <h1 className="site-heading">{title}</h1>
      <p className="mt-4 text-body text-muted-foreground">
        {copy.legal.updated} {legalUpdated}
      </p>
      <div className="mt-10 grid gap-4 text-body font-normal text-foreground/85">{intro}</div>
      <div className="mt-14 grid gap-14">{children}</div>
    </article>
  );
}

export function Section({
  id,
  title,
  children,
}: {
  id: string;
  title: string;
  children: ReactNode;
}) {
  return (
    <section id={id} aria-labelledby={`${id}-title`} className="scroll-mt-24">
      <h2 id={`${id}-title`} className="text-section">
        {title}
      </h2>
      <div className="mt-4 grid gap-4 text-body font-normal text-foreground/85">{children}</div>
    </section>
  );
}

export function List({ children }: { children: ReactNode }) {
  return <ul className="grid list-disc gap-2 ps-5 marker:text-muted-foreground">{children}</ul>;
}

/** Label and value rows without rules between them; on a phone the label sits above its value. */
export function Facts({ rows }: { rows: { label: ReactNode; value: ReactNode }[] }) {
  return (
    <dl className="grid gap-x-8 gap-y-4 sm:grid-cols-[11rem_1fr]">
      {rows.map((row, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: the rows are static and never reorder.
        <div key={index} className="contents">
          <dt className="font-medium text-foreground">{row.label}</dt>
          <dd className="-mt-3 text-foreground/85 sm:mt-0">{row.value}</dd>
        </div>
      ))}
    </dl>
  );
}

export function TextLink({ href, children }: { href: string; children: ReactNode }) {
  return (
    <a
      href={href}
      className="text-foreground underline decoration-muted-foreground underline-offset-4 transition-colors hover:decoration-foreground"
    >
      {children}
    </a>
  );
}
