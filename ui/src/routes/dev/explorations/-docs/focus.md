# Exploration 05: Koi Focus

Open `/dev/explorations/focus/overview` on the Vite server at `http://localhost:5173`.
The fifth direction is also available in the exploration gallery and preview picker.

This blends Fey's quiet canvas with the linked context from Koi's existing Trade Republic
overview. A single selection connects the summary, multi-brand chart, competitor ranking,
and answer inspector. The context column starts with engine presence and the lowest-presence
buyer question; selecting a question turns it into evidence. The agent brief stays available
alongside that evidence.

The existing mock provider supplies the same deterministic data, filters, collection states,
prompt and competitor management, citations, and Markdown brief as the other explorations.
There are no new dependencies or production route changes. A sample scan returns to the
overall view, and navigation to another section resets the overview selection.

## Visual references

These screens were examined through Mobbin MCP. The implementation recreates relevant
interaction and layout patterns with existing Koi components; it does not embed screenshots.

- [Fey portfolio](https://mobbin.com/screens/439de82b-140c-4673-b36c-1e8f61b015f1):
  restrained chart chrome, compact watchlist rows, and a floating navigation dock.
- [Fey overview](https://mobbin.com/screens/9be7efbe-d004-4a8a-b31d-f14c49c73664):
  a signal area next to an editorial/context column on a dark canvas.
- [Linear inbox detail](https://mobbin.com/screens/beb9d6b3-ec34-46d7-9332-320fcb32a338):
  selection in a list reveals context alongside it rather than navigating away.
- [Coinbase portfolio](https://mobbin.com/screens/b6cc1e50-d9d1-4fde-8bde-be39ee423912):
  aligned comparison rows that keep names and numeric values easy to scan.

Mobbin searches for Trade Republic returned other products. Its influence comes from
the existing Koi Trade Republic exploration: shared prompt selection, comparison chart,
and answer inspector. Earlier chat history was not available; repository source and
commits supplied the implementation context.

## Manual checks

1. Open the gallery. Confirm the fifth entry and switch between all five directions.
2. On the Focus overview, compare the simplicity of the main signal with the context column.
   Decide whether the chart, competitive ranking, and agent brief have the right emphasis.
3. Select the lowest-presence question or a question below the ranking. Check that the
   headline rates, chart scope, ranking, and inspector all follow the same question.
4. Change the inspector's engine. Read the outcome, answer, date, and sources. Open
   “Inspect answer”, then close the dialog with Escape and check that focus returns.
5. Use “Back to all prompts”. Confirm that the overall metrics and context return.
6. Change the 7/30/90-day window and global engine filter. Check that the summary, engine
   rates, ranking, and question list update. Hover the chart, then inspect a date by keyboard.
7. Open all prompts, competitors, mentions, citations, and the agent brief using the dock
   and its More menu. Try a local edit, an answer inspection, and a Markdown download.
8. Use Preview controls to switch to Empty and Loading. From Empty, run a sample scan
   and check the populated state returns. Confirm no stale inspector is shown.
9. At 390px and 320px widths, check wrapping and horizontal overflow. Select a question:
   the context should scroll into view above the ranking and question list. Check the dock
   does not prevent reaching the last row or footer. Also try 200% browser zoom.
10. Tab through filters, scope selection, chart date slider, context actions, questions,
    and navigation. Check visible focus, descriptive labels, and Enter/Space activation.

At the user's request, further browser and interaction testing is left to them. Before
that request, the desktop overview was opened once and a TypeScript check passed.
