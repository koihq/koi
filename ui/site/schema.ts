export type Question = { q: string; a: string };

/** Structured data for the pages. No ratings or reviews: we do not collect any. */

const context = "https://schema.org";

export const softwareSchema = {
  "@context": context,
  "@type": "SoftwareApplication",
  name: "Koi",
  applicationCategory: "BusinessApplication",
  operatingSystem: "Web",
  url: "https://getkoi.co",
  license: "https://opensource.org/license/mit",
  codeRepository: "https://github.com/koihq/koi",
  description:
    "Open source tool that measures whether ChatGPT, Gemini, Google AI Mode and Google AI Overviews mention or cite a brand, who they recommend instead, and hands a markdown brief to the founder's agent.",
};

export function faqSchema(items: Question[]) {
  return {
    "@context": context,
    "@type": "FAQPage",
    mainEntity: items.map((item) => ({
      "@type": "Question",
      name: item.q,
      acceptedAnswer: { "@type": "Answer", text: item.a },
    })),
  };
}

export function articleSchema(page: {
  title: string;
  description: string;
  url: string;
  updated: string;
}) {
  return {
    "@context": context,
    "@type": "Article",
    headline: page.title,
    description: page.description,
    mainEntityOfPage: page.url,
    dateModified: page.updated,
    author: { "@type": "Organization", name: "Koi", url: "https://getkoi.co" },
    publisher: { "@type": "Organization", name: "Koi", url: "https://getkoi.co" },
  };
}

export function breadcrumbSchema(trail: { name: string; url: string }[]) {
  return {
    "@context": context,
    "@type": "BreadcrumbList",
    itemListElement: trail.map((step, index) => ({
      "@type": "ListItem",
      position: index + 1,
      name: step.name,
      item: step.url,
    })),
  };
}
