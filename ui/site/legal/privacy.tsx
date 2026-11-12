import { business } from "./business";
import { CookieSettingsButton } from "./cookie-settings";
import { Facts, LegalPage, List, Section, TextLink } from "./legal-page";

const processors = [
  {
    label: "netcup",
    value: "Hosts the server and the database. Data centers in the EU (Germany or Austria).",
    href: "https://www.netcup.com/en/contact/privacy-policy",
  },
  {
    label: "Cloudflare",
    value:
      "Domain, network delivery, access control and encrypted database backups (stored in the EU, deleted after 30 days).",
    href: "https://www.cloudflare.com/privacypolicy/",
  },
  {
    label: "Stripe",
    value:
      "Checkout, payments, taxes, invoices and refunds, as merchant of record through Stripe Managed Payments. Receives the email, billing and card details you enter at checkout. United States and Ireland.",
    href: "https://stripe.com/privacy",
  },
  {
    label: "Cloro",
    value:
      "Asks the AI engines your tracked prompts and returns their answers and sources. Receives the prompt text and target country.",
    href: "https://cloro.dev/privacy",
  },
  {
    label: "OpenRouter",
    value:
      "Runs the language models that read your website, draft prompts and competitors, and check whether an answer is about your brand. Receives your brand details, website address, prompts and AI answers. United States.",
    href: "https://openrouter.ai/privacy",
  },
  {
    label: "Resend",
    value:
      "Sends account emails: welcome, first results, password reset and password changed. Receives your email address and your brand name. United States.",
    href: "https://resend.com/legal/privacy-policy",
  },
  {
    label: "PostHog",
    value:
      "Product analytics, session recordings of setup when you allow them, and support chat. Receives your user ID, email address, usage events and IP address, used to estimate your country. EU (Germany).",
    href: "https://posthog.com/privacy",
  },
  {
    label: "Logo.dev",
    value:
      "Shows company logos in the app. Your browser requests each logo directly, so Logo.dev receives the brand and competitor domains and your IP address. United States.",
    href: "https://www.logo.dev/privacy",
  },
  {
    label: "GitHub",
    value:
      "If you give a public repository instead of a website, Koi reads it through the GitHub API. United States.",
    href: "https://docs.github.com/en/site-policy/privacy-policies/github-general-privacy-statement",
  },
];

