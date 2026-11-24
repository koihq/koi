# Koi

Koi is an open-source alternative to Profound and Peec AI. It asks ChatGPT, Gemini, Google AI Mode and Google AI Overviews the questions your buyers ask. It records whether your product is cited, mentioned or missing, who is named instead, and which pages each answer used.

This plugin connects Claude to Koi Cloud and adds three skills:

- **koi-setup** chooses the buyer questions and competitors to track, then runs the first check.
- **koi-brief** explains why answers leave you out and what to do first, with the evidence and the SEO or GEO practice behind each action.
- **koi-fix** carries out one action in your repository, or prepares the text if there is none.

## Start

Install the plugin. In Claude Code:

```
/plugin install koi --marketplace koihq/koi
```

Sign in to Koi when Claude asks, then try:

> Set up Koi and tell me who AI recommends instead of my product.

> Why does ChatGPT recommend my competitor? Prepare a Koi brief.

> Apply the first action from my Koi brief to this repository.

The first check asks 5 questions once on all four engines. Daily tracking needs a Koi plan, which you manage in the app. The plugin cannot buy or change one.

## What it does with your data

The plugin runs no code on your computer. Claude calls the Koi server at `app.getkoi.co` after you sign in with OAuth. It sends what a task needs, such as your brand name, website, questions and competitors, and reads back collected answers and their sources. Koi collects answers through a provider on its servers.

When Claude can open web pages, the skills may also ask it to read your site and the sources an answer cites. Revoke access at any time from the MCP page in Koi. The [privacy policy](https://getkoi.co/privacy) covers storage and retention.

## Self-hosting

This plugin connects to Koi Cloud. If you run your own Koi server, follow the [MCP guide](https://github.com/koihq/koi/blob/main/docs/mcp.md) instead.

## Help

- [Report a problem](https://github.com/koihq/koi/issues)
- [MCP guide](https://github.com/koihq/koi/blob/main/docs/mcp.md)
- [Website](https://getkoi.co) and [source code](https://github.com/koihq/koi), MIT licensed
- [Privacy policy](https://getkoi.co/privacy) and [terms](https://getkoi.co/terms)
