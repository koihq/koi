# MVP exploration walkthrough

Open `http://localhost:5173/dev/explorations/home-context` during `make dev`.
The existing overview is now Visibility. The four dock destinations are Visibility,
Mentions, Prompts, and Competitors; Settings is in the account pill.

This is the exploration implementation. Lists and settings change locally for the
current session, including when switching brands. Reloading resets them. Log out
shows a local preview state and does not change the real cookie session. Collection
and drafting use fixed examples, not external services or a simulated background job.
Historical answers keep their original names and question text after tracking edits.

## Visual comparison

Compare with commit `0216998`, using the same viewport, brand, period, and browser.

1. Visibility retains its asymmetric columns, chart, recap, fonts, brand pills,
   and answer tiles. Only the removed search/brief, account control, navigation,
   and small Mentions link should differ in the normal overview.
2. Open an answer. Its glass panel remains 430px wide at desktop with 20px outer
   insets. Open a Mentions answer and compare the same panel.
3. Check the new list surfaces and 16px headings against the overview's material
   and type hierarchy. There is no production sidebar or generic table chrome.
4. Check 320px, 390px, desktop, and 200% zoom. Long questions/URLs wrap; the dock
   leaves the final action reachable. Menus and forms retain the same typeface.
5. Use Tab, Enter, arrows within menus, and Escape. Closing an answer or editor
   returns focus to its trigger (or the heading if the trigger was removed).

Browser/visual testing remains assigned to the user. Source comparison established
that 18 retained overview CSS blocks have unchanged declarations; that does not
establish pixel-level browser fidelity or rendered contrast.

## Behavior walkthrough

- Switch all six brands; choose different 7D/30D/90D periods, navigate away and back.
  Each brand keeps its period. Browser Back restores the previous destination.
- Open Account → Settings. Save a valid domain-or-GitHub identity and country;
  navigate away and back. Koi has no invented default domain: provide an identity
  when testing its settings save, or use another populated brand.
- In Prompts, add, edit, search, filter, confirm a drafted item, and delete.
  Try blank text. Cancel and re-open a form. Historical answer text stays unchanged.
- In Competitors, use manual entry and the small sample-directory lookup. Open
  More details for GitHub/aliases/URLs. Try a duplicate identity and an invalid URL.
- Resolve the name-only `Unresolved alternative` suggestion by adding identity,
  then confirm it. Review completion must leave unresolved observed names unconfirmed.
- Open Mentions, choose a historical collection, filter outcomes/engines, open details,
  close, and clear filters. All answers retain sample provenance; baseline samples
  intentionally have no collected sources.
- Choose Log out, then Return to preview. No real session is terminated.

## Explicit state examples

Append the parameters below to `/dev/explorations/home-context`. They are development
fixtures, not controls in the product UI. Changing `state` resets the sample session.

| Parameters | Inspect |
| --- | --- |
| `?view=prompts&state=empty` | Empty list, manual add, review readiness |
| `?view=competitors&state=review` | Review instruction; observed suggestions excluded from bulk confirmation |
| `?view=mentions&state=loading` | Stable shell and loading content; intentionally static fixture |
| `?view=mentions&state=error` | Inline error and functioning Retry |
| `?view=mentions&state=running` | Pending answers and completed/total status |
| `?view=mentions&state=partial` | Failed/stopped collection; pending slots read Not collected |
| `?view=prompts&state=save-error` | First mutation fails; input survives and retry succeeds |
| `?view=prompts&state=draft-failed` | Draft retry with a fixed sample result |
| `?view=prompts&state=draft-empty` | No-new-suggestions feedback |
| `?view=prompts&state=draft-queued` | Static queued state |
| `?view=prompts&state=draft-running` | Static drafting state |
| `?view=competitors&state=optional-off` | Manual controls usable without optional drafting/lookup |

Fixed empty/review scenarios stay collection-free after review; their ready message
marks the end of setup. They do not synthesize a new collection when a button is pressed.
Drafting supplies one deterministic additional prompt/competitor per brand; repeating
it reports no new suggestions. Management supports pagination at 10/20/50 items;
add enough prompts to inspect the page boundary. Mentions paginates at 40.

## Remaining production integration

The current production APIs provide management, drafting, tracking readiness, run
history, and result detail. Integrating the design with those routes remains separate.
Period aggregation, complete engine/outcome filtering across server pages, and a
production multi-brand shell need explicit contracts before live parity can be claimed.

Validation commands: `make format`, `pnpm --dir ui check`,
`pnpm --dir ui check-types`, `pnpm --dir ui test`, `make ui-build`.
