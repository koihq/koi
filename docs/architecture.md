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
Stopping or resuming does not reset the prompt review. Editing the text or deleting does.

`POST /prompts/confirm` and `POST /competitors/confirm` accept `{"brandId":"..."}`.
They finish a review and can enable tracking, so they remain explicit domain
commands rather than generic CRUD updates.

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

GORM initializes the current schema with `AutoMigrate`. Memberships have foreign
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
then writes and invalidates the review in the same transaction. A failure at any step
rolls back the whole command. `BillingRepository.ReservePromptCapacity` serializes
prompt activation across an organization on Koi Cloud; self-hosted instances have no
allowance.

Editing a competitor's identity keeps its review, while adding or removing a reviewed
competitor invalidates it. Deleting an observed competitor saves a dismissal, so later
answers do not suggest it again.

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
