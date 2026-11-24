# SEO and GEO playbook

Sources checked on 2026-11-24. If you can open pages, check a source again before relying on a
detail.

- **Official**: stated by the company that runs the engine.
- **Research**: measured in a published study, under its own conditions.
- **Hypothesis**: reasonable, not established. Test it and say so.

## Google AI Overviews and AI Mode

Official, from [Google's guide to generative AI features](https://developers.google.com/search/docs/fundamentals/ai-optimization-guide).
Google treats this work as SEO: these features draw on its Search index and ranking systems.

Check in this order:

1. **Eligible.** The page must be indexed and able to appear with a snippet. Look for `noindex`,
   `nosnippet`, blocked crawling, a wrong canonical and non-200 responses. See the
   [technical requirements](https://developers.google.com/search/docs/essentials/technical).
2. **Included.** The site must be included in Search generative AI features in Search Console. Only
   the owner can see this, so ask.
3. **Readable.** Main content can be crawled and does not depend on blocked JavaScript. See
   [JavaScript SEO](https://developers.google.com/search/docs/crawling-indexing/javascript/javascript-seo-basics).
4. **Worth using.** A point of view, first-hand experience or data others do not have, organized for
   a reader with clear headings. See
   [helpful content](https://developers.google.com/search/docs/fundamentals/creating-helpful-content).
5. **Not duplicated.** Merge pages that compete for the same question.

Google says these are not needed for its AI features:

- `llms.txt` or other AI text files. Google Search ignores them; they neither help nor harm there.
- Splitting content into small chunks.
- Rewriting content in a special way for AI.
- Special schema. Structured data is still useful for rich results.

Do not recommend:

- A page for every variation of a question. Doing this to manipulate rankings or AI answers breaks
  the [spam policies](https://developers.google.com/search/docs/essentials/spam-policies).
- Inauthentic mentions.

The owner can measure this in Search Console's generative AI performance report.

## ChatGPT

Official, from [OpenAI's crawler documentation](https://developers.openai.com/api/docs/bots):

- Allow `OAI-SearchBot`. Sites that opt out are not shown in ChatGPT search answers, though they can
  still appear as navigational links.
- `GPTBot` concerns model training and is a separate choice. Blocking it does not opt out of search.
- `ChatGPT-User` fetches pages when a user asks. It does not decide what appears in search.
- A `robots.txt` change can take about 24 hours to apply.
- Check that a firewall or CDN does not block requests from OpenAI's published IP ranges.

Hypothesis: which pages ChatGPT prefers beyond access is not documented. Study the sources in the
collected answers instead of assuming a rule.

## Gemini

Google's guide above is about Google Search. Do not present it as official guidance for the Gemini
app. Hypothesis: the same accessible, indexed, specific pages are a sensible start. Rely on the
sources Koi collected from Gemini.

## Basics on the page

General SEO practice:

- One clear subject per page, with a descriptive title and heading.
- Self-referencing canonical, internal links from relevant pages, an entry in the sitemap.
- Main content in the HTML response.
- Real authors where they matter; a changed date only when the content changed.

Hypothesis for AI answers: state plainly what the product is, for whom and how it differs, near the
top, so that a reader who lands there needs no other page.

## Evidence in the content

Research: [GEO: Generative Engine Optimization](https://arxiv.org/abs/2311.09735) tested content
changes on a benchmark and reports visibility gains of up to 40% in generative engine responses. It
also found that results vary by domain. It is not a forecast for a given site, and today's engines
differ from the ones tested.

Use evidence when it is true and useful to the reader: measured results with the method and date,
named sources for claims, quotations from real people who agreed to them. Never invent a number, a
customer or a quote. Keyword stuffing is not evidence.

## Being recognized

For `no_info`, `other_company` and `other_meaning`:

- Use one name and one short description, with the category, on the site, repository and profiles.
- Official: `Organization` markup can help Google tell an organization apart from others, and
  `sameAs` can list its real profiles. See
  [Organization structured data](https://developers.google.com/search/docs/appearance/structured-data/organization).
  Mark up only what the page shows or what is true of the organization.
- Hypothesis: the same consistency helps other engines.

## Pages owned by others

Use this when answers rely on comparisons, directories, communities or reviews.

1. Read the page and find how it chooses what to include.
2. Pick the honest route: correct a mistake, submit to a directory, offer the author access and
   facts, answer a question while saying who you are, ask real customers for reviews without paying
   for a positive one.
3. If there is no honest route, say so and look at the next source.

Never recommend fake reviews, undisclosed promotion, sock puppets or mass-posted mentions.

## Common questions

| Question | Answer |
| --- | --- |
| Should I add `llms.txt`? | Not for Google Search. Fine for other tools that read it. Low priority here. |
| Do I need schema? | Not for Google's AI features. Use it where it is accurate and supports rich results or identity. |
| Should I block AI crawlers? | It is the owner's choice. Separate training (`GPTBot`) from search (`OAI-SearchBot`). |
| Should I write 50 comparison pages? | No. Write the few where you have first-hand facts. |
| When will it work? | Unknown. Recheck on the next runs and look for a repeated change. |
