# Koi: agentic workspace discovery

Status: approach 1 chosen on 13 October 2026. Visual design and implementation remain in discovery.

## Product premise

Koi is a full agentic AI visibility layer. People can work with it through their existing
agents. The UI adapts to their goals and preferences, giving people a way to understand,
inspect, and direct that work.

Use two separate owned brands throughout discovery:

- **Deja:** an open-source zsh tool that predicts and ranks commands from shell history.
  Its documented positioning is a replacement for zsh-autosuggestions. See the brand brief below.
- **Koi:** the AI visibility product itself, tracked as a brand within Koi.

The founder can ask their agent to shape Home around a goal for either brand. Deja and
Koi are not competitors in this example. Multi-brand support is planned for the near
future; using both in discovery does not claim that support is implemented today.

The goal affects the questions being investigated, the comparisons shown, the evidence
surfaced, and the next actions. It must change more than the arrangement of widgets.

## Deja brand brief

Repository reviewed on 13 October 2026:
[Giammarco-Ferranti/deja](https://github.com/Giammarco-Ferranti/deja).

The [README](https://github.com/Giammarco-Ferranti/deja/blob/main/README.md) describes
inline command suggestions for zsh on macOS and Linux, including WSL. Deja is MIT-licensed
and explicitly positioned as a replacement for zsh-autosuggestions. Suggestions come from
existing history; it does not generate new commands or execute a suggestion automatically.

The [architecture guide](https://github.com/Giammarco-Ferranti/deja/blob/main/docs/architecture.md)
explains four ranking signals: fuzzy matching, frequency/recency, directory affinity, and
learned command sequences. One local daemon serves terminal windows, backed by SQLite.
These provide concrete positioning angles: recalling abbreviated commands, remembering
project-specific workflows, and anticipating a command that usually follows another.

Audience inference: developers and other frequent terminal users who use zsh. The repository
establishes compatibility and functionality, not a validated customer segmentation.

Proposed first visibility goal:

> Get Deja recommended when zsh users ask for alternatives to zsh-autosuggestions.

The named comparison comes from Deja's own positioning. It is not a measured visibility gap
or a confirmed list of every competing product. Suggested research prompts, not collected data:

- “What are good alternatives to zsh-autosuggestions?”
- “Which zsh tools suggest commands based on the current project directory?”
- “How can I get fuzzy inline command suggestions in zsh?”

Use the repository URL as the known brand reference; no standalone product domain has been
provided. Match mentions using product and shell context so unrelated uses of “deja” are
not treated as brand evidence.

## What is established, proposed, and open

Established by the product direction:

- Work with people's agents, rather than requiring a separate Koi assistant.
- Adapt the UI around user goals and desires.
- Use a personalized home as the primary experience, with stable sections for prompts,
  competitors, mentions/answers, and citations.
- Use Deja and Koi as separate brand examples, anticipating future multi-brand support.
- Explore one section and one concept at a time, using Mobbin for research.
- The current design system can be reconsidered during discovery.
- Closely follow Fey's original product visual system. Working rules and font research
  live in [fey-visual-system.md](fey-visual-system.md).
- A sidebar is not a requirement. Explore a navigation island or no persistent navigation.
- Leave UI testing to the user and supply manual checks.

Proposals in this document remain hypotheses. The previous Koi Focus exploration is not
the baseline for this discovery. Its static dashboard blend did not establish the goal-driven
interaction model. This document starts from that model; it does not change existing UI code.

Open decisions include the first section's purpose, how much composition freedom agents
have, how preferences persist, and whether navigation needs a permanent visible surface.

## Product direction: personalized home with stable sections

Decision: choose approach 1, inspired by Fey's home model. Home adapts to the person's
goal; the underlying management and evidence sections remain predictable and directly usable.
This selects the product structure, not Fey's palette, dock, or exact layout.

For either Deja or Koi:

1. The founder arrives at a useful home based on the tracked brand and available evidence.
2. They ask their existing agent to focus Home on a goal for that specific brand.
3. Home adapts its comparisons, priorities, evidence, and next actions to that goal.
4. Prompts, competitors, answers, and citations remain accessible in stable sections.
5. Direct edits and inspection remain possible without the agent.

Panel selection and arrangement are not the primary interaction. Supported blocks can
still power composition internally. Multiple saved goal views are a future possibility,
not a requirement for the initial experience. No view picker or layout builder is implied.

The intended advantage is that an agent shapes a persistent, interactive home around a goal,
while the person can understand and inspect the evidence behind it. Agent connectivity alone
is not sufficient differentiation.

Market research reviewed on 13 October 2026:

- [Profound](https://help.tryprofound.com/articles/5194011335-interpret-answer-engine-insights-v2)
  documents stable analytical tabs, saved filter views, configurable comparisons, and AI
  investigation from selected charts.
- [Scrunch](https://scrunch.com/faqs/does-scrunch-do-competitive-benchmarking-against-multiple-answer-engines/)
  documents dashboard comparisons and competitor management in its AI Context area.
- [Peec](https://peec.ai/blog/peec-ai-mcp) documents MCP access for existing AI tools and workflows.
- [Otterly](https://otterly.ai/marketplace/custom-dashboard) provides a workflow for creating
  a custom dashboard as an external Claude artifact.

Interpretation: familiar product sections can coexist with agent-driven workflows.
These sources establish available features, not customer preference or retention evidence.
The recommendation to start with a personalized home is product judgment, now accepted
by the user; it is not a claim that the market has validated Koi's exact approach.

## First discovery unit: brand and goal context on Home

Start with the top of Home. Before choosing charts, establish whose visibility the person
is inspecting, what they want to achieve, and which evidence the screen represents.

Proposed hierarchy:

1. **Active brand:** Deja or Koi, with a discoverable brand switcher.
2. **Active goal:** the question or outcome driving this brand's Home.
3. **Evidence scope:** prompts/topics, engines, time window, and collection freshness.

Brand switching changes the subject of the product. Goal changes adapt Home within that
brand. Navigation changes the section within that brand. Keep these three operations
distinct even if they eventually share a compact control surface.

Concept sketches; the example goals below are proposals, not approved goals or findings:

```text
Deja ▾                                           Goal controls
Get recommended as an alternative to zsh-autosuggestions
[proposed discovery + comparison prompts] · selected engines · last 30 days

[Home content chosen for Deja's goal]
```

```text
Koi ▾                                            Goal controls
Understand Koi's visibility for AI visibility tools
[prompt scope to define] · selected engines · last 30 days

[Home content chosen for Koi's goal]
```

The brand label should remain in a predictable location when Home adapts. Selecting Koi
should restore Koi's context; returning to Deja should restore Deja's context. Whether the
current section carries across brand switches remains an open interaction decision.

There is no cross-brand aggregate dashboard in this first concept. Personal presentation
defaults may be shared, but goals, prompts, competitors, answers, and home compositions
are scoped to the selected brand. Agent operations must identify their target brand explicitly.

The first Home concept combines brand selection, goal expression, and scope with a useful
analytical section: mention-rate history, comparison against alternatives, inspectable demo
answers, and a short recap of missing questions. A markdown export carries this context to
the user's existing agent. All evidence in this exploration is synthetic and labeled.
Defer the full navigation shell, task management, brief editing, and layout customization.

## Visual and UX hypotheses

**Brand and goal give Home its identity.** The brand tells the person whose visibility
they are inspecting; the goal explains why this composition exists. The scope sits close
to that context rather than disappearing into an agent conversation.

**Composition follows the question.** A competitive goal may need a paired comparison;
an engine-specific goal may need a different comparison. A source investigation may begin
with citations. We should not force every goal into the same metric row and chart.

**Evidence has a stable way in.** People can select a claim, question, or data point to inspect
its basis. Agent-authored interpretation and measured facts should remain distinguishable.

**Adaptation is understandable.** A goal change can produce a new composition. A routine
data refresh updates the current composition without moving controls or resetting selection.
Show what an agent changed and allow the person to restore the previous view. Whether
changes apply immediately or arrive as proposals is still open.

**Preferences constrain adaptation.** “Keep the comparison visible”, “show me a table”,
or “only show Perplexity” should survive future agent changes. We still need to decide
whether these belong to a saved view, a goal, or a person's defaults.

**Direct use remains possible.** People can adjust scope, inspect evidence, and recover
from a failed operation without sending another message to their agent.

The visual reference is now Fey's original product UI. Use layered dark surfaces, compact
controls, and its restrained spacing and hierarchy. Koi currently uses Inter Variable;
secondary references identify Fey's font as Calibre. Exact font adoption remains open.
See [fey-visual-system.md](fey-visual-system.md) for evidence and working values.

## Navigation: two candidates, no decision yet

“No navigation” here means no permanent application navigation. Brand switching, Home,
stable product sections, return paths, scope controls, and account or connection settings
must still be discoverable.

| Candidate | What stays visible | What we need to learn |
| --- | --- | --- |
| Navigation island | A compact, labeled surface for accessing Home and stable product sections | Does it help people orient themselves, or pull attention away from the active goal? |
| No persistent navigation | Goal identity and local controls; broader destinations available on demand | Can people find management sections, return from evidence, and find settings without guessing? |

Neither candidate should reproduce the old app sections as a row of unlabeled icons.
If an island exists, its contents should reflect what people need to move between.

Proposed sequence: establish brand/goal context and the first analytical section, then compare both navigation
treatments around exactly the same content. This avoids confusing a navigation preference
with a preference for different dashboards.

## Architecture hypothesis: shared evidence, declarative home composition

This is a product-level architecture proposal, not an API commitment.

Separate four responsibilities:

| Responsibility | Owns |
| --- | --- |
| Goal | Brand-scoped user intent, competitors, scope, and unresolved assumptions |
| Evidence | Collected answers, engine identity, sources, collection status, and dates |
| View definition | The questions, supported blocks, order, and shared selections used to inspect a goal |
| Execution | Collection and analysis work, progress, failures, and proposed next actions |

An agent can operate on the goal and view through the same domain capabilities as the UI.
The UI renders a saved home definition against evidence. Stable product sections use that
same evidence independently of the home composition. The agent does not need to generate
arbitrary React or CSS to change home.

A provisional definition could look like this; names and schema are illustrative:

```json
{
  "version": 1,
  "goalId": "koi-discovery",
  "title": "Koi's AI visibility",
  "scope": {
    "brandId": "koi",
    "competitorIds": [],
    "promptSetId": "koi-discovery-prompts",
    "engineIds": ["chatgpt", "perplexity"],
    "windowDays": 30
  },
  "blocks": [
    { "id": "presence", "kind": "brandPresenceTrend" },
    { "id": "questions", "kind": "promptPresence" },
    { "id": "evidence", "kind": "answerEvidence", "selectionFrom": "questions" }
  ]
}
```

Example data flow:

```text
User intent → their agent → goal + supported view definition
                                      ↓
                             validate and save a revision
                                      ↓
Collected evidence → scoped measurements → UI rendering
                                      ↓
                          shared selection → answer detail
```

The renderer owns responsive layout, control behavior, keyboard access, and measurement
presentation. The agent selects supported analytical blocks and their scope. How freely
it can choose size, placement, or new block types is an explicit future decision.

Persist shared goal/view configuration separately from transient inspection state.
Selecting an answer should not rewrite everyone's saved workspace. Use revisions so
concurrent agent and human edits can be detected instead of silently overwriting each other.
The illustrative definition above belongs to Koi as the tracked brand. Deja would have
its own definition and evidence scope. The eventual multi-brand implementation must validate
that referenced goals, prompt sets, and answers belong to the target brand.

Current cookie authentication, owner-supplied Cloro collection, optional integrations,
and engine/provider separation remain constraints. Agent authentication and CLI/MCP
transport need separate design; this document does not treat them as already implemented.
Existing answers remain usable evidence, and saved views reference them rather than copying
them into agent-written content. No historical database compatibility work is implied.

## Failure cases the concept must explain

- **Ambiguous goal:** expose the proposed scope and unresolved assumptions; do not silently
  invent a success target or equate visibility with revenue.
- **Insufficient evidence:** show coverage and an explicit collection action. No fabricated
  rates, narrative conclusions, or zero values presented as measured absence.
- **Partial collection:** distinguish failed answers from completed answers. Compare brands
  over the same completed answer set, with the denominator visible.
- **Agent unavailable:** the saved view and evidence remain usable directly.
- **Invalid composition:** preserve the last usable view and identify the unsupported block
  or scope value, with a way to revise it.
- **Concurrent changes:** show the revision conflict and preserve both intents for resolution.
- **New goal during inspection:** make the scope change explicit and give a clear route back.
- **Brand switch during loading or inspection:** do not display the previous brand's answers
  or apply a pending agent update to the newly selected brand.

## Mobbin research, round one

These are observations from the returned screen images. Their application to Koi is a
design hypothesis, not evidence that the products implement Koi's agentic architecture.

| Reference | Observed pattern | Hypothesis for Koi |
| --- | --- | --- |
| [Notion view creation](https://mobbin.com/screens/2f7bbc1f-a8a0-4fd4-ba5f-aa6c5ef86ceb) | A named database view has several available representations and an “Or describe a view…” field | A goal can create a named view over shared evidence; the person can still edit it directly |
| [Notion view settings](https://mobbin.com/screens/8859ba39-fc79-499d-a756-0085c2bf502f) | Layout choices and “Open pages in” behavior are configured locally | Presentation preferences and evidence-opening behavior can be separate from the data |
| [Fey overview](https://mobbin.com/screens/53357e14-9365-46af-8fc9-cd1347d0fca5) | A floating dock leaves most of the screen to a chart, recap, and contextual panel | Test a navigation island without assuming the icon inventory or three-column layout |
| [Fey contextual search](https://mobbin.com/screens/957fd895-354c-4910-acca-d7e3d0a13fd8) | A search surface occupies the right-hand context area while the overview remains visible | A focused task can temporarily occupy context without replacing the whole workspace |
| [ChatGPT research result](https://mobbin.com/screens/1d43b453-a7b4-438f-8038-f9d5c16c903e) | A readable research result sits beside activity and source access | Keep the result primary and make its provenance inspectable on demand |

The ChatGPT search returned research screens rather than the requested editable canvas.
It supports the provenance observation only. These references do not validate agent-driven
workspace composition or settle the navigation decision.

Next research should begin with brand switching and home goal context, then follow the first
analytical question once chosen. Avoid collecting whole dashboards
before knowing which relationship we want the first section to explain.

## What the user should evaluate in the first mockup

1. Can you immediately identify whether Home is showing Deja or Koi?
2. Can you understand the active goal and evidence scope without opening an agent chat?
3. Can you switch brands and return to each brand's own context?
4. Can an agent update Deja's Home without changing Koi's Home?
5. Can you distinguish brand selection, goal editing, and section navigation?
6. On mobile and by keyboard, can you reach and use the brand and goal controls?

Deja's audience and use case are now grounded in its repository. Next decision: whether to
use the proposed zsh-autosuggestions alternative goal for Deja, and which goal to use for Koi.


## October 13 Home visual revision

The founder rejected the large workspace, goal, tracking-scope header and the design-study
footer. The new Home starts directly with evidence, using Fey's compact overview
composition: chart and ranking left, recap and answer feed right, and a floating dock.
Brand selection lives in a slim greeting; period controls live below the chart.
Goal-driven composition remains the product direction, but goal authoring is removed
from this screen for this iteration. The dock remains an exploration choice.
