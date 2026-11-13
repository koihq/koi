# Architecture

Koi runs one Go binary with a React SPA. HTTP handlers adapt transport types,
domain commanders and queriers implement use cases, and GORM repositories own
scoped queries and persistence. The app package composes those dependencies and
runs River workers.

Product changes run in the domain: a commander takes the brand lock inside one
transaction, calls entity methods and writes through thin repositories. HTTP and
MCP share the same commanders and queriers.

## Identity and isolation

The `koi_session` cookie identifies a server-side SCS session. Each authenticated
request resolves the user's membership once and places `UserID`, `OrganizationID`
and `Role` in the context. An organization can have several members and brands;
a user belongs to exactly one organization, enforced by `memberships.user_id`.

Brand repositories filter directly by the verified organization. Child repositories
check their brand's organization. A missing tenant scope is rejected. Reads and
mutations by UUID remain scoped, including when a caller knows another tenant's ID.
An inaccessible resource returns 404.

Authentication and billing middleware live under `api/middlewares`. Cookie
creation and renewal stay in the auth handler. Worker and webhook operations have
explicit internal repository entry points; they do not depend on a browser session.

## HTTP contract

Public API names are camelCase. Persisted domain structs keep their own database
and snapshot serialization; response DTOs explicitly select public fields.

Examples, relative to `/api/v1`:

```http
GET /prompts?brandId=<uuid>&page=1&pageSize=20
POST /prompts
Content-Type: application/json

{"brandId":"<uuid>","text":"Which tools should a founder consider?"}
```

```http
GET /prompts/<prompt-uuid>
PATCH /prompts/<prompt-uuid>
Content-Type: application/json

{"text":"Which analytics tools should a founder consider?"}
```

Collections require `brandId`; creation and collection commands carry it in the
body. Details and mutations identify the resource by its own UUID. The server
resolves its parent. The same rules apply to competitors and generations. Runs use
`/runs?brandId=...`, `/runs/{runID}` and `/runs/{runID}/results`.

`/prompt-history/{promptID}/runs` and `/prompt-history/{promptID}/visibility` read saved
results and keep a required `brandId` query parameter, because results have no foreign key to
a live prompt.

A prompt with saved results can only be stopped (`inactive: true`), so its history stays
reachable and resuming continues the same series. `DELETE /prompts/{id}` returns
`resource_conflict` for such a prompt; `hasResults` tells clients which action applies.

`POST /tracking/start` accepts `{"brandId":"..."}`. It confirms the brand's manual and
drafted rows and starts daily collection, once per brand; repeating it changes nothing.
It stays an explicit domain command rather than a generic CRUD update. After it, changes
to either list never block collection and apply from the next scheduled run.

`POST /generations` starts the brand's one draft: at most 5 prompts and every competitor
it can verify. A failed draft can run again until one succeeds or tracking starts; after
that it returns `initial_already_completed`.

`POST /brands/{id}/activate` stores a UI preference. It never changes the scope
of another request. Open forms and separate tabs keep their explicit destinations.
No aliases for the former `/brands/{brandID}/...` product paths are provided.

`GET /auth/me` returns `activeBrand: BrandRes | null`, resolved within the verified
organization. An absent, malformed or inaccessible session preference falls back
to the newest accessible brand (`created_at DESC, id DESC`); no brands means `null`.
This read does not write the fallback to the session. Login and signup return only
the account identity. `GET /brands?q=...` searches name, domain and GitHub repository
within the organization, retaining the normal paginated response and newest-first order.

Persisted lists use `Page[T]`, `page` starting at 1 and `pageSize` between 1 and 100.
Empty results contain `items: []` and all page metadata. Country and competitor
search remain bounded lookups.

Errors retain stable codes and camelCase field names. Responses expose
`X-Request-ID`; error bodies repeat it as `error.requestId`. SQL and provider
errors are never exposed as arbitrary internal messages.

Small `Get`, `List`, `Create`, `Update`, `Delete`, `Action`, `Command` and `Read`
HTTP adapters share transport handling and take typed functions and mappers. A
resource's request, response and mapper live in its handler file. Resource routes
name their identifier `{id}`; prompt history keeps `{promptID}` because the value
may refer to a deleted snapshot. Authentication, billing, OAuth and generation
creation handlers remain explicit. `ServerOptions` lists an instance's optional
capabilities, and `Billing` is set only on Koi Cloud.

## Database and background work

Goose migrations in `pkg/database/migrations` define the schema; `database.Migrate` applies
them at startup under a Postgres advisory lock, and River applies its own. The GORM structs must
match the migrations, which `TestMigrationsMatchModels` checks against `AutoMigrate`. Memberships have foreign
keys to users and organizations with cascading deletion. Brands and billing rows
reference organizations with restricted deletion. Brand-owned product rows and
run results keep their existing cascading foreign keys.

Historical result `PromptID` values intentionally have no FK to current prompts.
Stripe IDs are external references; River job IDs are independent of job retention.
Snapshots and bounded arrays remain JSONB. Organization IDs are not duplicated
across all child tables.

`Repository[T]` provides scoped `Create`, `Get`, `Update` and `Delete`, and
`ListRepository[T, F]` adds `List` with a typed filter and a stable order. A concrete
repository declares its scope, filter, order and writable columns in the constructor,
then adds only the queries specific to it. `Update` writes only the declared columns.
A scoped repository rejects a `Create` it cannot authorize, including one for a brand
of another organization.

