---
name: koi-setup
description: Set up or review a brand, prompts, and competitors in Koi through its MCP server. Use when asked to configure Koi monitoring inputs; not for reporting on collected visibility.
---

# Koi setup

Use the connected Koi MCP server. If it is unavailable, explain how to connect to the user's Koi instance.

1. Find the brand with `brands_list` or, when the user requests a new one, use `brands_countries` and `brands_create`. Resolve ambiguity before changing a brand. Pass its explicit `brandId` to collection tools; browser active-brand state does not apply to MCP.
2. Inspect existing inputs with `prompts_list` and `competitors_list`, following pagination. Use `competitors_search` for suggestions when available. Manual prompt and competitor tools remain usable without optional drafting services.
3. Apply only the additions and edits the user requested, using the corresponding create, update, or delete tools. To remove a prompt with saved answers (`hasResults`), stop it with `prompts_update` and `inactive: true`; only prompts with no answers can be deleted. `prompts_confirm`, `prompts_confirm_all`, `competitors_confirm`, and `competitors_confirm_all` finish review and can enable tracking; call them when the user asks to finish or enable monitoring.
4. Re-read affected items or lists and report what changed and what still needs review. Use `tracking_get` to report tracking state when relevant.

If the user requests AI drafting, `generations_create` starts a job: supply a stable `idempotencyKey` and poll `generations_get`. If drafting is unavailable, continue with manual inputs. Do not use billing tools to complete setup.
