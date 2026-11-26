import { business } from "./legal/business";

/** Every visible string on the page, kept together so the copy can be read and edited in one place. */

export const links = {
  app: "https://app.getkoi.co/sign-up",
  signIn: "https://app.getkoi.co/sign-in",
  github: "https://github.com/koihq/koi",
  docs: "https://github.com/koihq/koi#readme",
  selfHost: "https://github.com/koihq/koi#self-host-with-docker-compose",
  skill: "/SKILL.md",
  mcp: "https://github.com/koihq/koi/blob/main/docs/mcp.md",
  license: "https://github.com/koihq/koi/blob/main/LICENSE",
  issues: "https://github.com/koihq/koi/issues",
  correction: "https://github.com/koihq/koi/issues/new?template=correction.yml",
  privacy: "/privacy",
  terms: "/terms",
};

export const agentPrompt =
  "Set up Koi and tell me who AI recommends instead of my product: getkoi.co/SKILL.md";

export const engineNames = {
  chatgpt: "ChatGPT",
  gemini: "Gemini",
  googleAiMode: "Google AI Mode",
  google: "Google AI Overviews",
} as const;

/** Beside a mark the logo already says whose engine it is, so Google's two keep only the product. */
export const engineShortNames = {
  chatgpt: "ChatGPT",
  gemini: "Gemini",
  googleAiMode: "AI Mode",
  google: "AI Overviews",
} as const;

