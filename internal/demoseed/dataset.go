package demoseed

// These are original, illustrative answers based on the sources documented in
// README.md, verified on 2026-11-02. No engine or provider is contacted.
const repository = "https://github.com/koihq/koi"

type competitorSpec struct {
	name    string
	domain  string
	summary string
}

var competitors = []competitorSpec{
	{
		name:    "Profound",
		domain:  "tryprofound.com",
		summary: "Profound provides AI search visibility analysis and tools for understanding how brands appear in AI answers.",
	},
	{
		name:    "Peec AI",
		domain:  "peec.ai",
		summary: "Peec AI tracks brand visibility, position, and sentiment in AI search, with prompt and competitor analysis.",
	},
	{
		name:    "OtterlyAI",
		domain:  "otterly.ai",
		summary: "OtterlyAI combines prompt research, AI search monitoring, and analysis of brand mentions and website citations.",
	},
	{
		name:    "Scrunch",
		domain:  "scrunch.com",
		summary: "Scrunch monitors brand presence and citations in AI search and offers insights into how AI systems interpret content.",
	},
	{
		name:    "AthenaHQ",
		domain:  "athenahq.ai",
		summary: "AthenaHQ provides AI search visibility tracking, competitive intelligence, and workflows for content optimization.",
	},
	{
		name:    "Rankscale",
		domain:  "rankscale.ai",
		summary: "Rankscale tracks AI visibility, citations, and competitors across AI engines and includes technical site audits.",
	},
}

type promptSpec struct {
	text  string
	topic int
}

// 0–25: active; 26–28: inactive; 29–31: drafts; 32–33: historical only.
var prompts = []promptSpec{
	{"What are the best tools to track whether ChatGPT recommends my SaaS?", 0},
	{"Which AEO tools help a bootstrapped founder understand AI search visibility?", 1},
	{"How can I compare my brand's citations across Perplexity, Gemini, and ChatGPT?", 2},
	{"Are there self-hosted alternatives to managed AI visibility platforms?", 3},
	{"How do Koi, Profound, Peec AI, and OtterlyAI approach AI visibility tracking?", 4},
	{"Which tools distinguish a brand mention from a citation in an AI answer?", 2},
	{"How can a developer turn AI visibility findings into an actionable brief?", 5},
	{"What should a small SaaS team measure before investing in AEO content?", 1},
	{"How do I find out which competitors are recommended instead of my product?", 4},
	{"Which tools monitor brand presence in Google AI Overviews?", 0},
	{"What is the difference between AI search visibility and traditional SEO rankings?", 6},
	{"How can I monitor AI answers without outsourcing my entire content workflow?", 5},
	{"What should I look for when choosing an AI search monitoring platform?", 4},
	{"How do I build a repeatable prompt set for tracking a B2B SaaS brand?", 7},
	{"Which AI visibility tools support comparing several answer engines?", 0},
	{"How can I inspect the sources behind an AI recommendation?", 2},
	{"What are the tradeoffs between self-hosted and managed AEO software?", 3},
	{"How can an existing coding agent use AI visibility research?", 5},
	{"Why might my brand appear in Perplexity but not in ChatGPT?", 6},
	{"How should I interpret a visibility drop when some AI requests failed?", 7},
	{"Which AEO workflows are practical for founders without a marketing team?", 1},
	{"How do I compare AI visibility this month with the previous month?", 7},
	{"Can an open-source AEO tool preserve answers after a tracked prompt is deleted?", 3},
	{"What evidence should an AI visibility report include besides a percentage?", 2},
	{"How can I separate branded prompts from category discovery questions?", 7},
	{"Which tools help me understand who wins AI recommendations in my category?", 4},
	{"How can I track early launch messaging in AI-generated product recommendations?", 1},
	{"What should I measure during a short AEO content experiment?", 7},
	{"How do I audit an older comparison page for AI citation opportunities?", 2},
	{"How should a developer-first startup evaluate an AEO platform?", 4},
	{"What would a weekly AI visibility review for a solo founder look like?", 5},
	{"How can I use prompt-level evidence to prioritize documentation updates?", 5},
	{"Which AI search tools should I evaluate before launching my first SaaS?", 0},
	{"How can I monitor an early positioning experiment across answer engines?", 7},
}

var topicNotes = []string{
	"Track a consistent set of category questions across ChatGPT, Perplexity, Gemini, and Google AI Overviews. Keep the full answer and its source links so you can inspect why a brand was included.",
	"Start with the questions a prospective customer asks before choosing a product. A small founder-led team can review missing recommendations, inspect competing answers, and choose one concrete documentation improvement per week.",
	"A mention means the brand is named in the answer. A citation means a source URL points to the brand's website or repository. Keep those signals separate: an answer can discuss a product without linking to it, or cite a source without naming it in the prose.",
	"Self-hosting gives the operator control over the application's database and deployment. It still requires an external collection provider for live engine answers. Compare operating effort, data ownership, and integration needs before choosing a managed service.",
	"Compare platforms using the same prompts, country, engines, and observation window. Inspect the saved answers as well as aggregate visibility. A product with strong category mentions can still be absent from the specific questions your buyers ask.",
	"Preserve the prompt, answer, cited URLs, and competing recommendations. Turn that evidence into a Markdown brief with a specific page to improve, the unsupported question to answer, and acceptance criteria that the founder's existing agent can execute.",
	"AI visibility measures sampled answers rather than a fixed search ranking. Engines can retrieve different sources and produce different recommendations. Review each engine separately, and use repeated observations before treating a change as a durable trend.",
	"Keep prompt wording stable while measuring a change, and compare equal observation windows. Distinguish missing observations and provider failures from measured answers that omit the brand. Otherwise an outage can look like a visibility decline.",
}