Prompt, competitor and generation commands run under `withBrandLock`. Commands on an
existing child read it again under the lock through `withLockedChild`. Each one
validates through entity methods such as `Prompt.Update` and `Competitor.Confirm`,
then writes in the same transaction. A failure at any step
rolls back the whole command. `BillingRepository.ReservePromptCapacity` serializes
prompt activation across an organization on Koi Cloud; self-hosted instances have no
allowance.

Deleting an observed competitor saves a dismissal, so later answers do not suggest it
again.

Worker state machines (`visibility_attempts`, `generation_attempts`,
`visibility_schedule`) keep their transactions in `database`, because they advance
attempts and River jobs atomically. They take the same brand lock and share
`Competitor.ConflictsWith`. Enqueue and run creation commit or roll back together.

Visibility analytics selects cell data, run timestamp and brand snapshot in one
joined query that applies the repository organization scope, without a second run preload. Current competitor identities are still
applied at read time.

Every analytics report includes each confirmed competitor's ID, name, domain,
metric and series, even when the period has no completed answers. The summary adds
`hasConfirmedPrompts`, checked independently of period and engines and including
stopped prompts. Handlers do not load prompt pages to derive that state.

Workers consume the application `CollectedAnswer` and `ProviderError` contracts.
Provider adapters retain their protocol details and stable failure codes. Cloro is
required for live collection; OpenRouter drafting and Logo.dev remain optional.

### Workers and scheduling

River runs jobs from Postgres; queue discovery polls approximately every second with jitter.
`KOI_WORKER_CONCURRENCY` defaults to four workers per enabled generation/visibility queue.
A separate maintenance queue uses one worker and runs on startup, then every minute to
reconcile interrupted runs and dispatch visibility work.

Each brand gets one AI draft: at most 5 prompts and every competitor it can verify. After
that, add entries by hand or through your agent over MCP. "Start tracking" confirms the lists
once and starts collection; later changes never pause it and apply from the next check. Each
brand runs every 24 hours
from its persisted UTC anchor; missed slots are skipped rather than collected in a burst.
A visibility job processes cells for approximately 40 seconds, persists Cloro task IDs and
answers, then snoozes for ten seconds if work remains. Actual resumption also depends on
River's scheduler and available workers. This is a bounded loop inside a queued job.

Visibility cells show matched competitor names and supporting evidence. Matching uses current
names, aliases, official domains, and GitHub repositories, so edits also update historical links
without changing saved answers or brand outcomes. Live Cloro entities and leading Markdown
links to product homepages or repositories provide conservative discovery: two distinct cells
create an unconfirmed observed suggestion; retries do not count twice. Names found only in
ordinary prose may be missed. Name-only suggestions need a primary domain or repository before
individual confirmation. Starting tracking excludes observed suggestions, and deleting one suppresses
its rediscovery for that brand without interrupting tracking.

Generation runs when a job is queued. `KOI_PROVIDER_TIMEOUT` defaults to three minutes per
provider call. Generation makes two sequential calls, so its job timeout is twice that value
plus 30 seconds (6m30s by default). Retry delays after failures are five and twenty seconds;
there are at most three attempts. Graceful shutdown allows jobs 60 seconds before cancellation.

## UI state and generated clients

`main.tsx` creates one API instance and QueryClient, shared with router loaders and
React providers. OpenAPI remains the source of HTTP types. `make openapi` uses the
pinned generator and emits request objects with `useSingleRequestParameter=true`.
Do not edit generated files manually.

Thin resource wrappers expose calls such as:

```ts
api.prompts.list({ brandId, page: 1, pageSize: 20 });
api.prompts.create({ brandId, text });
api.prompts.update(id, { text });
```

The generated client's response middleware normalizes errors once. Shared
`queryOptions` keep loader and hook requests consistent. Organization-level lists
and billing include organization identity in their cache keys; product keys include
the globally unique brand UUID. Account changes cancel requests and clear the cache.
Capabilities share one cached query. Query options forward the abort signal through
the API wrappers. The authenticated loader uses the resolved active brand without
loading the brand collection. The shared brand selector searches only while open
and fetches one page of 20 results at a time, with a 300 ms search debounce.
Analytics screens and prompt previews use the identities in their own reports;
they do not load competitor collections to complete those reports.

Feature-specific mutations share invalidation sets. Prompt list queries and batch
actions are local hooks; failed batch IDs remain selected for retry. Changing brand
remounts the route outlet so forms, selection and dialogs start in the new context.

## MCP and OAuth

The MCP Go SDK implements the resource server and client primitives, not an authorization server. Koi therefore implements the issuer-specific endpoints in `api`, while credential state and token lifecycle live in `domain` and GORM persistence lives in `database`. We evaluated [Fosite](https://github.com/ory/fosite), [go-oauth2](https://github.com/go-oauth2/oauth2), and [Hydra](https://www.ory.com/hydra). Fosite would replace the token protocol but require its request/session storage adapters; Hydra would add a separate service to every self-hosted deployment; go-oauth2 does not clearly document the MCP PKCE/CIMD requirements. The current narrow issuer avoids those additional adapters and services. Revisit this decision before broad public deployment, after interoperability and security tests against real clients.