export function Privacy() {
  return (
    <LegalPage
      title="Privacy Policy"
      intro={
        <>
          <p>
            This policy explains what personal data Koi Cloud collects, why, who receives it, and
            the rights you have over it. It covers getkoi.co, app.getkoi.co and Koi's MCP server.
          </p>
          <p>
            It does not cover copies of Koi that other people run on their own servers from the open
            source code. Whoever runs such a copy is responsible for the data in it.
          </p>
        </>
      }
    >
      <Section id="controller" title="Who is responsible">
        <p>The data controller is:</p>
        <Facts
          rows={[
            { label: "Name", value: `${business.name} (sole proprietor)` },
            { label: "Address", value: business.address },
            { label: "VAT number", value: business.vat },
            {
              label: "Email",
              value: <TextLink href={`mailto:${business.email}`}>{business.email}</TextLink>,
            },
          ]}
        />
        <p>Write to that address for anything about your data.</p>
      </Section>

      <Section id="data" title="What we collect">
        <Facts
          rows={[
            {
              label: "Account",
              value:
                "Your email address and your password, stored only as a one-way hash. Your organization's name, which starts as the part of your email before the @.",
            },
            {
              label: "Your brand",
              value:
                "The brands you track: name, website or public GitHub repository, country and description. Also your prompts, competitors and the AI answers and sources Koi collects for them. If you trade under your own name, some of this is personal data.",
            },
            {
              label: "Connected agents",
              value:
                "When you connect an agent over MCP: the client's name and redirect addresses, and access tokens, stored only as hashes.",
            },
            {
              label: "Billing",
              value:
                "Your subscription status, plan, trial and billing dates, and the Stripe IDs that link them. Koi never receives your card details.",
            },
            {
              label: "Usage",
              value:
                "Events about how the product is used, such as signing up, finishing a setup step, starting a trial, changing plan and agent tool calls. With your consent, also page views and recordings of the setup screens.",
            },
            {
              label: "Security",
              value:
                "To limit abuse, Koi counts requests per IP address and per email address. These counters live only in memory and reset whenever the server restarts. Server logs record the request path, status and timing, not your IP address.",
            },
          ]}
        />
        <p>
          You give us your email address and password to create an account; without them we cannot
          provide the service. Everything else is optional or generated as you use Koi.
        </p>
      </Section>

      <Section id="purposes" title="Why we use it">
        <Facts
          rows={[
            {
              label: "To provide Koi",
              value:
                "Your account, sign-in, tracking your brand, the emails the service needs, billing and support. Legal basis: performing our contract with you (GDPR Art. 6(1)(b)).",
            },
            {
              label: "To keep Koi secure",
              value:
                "Rate limits, fraud prevention and preventing abuse of the free check. Legal basis: our legitimate interest in protecting the service (Art. 6(1)(f)).",
            },
            {
              label: "To improve Koi",
              value:
                "Usage events sent by our server, without cookies, so we can see which steps work and which do not. Legal basis: our legitimate interest in improving the product (Art. 6(1)(f)). You can object at any time.",
            },
            {
              label: "Analytics in your browser",
              value:
                "The analytics cookie and setup recordings. Legal basis: your consent (Art. 6(1)(a)), which you can withdraw at any time.",
            },
            {
              label: "Legal obligations",
              value:
                "Keeping records the law requires and answering lawful requests from authorities (Art. 6(1)(c)).",
            },
          ]}
        />
        <p>We do not sell your data, use it for advertising, or share it with data brokers.</p>
      </Section>

      <Section id="ai" title="How Koi uses AI">
        <List>
          <li>
            Koi sends your tracked prompts to ChatGPT, Gemini, Google AI Mode and Google AI
            Overviews through Cloro, and stores the answers and sources they return.
          </li>
          <li>
            When you add a brand, Koi and OpenRouter read your website or public repository to
            suggest prompts and competitors. Nothing is tracked until you confirm the suggestions.
          </li>
          <li>
            A language model on OpenRouter checks whether an answer that names your brand is really
            about it.
          </li>
        </List>
        <p>
          None of this makes decisions about you as a person. Do not put other people's personal
          data in prompts or brand descriptions.
        </p>
      </Section>

      <Section id="processors" title="Who receives your data">
        <p>
          These providers process data for us, only to provide their service, under data processing
          agreements:
        </p>
        <Facts
          rows={processors.map((processor) => ({
            label: <TextLink href={processor.href}>{processor.label}</TextLink>,
            value: processor.value,
          }))}
        />
        <p>
          We may also disclose data when the law requires it, or to a buyer if Koi is ever sold, in
          which case this policy keeps applying to your data.
        </p>
      </Section>

      <Section id="transfers" title="Transfers outside the EU">
        <p>
          Some providers above are in the United States or elsewhere outside the European Economic
          Area. Those transfers rely on the EU-US Data Privacy Framework where the provider is
          certified, or on the European Commission's Standard Contractual Clauses. Write to us for a
          copy of the safeguards.
        </p>
      </Section>

      <Section id="retention" title="How long we keep it">
        <Facts
          rows={[
            {
              label: "Account and brands",
              value: "Until you ask us to delete your account.",
            },
            {
              label: "Backups",
              value: "Up to 30 days after deletion, until the backup expires.",
            },
            {
              label: "Sessions",
              value: "8 hours from sign-in.",
            },
            {
              label: "Tokens",
              value:
                "Password reset links expire after 1 hour. Agent access tokens expire after 1 hour and their refresh tokens after 30 days, and are then deleted.",
            },
            {
              label: "Background jobs",
              value:
                "Up to 7 days after they finish, including the address a reset email was sent to.",
            },
            {
              label: "Analytics",
              value: "Deleted with your account, and otherwise kept for no more than 24 months.",
            },
            {
              label: "Payments",
              value:
                "Stripe keeps transaction records for as long as tax and financial law requires.",
            },
          ]}
        />
      </Section>

      <Section id="rights" title="Your rights">
        <p>Under the GDPR you can ask us to:</p>
        <List>
          <li>give you a copy of your data, also in a portable format;</li>
          <li>correct data that is wrong;</li>
          <li>delete your account and its data;</li>
          <li>
            restrict how we use your data, or stop using it where we rely on legitimate interest;
          </li>
          <li>withdraw your consent, without affecting what happened before.</li>
        </List>
        <p>
          Email <TextLink href={`mailto:${business.email}`}>{business.email}</TextLink> from the
          address on your account. We answer within one month, free of charge. To delete an account
          with an active subscription, cancel it first in billing.
        </p>
        <p>
          You can also complain to the Italian data protection authority, the{" "}
          <TextLink href="https://www.garanteprivacy.it">
            Garante per la protezione dei dati personali
          </TextLink>
          , or to the authority where you live or work.
        </p>
      </Section>

      <Section id="cookies" title="Cookies">
        <Facts
          rows={[
            {
              label: "koi_session",
              value:
                "Keeps you signed in to app.getkoi.co. Necessary. Lasts 8 hours. Holds only a random token.",
            },
            {
              label: "koi_consent",
              value:
                "Remembers whether you allowed analytics, on getkoi.co and app.getkoi.co. Necessary. Lasts 180 days.",
            },
            {
              label: "ph_* (PostHog)",
              value:
                "Recognizes your browser across visits so we can see which pages help people get started. Analytics. Set only if you allow it. Lasts up to 1 year.",
            },
          ]}
        />
        <p>
          If you decline, PostHog counts the page view anonymously and stores nothing on your
          device. You can change your choice at any time.
        </p>
        <CookieSettingsButton className="justify-self-start text-foreground underline decoration-muted-foreground underline-offset-4 transition-colors hover:decoration-foreground" />
      </Section>

      <Section id="security" title="Security">
        <p>
          Connections use HTTPS. Passwords are hashed with Argon2id, and session and access tokens
          are stored only as hashes. Only the operator can reach the server and database. No system
          is perfectly secure; if a breach puts your data at risk, we will tell you and the
          authority as the law requires.
        </p>
      </Section>

      <Section id="age" title="Business use and age">
        <p>
          Koi Cloud is for businesses and professionals, and you must be at least 18 to use it. We
          do not knowingly collect data from children.
        </p>
      </Section>

      <Section id="changes" title="Changes">
        <p>
          When this policy changes, we update the date above. If a change matters, we tell account
          holders by email before it takes effect.
        </p>
      </Section>
    </LegalPage>
  );
}
