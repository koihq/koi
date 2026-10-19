---
name: ui-plan
description: Explore the design of a UI view or component through Mobbin research, design principles, and a precise implementation handoff. Use when the user asks to explore a view or component, compare design directions, or explicitly invokes ui-plan. Do not trigger for routine UI plans or implementation tasks.
---

# UI plan

Use this skill for a requested design exploration of a UI view or component. A page
counts as a view. A routine plan or implementation request does not by itself require
this exploration. Own the research and design judgment before handing implementation
to another model.
The executor may have less reasoning ability and no conversation history. Resolve the
design here; give it concrete edits and observable outcomes.

Follow this order: understand the target → research Mobbin → report the exploration →
define principles → check applicable skills → finalize the implementation plan.
Apply it to individual components as well as whole views. Scale depth to the exploration,
but keep research and reasoning in each exploration that uses this skill.

An exploration-only request produces a plan, without changing application source. If the user
also authorized implementation, complete this sequence before implementing; this skill
does not add a separate approval step or authorize changing models or spawning agents.

## 1. Understand the target and its constraints

Read the repository instructions, then [product-design](../product-design/SKILL.md)
and [codebase-patterns](../codebase-patterns/SKILL.md), including its
[frontend section](../codebase-patterns/frontend.md). Inspect the actual target,
its callers, shared components, existing tokens, installed dependencies, and nearby
examples before choosing file boundaries or implementation patterns.

Establish:

- Who uses this view or component, what triggers the visit or interaction, and the
  decision or action it must support. For a component, include its host view and consumers.
- What exists today, what changes, and what remains outside this task.
- Navigation context, data availability, important states, supported input methods,
  and the current preview and verification commands.
- The project's chosen visual references and which qualities each governs: composition,
  density, typography, surfaces, interaction, or another named quality.

For Koi, read relevant parts of `docs/agentic-workspace-discovery.md` and
`docs/fey-visual-system.md` when planning discovery surfaces. Inspect the applicable
Fey and Visitors explorations in `ui/src/routes/dev/explorations/`. Distinguish chosen
direction from abandoned variants, proposed behavior, and production conventions.
An exception authorized for an exploration applies to that scope; it is not a global
design-system change. Current user instructions take precedence over older documents.

Inspect the running target when available and permitted. Source inspection cannot
establish rendered fidelity. Record missing visual evidence instead of claiming it.
Clarify a material unknown early; continue research that does not depend on its answer.

## 2. Research Mobbin before choosing the design

Every exploration using this skill requires actual Mobbin research. Discover and use the available
Mobbin connector tools; a list of familiar products or remembered screenshots does not
count. Revisions of the same plan may reuse its inspected results when the UI problem
has not changed; identify that reuse and research any newly introduced pattern.

Search these lanes:

1. **Project references.** Look for the relevant view or component in chosen references
   such as Fey or Visitors. Record a missing match rather than inventing one.
2. **The same user job.** Find analogous views and components independent of brand or
   industry. For a page, investigate both overall composition and its decisive controls;
   for a component, investigate the component and enough host context to judge its fit.
3. **Another sector.** Search for the same interaction or information problem in a
   different field. For example, portfolio comparison can inform brand comparison,
   and an editor's inspector can inform inspecting answer evidence. Explain what transfers
   and where the analogy breaks. Do not adopt a pattern merely to include another sector.

Use `search_screens` for product screens and embedded controls, `search_flows` when
sequence matters, and `search_sections` for website sections. These are capability names;
discover their actual tool identifiers in the current environment. Default to `web` for
Koi. Use another platform only for a named transferable behavior. Write focused queries
describing visible elements and relationships, not adjective lists such as “clean UI.”
Keep task context and destination fields consistent according to the tool contract.

Inspect the returned images. Follow useful results with a narrower query when a decisive
layout, state, or interaction is still unclear. A small set of well-understood examples
is enough once the relevant choices are supported; more screenshots are not progress.

Record the actual queries, what was inspected, and useful negative results. Cite each
discussed reference with its canonical `mobbin_url`. Separate visible observations from
inferences: a static image does not prove keyboard behavior, motion, exact CSS values,
or usability. Label proposed measurements as implementation choices. If saving images
is useful, download the high-resolution `image_url`; expiring image URLs are not durable
citations. Display any tool-required usage notice exactly as returned.

If Mobbin is inaccessible, report the attempted access and what is missing. Continue
independent repository analysis and produce a clearly marked draft if useful, with
research still pending. Do not silently substitute another source or call it ready
for implementation. If searches succeed but yield weak matches, document that evidence
and the remaining design uncertainty; do not force a misleading reference.

## 3. Report the exploration, then define the principles

Report the exploration to the user before finalizing the plan, and preserve it in the
handoff. Show the search coverage, linked examples, what to borrow or adapt, what to
reject, and why. Explain what stays faithful to the project's references and what is
borrowed from elsewhere. A mood board without decisions is not a research report.

Use that evidence to define principles for the view and each materially changed
component. A principle must connect:

**User need → intended behavior → visual expression → fit in this app → tradeoff.**

