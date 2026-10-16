# Home-context migration

The production workspace now shows one active brand. The domain model and HTTP resources remain brand scoped so the current account and ownership checks continue to work.

| Area | Existing application | Production result |
| --- | --- | --- |
| Design system | Inter, shadcn primitives, sidebar shell | Home-context palette, Inter type stack, and compact header; see [design-system.md](design-system.md) |
| Brand | Brand CRUD and settings | Existing form retained; save invalidates route data so the header updates |
| Prompts | Paginated CRUD, confirmation, optional drafting | Existing flows on Home below the signal; each live prompt links to detail |
| Competitors | Paginated CRUD, review, search, optional drafting | Existing route retained under the compact header |
| Visibility collection | Cloro runs and saved result cells | Existing collection and tracking state retained; saved answers and citations appear in prompt detail |
| Historical prompts | Run snapshots existed but had no prompt lookup | New paginated prompt-history and prompt-run endpoints; deleted prompts remain inspectable |
| Analytics | No aggregate trend API | Deterministic temporary series, rates, changes, engine scope, and competitor comparison |

The temporary analytics adapter is `ui/src/lib/mock-visibility.ts`. It uses only confirmed prompts and competitors, produces the same result across navigation, and responds to the selected time range and engines. These values have no storage or backend contract yet. Live run answers and citations are never sourced from the adapter.

New API reads:

- `GET /api/v1/brands/{brandID}/prompt-history?page&pageSize`
- `GET /api/v1/brands/{brandID}/prompt-history/{promptID}/runs?page&pageSize`

Both use the existing cookie session and brand ownership checks. The run response includes public run status and only that prompt's results. The OpenAPI spec and generated TypeScript client are aligned.

The old `/prompts` and `/mentions` URLs redirect to Home, while `/citations` and `/brief` placeholder pages were removed. The `/dev/explorations/home-context` study remains available as a reference.

The next data integration is an aggregate visibility read over saved collection results. It should replace the adapter without changing Home or prompt detail controls. Pending and errored collection cells must remain excluded from completed-answer rates, and comparison must use the same prompt, engine, and date scope as the primary rate.
