import { useState } from "react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  BRANDS,
  type Competitor,
  dateLabel,
  ENGINES,
  engineName,
  metrics,
  type Prompt,
  percent,
  type Section,
} from "../-lib/data";
import { useMock } from "../-lib/state";
import { C, Choice, ExploreLink, Icon, Outcome } from "./common";

export function SectionHeading({
  section,
  children,
}: {
  section: Section;
  children?: React.ReactNode;
}) {
  const leads = {
    overview: C.verdictDetail,
    mentions: C.mentionsLead,
    prompts: C.promptsLead,
    competitors: C.competitorsLead,
    citations: C.citationsLead,
    brief: C.briefScreenLead,
  };
  return (
    <div className="exp-section-heading">
      <div>
        <h1>{C[section]}</h1>
        <p>{leads[section]}</p>
      </div>
      {children}
    </div>
  );
}

function Pagination({
  page,
  total,
  onChange,
}: {
  page: number;
  total: number;
  onChange: (value: number) => void;
}) {
  const pages = Math.max(1, Math.ceil(total / 12));
  return (
    <div className="exp-pagination">
      <span>{`${C.page} ${page} ${C.of} ${pages} · ${total} ${C.items}`}</span>
      <div>
        <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => onChange(page - 1)}>
          {C.previous}
        </Button>
        <Button
          variant="outline"
          size="sm"
          disabled={page >= pages}
          onClick={() => onChange(page + 1)}
        >
          {C.next}
        </Button>
      </div>
    </div>
  );
}

