import koiMark from "@/assets/koi-mark.png";
import { ENGINES, hash } from "./data";

// No domain is invented for Deja or zsh-autosuggestions: both are repository-only here.
export const HOME_LOGOS: Record<string, { domain?: string; src?: string }> = {
  Deja: {},
  "zsh-autosuggestions": {},
  Koi: { src: koiMark },
  "Peec AI": { domain: "peec.ai" },
  Profound: { domain: "tryprofound.com" },
  Mintlify: { domain: "mintlify.com" },
  GitBook: { domain: "gitbook.com" },
  ReadMe: { domain: "readme.com" },
  beehiiv: { domain: "beehiiv.com" },
  Substack: { domain: "substack.com" },
  Kit: { domain: "kit.com" },
  Tally: { domain: "tally.so" },
  Typeform: { domain: "typeform.com" },
  Fillout: { domain: "fillout.com" },
  PostHog: { domain: "posthog.com" },
  Mixpanel: { domain: "mixpanel.com" },
  Amplitude: { domain: "amplitude.com" },
};

export const HOME_BRANDS = {
  deja: {
    name: "Deja",
    rivals: ["zsh-autosuggestions"],
    topics: {
      alternatives: [
        "best zsh plugin for inline predictive command autosuggestions",
        "What are alternatives to zsh-autosuggestions?",
        "Which zsh suggestion tools should I try?",
      ],
      fuzzy: [
        "How can I get fuzzy command suggestions in zsh?",
        "Which shell tools match commands when I mistype?",
      ],
      workflow: [
        "Can shell suggestions use my current project directory?",
        "How can I recall commands from my project history?",
      ],
    },
  },
  koi: {
    name: "Koi",
    rivals: ["Peec AI", "Profound"],
    topics: {
      discovery: [
        "How can a founder track their brand in AI answers?",
        "Which tools monitor brand visibility in ChatGPT?",
      ],
      agents: [
        "Which AI visibility tools track citations across answer engines?",
        "How can I review changes in my brand’s AI visibility?",
      ],
      competitors: [
        "How can I see who AI recommends instead of my brand?",
        "Which tools compare competitor mentions in AI answers?",
      ],
    },
  },
  // Real demo brands; all visibility evidence remains synthetic.
  mintlify: {
    name: "Mintlify",
    rivals: ["GitBook", "ReadMe"],
    topics: {
      discovery: [
        "Which tools help small teams publish developer documentation?",
        "What documentation tools work well for an API product?",
      ],
    },
  },
  beehiiv: {
    name: "beehiiv",
    rivals: ["Substack", "Kit"],
    topics: {
      discovery: [
        "Which newsletter tools work best for independent creators?",
        "How can I start a paid newsletter with a small audience?",
        "What is the easiest way to launch a newsletter without a website?",
        "Which platform should I use to move an email list from Substack?",
        "How do beehiiv and Kit compare for a new creator newsletter?",
        "What should I look for in a newsletter platform before I grow my list?",
      ],
      growth: [
        "Which newsletter platforms help creators get their first thousand subscribers?",
        "How can I grow a newsletter through referrals?",
        "What tools help newsletter writers understand where subscribers come from?",
        "Which email platforms support recommendations between newsletters?",
        "How can a small newsletter improve signup conversion?",
        "What is a good newsletter tool for running audience surveys?",
      ],
      monetization: [
        "Which newsletter platforms support paid subscriptions and sponsorships?",
        "How can I sell ads in a newsletter with a small audience?",
        "What is the best way to offer free and paid newsletter tiers?",
        "Which email tools make it easy to test subscription pricing?",
        "Can I run a paid newsletter without giving up ownership of my audience?",
        "How do newsletter platforms handle payments for international subscribers?",
      ],
      operations: [
        "Which newsletter tool is best for a weekly editorial workflow?",
        "How can I automate a welcome sequence for new newsletter readers?",
        "What platform lets a team collaborate on newsletter drafts?",
        "How can I segment subscribers based on what they read?",
        "Which newsletter platforms provide useful open and click reporting?",
        "What should I check before migrating a large newsletter to a new platform?",
      ],
    },
  },
  tally: {
    name: "Tally",
    rivals: ["Typeform", "Fillout"],
    topics: {
      discovery: [
        "Which form builders are good for collecting customer feedback?",
        "What is a simple form builder for a small business?",
      ],
    },
  },
  posthog: {
    name: "PostHog",
    rivals: ["Mixpanel", "Amplitude"],
    topics: {
      discovery: [
        "Which analytics tools help founders understand product activation?",
        "How can a small SaaS team track retention?",
      ],
    },
  },
};

export type HomeBrand = keyof typeof HOME_BRANDS;
export type HomeAnswer = {
  id: string;
  date: string;
  day: number;
  topic: string;
  prompt: string;
  engine: string;
  mentions: string[];
};

// Fixed, synthetic evidence for the design study. Never presented as live collection.
export function homeEvidence(brand: HomeBrand): HomeAnswer[] {
  const definition = HOME_BRANDS[brand];
  return Array.from({ length: 180 }, (_, index) => {
    const day = index - 90;

    return Object.entries(definition.topics).flatMap(([topic, prompts], topicIndex) =>
      prompts.flatMap((prompt, promptIndex) =>
        ENGINES.map((engine, engineIndex) => {
          const id = `${brand}:${day}:${topic}:${promptIndex}:${engine.id}`;
          const threshold =
            (brand === "deja" ? 18 : 24) + day * 0.3 + topicIndex * 8 - engineIndex * 3;
          const mentions = [definition.name, ...definition.rivals].filter(
            (_, index) =>
              hash(`${id}:${index}`) % 100 < (index === 0 ? threshold : 82 - index * 14),
          );
          return {
            id,
            date: new Date(Date.UTC(2026, 6, 16 + day)).toISOString().slice(0, 10),
            day,
            topic,
            prompt,
            engine: engine.id,
            mentions,
          };
        }),
      ),
    );
  }).flat();
}

export function mentionRate(answers: HomeAnswer[], name: string) {
  return answers.length
    ? (answers.filter((answer) => answer.mentions.includes(name)).length / answers.length) * 100
    : 0;
}
