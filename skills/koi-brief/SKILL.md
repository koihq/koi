---
name: koi-brief
description: Prepare an evidence-based Markdown AEO visibility brief for a brand using the Koi MCP server. Use when asked how a brand appears in AI answers or what its founder should improve.
---

# Koi brief

Use the connected Koi MCP server. If it is unavailable, explain how to connect to the user's Koi instance; do not invent visibility data.

1. Find the requested brand with `brands_list`. If several brands fit, ask which one. Use its explicit `brandId` for every brand-scoped call.
2. Read `tracking_get` and `analysis_summary`. Use the requested period and engines, or the 30-day default and all engines. If there are no completed results, say so; do not interpret missing data as a miss.
3. For material claims, inspect relevant runs with `runs_list` and `runs_results`, following pagination where needed. Use `prompt_history_runs` or `prompt_visibility` when a prompt's history explains a finding; `prompts_list` with `inactive: true` lists stopped prompts. Distinguish engine names from the collection provider.
4. Write a Markdown brief for the founder's existing agent: brand and period; visibility by engine; notable mentions, citations, missed opportunities, and recommended next steps. Attribute claims to the Koi results and include source URLs when present. Label recommendations as recommendations, not observed facts.

Do not create or change prompts, competitors, billing, or tracking as part of preparing a brief unless the user requests that work.
