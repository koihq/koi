# UI architecture

React 19 SPA under `ui/` (create with Vite, package manager **pnpm**) with a simplified API-layer architecture:

```
ui/src/
├── api/              # Entity types, factories, mappers, DI provider
├── services/         # Non-HTTP services (none yet)
├── context/          # React Context providers
├── routes/           # Presentation layer (UI, hooks, components)
├── components/       # Shared components (ui/, blocks/, custom/)
├── hooks/            # Generic hooks (useEntityList, useEntityMutations, useEntityForm)
├── utils/            # parseError, date, mappers
├── lib/              # Parsers, router context, utils
└── config/           # API base URL, i18n later
```

Package manager is **pnpm**. Do not add npm or yarn lockfiles.

Import alias is `@/*` → `src/*` (tsconfig `baseUrl` + `paths`). Do not use `#/` or add `package.json#imports`. shadcn `components.json` aliases stay `@/components`, `@/lib/utils`, `@/components/ui`.

## API layer (`src/api/`)

SDK/client instances are created **once** in `apiProvider.tsx` and passed into thin per-entity wrappers. Entity files never construct the HTTP client themselves.

- `featureName.api.ts` — Entity types, Create/Update types, ListParams, and `createFeatureNameApi(client)` wrapper that receives a pre-built client.
- `shared/types.ts` — `BaseEntity`, `ApiListResponse`, `BaseListParams`.
- `shared/mappers.ts` — Entity mappers + field utilities.
- `shared/apiClient.ts` — `createApiClient()` (base URL) and `callApi(fn)` error wrapper.
- `apiProvider.tsx` — instantiates clients, builds wrappers, exposes `ApiProvider` + `useApi()` for DI.

```typescript
// src/api/health.api.ts
export interface Health {
  status: string;
  db: string;
}

export function createHealthApi(client: ApiClient) {
  return {
    get: async (): Promise<Health> => {
      const response = await callApi(() => client.get<Health>("/health"));
      return response;
    },
  };
}
```

**Naming**: generic method names (`list`, `create`, `update`, `delete`, `getById`) without entity suffix. Read-only or singleton resources may expose `get` / `list` only.

To register a new entity:

1. Add `<entity>: ReturnType<typeof create<Entity>Api>` to `ApiContextValue`.
2. Inside the `useMemo` block, reuse an existing client or create a thin wrapper next to the others.
3. Add `<entity>: create<Entity>Api(client)` to the returned object.

```typescript
const { health } = useApi();
health.get();
```

## Services layer (`src/services/`)

Leave empty until a non-HTTP service exists. Do not add Keycloak or role-permission services in setup.

## Dependency injection (`src/api/apiProvider.tsx`)

`ApiProvider` creates one HTTP client via `createApiClient()`, instantiates wrappers inside a `useMemo`, and returns them via Context.

## Presentation layer (`src/routes/`)

TanStack Router file-based routing:

| Prefix | Purpose                            | Example                    |
| ------ | ---------------------------------- | -------------------------- |
| `_`    | Layout (non-URL segment)           | `_root.tsx`                |
| `$`    | Dynamic parameter                  | `$id.tsx` → `/route/:id`   |
| `-`    | Private folder or read-only module | `-components/`, `-hooks/`  |

Pathless `_authenticated` layout gates `/`. `beforeLoad` calls `context.api.auth.me()` (the same `createApi()` instance is passed into the router in `main.tsx`; hooks cannot run in `beforeLoad`) and redirects to `/sign-in` on failure. Public routes: `/sign-in`, `/sign-up`. Do not add Keycloak or JWT.

Full-CRUD feature structure:

```
routes/featureName/
├── index.tsx           # List screen and route composition (/featureName)
├── create/index.tsx    # Create form
├── read/$id.tsx        # Detail view
├── update/$id.tsx      # Edit form
├── -components/
├── -hooks/             # useFeatureList.ts, useCreateFeature.ts
├── -types/             # Zod schemas, form types
└── -constants/
```

Keep the page composition in the file-based route's `index.tsx`. Extract focused reusable pieces (dialogs, forms, row actions, etc.) into `-components/`; do not add a feature-management/page component there whose only job is to render the route's list screen. This keeps the route entry point as the screen owner without preventing smaller components from being colocated.

Entity list/mutation hooks wrap TanStack Query and call `useApi()`.

## Components

Visual language, allowed primitives, and named anti-patterns: `.agents/skills/product-design` (then [primitives.md](../product-design/primitives.md)).

```
components/ui/      shadcn primitives (add with the CLI, don’t fork)
components/blocks/  product compositions built only from ui/
routes/             screens that assemble blocks, not restyle primitives
```

Routes import `@/components/ui/*` and `@/components/blocks/*`. Do not change a primitive’s size or type from a route (`className="h-9 text-sm"` on `Input` is forbidden). Do not add `Card` to group metrics or metadata. `FieldGroup` gap groups form fields; do not insert `FieldSeparator` between them. Tailwind v4 for styling.

## Dev

`make dev` runs Vite on `:5173` and proxies `/health` and `/api` to the Go API on `:8080`. Browse `http://localhost:5173`. Keep `createApiClient()` on a relative/empty base URL so requests go through the proxy — do not point the browser at `:8080`.
