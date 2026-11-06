---
name: koi-setup
description: Set up or review a brand, prompts, and competitors in Koi through its MCP server, and run the first visibility check. Use when asked to set up Koi or configure its monitoring inputs; not for reporting on collected visibility.
---

# Koi setup

Use the connected Koi MCP server. If it is unavailable, explain how to connect to the user's Koi instance.

## First setup

When the user asks to set up Koi and has no brand yet, take them to their first real result.

1. Check `brands_list`. If there is no brand, read the current project to find its name, website or GitHub repository, and market. Ask only for what you cannot find. Use `brands_countries`, then `brands_create`. Pass the explicit `brandId` to later tools; browser active-brand state does not apply to MCP.
2. Start the draft with `generations_create`, using a stable `idempotencyKey`, and poll `generations_get`. If drafting is unavailable or fails, write prompts yourself with `prompts_create`.
3. Read `prompts_list` and `competitors_list`. Correct prompts that do not describe this product with `prompts_update`, using what you learned from the project. Koi Cloud's free check allows 5 prompts.
4. Call `tracking_start`. Then poll `runs_list` and `runs_results` for the run. Results fill in while the run is in progress; pending ones have no outcome.
5. As soon as answers arrive, tell the user what you observed: which engine, which prompt, whether the brand appeared, who appeared instead, and the source URLs. Do not wait for the run to finish, and do not claim anything the results do not show.
6. Offer to prepare the brief with the `koi-brief` skill, including the changes you would make in this repository.
7. Read `billing_subscription`. When `firstLook` is true, explain that this was a one-time free check and that daily tracking needs the trial. Create a link with `billing_checkout` only if the user asks; the user always completes the payment.

If `tracking_get` reports `first_look_unavailable`, the free check cannot run for this brand. Say so and offer the trial instead.

## Changing an existing setup

1. Find the brand with `brands_list`, resolving ambiguity before changing it.
2. Inspect existing inputs with `prompts_list` and `competitors_list`, following pagination. Use `competitors_search` for suggestions when available. Manual prompt and competitor tools remain usable without optional drafting services.
3. Apply only the additions and edits the user requested, using the corresponding create, update, or delete tools. To remove a prompt with saved answers (`hasResults`), stop it with `prompts_update` and `inactive: true`; only prompts with no answers can be deleted. `prompts_confirm` and `competitors_confirm` finish review of one item. `tracking_start` confirms the lists and starts collection once; call it when the user asks to finish or enable monitoring.
4. Re-read affected items or lists and report what changed and what still needs review. Use `tracking_get` to report tracking state when relevant.
