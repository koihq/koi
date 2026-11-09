# Landing revision: hero, agent pill, open source, questions

Status: **Ready**. Baseline: `3949e14` on `feat/landing-page`, with `ui/site/` untracked and the
uncommitted edits listed in git status. First task: copy this file to
`plans/ui/landing-revision.md` (the project's plan location). Do not commit (user commits).

## Context

The founder reviewed the rendered landing (`ui/site`, `make site-dev` on :5174) and rejected:

1. **Hero hierarchy.** Headline and lead sit on the left; the two CTAs float on the far right,
   bottom-aligned to the lead. The eye splits and the right half of the headline is empty.
2. **Too little colour.** Real engine and agent logos exist (`ui/site/logos.tsx`) but the hero shows none.
3. **Too narrow.** Container `max-w-295` (1180px).
4. **Agent CTA.** Wants the Clerk pattern: one pill holding overlapping agent logos + "Copy prompt".
5. **Pricing comparison table** (Koi vs Peec/Otterly/Profound): remove it. Replace its value with a
   proper **open source section**.
6. **Pond** (halftone koi in pricing) is liked: reuse it once more, not in the hero (it pulled the
   eye there before). Decision: the **closing CTA**.
7. **Questions** are glossary questions; replace them with the doubts a founder actually has.

Decisions confirmed with the user: centered hero; pond reused in the closing CTA.

## Research (Mobbin, `search_sections`, web)

Queries: "developer tool hero section dark with headline, subheadline and logos of supported
integrations"; "centered hero section with copy command pill button for AI coding agents"; "open
source section on developer tool landing page with GitHub stars and self-host option"; "open source
section showing GitHub repository, license and contributors on dark developer landing page"; "FAQ
section on SaaS landing page dark". No Fey/Family/Trade Republic marketing match (they govern the
product, not this page).

