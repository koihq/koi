import { business } from "./legal/business";

/** Every visible string on the page, kept together so the copy can be read and edited in one place. */

export const links = {
  app: "https://app.getkoi.co/sign-up",
  signIn: "https://app.getkoi.co/sign-in",
  github: "https://github.com/koihq/koi",
  docs: "https://github.com/koihq/koi#readme",
  selfHost: "https://github.com/koihq/koi#self-host-with-docker-compose",
  skill: "/SKILL.md",
  license: "https://github.com/koihq/koi/blob/main/LICENSE",
  issues: "https://github.com/koihq/koi/issues",
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
      before: "Koi asks",
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
  legal: {
    updated: "Last updated",
    cookieSettings: "Cookie settings",
  },
  footer: {
    source: "Source code",
    skill: "Agent instructions",
    privacy: "Privacy",
    terms: "Terms",
    rights: "Koi",
    owner: `© 2026 ${business.name} · VAT ${business.vat}`,
  },
};
