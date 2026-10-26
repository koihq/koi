---
name: codebase-patterns
description: Koi Go and React layering — chi handlers, domain commanders/queriers, GORM Store, ApiProvider factories, TanStack Router, and shadcn. Use when adding HTTP endpoints, domain entities, repositories, React routes, API wrappers, or UI features.
---

# Codebase patterns

Read the section that matches the work. Do not invent a parallel layout.

- **Go API, domain, GORM, chi, Store, handlers, commanders, queriers** → [backend.md](backend.md)
- **React routes, ApiProvider, factories, TanStack Router, shadcn, entity hooks** → [frontend.md](frontend.md)
- **Both** (new entity end to end) → backend first, then frontend.

Cookie session auth exists (`Authorizer` on `/me`, `_authenticated` layout). Do not add Keycloak or JWT.
