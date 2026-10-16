import { createFileRoute, Link } from "@tanstack/react-router";
import { useHomeWorkspace } from "./-components/home-workspace-context";
import V from "./-locales/prompt-visibility.en.json";
import { Management } from "./home-context";

export const Route = createFileRoute("/dev/explorations/home-context/competitors")({
  component: CompetitorsPage,
});

function CompetitorsPage() {
  const { brand, state, search, rows, update } = useHomeWorkspace();

  return (
    <>
      <Link
        to="/dev/explorations/home-context"
        search={{ ...search, view: undefined, promptId: undefined, runId: undefined }}
        className="hc-text-link"
      >
        {V.back}
      </Link>
      <Management
        key={`${brand}:competitors`}
        kind="competitors"
        brand={brand}
        state={state}
        scenario={search.state}
        rows={rows}
        search={search}
        onSearch={update}
        onOpen={() => {}}
      />
    </>
  );
}