| Reference | Borrow | Reject |
| --- | --- | --- |
| [Supabase hero](https://mobbin.com/sites/sections/0eb5722a-4576-49bd-9d04-b2d9556690fc) · [Sana](https://mobbin.com/sites/sections/dc99acb9-1b40-4dc3-bad4-ffa783062589) · [Cursor](https://mobbin.com/sites/sections/a7f2c059-5d26-466d-9337-33ab8583c3d4) | One centered axis: headline → lead → CTAs → product frame | Gradients, customer-logo strips (we have none) |
| [Retool hero](https://mobbin.com/sites/sections/becbcb71-ca9f-40da-b93f-89268cc78e46) | Negative example: CTAs split right only work with a short title — our current bug | — |
| [Browserbase](https://mobbin.com/sites/sections/c5c98154-679e-45ff-9a81-58feae8c0ca5) · [Vercel](https://mobbin.com/sites/sections/b6e0d794-fb8f-491b-a2c7-c72834365966) + user's Clerk screenshot | "Copy prompt" as a primary CTA; the logo stack says "works with your agent" without a sentence | Mono command text (Koi "Mono chrome" anti-pattern) |
| [Retool self-hosted](https://mobbin.com/sites/sections/ec886bd7-cfbf-4c80-861d-bb69db7fe9fb) | Text + real commands side by side, then a row of 3 facts | Illustrated clouds, chat bubble |
| [Glide open source](https://mobbin.com/sites/sections/efa8f0b3-1c0f-4b73-b7df-ea7a2a8f7d53) | Halftone texture behind a block (validates the pond reuse) | Coloured card backgrounds |
| [1Password open source](https://mobbin.com/sites/sections/3c0ecbe4-eef7-4e12-9fe4-1dea9dd3b8ca) | — | Three white cards: Koi "Card kit" anti-pattern |
| [Linear FAQ](https://mobbin.com/sites/sections/1d6082b2-fd08-479c-9367-cccd1b14e62c) · [Dovetail](https://mobbin.com/sites/sections/305c3f1c-1497-4d04-9803-7b977805d603) · [Giga](https://mobbin.com/sites/sections/2d1ea869-c2c4-4d67-ba52-34a76fd84ebd) | Title left + accordion right; first item open; "still unsure?" link under the title | Boxed rows (Resend) |

## Principles

- **Hero — one reading line.** A founder lands and must get: the promise, which engines, how to
  start. Center everything on the frame's axis so headline → engines → pill → app is one vertical
  scan. The four engine logos sit inline in the lead, in colour, before each name: colour carries
  information ("these four"), not decoration. Tradeoff: centered loses some editorial left-edge
  tension; at 1360px it removes the empty right half and the split CTAs.
- **Agent pill — the primary action names its audience.** Overlapping logos of Claude Code, Codex
  and Cursor + a primary inner button. Replaces "Start with an agent" + the separate "Works with"
  logo row (redundant). Browser path stays as a quiet text link below/beside.
- **Open source — prove it, then let them choose.** The comparison table's real differentiators
  were "Open" and "Run it yourself". Make them a section: claim, repo identity (GitHub mark,
  `koihq/koi`, MIT), the real self-host commands, and the Cloud vs Self-hosted choice as a
  two-column definition, no cards. Tradeoff: we lose the price-per-competitor grid; the pricing
  headline still carries the "$125 with Peec AI" comparison with a dated footnote.
- **Pond — one more appearance, same meaning.** Closing CTA over a pond; orange still means "AI
  named you". Copy stays on the left so the existing `slotsAt` logic (fish only to the right of the
  avoided zone) works unchanged.
- **Questions — answer the objections.** Volatility, choosing prompts, namesakes, what the agent
  changes, how fast, trial, money, self-host cost. Native `<details>` keeps it readable without JS.

## Global changes

- Container: replace every `max-w-295` in `ui/site/app.tsx` with `max-w-340` (1360px). Keep
  `px-4 sm:px-7`. Text measures stay capped by their own `max-w-*` (lead `max-w-160`, body `max-w-140`).

## Tasks (in order)

### 1. `ui/site/logos.tsx` — agent marks for the pill
- Export `agentList` built from the existing `agentArt` (Claude Code / `claude.svg`, Codex /
  `chatgpt` + `chatgptDark`, Cursor / `cursor.svg` inverted in dark).
- Add `AgentStack({ size = 32 })`: a `<span aria-hidden="true" class="flex -space-x-2">` of
  circles. Each circle: `size-8 rounded-full grid place-items-center bg-[#16161a] ring-2 ring-[#0b0b0d]`
  (ring = pill background so the overlap reads as cut), logo 18px inside via the existing `Art`
  with `label=""` (Cursor keeps `dark:invert`).
- Delete `AgentLogos` once task 3 removes its only caller.

### 2. `ui/site/app.tsx` + `site.css` — `AgentPill`
- Rename `AgentButton` → `AgentPill`; keep its copy/timer/`onFailure` logic and the `.site-swap`
  label swap unchanged. Markup:
  ```tsx
  <div className="site-agent-pill">
    <AgentStack />
    <Button size="cta" className="rounded-full" onClick={copyPrompt}
            aria-label={copy.hero.agentLabel}>…swap spans…</Button>
  </div>
  ```
  Note: `rounded-full` on `Button` is a radius-only override, accepted for the landing scope (pill
  shape is the pattern). Do not change size/type classes.
- `site.css`:
  ```css
  .site-agent-pill { display:inline-flex; align-items:center; gap:1rem; padding:0.375rem 0.375rem 0.375rem 0.5rem;
    border-radius:9999px; background:#0b0b0d; box-shadow: inset 0 0 0 1px var(--border); }
  ```
- Copy (`copy.ts` → `hero`): `agent: "Copy agent prompt"`, add
  `agentLabel: "Copy the setup prompt for Claude Code, Codex or Cursor"`. Keep `copied`,
  `copyFailed`, `copiedStatus`.
- `Actions` (hero + closing): `flex flex-col items-center gap-4` (closing: `items-start`) →
  `<AgentPill/>` then a text link `copy.hero.browser` + `ArrowRight01Icon` (`text-body
  text-muted-foreground hover:text-foreground`, `rounded-control px-2 py-2`). The failure `<p>` with
  the prompt stays, centered under it in the hero.

### 3. Hero
- Section: `mx-auto max-w-340 px-4 pt-36 pb-32 sm:px-7 sm:pt-44 text-center`, inner stack
  `flex flex-col items-center`.
- `h1.site-display` `max-w-240 mx-auto` (headline breaks into 2 balanced lines).
- Lead `p.site-lead mt-6 max-w-160 text-muted-foreground`, rendered from a new
  `copy.hero.lead` structure so logos sit inline:
  ```ts
  lead: { before: "Koi asks", after: "what your buyers ask, every day. Your agent makes the fix." }
  ```
  Render: `before`, then for each `engineList` entry `<EngineLogo engine size={20} decorative/>`
  + visible `engineNames[engine]` (Google entry shows "Google"), joined with ", " and " and ",
  then `after`. Each logo `inline-block align-[-0.2em] me-1.5`. The text alone must still read as
  the full sentence (logos are decorative).
- `Actions` with `mt-9`. `ProductFrame` unchanged except it now spans 1360px (APP_WIDTH 1320 ≤
  width, so scale caps at 1; the frame stays 1320 wide centered — set the figure `mx-auto
  max-w-[1320px]`).
- `Facts`: keep, `max-w-340`; drop the engine-logo row on fact 0 (now in the hero) and give all
  three facts the same structure (title + body) — no decorative glyphs.

### 4. Agent section
- Replace `<AgentButton />` with `<AgentPill />`; delete the `worksWith` block and
  `copy.agent.worksWith` (the stack replaces it). Keep `readSkill` link beside the pill.

### 5. Pricing — remove the comparison
- Delete `Comparison`, `ColumnHead`, `rivalPrices`, `rivalColumns`, `factTopics`, and the
  `<Comparison>` block; remove `BrandIcon` import if unused.
- `copy.pricing`: delete `koi`, `compareCaption`, `promptsHead`, `noPlan`, `talkToSales`, `topics`,
  `koiFacts`, `rivalFacts`. Replace `footnote` with:
  `"Peec AI's list price for 50 prompts on all four engines, read on Nov 7, 2026. Koi's price includes tax."`
  and render it under the pond (`mx-auto mt-8 max-w-340 px-4 sm:px-7`, `text-xs text-muted-foreground`).
- `site.css`: delete `.site-koi-column`, `.site-compare-row*`, `.site-scroll` if unused.
- Keep `assets/logos/rivals/*` (SOURCES.md says the demo uses them; verify with grep, delete only
  if nothing references them).

### 6. Open source section (rewrite `OpenSource`)
Layout `mx-auto max-w-340 px-4 pb-32 sm:px-7`:
- Heading row: `GithubLogo size={28}` above `h2.site-heading` "Open source, MIT licensed."
  + `p.site-lead mt-4 max-w-140 text-muted-foreground`: "Koi Cloud runs the same code that is on
  GitHub. Read it, fork it, or run it on your own server."
- Body grid `mt-12 grid gap-10 md:grid-cols-[minmax(0,7fr)_minmax(0,5fr)] md:gap-14 items-start`:
  - **Left — the commands**: `div.site-product rounded-panel overflow-hidden` holding the kit
    `Code` component (`ui/site/kit/code.tsx`, already used by the demos; mono here is the
    stageshow-scene exception, same as `FixDemo`) with `numbers={false}` and lines verified against
    README "Self-host with Docker Compose" and the compose file in the repo:
    ```
    git clone https://github.com/koihq/koi && cd koi
    cp .env.selfhosted.example .env.selfhosted
    # set POSTGRES_PASSWORD, KOI_PUBLIC_URL and KOI_CLORO_API_KEY
    docker compose --env-file .env.selfhosted up -d
    ```
    If the README's command or compose file differs, use the README's exact form and report it.
    Wrap in `role="img"` is wrong here — it is real text: plain `<figure>` with `<figcaption
    className="sr-only">Self-hosting commands</figcaption>`.
  - **Right — the choice**, a `dl` with two groups separated by `gap-8` (no borders, no cards):
    - `dt.text-section` "Koi Cloud" + muted " we host"; `dd` "From €29 a month. Collection,
      updates and backups included." then `PrimaryLink` "Run a free check".
    - `dt` "Your server" + muted " you host"; `dd` "Free. Live answers come through your own Cloro
      account. OpenRouter drafting is optional." then text link "Read the self-hosting guide" →
      `links.selfHost` with `ExternalIcon`.
    - Under both, a quiet repo line: `GithubLogo 16` + "koihq/koi" + muted "· MIT" as a link to
      `links.github`.
- Below the grid, `mt-14 grid gap-8 sm:grid-cols-3` facts (title `text-section`, body muted):
  "Same app, no lite edition" / "The self-hosted build has every screen Koi Cloud has.";
  "Your data stays with you" / "Answers, prompts and sessions live in your Postgres.";
  "First user only" / "Sign-up closes after you, so nobody spends your Cloro credits."
- Replace `copy.openSource` with these strings. "updates and backups included" — verify with the
  user/README before shipping; if unconfirmed, use "From €29 a month, collection included." (current copy).

### 7. Questions (rewrite `Faq`)
- Layout `mx-auto grid max-w-340 gap-10 px-4 pb-32 sm:px-7 md:grid-cols-[minmax(0,4fr)_minmax(0,8fr)]`.
- Left: `h2.site-heading` "Questions" + `p.mt-3 text-body text-muted-foreground` "Something else?"
  link "Ask on GitHub" → `https://github.com/koihq/koi/issues` (add `links.issues`).
- Right: list of `<details name="faq">` (exclusive accordion; first has `open`). Each:
  `border-b border-border` (one structural rule per row is the accepted accordion affordance on the
  landing), `summary` = `flex justify-between items-center py-5 text-section cursor-pointer
  list-none [&::-webkit-details-marker]:hidden` with a `PlusSignIcon` (hugeicons, `size-4
  text-muted-foreground`) rotating 45° when open (`transition-transform 200ms var(--ease-out)`,
  none under reduced motion); answer `pb-6 pe-10 text-body text-muted-foreground max-w-150`.
- Copy (`copy.faq.items`):
  1. **AI answers change every time. Is one check worth anything?** — Not much. That is why Koi asks every question on all four engines every day and shows the trend, not a single answer.
  2. **How do I know which questions to track?** — Your agent reads your site and drafts the questions your buyers ask, with the competitors they weigh you against. Nothing is asked until you confirm them.
  3. **What if another company has my name?** — Koi reads every answer that names you and checks it is about your product. A namesake does not count as a mention.
  4. **What does my agent actually change?** — What the brief points to: a missing comparison page, a claim with no source, a page engines cite for your rivals but not you. It edits your repo and you review the diff.
  5. **How soon will a fix show up?** — Koi asks again the next day, so you see the day an engine starts naming you. Engines that search the web as they answer tend to move first.
  6. **Can I try it before paying?** — Yes. The first check is free, needs no card and covers 5 prompts. Daily tracking starts with a 7-day trial.
  7. **Can my agent spend my money?** — No. It can create a checkout link when you ask, and only you can complete the payment.
  8. **Is self-hosting really free?** — The code is MIT licensed. Live answers come through your own Cloro account, which Cloro bills you for.
  Claims verified in this session: daily checks, 5-prompt free check without card, 7-day trial,
  checkout-link-only billing (`ui/site/public/SKILL.md`); namesake judging (commit `34594cc`);
  MIT (`LICENSE`); Cloro requirement (AGENTS.md). Q4's brief content is a recommendation framing,
  consistent with `skills/koi-brief`.

### 8. Closing over the pond
- Section `relative overflow-hidden` full-bleed, `min-h-[520px]`; `<PricingPond count={120}
  avoid={copyRef} />` behind; inner `relative mx-auto max-w-340 px-4 py-28 sm:px-7`, copy block
  `ref={copyRef}` `max-w-130 rounded-panel bg-background p-6 sm:p-8` (same panel treatment as
  pricing so fish don't run under text) containing `h2.site-display` "Find out what AI says about
  you." and `<Actions align="start" />`.
- Rename `PricingPond` → `Pond` in `pond/ponds.tsx` (update its doc comment: "One koi per prompt in
  pricing; a fixed shoal behind the closing call to action"). No engine changes. On mobile
  (`max-lg`) mirror pricing: `max-lg:pb-100` so fish sit below the panel.

### 9. Docs
- `docs/design-system.md` → Public landing: pond is used in pricing and the closing CTA; pricing
  names Peec AI in its headline with a dated footnote (no comparison table); add the agent pill
  (`AgentStack`, `.site-agent-pill`) and the 1360px container.

## States

- Copy: idle → copied (2.6s) → idle; failure shows the prompt text under the pill (existing). The
  `<output>` announces "Prompt copied…" (existing).
- No JS: copy, open source and `<details>` work; the pill's button is inert (same as today) and the
  agent section's "Read SKILL.md" link remains the fallback; pond canvas is blank.
- Reduced motion: pond holds still (existing); plus icon doesn't rotate; label swap drops blur (existing).
- Narrow (<640px): hero lead wraps with logos inline; pill stays one row (logos 3×32 overlapped ≈
  80px + button ≈ 180px fits 320px); open-source grid stacks commands then choice; `Code` scrolls
  horizontally inside its panel (`overflow-x-auto`); FAQ stacks title above list.
- Loading/empty/error states of data do not apply (static page).

## Verification

1. `pnpm --dir ui check`, `pnpm --dir ui check-types`, `pnpm --dir ui exec vitest run`, `make site-build`.
2. `make site-dev`, open `http://localhost:5174` at 1440 and 375 wide:
   - Hero: headline, logo sentence, pill and frame share one center axis; nothing on the right edge alone.
   - Click "Copy agent prompt" → label becomes "Copied. Now paste it"; clipboard holds `agentPrompt`.
   - Pricing: no table; footnote under the pond; slider still adds/removes fish.
   - Open source: commands match README; both links resolve.
   - Questions: one open at a time; keyboard Tab/Enter/Space toggle; works with JS disabled.
   - Closing: fish fill the right of the panel, none under text; pauses off screen.
3. Stageshow `scripts/layout.mjs` against :5174 (reports outside the repo, e.g. the scratchpad) for
   crops/insets; `grep -rn "max-w-295\|AgentButton\|AgentLogos\|Comparison" ui/site` returns nothing.
