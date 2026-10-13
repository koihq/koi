import { ENGINES, hash } from "./data";

export const HOME_BRANDS = {
  deja: {
    name: "Deja",
    rivals: ["zsh-autosuggestions"],
    topics: {
      alternatives: [
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
        "Which AI visibility tools work with my existing agent?",
        "How can I give my coding agent an AI visibility brief?",
      ],
      competitors: [
        "How can I see who AI recommends instead of my brand?",
        "Which tools compare competitor mentions in AI answers?",
      ],
    },
  },
  // Additional fictional workspace brands for exercising selection and overflow.
  orbit: {
    name: "Orbit",
    rivals: ["DocNest", "PageKit"],
    topics: {
      discovery: [
        "Which tools help small teams publish developer documentation?",
        "What documentation tools work well for an API product?",
      ],
    },
  },
  sprout: {
    name: "Sprout",
    rivals: ["Letterbox", "MailGarden"],
    topics: {
      discovery: [
        "Which newsletter tools work best for independent creators?",
        "How can I start a paid newsletter with a small audience?",
      ],
    },
  },
  forma: {
    name: "Forma",
    rivals: ["Formleaf", "Inputly"],
    topics: {
      discovery: [
        "Which form builders are good for collecting customer feedback?",
        "What is a simple form builder for a small business?",
      ],
    },
  },
  northstar: {
    name: "Northstar",
    rivals: ["MetricNest", "SignalDesk"],
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
  return Array.from({ length: 90 }, (_, day) =>
    Object.entries(definition.topics).flatMap(([topic, prompts], topicIndex) =>
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
    ),
  ).flat();
}

export function mentionRate(answers: HomeAnswer[], name: string) {
  return answers.length
    ? (answers.filter((answer) => answer.mentions.includes(name)).length / answers.length) * 100
    : 0;
}
