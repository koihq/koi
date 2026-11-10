## What and why

<!-- What changes, and the problem it solves. Link the issue: Closes #123 -->

## How to verify

<!-- Commands, screenshots, or steps a reviewer can follow. -->

## Checklist

- [ ] The title follows Conventional Commits (`feat:`, `fix:`, `docs:` …)
- [ ] `make test`, `pnpm --dir ui check`, `check-types` and `test` pass
- [ ] Schema changes come with a new migration
- [ ] OpenAPI spec and generated client are updated if the API changed
