import { copy, links } from "../copy";
import { ExternalIcon, Faq } from "../layout";
import { TextLink } from "../legal/legal-page";
import type { OpenSourceContent, Project } from "./open-source";
import { Closing, formatDay, lastChecked, Name } from "./parts";
import { tools } from "./tools";

const { alternatives: shared } = copy;
const words = shared.openSource;

function Intro({ content }: { content: OpenSourceContent }) {
  const checked = lastChecked(content.projects.map((project) => tools[project.tool]));

  return (
    <header className="pt-32 pb-24 sm:pt-40">
      <h1 className="site-display max-w-240">{content.heading}</h1>
      <p className="site-lead mt-6 max-w-180">{content.answer}</p>
      <p className="mt-6 max-w-180 text-body text-pretty text-muted-foreground">
        {`${words.disclosure} ${words.checked} ${formatDay(checked)}, ${words.read} `}
        <TextLink href={links.correction}>{shared.correct}</TextLink>
      </p>
    </header>
  );
}

/** Four short columns, so it stays a table on a phone. */
function Glance({ projects }: { projects: Project[] }) {
  const { columns } = words;

  return (
    <section aria-labelledby="glance" className="pb-28">
      <h2 id="glance" className="site-heading">
        {words.glance}
      </h2>
      <table className="site-compare mt-10 max-w-220">
        <thead>
          <tr>
            <th scope="col">{columns.project}</th>
            <th scope="col">{columns.license}</th>
            <th scope="col">{columns.engines}</th>
            <th scope="col">{columns.hosted}</th>
          </tr>
        </thead>
        <tbody>
          {projects.map((project) => (
            <tr key={project.tool}>
              <th scope="row">
                <Name tool={tools[project.tool]} />
              </th>
              <td>{project.license}</td>
              <td>{project.engines}</td>
              <td>{project.hosted}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}

function Picks({ projects }: { projects: Project[] }) {
  return (
    <section aria-labelledby="pick" className="pb-28">
      <h2 id="pick" className="site-heading">
        {words.pick}
      </h2>
      <ul className="mt-10 grid max-w-220 gap-x-12 gap-y-6 sm:grid-cols-2">
        {projects.map((project) => (
          <li key={project.tool} className="text-body text-pretty text-muted-foreground">
            <strong className="font-medium text-foreground">
              {`${shared.choose} ${tools[project.tool].name}`}
            </strong>
            {` ${project.pick}`}
          </li>
        ))}
      </ul>
    </section>
  );
}

function Projects({ projects }: { projects: Project[] }) {
  return (
    <section aria-labelledby="projects" className="pb-28">
      <h2 id="projects" className="site-heading">
        {words.projects}
      </h2>
      <div className="mt-12 grid gap-16">
        {projects.map((project) => {
          const tool = tools[project.tool];
          const [repository] = tool.sources.filter((source) => source.url.includes("github.com"));

          return (
            <article
              key={project.tool}
              className="grid gap-x-10 gap-y-6 md:grid-cols-[minmax(0,4fr)_minmax(0,8fr)]"
            >
              <div>
                <h3 className="text-section font-medium">
                  <Name tool={tool} />
                </h3>
                {repository && (
                  <a
                    href={repository.url}
                    className="mt-3 inline-flex items-center gap-1 text-body text-muted-foreground transition-colors hover:text-foreground"
                  >
                    {words.source}
                    <ExternalIcon />
                  </a>
                )}
              </div>
              <div>
                <p className="max-w-180 text-body text-pretty">{project.summary}</p>
                <dl className="mt-6 grid gap-x-10 gap-y-5 sm:grid-cols-3">
                  {(["collects", "runs", "agents"] as const).map((fact) => (
                    <div key={fact}>
                      <dt className="text-body font-medium">{words.facts[fact]}</dt>
                      <dd className="mt-1.5 text-body text-pretty text-muted-foreground">
                        {project[fact]}
                      </dd>
                    </div>
                  ))}
                </dl>
              </div>
            </article>
          );
        })}
      </div>
    </section>
  );
}

export function OpenSourcePage({ content }: { content: OpenSourceContent }) {
  return (
    <>
      <article className="mx-auto max-w-340 px-4 sm:px-7">
        <Intro content={content} />
        <Glance projects={content.projects} />
        <Picks projects={content.projects} />
        <Projects projects={content.projects} />
      </article>
      <Faq items={content.questions} />
      <Closing />
    </>
  );
}