Explain both how it works and why it should work that way. Name the information order,
primary relationship, action priority, disclosure, and feedback that serve the user.
Connect UI choices such as grouping, type emphasis, surface, density, and placement to
those needs. “Minimal,” “intuitive,” and “consistent” are not sufficient explanations.

For example: “The founder must inspect a citation without losing the brand comparison.
Keep the comparison visible and open evidence in a contextual inspector. Its title and
source establish provenance; metadata stays subordinate. This supports Koi's executive
and audit reading modes. On narrow screens the inspector takes the available width and
returns to the same comparison state when closed.” The principle explains the choice;
the final spec must still settle exact geometry, focus, navigation, and state behavior.

Evaluate plausible alternatives where they change the result, then choose one direction
and explain the rejected tradeoff. Preserve the reference's useful logic while adapting
its domain content and controls to Koi. Do not transplant finance vocabulary or controls
that do not serve the AEO task. References guide judgment; they do not bound the search.

## 4. Check the proposal against the other skills

Read and apply the owning skills rather than paraphrasing them from memory. Use their
rules as a design preflight; a proposal has not passed a rendered interface review.
Load supporting references only where they bear on an actual decision.

| Skill | Apply it to |
| --- | --- |
| [product-design](../product-design/SKILL.md) and [primitives](../product-design/primitives.md) | Koi's reader job, applicable visual direction, shared tokens and primitives, named anti-patterns |
| [codebase-patterns](../codebase-patterns/frontend.md) | Routes, component ownership, API wrappers, hooks, query keys and data flow; read `backend.md` too if the plan changes backend contracts |
| [better-accessibility](../better-accessibility/SKILL.md) | Semantics, names, focus, keyboard/touch paths and dynamic announcements |
| [better-layout](../better-layout/SKILL.md) | Grouping, reading order, responsive behavior and disclosure |
| [better-writing](../better-writing/SKILL.md) | Exact labels, explanations, errors, recovery and empty-state copy |
| [better-typography](../better-typography/SKILL.md) | Type roles, hierarchy, wrapping, truncation and zoom |
| [better-colors](../better-colors/SKILL.md) | Semantic roles, theme coverage and contrast verification |
| [better-ui](../better-ui/SKILL.md) | Control geometry, alignment, surfaces, icons and hit areas |
| [frontend-design](../frontend-design/SKILL.md) | A deliberate composition grounded in this brief; work within the selected project system |
| [ponytail](../ponytail/SKILL.md) | Reuse and the simplest implementation that fulfills the design |
| [vercel-react-best-practices](../vercel-react-best-practices/SKILL.md) | Applicable React rendering, data and bundle decisions; use SPA rules for Koi |

When planning motion, also read [emil-design-eng](../emil-design-eng/SKILL.md) and
[animate](../animate/SKILL.md). Use their decision rules to specify the plan without
starting implementation. Load the relevant TanStack Query, Router, or Table skill when
its behavior changes. Use other specialized skills only when the proposal needs them.

For an actual screen or flow review, use [better-interface](../better-interface/SKILL.md).
A user-requested branch, PR, or uncommitted UI diff review belongs to
[interface-review](../interface-review/SKILL.md). Keep those review workflows distinct
from this plan preflight; do not issue a runtime approval based on a proposed design.

Report a compact table: **skill/rule → decision checked → adjustment or rationale →
verification still needed**. Resolve conflicts explicitly against the current user
instructions and scoped project direction. Revise the proposal before finalizing it.
Name missing skills or unresolved conflicts; do not claim those checks passed.

## 5. Write the implementation handoff

Use [implementation-plan.md](references/implementation-plan.md) when finalizing.
Follow an existing plan location if the project has one; otherwise save to
`plans/ui/<descriptive-slug>.md`. Record the current commit and relevant uncommitted
baseline. Keep the handoff self-contained: the executor should not need this conversation
or access to Mobbin to know what to build. Links support decisions; they do not replace
the actual specification.

Settle the design and implementation choices: exact files and ownership, verified
symbols and imports to reuse, component props, data calls, state transitions, copy,
tokens, dimensions, breakpoints, and acceptance criteria. Include concrete type or call
examples for nontrivial contracts. Specify behavior for relevant loading, empty, error,
selected, disabled, overflow, narrow-width, keyboard and reduced-motion states. Explain
which states do not apply rather than inventing them.

Make ordered tasks small enough to execute directly. Each task names a file, the change,
any dependencies, and its observable result. Set numeric values or exact existing tokens;
do not leave “adjust spacing,” “make it feel like Fey,” “handle errors,” or multiple
unresolved design options for the executor. Add abstractions, dependencies, backend
changes, or compatibility work only where required by the actual scope.

Choose meaningful checks proportional to the change. Include exact project commands,
interaction scenarios, and visible outcomes; respect any user restriction on browser
testing and assign manual checks when needed. Distinguish checks already performed
from checks the executor or user still needs to run.

Before marking the plan **Ready**, walk through it as an executor with no prior context.
Could it build the wrong layout, guess an API, invent a token, or miss a recovery path
while still following the text? Close those gaps. Mark **Draft** if research, a material
decision, or a required contract remains unresolved. Do not leave product or design
judgment to the implementing model. If baseline contracts drift during implementation,
have it report the specific mismatch instead of silently redesigning the solution.