export function MentionsScreen({
  compact = false,
  onSelectPrompt,
}: {
  compact?: boolean;
  onSelectPrompt?: (id: string) => void;
}) {
  const { observations, prompts, filters, setSelected } = useMock();
  const [day, setDay] = useState("179");
  const [query, setQuery] = useState("");
  const [outcome, setOutcome] = useState("all");
  const [page, setPage] = useState(1);
  const answers = observations.filter((item) => item.day === Number(day));
  const rows = [
    ...new Map(
      [...prompts, ...answers.map((item) => ({ id: item.promptId, text: item.promptText }))].map(
        (item) => [item.id, item],
      ),
    ).values(),
  ].filter(
    (item) =>
      item.text.toLowerCase().includes(query.toLowerCase()) &&
      (outcome === "all" ||
        answers.some(
          (answer) =>
            answer.promptId === item.id &&
            answer.outcome === outcome &&
            (filters.engine === "all" || answer.engine === filters.engine),
        )),
  );
  const engines = ENGINES.filter(
    (engine) => filters.engine === "all" || engine.id === filters.engine,
  );
  const currentPage = Math.min(page, Math.max(1, Math.ceil(rows.length / 12)));

  return (
    <div className={`exp-audit ${compact ? "is-compact" : ""}`}>
      <div className="exp-table-tools">
        <Input
          aria-label={C.searchAnswers}
          placeholder={C.searchAnswers}
          value={query}
          onChange={(event) => {
            setQuery(event.target.value);
            setPage(1);
          }}
        />
        {!compact ? (
          <Choice
            label={C.runHistory}
            value={day}
            options={[
              { value: "179", label: C.latestRun },
              { value: "178", label: C.priorRun },
            ]}
            onChange={setDay}
          />
        ) : null}
        <Choice
          label={C.outcome}
          value={outcome}
          options={[
            { value: "all", label: C.allOutcomes },
            ...(["cited", "mentioned", "miss", "error"] as const).map((value) => ({
              value,
              label: C[value],
            })),
          ]}
          onChange={(value) => {
            setOutcome(value);
            setPage(1);
          }}
        />
      </div>
      <div className="exp-table-scroll">
        <Table aria-label={C.mentions}>
          <TableHeader>
            <TableRow>
              <TableHead>{C.promptText}</TableHead>
              {engines.map((engine) => (
                <TableHead key={engine.id}>{engine.name}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.slice((currentPage - 1) * 12, currentPage * 12).map((prompt) => (
              <TableRow key={prompt.id}>
                <TableHead scope="row" className="exp-prompt-cell">
                  <button
                    type="button"
                    onClick={() => {
                      onSelectPrompt?.(prompt.id);
                      const observation = answers.find(
                        (item) =>
                          item.promptId === prompt.id &&
                          (filters.engine === "all" || item.engine === filters.engine),
                      );
                      if (observation && !onSelectPrompt) setSelected(observation);
                    }}
                  >
                    {prompt.text}
                  </button>
                </TableHead>
                {engines.map((engine) => {
                  const observation = answers.find(
                    (item) => item.promptId === prompt.id && item.engine === engine.id,
                  );
                  return (
                    <TableCell key={engine.id}>
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={!observation}
                        aria-label={`${prompt.text}: ${engine.name}`}
                        onClick={() => {
                          if (observation) setSelected(observation);
                          onSelectPrompt?.(prompt.id);
                        }}
                      >
                        <Outcome observation={observation} />
                      </Button>
                    </TableCell>
                  );
                })}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      {!rows.length ? (
        <div className="exp-no-results">
          <p>{C.noResults}</p>
          <Button
            variant="outline"
            onClick={() => {
              setQuery("");
              setOutcome("all");
            }}
          >
            {C.clearFilters}
          </Button>
        </div>
      ) : null}
      <Pagination page={currentPage} total={rows.length} onChange={setPage} />
    </div>
  );
}

export function ManagementScreen({ kind }: { kind: "prompts" | "competitors" }) {
  const state = useMock();
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("all");
  const [category, setCategory] = useState("all");
  const [sort, setSort] = useState("presence");
  const [page, setPage] = useState(1);
  const [edit, setEdit] = useState<Prompt | Competitor | "new" | null>(null);
  const [remove, setRemove] = useState<Prompt | Competitor | null>(null);
  const source = kind === "prompts" ? state.prompts : state.competitors;
  const rows = source
    .filter(
      (item) =>
        ("text" in item ? item.text : `${item.name} ${item.domain}`)
          .toLowerCase()
          .includes(query.toLowerCase()) &&
        (status === "all" || (status === "confirmed") === item.confirmed) &&
        (category === "all" || !("category" in item) || item.category === category),
    )
    .map((item) => ({
      item,
      rate: metrics(
        kind === "prompts"
          ? state.results.filter((answer) => answer.promptId === item.id)
          : state.results,
        kind === "prompts" ? "tally" : item.id,
      ).presence,
    }))
    .sort((a, b) =>
      sort === "name"
        ? ("text" in a.item ? a.item.text : a.item.name).localeCompare(
            "text" in b.item ? b.item.text : b.item.name,
          )
        : b.rate - a.rate,
    );
  const currentPage = Math.min(page, Math.max(1, Math.ceil(rows.length / 12)));
  const categories = [...new Set(state.prompts.map((prompt) => prompt.category))];

  function confirm(item: Prompt | Competitor) {
    if (kind === "prompts")
      state.setPrompts((current) =>
        current.map((value) => (value.id === item.id ? { ...value, confirmed: true } : value)),
      );
    else
      state.setCompetitors((current) =>
        current.map((value) => (value.id === item.id ? { ...value, confirmed: true } : value)),
      );
    state.setNotice("Confirmed for the next sample scan.");
  }

  function removeItem() {
    if (!remove) return;
    if (kind === "prompts")
      state.setPrompts((current) => current.filter((item) => item.id !== remove.id));
    else state.setCompetitors((current) => current.filter((item) => item.id !== remove.id));
    state.setNotice("Removed from tracking. Historical answers are still available.");
    setRemove(null);
  }

  return (
    <div className="exp-management">
      <div className="exp-table-tools">
        <Input
          aria-label={kind === "prompts" ? C.searchPrompts : C.searchCompetitors}
          placeholder={kind === "prompts" ? C.searchPrompts : C.searchCompetitors}
          value={query}
          onChange={(event) => {
            setQuery(event.target.value);
            setPage(1);
          }}
        />
        <Choice
          label={C.status}
          value={status}
          options={[
            { value: "all", label: C.allStatuses },
            { value: "confirmed", label: C.confirmed },
            { value: "draft", label: C.draft },
          ]}
          onChange={(value) => {
            setStatus(value);
            setPage(1);
          }}
        />
        {kind === "prompts" ? (
          <Choice
            label={C.category}
            value={category}
            options={[
              { value: "all", label: C.allCategories },
              ...categories.map((value) => ({ value, label: value })),
            ]}
            onChange={(value) => {
              setCategory(value);
              setPage(1);
            }}
          />
        ) : null}
        <Choice
          label={C.sort}
          value={sort}
          options={[
            { value: "presence", label: C.sortPresence },
            { value: "name", label: C.sortName },
          ]}
          onChange={setSort}
        />
        <Button onClick={() => setEdit("new")} size="lg">
          <Icon name="add" size={16} />
          {kind === "prompts" ? C.addPrompt : C.addCompetitor}
        </Button>
      </div>
      {kind === "competitors" ? (
        <div className="exp-compare-note">
          <span className="exp-letter">{C.tally[0]}</span>
          <strong>{C.tally}</strong>
          <span>{C.presence}</span>
          <strong>{percent(state.summary.presence)}</strong>
        </div>
      ) : null}
      <div className="exp-table-scroll">
        <Table aria-label={C[kind]}>
          <TableHeader>
            <TableRow>
              <TableHead>{kind === "prompts" ? C.promptText : C.name}</TableHead>
              <TableHead>{kind === "prompts" ? C.category : C.domain}</TableHead>
              <TableHead>{C.status}</TableHead>
              <TableHead>{C.presence}</TableHead>
              <TableHead className="exp-actions-cell">{C.actions}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.slice((currentPage - 1) * 12, currentPage * 12).map(({ item, rate }) => (
              <TableRow key={item.id}>
                <TableHead scope="row" className="exp-prompt-cell">
                  <button
                    type="button"
                    onClick={() => {
                      const answer = [...state.results]
                        .reverse()
                        .find((value) =>
                          kind === "prompts"
                            ? value.promptId === item.id
                            : value.brands.includes(item.id),
                        );
                      if (answer) state.setSelected(answer);
                      else setEdit(item);
                    }}
                  >
                    {"text" in item ? (
                      item.text
                    ) : (
                      <span className="exp-name-with-letter">
                        <span className="exp-letter">{item.name[0]}</span>
                        {item.name}
                      </span>
                    )}
                  </button>
                </TableHead>
                <TableCell>{"category" in item ? item.category : item.domain}</TableCell>
                <TableCell>
                  <span className={`exp-status ${item.confirmed ? "confirmed" : "draft"}`}>
                    <i />
                    {item.confirmed ? C.confirmed : C.draft}
                  </span>
                </TableCell>
                <TableCell className="tabular-nums">
                  {state.results.some((value) =>
                    kind === "prompts"
                      ? value.promptId === item.id
                      : value.brands.includes(item.id),
                  )
                    ? percent(rate)
                    : C.notCollected}
                </TableCell>
                <TableCell>
                  <DropdownMenu>
                    <DropdownMenuTrigger
                      render={<Button variant="ghost" size="icon-lg" />}
                      aria-label={`${C.actions}: ${"text" in item ? item.text : item.name}`}
                    >
                      <Icon name="more" />
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem onClick={() => setEdit(item)}>{C.edit}</DropdownMenuItem>
                      {!item.confirmed ? (
                        <DropdownMenuItem onClick={() => confirm(item)}>
                          {C.confirm}
                        </DropdownMenuItem>
                      ) : null}
                      <DropdownMenuItem variant="destructive" onClick={() => setRemove(item)}>
                        {C.delete}
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      {!rows.length ? (
        <div className="exp-no-results">
          <p>{source.length ? C.noResults : C.noTracked}</p>
          <Button
            variant="outline"
            onClick={() => {
              setQuery("");
              setStatus("all");
              setCategory("all");
            }}
          >
            {C.clearFilters}
          </Button>
        </div>
      ) : null}
      <Pagination page={currentPage} total={rows.length} onChange={setPage} />
      <p className="exp-management-hint">{C.manageHint}</p>
      <Dialog
        open={edit !== null}
        onOpenChange={(open) => {
          if (!open) setEdit(null);
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {kind === "prompts"
                ? edit === "new"
                  ? C.addPrompt
                  : C.editPrompt
                : edit === "new"
                  ? C.addCompetitor
                  : C.editCompetitor}
            </DialogTitle>
            <DialogDescription>{C.manageHint}</DialogDescription>
          </DialogHeader>
          {edit ? (
            <EditForm
              key={edit === "new" ? "new" : edit.id}
              kind={kind}
              item={edit}
              onClose={() => setEdit(null)}
            />
          ) : null}
        </DialogContent>
      </Dialog>
      <AlertDialog
        open={Boolean(remove)}
        onOpenChange={(open) => {
          if (!open) setRemove(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{C.deleteTitle}</AlertDialogTitle>
            <AlertDialogDescription>{C.deleteLead}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{C.cancel}</AlertDialogCancel>
            <AlertDialogAction onClick={removeItem}>{C.delete}</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function EditForm({
  kind,
  item,
  onClose,
}: {
  kind: "prompts" | "competitors";
  item: Prompt | Competitor | "new";
  onClose: () => void;
}) {
  const state = useMock();
  const [name, setName] = useState(item === "new" ? "" : "text" in item ? item.text : item.name);
  const [domain, setDomain] = useState(item !== "new" && "domain" in item ? item.domain : "");
  const [category, setCategory] = useState(
    item !== "new" && "category" in item ? item.category : "Discovery",
  );
  const [confirmed, setConfirmed] = useState(item === "new" || item.confirmed);
  const [error, setError] = useState("");

  function save(event: React.FormEvent) {
    event.preventDefault();
    const normalizedDomain = domain
      .trim()
      .replace(/^https?:\/\//, "")
      .replace(/\/$/, "")
      .toLowerCase();
    if (
      kind === "prompts"
        ? name.trim().length < 5
        : !name.trim() || !/^[a-z0-9-]+(?:\.[a-z0-9-]+)+(?:\/[a-z0-9/_-]*)?$/.test(normalizedDomain)
    ) {
      setError(kind === "prompts" ? C.promptRequired : C.nameRequired);
      return;
    }

    if (kind === "prompts")
      state.savePrompt(
        { text: name.trim(), category, confirmed },
        item === "new" ? undefined : item.id,
      );
    else
      state.saveCompetitor(
        { name: name.trim(), domain: normalizedDomain, confirmed },
        item === "new" ? undefined : item.id,
      );
    onClose();
  }

  return (
    <form onSubmit={save}>
      <FieldGroup>
        <Field data-invalid={Boolean(error)}>
          <FieldLabel htmlFor="exp-edit-name">
            {kind === "prompts" ? C.promptText : C.name}
          </FieldLabel>
          <Input
            id="exp-edit-name"
            value={name}
            onChange={(event) => setName(event.target.value)}
            aria-invalid={Boolean(error)}
            aria-describedby={error ? "exp-edit-error" : undefined}
          />
          {error ? <FieldError id="exp-edit-error">{error}</FieldError> : null}
        </Field>
        {kind === "competitors" ? (
          <Field>
            <FieldLabel htmlFor="exp-edit-domain">{C.domain}</FieldLabel>
            <Input
              id="exp-edit-domain"
              value={domain}
              onChange={(event) => setDomain(event.target.value)}
              aria-invalid={Boolean(error)}
            />
          </Field>
        ) : (
          <Field>
            <FieldLabel>{C.category}</FieldLabel>
            <Choice
              label={C.category}
              value={category}
              options={[
                "Discovery",
                "Comparison",
                "Integrations",
                "Features",
                "Security",
                "Enterprise",
              ].map((value) => ({ value, label: value }))}
              onChange={setCategory}
            />
          </Field>
        )}
        <Field>
          <FieldLabel>{C.status}</FieldLabel>
          <Choice
            label={C.status}
            value={confirmed ? "confirmed" : "draft"}
            options={[
              { value: "confirmed", label: C.confirmed },
              { value: "draft", label: C.draft },
            ]}
            onChange={(value) => setConfirmed(value === "confirmed")}
          />
        </Field>
        <div className="exp-form-actions">
          <Button type="button" variant="outline" onClick={onClose}>
            {C.cancel}
          </Button>
          <Button type="submit">{item === "new" ? C.add : C.save}</Button>
        </div>
      </FieldGroup>
    </form>
  );
}

export function CitationsScreen() {
  const { results, setSelected } = useMock();
  const [query, setQuery] = useState("");
  const [sort, setSort] = useState("count");
  const domains = new Map<
    string,
    { count: number; prompts: Set<string>; answer: (typeof results)[number] }
  >();
  for (const answer of results)
    for (const source of answer.sources) {
      const domain = new URL(source).hostname;
      const current = domains.get(domain);
      if (current) {
        current.count += 1;
        current.prompts.add(answer.promptId);
        current.answer = answer;
      } else domains.set(domain, { count: 1, prompts: new Set([answer.promptId]), answer });
    }
  const rows = [...domains]
    .filter(([domain]) => domain.includes(query.toLowerCase()))
    .sort((a, b) => (sort === "name" ? a[0].localeCompare(b[0]) : b[1].count - a[1].count));

  return (
    <div className="exp-citations">
      <div className="exp-table-tools">
        <Input
          aria-label={C.searchDomains}
          placeholder={C.searchDomains}
          value={query}
          onChange={(event) => setQuery(event.target.value)}
        />
        <Choice
          label={C.sort}
          value={sort}
          options={[
            { value: "count", label: C.sortCount },
            { value: "name", label: C.sortName },
          ]}
          onChange={setSort}
        />
      </div>
      <Table aria-label={C.citations}>
        <TableHeader>
          <TableRow>
            <TableHead>{C.domainTitle}</TableHead>
            <TableHead>{C.references}</TableHead>
            <TableHead>{C.uniquePrompts}</TableHead>
            <TableHead>{C.latest}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map(([domain, value]) => (
            <TableRow key={domain}>
              <TableHead scope="row">
                <button
                  type="button"
                  className="exp-source-domain"
                  onClick={() => setSelected(value.answer)}
                >
                  <span className="exp-letter">{domain[0].toUpperCase()}</span>
                  {domain}
                  <Icon name="next" size={14} />
                </button>
              </TableHead>
              <TableCell>{value.count}</TableCell>
              <TableCell>{value.prompts.size}</TableCell>
              <TableCell>{dateLabel(value.answer.day)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      {!rows.length ? (
        <div className="exp-no-results">
          <p>{C.noResults}</p>
          <Button variant="outline" onClick={() => setQuery("")}>
            {C.clearFilters}
          </Button>
        </div>
      ) : null}
    </div>
  );
}

export function BriefScreen({ compact = false }: { compact?: boolean }) {
  const { brief, briefVersion, regenerateBrief, setNotice, results, setSelected, summary } =
    useMock();

  async function copy() {
    try {
      await navigator.clipboard.writeText(brief);
      setNotice(C.copied);
    } catch {
      setNotice(C.clipboardError);
    }
  }

  function download() {
    const url = URL.createObjectURL(new Blob([brief], { type: "text/markdown;charset=utf-8" }));
    const link = document.createElement("a");
    link.href = url;
    link.download = "tally-ai-visibility-brief.md";
    link.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
    setNotice(C.downloaded);
  }

  const gap =
    results.find((item) => item.promptText.includes("SSO") && item.outcome === "miss") ??
    results.find((item) => item.outcome === "miss");
  return (
    <div className={`exp-brief-screen ${compact ? "is-compact" : ""}`}>
      <div className="exp-brief-actions">
        <Button onClick={() => void copy()} size="lg" disabled={!results.length}>
          <Icon name="copy" size={16} />
          {C.copyBrief}
        </Button>
        <Button variant="outline" size="lg" onClick={download} disabled={!results.length}>
          <Icon name="download" size={16} />
          {C.downloadBrief}
        </Button>
        <Button variant="ghost" onClick={regenerateBrief} disabled={!results.length}>
          {C.regenerate}
        </Button>
      </div>
      {!results.length ? (
        <p>{C.noBrief}</p>
      ) : (
        <>
          <div className="exp-brief-header">
            <span className="exp-overline">{`${C.briefVersion} ${briefVersion} · ${C.asOf}`}</span>
            <h2>{C.briefTitle}</h2>
            <p>{`${C.tally} ${C.presence.toLowerCase()}: ${percent(summary.presence)}. ${C.briefLead}`}</p>
          </div>
          <div className="exp-brief-tasks">
            {[
              { title: C.task1, description: C.task1Lead },
              { title: C.task2, description: C.task2Lead },
              { title: C.task3, description: C.task3Lead },
            ].map((task, index) => (
              <section key={task.title}>
                <span>{index + 1}</span>
                <div>
                  <h3>{task.title}</h3>
                  <p>{task.description}</p>
                </div>
              </section>
            ))}
          </div>
          <div className="exp-brief-acceptance">
            <h3>{C.acceptance}</h3>
            <p>{C.acceptanceLead}</p>
            {gap ? (
              <Button variant="outline" onClick={() => setSelected(gap)}>
                {C.readEvidence}
                <Icon name="next" size={14} />
              </Button>
            ) : null}
          </div>
          {!compact ? (
            <details className="exp-markdown">
              <summary>{C.markdown}</summary>
              <pre>{brief}</pre>
            </details>
          ) : null}
        </>
      )}
    </div>
  );
}

export function SignalFeed() {
  const { results, setSelected } = useMock();
  const items = [
    {
      title: C.changeOne,
      body: C.changeOneBody,
      label: "Perplexity",
      match: results.find((item) => item.engine === "perplexity" && item.outcome === "cited"),
    },
    {
      title: C.changeTwo,
      body: C.changeTwoBody,
      label: "Enterprise",
      match: results.find((item) => item.promptText.includes("SSO") && item.outcome === "miss"),
    },
    {
      title: C.changeThree,
      body: C.changeThreeBody,
      label: "Comparison",
      match: results.find(
        (item) => item.promptText.includes("Typeform") && item.outcome !== "error",
      ),
    },
  ];
  return (
    <div className="exp-signal-feed">
      {items.map((item) => (
        <article key={item.title}>
          <div className="exp-feed-meta">
            <span>
              <i />
              {item.label}
            </span>
            <span>{C.today}</span>
          </div>
          <h3>{item.title}</h3>
          <p>{item.body}</p>
          <Button
            variant="link"
            onClick={() => {
              if (item.match) setSelected(item.match);
            }}
            disabled={!item.match}
          >
            {C.readEvidence}
            <Icon name="next" size={14} />
          </Button>
        </article>
      ))}
    </div>
  );
}

export function Heatmap({ onSelect }: { onSelect: (id: string) => void }) {
  const { prompts, results, filters } = useMock();
  return (
    <div className="exp-heatmap">
      {prompts
        .filter((prompt) => prompt.confirmed)
        .slice(0, 12)
        .map((prompt) => {
          const rate = metrics(results.filter((item) => item.promptId === prompt.id)).presence;
          return (
            <button
              key={prompt.id}
              type="button"
              className={rate < 35 ? "is-loss" : "is-gain"}
              style={{ flexGrow: Math.max(1, rate / 10) }}
              onClick={() => onSelect(prompt.id)}
              title={`${prompt.text}: ${percent(rate)} (${filters.days} days)`}
            >
              <span>{prompt.text}</span>
              <strong>{percent(rate)}</strong>
            </button>
          );
        })}
    </div>
  );
}

export function PromptScreener({
  selected,
  onSelect,
}: {
  selected: string | null;
  onSelect: (id: string) => void;
}) {
  const { prompts, results } = useMock();
  const [query, setQuery] = useState("");
  const rows = prompts.filter((item) => item.text.toLowerCase().includes(query.toLowerCase()));
  return (
    <div className="exp-screener">
      <Input
        aria-label={C.searchPrompts}
        placeholder={C.searchPrompts}
        value={query}
        onChange={(event) => setQuery(event.target.value)}
      />
      <div className="exp-screener-head">
        <span>{C.promptText}</span>
        <span>{C.presence}</span>
        <span>{C.citationRate}</span>
      </div>
      {rows.map((prompt) => {
        const summary = metrics(results.filter((item) => item.promptId === prompt.id));
        return (
          <button
            type="button"
            key={prompt.id}
            className={`exp-screener-row ${selected === prompt.id ? "is-selected" : ""}`}
            onClick={() => onSelect(prompt.id)}
          >
            <span>{prompt.text}</span>
            <span className={summary.presence < 35 ? "exp-negative" : "exp-positive"}>
              {percent(summary.presence)}
            </span>
            <span>{percent(summary.cited)}</span>
          </button>
        );
      })}
      {!rows.length ? <p>{C.noMatching}</p> : null}
    </div>
  );
}

export function AnswerInspector({ promptId }: { promptId: string | null }) {
  const { results, filters, setFilters, setSelected } = useMock();
  const [engine, setEngine] = useState(ENGINES[0].id);
  const answer = [...results]
    .reverse()
    .find(
      (item) =>
        item.promptId === promptId &&
        item.engine === (filters.engine === "all" ? engine : filters.engine),
    );
  return (
    <div className="exp-inspector">
      <Choice
        label={C.engine}
        value={filters.engine === "all" ? engine : filters.engine}
        options={ENGINES.map((item) => ({ value: item.id, label: item.name }))}
        onChange={(value) => {
          setEngine(value as typeof engine);
          if (filters.engine !== "all") setFilters({ ...filters, engine: value as typeof engine });
        }}
      />
      {answer ? (
        <>
          <h3>{answer.promptText}</h3>
          <Outcome observation={answer} />
          <p>{answer.rawAnswer}</p>
          <h4>{C.sources}</h4>
          <ul>
            {answer.sources.map((source) => (
              <li key={source}>
                <a href={source} target="_blank" rel="noopener noreferrer">
                  {new URL(source).hostname}
                </a>
              </li>
            ))}
          </ul>
          {!answer.sources.length ? <p className="exp-muted">{C.noSources}</p> : null}
          <Button variant="outline" onClick={() => setSelected(answer)}>
            {C.inspect}
          </Button>
          <span className="exp-muted">{`${engineName(answer.engine)} · ${dateLabel(answer.day)}`}</span>
        </>
      ) : (
        <div className="exp-inspector-empty">
          <Icon name="quote" size={28} />
          <h3>{C.selectPrompt}</h3>
          <p>{C.panelLead}</p>
        </div>
      )}
    </div>
  );
}

export function FullSection({ section }: { section: Section }) {
  if (section === "mentions") return <MentionsScreen />;
  if (section === "prompts" || section === "competitors")
    return <ManagementScreen kind={section} />;
  if (section === "citations") return <CitationsScreen />;
  if (section === "brief") return <BriefScreen />;
  return null;
}

export function ContextNote({ section }: { section: Section }) {
  const { summary } = useMock();
  return (
    <aside className="exp-context-note">
      <span className="exp-overline">{C.atAGlance}</span>
      <strong>{percent(summary.presence)}</strong>
      <p>{C.presenceHint}</p>
      <ExploreLink section={section === "brief" ? "mentions" : "brief"} className="exp-text-link">
        {section === "brief" ? C.historical : C.openBrief}
        <Icon name="next" size={16} />
      </ExploreLink>
      <div className="exp-context-brands">
        {BRANDS.slice(0, 4).map((brand) => (
          <span key={brand.id} className="exp-letter" title={brand.name}>
            {brand.name[0]}
          </span>
        ))}
      </div>
    </aside>
  );
}
