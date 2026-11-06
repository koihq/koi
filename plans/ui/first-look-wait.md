# First Look wait

- **Status:** Implemented, uncommitted, on `feat/add-a-judger-for-mentions`.
- **Target:** `ui/src/routes/_authenticated/_app/-components/first-look.tsx`, the states between
  the end of setup and the first answer. Koi Cloud only (`billing.firstLook`).
- **Baseline:** `73587a3` plus the branch's uncommitted mention-judge work.
- **Scope:** UI and copy. No backend or OpenAPI change.

## User job and problem

A founder has just finished setup. They need to know that something is happening, and whether
Koi understood their product, before the first answer arrives. The page showed one still title,
then a still summary, with the trial offer as the brightest object.

Measured: one local draft took 29.6s (0.7s queued), then 20 answers (5 questions × 4 engines).
One sample only, and the split between reading and drafting is not recorded. The wait inside
`suggest` is one non-streaming call, so nothing can honestly advance during it.

## Research

| Lane | Query (`search_screens` unless noted, web) | Result |
| --- | --- | --- |
| Project references | Fey loading state; Family loading/syncing; Trade Republic loading | Fey: logo splash only. Family and Trade Republic: no match |
| Same job | Onboarding analyzing website with checklist; `search_flows` AI scans company website | Arcade, Clay, Unify. Most checklist hits were setup to-do lists, not waits |
| Same job | Deployment/build progress; AI deep research in progress | Searchable, Elicit, Copilot, Mistral, Customer.io, Mercor |
| Another sector | Flight/hotel search loading with progress | Weak: returned AirOps, Artlist and generic spinners. Not repeated; a gap |

| Reference | Observed | Decision |
| --- | --- | --- |
| [Elicit](https://mobbin.com/screens/232b3136-ccbb-42e7-a45c-412ea436f330) | Finished steps state what they found ("50 papers found") | Borrow: the core idea |
| [Unify](https://mobbin.com/flows/4664da9b-be47-4971-8ca8-f5069107cc66) | Step list that gives way to results | Borrow the handover |
| [AirOps](https://mobbin.com/screens/0a9b3407-3578-4f6b-b18b-b6d9ae306fc4) | Bar, step list and rotating tips, in the same market | Adapt the list. Reject the bar and tips: no real percentage, and tips compete with real findings |
| [Searchable](https://mobbin.com/screens/7fdc6f75-0dca-4110-a281-610908e36c91) | Says the report runs without you | Borrow leaving as a first-class option |
| [Arcade](https://mobbin.com/screens/b217a5b4-7c31-4a53-a589-2ed736116acd) | Skeleton of the result | Reject: 30s or more of gray bars hides data Koi already has |
| [Artlist](https://mobbin.com/screens/d171b06f-7b2a-419b-94b3-744f6ac476d2) | Percentage | Reject: it would be invented |

Static images do not prove motion or timing behavior.

## Principles

- **Only real signals advance a step.** `summary` finishes reading; `state: succeeded` finishes
  drafting. No timers or percentages drive progress. Tradeoff: nothing advances during `suggest`.
- **The wait pays for itself.** A finished step keeps what it found, in foreground text, while
  process labels stay muted. The eye lands on findings, in keeping with Family's restraint.
- **One stable heading.** Steps carry the state, so the title does not swap three times.
- **Leaving is safe, and the page says so.** The worker runs without the page and the first check
  emails its result when at least one answer came back.
- **The offer waits for value.** The price column appears with the first verdict. Tradeoff: the
  layout goes from one centered column to two at that moment, along with the whole content swap.

Rejected: a skeleton of the result (above), and a composed sentence in Fey's style ("Koi is X for
Y, up against Z"), because generated summaries are not grammatical fragments.

## Specification

Steps come from `firstLookWait(draft)` in `lib/first-look.ts`:

| Draft | Read | Draft step | Ask |
| --- | --- | --- | --- |
| queued, or running without `summary.category` | active | pending | pending |
| running with `summary.category` | done | active | pending |
| succeeded | done | done | active |
| missing or failed | hidden | hidden | active |

- **Layout:** `max-w-3xl` centered column holding the progress indicator and panel. Rows use
  `grid-cols-[1.25rem_minmax(0,1fr)]`. Details use the existing 8rem definition-list column.
- **Marks:** `Tick02Icon` done, 14px ring active, 6px dot pending. Decorative, so `aria-hidden`.
- **Copy:** `firstLook.wait*` in `locales/en/home.json`.
- **Slow:** after `SLOW_WAIT_MS` (60s) from the run's `createdAt`, else the draft's, the
  description changes to `waitSlow`. A new phase restarts the clock.
- **Motion, state indication:** ring spin; reduced motion uses opacity pulse. New details expand
  with `grid-template-rows` over 200ms `ease-out`, so rows below move with them; none under
  reduced motion.
- **Accessibility:** one `h1`; a stable `sr-only` `output` announces the current step.

Failure paths are unchanged: a failed draft or start goes to `TrackingStart`; a finished run with
no verdict shows `firstLook.failed`.

## Verification

Done, by driving the page on a Vite server with a mocked API: each wait state at 1440px, sampled
at 390 and 320px and in light, with no horizontal overflow; then the live reading → verdict
sequence on real polling at 1440px dark. The "Up against" label and the column alignment came
after the narrow pass, so 390 and 320px were not re-rendered with them. Unit tests cover
`firstLookWait` and `displaySite`.

Not done: a real run with OpenRouter and Cloro, the narrow re-render above, keyboard and
screen-reader walks, 200% zoom, measured contrast, the reduced-motion rendering, and how long
`Asking` stays on screen before the first answer. Stageshow's `film` and `jumps` were not run.
Repeat the timing on more runs before putting a duration in the copy.

## Follow-up

Streaming the `suggest` call's web searches would let the draft step show real sites as it reads
them. That needs backend work and is out of scope here.