export const copy = {
  meta: {
    home: {
      title: "Koi: open source AI visibility tracker (AEO)",
      description:
        "Koi is an open source AI visibility tracker. It asks ChatGPT, Gemini, Google AI Mode and AI Overviews what your buyers ask, every day.",
    },
    homeSocial: {
      title: "Koi: see who AI recommends instead of you",
      description:
        "Open source AEO for founders. See who AI recommends instead of you, then let your agent fix it.",
    },
    privacy: {
      title: "Privacy Policy · Koi",
      description:
        "What personal data Koi Cloud collects, why, who receives it, how long it is kept and the rights you have over it.",
    },
    terms: {
      title: "Terms of Service · Koi",
      description:
        "The terms for using Koi Cloud: business use, plans and trial, cancellation and refunds, acceptable use and liability.",
    },
    notFound: {
      title: "Page not found · Koi",
      description: "This page does not exist.",
    },
  },
  notFound: {
    title: "This page does not exist.",
    body: "The address may have changed.",
    home: "Go to the Koi home page",
  },
  skipLink: "Skip to content",
  consent: {
    message: "Koi uses a cookie to see which pages help people get started.",
    accept: "Allow",
    reject: "Decline",
  },
  nav: {
    home: "Koi home",
    pricing: "Pricing",
    docs: "Docs",
    github: "GitHub",
    signIn: "Sign in",
    start: "Run a free check",
  },
  hero: {
    title: "See who AI recommends instead of you.",
    lead: {
      before: "Koi is an open source AI visibility tracker. It asks",
      and: "and",
      after: "what your buyers ask, every day. Your agent makes the fix.",
    },
    agent: "Copy agent prompt",
    agentLabel: "Copy agent prompt for Claude Code, Codex or Cursor",
    copied: "Copied. Now paste it",
    copyFailed: "Copy the prompt below",
    copiedStatus: "Prompt copied. Paste it in your agent.",
    browser: "Start in the browser",
    demoTitle:
      "Koi's summary page: AI visibility over 30 days, the competitors recommended instead, and the tracked prompts.",
  },
  bento: {
    title: "Keep your agent. Koi tells it what to fix.",
    lead: "It already knows your code and your voice. Now it knows who AI names instead of you.",
    setup: {
      title: "It sets Koi up from your repo.",
      body: "Paste one prompt. It finds your product, drafts your buyers' questions and runs the first check.",
      demo: "A prompt is pasted into an agent, which connects to Koi after a sign-in, runs the first check and reports who was named instead.",
      readSkill: "Read SKILL.md",
    },
    daily: {
      title: "Tomorrow shows if it worked.",
      body: "Koi asks again every day, on all four engines.",
    },
    namesake: {
      title: "Only mentions that are you.",
      body: "A company or a word that shares your name does not count.",
      label: "Three answers that name Koi, and which one counts",
      answers: [
        {
          text: "Koi is open source and hands a brief to the coding agent you already use.",
          verdict: "Counted",
          counted: true,
        },
        {
          text: "Koi Pond Supplies ships filters and pumps across the UK.",
          verdict: "Not counted: another company",
          counted: false,
        },
        {
          text: "Koi are ornamental carp, bred in Japan for their colours.",
          verdict: "Not counted: another meaning",
          counted: false,
        },
      ],
    },
    fix: {
      title: "It makes the fix. You review the diff.",
      body: "Copy any answer as a brief. No credits, and nothing is published for you.",
      demo: "An agent reads a brief and edits a repository. The next check shows the brand cited.",
    },
    evidence: {
      title: "The answer behind every number.",
      body: "What the engine said and the pages it read: what your agent works from.",
      engine: "ChatGPT",
      question: "Which AEO tool works with Claude Code?",
      before: "For teams that live in their editor,",
      brand: "Koi",
      after: "is open source and hands a brief to the coding agent you already use.",
      sources: ["getkoi.co", "github.com/koihq/koi"],
      outcome: "Cited",
    },
    rivals: {
      title: "Who gets named instead.",
      body: "For every question, the competitors engines pick over you, and how often.",
    },
  },
  pricing: {
    title: "Pay for prompts, not features.",
    titleMuted: "Every plan gets all four engines and the whole app.",
    koiKoi: "koi.",
    unit: "prompts",
    pickerLabel: "Prompts to track",
    perMonth: "a month",
    perPrompt: "per prompt",
    tax: "Tax included",
    cta: "Run a free check",
    note: "First check free, no card: 5 prompts, asked once. Daily tracking starts with a 7-day trial, card required.",
  },
  openSource: {
    title: "Open source, MIT licensed.",
    body: "The code behind Koi Cloud is on GitHub. Run it on your own server for free; live answers come through your own Cloro account.",
    repo: "koihq/koi",
    license: "MIT",
    selfCta: "Self-hosting guide",
  },
  faq: {
    title: "Frequently asked questions",
    more: "Something else?",
    ask: "Ask on GitHub",
    items: [
      {
        q: "What is Koi?",
        a: "Koi asks ChatGPT, Gemini, Google AI Mode and Google AI Overviews the questions your buyers ask, every day. It shows whether you are named, who is named instead and the pages behind each answer, then hands your coding agent a brief to fix it.",
      },
      {
        q: "What is AEO, and how is it different from SEO?",
        a: "SEO gets your page ranked in a list of links. AEO, answer engine optimization (also called GEO), gets your product named inside the answer itself, where there is often no list to click.",
      },
      {
        q: "How does Koi see what AI tells my buyers?",
        a: "It asks each engine through its public web page, in a real browser, logged out and from your buyers' country: the answer a new visitor reads, with the sources it shows. Not the model's API, which answers differently and shows no sources. That answer is what AEO changes, so it is what Koi tracks.",
      },
      {
        q: "Why does my own ChatGPT say something different?",
        a: "Your account has history and memory, and answers change on every ask. Koi asks logged out, the same questions every day, so read the trend, not a single answer.",
      },
      {
        q: "Which AI engines does Koi track?",
        a: "ChatGPT, Gemini, Google AI Mode and Google AI Overviews, on every plan.",
      },
      {
        q: "How is visibility measured?",
        a: "The share of answers that name you, per question and per engine. A citation, your link in the answer, is counted apart from a mention of your name. An answer about another company with your name does not count.",
      },
      {
        q: "Can I choose the questions and competitors?",
        a: "Yes. Koi drafts them from your site, nothing is asked until you confirm them, and you can change them at any time.",
      },
      {
        q: "Is there a free trial? Do I need a card?",
        a: "The first check is free and needs no card: 5 prompts, asked once on all four engines. Daily tracking starts with a 7-day trial, which asks for a card.",
      },
      {
        q: "Can I change my plan or cancel?",
        a: "Yes, at any time from billing. A canceled plan keeps running until the end of the period you paid for.",
      },
      {
        q: "Does my agent need an API key?",
        a: "No. Koi is an MCP server at app.getkoi.co/mcp, and Claude Code, Codex or Cursor connect with a sign-in you approve. Your agent can create a checkout link, but only you can pay.",
      },
      {
        q: "Can I self-host Koi?",
        a: "Yes, it is MIT licensed. Your data stays in your own Postgres, and live answers come through your own Cloro account, which Cloro bills you for.",
      },
    ],
  },
  closing: {
    title: "Find out what AI says about you.",
  },
  alternatives: {
    pick: (rival: string) => `Koi or ${rival}: which should you pick?`,
    choose: "Choose",
    table: (rival: string) => `Koi and ${rival}, feature by feature`,
    feature: "Feature",
    /** Read before a cell's words, since the marks are hidden from screen readers. */
    marks: { yes: "Yes", no: "No" },
    /** The bias comes first, before any claim. */
    disclosure: "We make Koi.",
    checked: "Prices and features checked on",
    sources: "from each company's own pages:",
    note: 'Prices are in each company\'s currency. "Not published" means its pages do not say.',
    correct: "Found a mistake? Tell us on GitHub.",
    differ: (rival: string) => `How Koi and ${rival} differ`,
    more: {
      before: "Comparing more tools? See",
      all: "every comparison",
      or: "or",
      openSource: "the open source AEO tools",
      end: ".",
    },
    topics: {
      pricing: "Pricing",
      data: "Engines and data",
      agents: "Working with your agent",
      team: "Team and security",
    },
    openSource: {
      disclosure: "We make Koi, one of the projects on this page.",
      checked: "Checked on",
      read: "on each project's repository and site.",
      glance: "At a glance",
      columns: {
        project: "Project",
        license: "License",
        engines: "Engines",
        hosted: "Hosted version",
      },
      pick: "Which one to pick",
      projects: "One by one",
      facts: {
        collects: "How it asks the engines",
        runs: "What you run",
        agents: "For agents",
      },
      source: "Source code",
    },
    index: {
      heading: "Koi compared with other AI visibility tools",
      lead: "Each page puts Koi next to one tool: prices for the same number of prompts, engines, agents and team features, checked on that company's own pages, with where the other tool is ahead.",
    },
    closing: "Check your own product before you choose.",
    closingBody:
      "The first Koi check is free and needs no card: 5 prompts, asked once on all four engines.",
  },
  legal: {
    updated: "Last updated",
    cookieSettings: "Cookie settings",
  },
  footer: {
    product: "Product",
    compare: "Compare",
    openSource: "Open source",
    legal: "Legal",
    source: "Source code",
    selfHost: "Self-hosting",
    mcp: "MCP server",
    skill: "Agent instructions",
    privacy: "Privacy",
    terms: "Terms",
    owner: `© 2026 ${business.name} · VAT ${business.vat}`,
  },
};
