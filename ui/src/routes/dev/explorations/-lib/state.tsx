import { createContext, type ReactNode, useContext, useEffect, useRef, useState } from "react";
import {
  buildBrief,
  type Competitor,
  collect,
  type DemoState,
  type Direction,
  FIXTURES,
  type Filters,
  INITIAL_COMPETITORS,
  INITIAL_PROMPTS,
  metrics,
  type Observation,
  type Prompt,
  selectObservations,
} from "./data";

function useMockState(direction: Direction, initialState: DemoState) {
  const [demoState, setDemoState] = useState<DemoState>(initialState);
  const [filters, setFilters] = useState<Filters>({ days: 30, engine: "all" });
  const [prompts, setPrompts] = useState(INITIAL_PROMPTS);
  const [competitors, setCompetitors] = useState(INITIAL_COMPETITORS);
  const [observations, setObservations] = useState(FIXTURES);
  const [selected, setSelected] = useState<Observation | null>(null);
  const [notice, setNotice] = useState("");
  const [briefVersion, setBriefVersion] = useState(1);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const sequence = useRef(0);

  useEffect(() => {
    setDemoState(initialState);
    if (timer.current) clearTimeout(timer.current);
  }, [initialState]);

  useEffect(
    () => () => {
      if (timer.current) clearTimeout(timer.current);
    },
    [],
  );
  useEffect(() => {
    if (!notice) return;
    const timeout = setTimeout(() => setNotice(""), 4500);
    return () => clearTimeout(timeout);
  }, [notice]);

  const visible = demoState === "populated" ? observations : [];
  const results = selectObservations(visible, filters);
  const summary = metrics(results);
  const previous = metrics(selectObservations(visible, filters, true));

  function scan() {
    if (timer.current) clearTimeout(timer.current);
    setDemoState("loading");
    timer.current = setTimeout(() => {
      sequence.current += 1;
      setObservations((current) => [
        ...current.filter((item) => item.day !== 179),
        ...collect(prompts, competitors, 179, sequence.current),
      ]);
      setDemoState("populated");
      setNotice("Sample scan complete. The latest answers are ready.");
    }, 650);
  }

  function reset() {
    if (timer.current) clearTimeout(timer.current);
    setPrompts(INITIAL_PROMPTS);
    setCompetitors(INITIAL_COMPETITORS);
    setObservations(FIXTURES);
    setFilters({ days: 30, engine: "all" });
    setSelected(null);
    setDemoState("populated");
    sequence.current = 0;
    setNotice("Demo restored to its original sample data.");
  }

  function savePrompt(value: Omit<Prompt, "id">, id?: string) {
    setPrompts((current) =>
      id
        ? current.map((item) => (item.id === id ? { ...value, id } : item))
        : [...current, { ...value, id: `local-prompt-${++sequence.current}` }],
    );
    setNotice(
      id
        ? "Prompt saved. Historical answers are unchanged."
        : "Prompt added. Run a sample scan to collect answers.",
    );
  }

  function saveCompetitor(value: Omit<Competitor, "id">, id?: string) {
    setCompetitors((current) =>
      id
        ? current.map((item) => (item.id === id ? { ...value, id } : item))
        : [...current, { ...value, id: `local-competitor-${++sequence.current}` }],
    );
    setNotice(id ? "Competitor saved." : "Competitor added. Run a sample scan to compare it.");
  }

  return {
    direction,
    demoState,
    filters,
    setFilters,
    prompts,
    setPrompts,
    competitors,
    setCompetitors,
    observations: visible,
    results,
    summary,
    previous,
    selected,
    setSelected,
    notice,
    setNotice,
    scan,
    reset,
    savePrompt,
    saveCompetitor,
    brief: buildBrief(visible, filters),
    briefVersion,
    regenerateBrief: () => {
      setBriefVersion((value) => value + 1);
      setNotice("Brief regenerated from the selected sample evidence.");
    },
  };
}

type MockState = ReturnType<typeof useMockState>;
const MockContext = createContext<MockState | null>(null);

export function MockProvider({
  direction,
  initialState,
  children,
}: {
  direction: Direction;
  initialState: DemoState;
  children: ReactNode;
}) {
  const state = useMockState(direction, initialState);
  return <MockContext.Provider value={state}>{children}</MockContext.Provider>;
}

export function useMock() {
  const state = useContext(MockContext);
  if (!state) throw new Error("Exploration components require MockProvider");
  return state;
}
