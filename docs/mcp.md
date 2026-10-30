# Koi MCP

Koi serves one Streamable HTTP endpoint from the Go binary: `http://localhost:8080/mcp` for local clients and `https://<your-host>/mcp` for remote clients. Set `KOI_MCP_PUBLIC_URL=http://localhost:8080` in Vite development, where the browser runs at `:5173`. In production, `KOI_MCP_PUBLIC_URL` defaults to `KOI_PUBLIC_URL` when the UI and MCP share an origin. Browser-based hosts must reach that origin from their own infrastructure; use a public reverse proxy or an HTTPS tunnel in development. Keep the MCP origin stable during OAuth authorization.

All MCP calls require an OAuth bearer access token. Koi hosts signup, login and consent for local and remote clients using the authorization-code flow with PKCE. Access tokens expire after one hour; refresh tokens rotate and expire after 30 days. The Agent access page lists OAuth connections and lets the user revoke them.

Koi publishes OAuth protected-resource metadata at `/.well-known/oauth-protected-resource/mcp` and authorization-server metadata at `/.well-known/oauth-authorization-server`. It accepts Client ID Metadata Documents (CIMD) and dynamic client registration (DCR). Tool metadata identifies read/write access and the required `koi:read` or `koi:write` scope. The MCP resource server uses the [official Go SDK](https://github.com/modelcontextprotocol/go-sdk) for Streamable HTTP, bearer verification and protected-resource metadata.

## Connect

- **Codex:** `codex mcp add koi --url https://<your-host>/mcp`. Complete OAuth when prompted. For local use, use `codex mcp add koi-local --url http://localhost:8080/mcp` and complete OAuth; if needed, run `codex mcp login koi-local`. Codex's shared app-server daemon may not inherit bearer-token environment variables from a new shell. [Codex MCP setup](https://developers.openai.com/resources/docs-mcp)
- **Claude Code:** `claude mcp add --transport http koi https://<your-host>/mcp`, then run `/mcp` to authorize. [Claude Code MCP setup](https://docs.claude.com/en/docs/claude-code/mcp)
- **Cursor:** add an HTTP MCP server named `koi` with URL `https://<your-host>/mcp` in Cursor's MCP settings. [Cursor MCP documentation](https://docs.cursor.com/context/model-context-protocol)
- **ChatGPT web:** add a custom MCP server under Plugins using the public HTTPS URL and OAuth. A local server needs a tunnel. [OpenAI documentation](https://developers.openai.com/api/docs/guides/custom-mcp-server)
- **Claude web:** add a custom remote connector with the same URL and choose OAuth sign-in. Claude connects from Anthropic's infrastructure. [Anthropic documentation](https://support.claude.com/en/articles/11175166-get-started-with-custom-connectors-using-remote-mcp)

Plugin or marketplace packages should point at this same endpoint. No host-specific product implementation is needed. Dots requires verification through a published plugin; Grok Bot needs a separate compatibility check before support is advertised.

## UI action → MCP tool

| UI action | MCP tool |
| --- | --- |
| Account identity | `account_me` |
| Search countries; list/get/create/update brands | `brands_countries`, `brands_list`, `brands_get`, `brands_create`, `brands_update` |
| List/get/create/update/delete/confirm prompts | `prompts_list`, `prompts_get`, `prompts_create`, `prompts_update`, `prompts_delete`, `prompts_confirm`, `prompts_confirm_all` |
| Search/list/get/create/update/delete/confirm competitors | `competitors_search`, `competitors_list`, `competitors_get`, `competitors_create`, `competitors_update`, `competitors_delete`, `competitors_confirm`, `competitors_confirm_all` |
| Start/list/get drafting jobs | `generations_create`, `generations_list`, `generations_get` |
| Tracking; list/get runs and answers | `tracking_get`, `runs_list`, `runs_get`, `runs_results` |
| Visibility summary and prompt history | `analysis_summary`, `prompt_visibility`, `prompt_history_list`, `prompt_history_runs` |
| Cloud catalog/quote/subscription/checkout/allowance/portal | `billing_catalog`, `billing_quote`, `billing_subscription`, `billing_checkout`, `billing_confirm_checkout`, `billing_change_allowance`, `billing_portal` |

`brandId` must be supplied on collection and brand-scoped commands. Resource details use the resource UUID, as in the product API. Browser `active_brand_id` is never read by MCP. `generations_create` and `billing_change_allowance` require a stable `idempotencyKey` on retries. Drafting returns a job ID; poll `generations_get`. Checkout returns the hosted Stripe URL; after payment poll `billing_subscription` or confirm a known session ID. Billing tools are absent in self-hosted mode. Brand activation is a browser-only preference, and login and credential management use the dedicated Koi flows.

## OAuth implementation choice

The MCP Go SDK implements the resource server and client primitives, not an authorization server. Koi therefore implements the issuer-specific endpoints in `koiapi`, while credential state and token lifecycle live in `koidomain` and GORM persistence lives in `koidatabase`. We evaluated [Fosite](https://github.com/ory/fosite), [go-oauth2](https://github.com/go-oauth2/oauth2), and [Hydra](https://www.ory.com/hydra). Fosite would replace the token protocol but require its request/session storage adapters; Hydra would add a separate service to every self-hosted deployment; go-oauth2 does not clearly document the MCP PKCE/CIMD requirements. The current narrow issuer avoids those additional adapters and services. Revisit this decision before broad public deployment, after interoperability and security tests against real clients.

## Verification

Automated tests cover the generic MCP handshake and tool discovery, OAuth code exchange with PKCE, replay rejection and token revocation. The deployment smoke test should run signup → consent → read/write tools → hosted Checkout → active subscription, then test another organization, credential revocation, retry, canceled checkout and delayed Stripe webhook. Confirm the same flow on Codex, Claude Code, Cursor, ChatGPT web and Claude web before listing each as verified. Host-specific tests and Dots/Grok checks require live accounts and a reachable deployment.
