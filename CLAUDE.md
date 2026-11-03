@AGENTS.md

## Claude Code

- Project skills live in `.agents/skills` and are exposed to Claude Code through the
  `.claude/skills` symlink. Edit them in `.agents/skills`; `agents/openai.yaml` files are
  Codex metadata and are ignored here.
- `ui-plan` research uses the Mobbin MCP tools (`search_screens`, `search_flows`,
  `search_sections`), platform `web` by default.
- Inspect rendered UI on `http://localhost:5173` during `make dev`. Stageshow's
  `scripts/layout.mjs`, `film.mjs` and `jumps.mjs` can validate real app pages; write their
  screenshots and reports outside the repository.
