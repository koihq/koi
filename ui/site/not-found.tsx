import { copy } from "./copy";

export function NotFound() {
  return (
    <section className="mx-auto max-w-340 px-4 pt-32 pb-48 sm:px-7 sm:pt-40">
      <h1 className="site-display max-w-240">{copy.notFound.title}</h1>
      <p className="site-lead mt-6 max-w-160 text-muted-foreground">
        {`${copy.notFound.body} `}
        <a
          href="/"
          className="text-foreground underline decoration-muted-foreground underline-offset-4 transition-colors hover:decoration-foreground"
        >
          {copy.notFound.home}
        </a>
      </p>
    </section>
  );
}
