import { Link } from "@tanstack/react-router";
import type { Prompt } from "@/api/prompts.api";
import { BrandPill } from "@/components/blocks/brand-pill";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ENGINES } from "../-lib/data";
import type { ExplorationSearch } from "../-lib/exploration-fixtures";
import { type PromptRow, percent, points } from "../-lib/prompt-visibility";
import C from "../-locales/mvp.en.json";
import V from "../-locales/prompt-visibility.en.json";
import { Icon } from "./common";
import { HomeBrandText } from "./home-brand-text";
import { copy } from "./list-controls";

export function PromptList({
  rows,
  search,
  onOpen,
  onEdit,
  onConfirm,
  onDelete,
}: {
  rows: PromptRow[];
  search: ExplorationSearch;
  onOpen: (id: string) => void;
  onEdit: (prompt: Prompt) => void;
  onConfirm: (id: string) => void;
  onDelete: (prompt: Prompt) => void;
}) {
  function cells(row: PromptRow) {
    const { prompt, past, stats } = row;
    const change = stats.changePoints;
    const details = {
      ...search,
      view: "visibility" as const,
      promptId: prompt.id,
      answerId: undefined,
      responseEngine: undefined,
      compareId: undefined,
    };

    return [
      <div key="prompt">
        <Link
          from="/dev/explorations/home-context"
          to="/dev/explorations/home-context"
          search={details}
          data-prompt-id={prompt.id}
          className="hc-prompt-link"
          onClick={(event) => {
            if (
              !event.metaKey &&
              !event.ctrlKey &&
              !event.shiftKey &&
              !event.altKey &&
              event.button === 0
            )
              onOpen(prompt.id);
          }}
        >
          <HomeBrandText text={prompt.text} />
        </Link>
        {(past || !prompt.confirmed) && (
          <small className="hc-prompt-status">{past ? V.pastPrompt : C.draft}</small>
        )}
      </div>,
      <span key="rate" className="hc-number">
        {stats.rate === null ? V.noAnswers : percent(stats.rate)}
      </span>,
      <span key="change" className="hc-number">
        {change === null
          ? V.noComparison
          : Math.abs(change) < 0.05
            ? V.unchanged
            : copy(change > 0 ? V.up : V.down, { points: points(change) })}
      </span>,
      <div key="engines" className="hc-engine-rates">
        {row.engines.map((item) => {
          const engine = ENGINES.find((entry) => entry.id === item.id);
          if (!engine) return null;

          return (
            <span
              key={item.id}
              title={copy(V.engineRate, {
                name: engine.name,
                rate: item.rate === null ? V.noData : percent(item.rate),
              })}
            >
              <BrandPill name={engine.name} domain={engine.domain} iconOnly />
              <span className="hc-number">{percent(item.rate)}</span>
            </span>
          );
        })}
      </div>,
      <div key="leader" className="hc-prompt-leaders">
        {row.leaders.length ? (
          <>
            {row.leaders.map((leader) => (
              <BrandPill
                key={leader.id}
                name={leader.name}
                domain={leader.domain}
                src={leader.src}
              />
            ))}
            <span className="hc-number">{percent(row.leaderRate)}</span>
          </>
        ) : stats.rate === null ? (
          V.noData
        ) : (
          V.noBrands
        )}
      </div>,
      <div key="actions" className="hc-actions">
        {!past && !prompt.confirmed && (
          <Button variant="ghost" size="sm" onClick={() => onConfirm(prompt.id)}>
            {C.confirm}
          </Button>
        )}
        {!past && (
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label={copy(C.actions, { name: prompt.text })}
                />
              }
            >
              <Icon name="more" />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="hc-brand-menu">
              <DropdownMenuItem onClick={() => onEdit(prompt)}>{C.edit}</DropdownMenuItem>
              <DropdownMenuItem onClick={() => onDelete(prompt)}>{C.delete}</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>,
    ];
  }
  const headers = [C.prompt, V.yourRate, V.change, V.byEngine, V.mostMentioned, V.actions];

  return (
    <>
      <div className="hc-prompt-table">
        <Table aria-label={V.tableLabel}>
          <TableHeader>
            <TableRow>
              {headers.map((header, i) => (
                <TableHead key={header} title={i === 4 ? V.leaderHelp : undefined}>
                  {header}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((row) => (
              <TableRow key={row.prompt.id}>
                {cells(row).map((cell, i) => (
                  <TableCell key={headers[i]}>{cell}</TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      <ul className="hc-prompt-mobile" aria-label={V.tableLabel}>
        {rows.map((row) => {
          const content = cells(row);

          return (
            <li key={row.prompt.id}>
              {content[0]}
              <dl>
                {headers.slice(1, 5).map((header, i) => (
                  <div key={header}>
                    <dt>{header}</dt>
                    <dd>{content[i + 1]}</dd>
                  </div>
                ))}
              </dl>
              {content[5]}
            </li>
          );
        })}
      </ul>
    </>
  );
}
