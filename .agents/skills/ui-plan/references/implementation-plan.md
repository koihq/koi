# Implementation plan format

Use this structure for the saved handoff. Scale each section to the change: a small
component can be concise, while a data-heavy page needs explicit contracts and states.
Remove inapplicable detail with a short reason. Replace prompts with actual decisions;
this is not a checklist to leave for the executor to fill in.

## Title and baseline

- **Status:** Ready or Draft. State exactly what prevents readiness.
- **Target:** Page route, flow, or component and its host surfaces.
- **Baseline:** Current commit, relevant uncommitted work, and inspected entry points.
- **Scope:** The user-visible outcome and explicit boundaries.
- **Constraints:** Current user instructions, selected visual direction and scoped exceptions.

## User job and current behavior

Name the user, entry trigger, information or decision needed, and next action. Explain
the current behavior and the intended change with a concrete before/after scenario.
Distinguish production functionality from exploration fixtures and proposed backend work.

## Research report

Record actual search coverage, including gaps:

| Lane | Mobbin tool and query | Inspected results or no match | Follow-up or limitation |
| --- | --- | --- | --- |

Cover chosen project references, the same user job, and an analogy from another sector.
Then explain the useful examples:

| Reference link | Observed in the image | Borrow, adapt, or reject | Application and tradeoff |
| --- | --- | --- | --- |

Use canonical Mobbin links. Separate observations from hypotheses and chosen reproduction
values. State where the chosen direction follows Fey, Visitors, or another project
reference, and where another pattern better serves this task. Include consequential
rejected options, rather than every search result.

## Principles and selected direction

For the view and each materially changed component, connect the user need, behavior,
visual expression, fit in the app, and tradeoff. Give each principle an observable
consequence, so the executor and reviewer can tell whether it survived implementation.
Choose a single direction and explain the meaningful alternatives you rejected.

## Skill preflight

| Skill and relevant rule | Decision checked | Adjustment or rationale | Verification still needed |
| --- | --- | --- | --- |

Record resolved conflicts, scoped exceptions, and missing coverage. A rule applied to a
plan is not proof of rendered contrast, focus behavior, or visual fidelity.

## View and component specification

Describe the actual target, not merely its inspiration. Include a labeled wireframe or
component tree when spatial relationships would otherwise remain ambiguous.

Specify the applicable details:

- **Composition:** Regions in reading order, their content, primary relationship,
  navigation context, primary and secondary actions, alignment and scroll ownership.
- **Geometry:** Container/column constraints, spacing, control and hit-area sizes,
  responsive breakpoints, wrapping and overflow. Use existing tokens or explicit values
  with units and conditions. Specify how the layout changes, not just “responsive.”
- **Visual roles:** Type roles and weights, semantic text/surface/border tokens, icons,
  radii and selected/focus/error treatments in supported themes. Name the source of any
  scoped values. Preserve the app's shared system unless a change is expressly in scope.
- **Components:** Existing primitives and blocks to reuse, exact exports/imports, new
  component responsibilities, typed props, callbacks, and the parent that owns state.
- **Copy:** Exact visible strings, accessible names, locale keys and catalogue files.
- **Interaction:** Activation, dismissal, selection, cancellation, focus movement and
  restoration, keyboard/touch equivalents, and whether state is preserved on return.
- **Motion:** None, or its purpose plus exact properties, duration, easing token,
  interruption/exit behavior, reduced-motion and pointer/keyboard treatment.

For a component-only plan, specify its integration with the host and the consumers
affected by a shared change. Do not invent new route or API work if none is needed.

## State and failure behavior

Use actual states and transitions, including applicable edge cases:

| State or trigger | Visible content and exact feedback | Enabled actions | State/focus/data outcome |
| --- | --- | --- | --- |

Cover initial/loading, success, true empty versus filtered empty, recoverable error,
pending mutations, disabled/selected states, long content, and narrow widths where
relevant. State what happens on retry, rapid repeated interaction, dismissal, navigation,
or brand/filter changes during a request. Keep feedback and a recovery path available.

## Data and code contracts

For each data-dependent behavior, specify the real data path:

`route/search params → owning state or hook → ApiProvider wrapper → request/response → UI`

Include verified file paths and symbols, a concrete TypeScript type or usage example,
query keys and their scope, pagination, mutation invalidation, and persistence/reset
behavior when relevant. Identify any required backend/OpenAPI changes explicitly;
do not assume an endpoint exists. Preserve the project's optional-service and auth rules.
Use deterministic, clearly identified fixtures for an exploration when live data is
outside scope. Name the fixture source and required edge examples.

Explain compatibility choices and why an existing abstraction or dependency is sufficient.
If a new dependency is necessary, settle its verified exact version and purpose in the
plan. Do not leave library selection to the executor.

## Ordered implementation tasks

| Order | File and symbol | Concrete edit | Depends on | Observable result |
| --- | --- | --- | --- | --- |

Use repository-relative paths. Distinguish files to create from files to change. Cite
current symbols or short excerpts where they disambiguate the edit. Include target code
for contracts or interactions whose shape matters; do not paste an entire speculative
implementation. Name generated files and how to regenerate them instead of hand-editing.
Keep tasks within the requested scope and identify shared consumers to recheck.

## Acceptance and verification

Tie checks to the principles and state table, using concrete fixtures and outcomes.
Examples of the required specificity:

- “At 320px, the full question wraps and the close action remains reachable without
  horizontal page scrolling,” rather than “works on mobile.”
- “Open evidence from the second answer, press Escape, and confirm focus returns to
  that answer while the selected brand and period remain unchanged,” rather than
  “keyboard support.”
- “Change brand while a request is pending; no result from the previous brand appears
  under the new brand,” rather than “handles races.”

List exact mechanical commands appropriate to the edits and their expected results.
Read the current project scripts rather than assuming a build proves type correctness.
For Koi, use the Vite surface at `http://localhost:5173` for UI checks. Separate automated
behavior checks from visual/interaction checks; follow any instruction assigning browser
testing to the user. Add focused tests only when they verify meaningful behavior.

Record checks already run, checks still required, and who performs restricted/manual
checks. Include supported themes, 320px reflow, 200% zoom, keyboard, touch and reduced
motion as relevant, without claiming unperformed runtime verification.

## Executor handoff

State the file-edit boundary, required skills/conventions and completion criteria. Inline
all necessary decisions; do not refer to “what we discussed” or require the executor to
infer the UI from reference links. If an inspected contract has materially changed or a
required resource is missing, report the mismatch before continuing dependent work.
Separate optional follow-up ideas from the work required for this plan. A Ready plan
has no unresolved question that changes its layout, interaction, data contract or scope.
