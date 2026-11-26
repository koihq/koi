import { Cancel01Icon, MinusSignIcon, Tick02Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";

import { copy } from "../copy";
import { Actions, Faq } from "../layout";
import { TextLink } from "../legal/legal-page";
import { type Cell, type Comparison, koiTopics, topics } from "./comparison";
import { comparisonsPage, openSourcePage } from "./list";
import { Closing, Name, Sources } from "./parts";
import { type RivalId, type Tool, tools } from "./tools";

const { alternatives: words } = copy;

/** The marks carry no colour: a yes is as bright as the words, a no and an unknown recede. */
const marks = {
  yes: { icon: Tick02Icon, tone: "text-foreground" },
  no: { icon: Cancel01Icon, tone: "text-muted-foreground" },
  unstated: { icon: MinusSignIcon, tone: "text-muted-foreground" },
};

function Intro({ content }: { content: Comparison }) {
  return (
    <header className="pt-32 pb-24 sm:pt-40">
      <h1 className="site-display max-w-240">{content.heading}</h1>
      <p className="site-lead mt-6 max-w-180 font-medium">{content.lead}</p>
      <p className="mt-4 max-w-180 text-body text-pretty text-muted-foreground">{content.intro}</p>
      <div className="mt-8">
        <Actions align="start" />
      </div>
    </header>
  );
}

function MarkCell({ cell }: { cell: Cell }) {
  const { icon, tone } = marks[cell.mark];
  const said = cell.mark === "unstated" ? undefined : words.marks[cell.mark];

  return (
    <td className={tone}>
      <span className="flex items-start gap-2">
        <HugeiconsIcon
          icon={icon}
          strokeWidth={2}
          aria-hidden="true"
          className="mt-0.5 size-4 shrink-0"
        />
        <span>
          {said && <span className="sr-only">{cell.text ? `${said}, ` : said}</span>}
          {cell.text}
        </span>
      </span>
    </td>
  );
}

/** Who each one suits, then the table behind that answer. */
function Pick({ content, rival }: { content: Comparison; rival: Tool }) {
  const choices = [
    { tool: tools.koi, text: content.pick.koi },
    { tool: rival, text: content.pick.rival },
  ];

  return (
    <section aria-labelledby="pick" className="pb-28">
      <h2 id="pick" className="site-heading max-w-220">
        {words.pick(rival.name)}
      </h2>
      <div className="mt-10 grid max-w-220 gap-x-12 gap-y-6 sm:grid-cols-2">
        {choices.map(({ tool, text }) => (
          <p key={tool.name} className="text-body text-pretty text-muted-foreground">
            <strong className="font-medium text-foreground">{`${words.choose} ${tool.name}`}</strong>
            {` ${text}`}
          </p>
        ))}
      </div>
      {/* Three columns fit a phone, so the table stays a table at every width. */}
      <table className="site-compare mt-12 max-w-220">
        <caption className="sr-only">{words.table(rival.name)}</caption>
        <thead>
          <tr>
            <th scope="col">
              <span className="sr-only">{words.feature}</span>
            </th>
            <th scope="col">
              <Name tool={tools.koi} />
            </th>
            <th scope="col">
              <Name tool={rival} />
            </th>
          </tr>
        </thead>
        <tbody>
          {content.rows.map((row) => (
            <tr key={row.feature}>
              <th scope="row">{row.feature}</th>
              <MarkCell cell={row.koi} />
              <MarkCell cell={row.rival} />
            </tr>
          ))}
        </tbody>
      </table>
      <Sources shown={[tools.koi, rival]} />
      <p className="mt-3 max-w-180 text-body text-pretty text-muted-foreground">
        {`${words.more.before} `}
        <TextLink href={comparisonsPage.path}>{words.more.all}</TextLink>
        {` ${words.more.or} `}
        <TextLink href={openSourcePage.path}>{words.more.openSource}</TextLink>
        {words.more.end}
      </p>
    </section>
  );
}

function Topics({ content, rival }: { content: Comparison; rival: Tool }) {
  return (
    <section aria-labelledby="differ" className="pb-28">
      <h2 id="differ" className="site-heading">
        {words.differ(rival.name)}
      </h2>
      <div className="mt-12 grid gap-16">
        {topics.map((topic) => (
          <div
            key={topic}
            className="grid gap-x-10 gap-y-6 md:grid-cols-[minmax(0,4fr)_minmax(0,8fr)]"
          >
            <h3 className="text-section font-medium">{words.topics[topic]}</h3>
            <div className="grid gap-x-12 gap-y-6 sm:grid-cols-2">
              {[
                { tool: tools.koi, text: koiTopics[topic] },
                { tool: rival, text: content.topics[topic] },
              ].map(({ tool, text }) => (
                <div key={tool.name}>
                  <p className="text-body font-medium">
                    <Name tool={tool} />
                  </p>
                  <p className="mt-3 text-body text-pretty text-muted-foreground">{text}</p>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}

export function ComparisonPage({ rival: id, content }: { rival: RivalId; content: Comparison }) {
  const rival = tools[id];

  return (
    <>
      <article className="mx-auto max-w-340 px-4 sm:px-7">
        <Intro content={content} />
        <Pick content={content} rival={rival} />
        <Topics content={content} rival={rival} />
      </article>
      <Faq items={content.questions} />
      <Closing />
    </>
  );
}
