# Contributing to Koi

Thanks for helping. Koi is a Go API with an embedded React UI, backed by PostgreSQL. Bug reports,
fixes, and small focused improvements are the easiest to review. Open an issue before starting a
large change, so we can agree on the approach first.

Report security problems privately, as described in [SECURITY.md](SECURITY.md), not in an issue.

## Set up

You need Go (the version in `go.mod`), Node 24 with pnpm (the version in `ui/package.json`), and
Docker for PostgreSQL.

```bash
cp .env.example .env
pnpm --dir ui install    # also installs the pre-commit hook
make dev                 # PostgreSQL, API on :8080, UI with HMR on http://localhost:5173
make seed-demo           # optional: a demo account with sample data
```

No external service is required for development. Live collection needs a
[Cloro](https://cloro.dev) key; OpenRouter, Logo.dev, Resend, Stripe and PostHog are optional.

## Before you open a pull request

```bash
make format              # Biome and gofmt fixes
make test                # Go tests, including PostgreSQL integration tests
pnpm --dir ui check && pnpm --dir ui check-types && pnpm --dir ui test
```

The pre-commit hook runs the same checks without rewriting files. CI runs them again, with
`go vet`, `govulncheck` and a Docker image build.

- Schema changes need a new migration: `make migration name=add_something`. Never edit a migration
  that has been released.
- Add an endpoint, MCP tool or UI API wrapper next to the closest existing one, and keep the
  OpenAPI spec (`docs/openapi.yaml`) and generated client (`make openapi`) in sync.
- Pin exact dependency versions. The UI uses pnpm only.
- [AGENTS.md](AGENTS.md) and [docs/architecture.md](docs/architecture.md) describe the layering,
  API conventions and readability rules. Coding agents read the same file.

## Pull request titles

PRs are squash-merged, and the title becomes the commit message on `main`. Release Please reads
those messages to choose the next version and write the changelog, so titles follow
[Conventional Commits](https://www.conventionalcommits.org/):

```
feat: show citation gaps per engine
fix(billing): keep the trial end after a plan change
docs: explain self-hosted backups
```

`feat` is a minor release and `fix` is a patch. Add `!` after the type for a breaking change. A
check fails when the title does not match.

## License

By contributing, you agree that your contribution is licensed under the [MIT License](LICENSE).
