import { links } from "../copy";
import { business } from "./business";
import { LegalPage, List, Section, TextLink } from "./legal-page";

export function Terms() {
  return (
    <LegalPage
      title="Terms of Service"
      intro={
        <>
          <p>
            These terms are an agreement between you and {business.name}, a sole proprietor
            registered in Italy (VAT {business.vat}, {business.address}), who runs Koi Cloud ("Koi",
            "we"). They cover getkoi.co, app.getkoi.co, Koi's MCP server and the paid subscription
            ("the service").
          </p>
          <p>
            By creating an account or starting a subscription, you accept these terms. If you accept
            them for a company, you confirm you can bind it, and "you" means that company.
          </p>
          <p>
            The open source code is a separate thing: it is licensed under the{" "}
            <TextLink href={links.license}>MIT License</TextLink>, and these terms do not apply to
            copies you run yourself.
          </p>
        </>
      }
    >
      <Section id="business" title="1. Business use only">
        <p>
          Koi Cloud is for businesses and professionals, used for purposes related to a trade,
          business or profession. You must be at least 18. By signing up you confirm you are not
          acting as a consumer. The service is not offered to consumers.
        </p>
      </Section>

      <Section id="account" title="2. Your account">
        <List>
          <li>Give a valid email address and keep your password and connected agents secure.</li>
          <li>
            You are responsible for what happens through your account, including actions taken by
            agents you connect over MCP.
          </li>
          <li>Tell us promptly if you think someone else has access to your account.</li>
        </List>
      </Section>

      <Section id="service" title="3. What Koi does">
        <p>
          Koi asks ChatGPT, Gemini, Google AI Mode and Google AI Overviews the prompts you track,
          records whether they mention or cite your brand and which competitors they name, and turns
          the results into briefs for you or your agent.
        </p>
        <List>
          <li>
            AI answers change from one day to the next, and differ by person, place and time. Koi
            reports what the engines answered when it asked; it does not guarantee that answers,
            rankings or visibility will improve.
          </li>
          <li>
            The answers are written by third parties. We do not check them for accuracy and are not
            responsible for what they say about you or anyone else.
          </li>
          <li>Koi is not affiliated with or endorsed by OpenAI, Google or any engine it tracks.</li>
          <li>
            Suggestions Koi drafts with AI, such as prompts, competitors and briefs, can be wrong.
            Review them before you rely on them.
          </li>
        </List>
      </Section>

      <Section id="billing" title="4. Plans, trial and payment">
        <List>
          <li>
            Koi Cloud is a monthly subscription priced by the number of tracked prompts. Current
            prices are on <TextLink href="/#pricing">getkoi.co</TextLink> and include taxes.
          </li>
          <li>
            Before subscribing, you can run one free check. Daily tracking starts with a 7-day free
            trial, one per organization, which needs a card. Unless you cancel before the trial
            ends, the subscription starts and your card is charged.
          </li>
          <li>
            The subscription renews automatically every month until you cancel. A larger allowance
            takes effect immediately and is charged pro rata; a smaller one takes effect at the next
            billing date.
          </li>
          <li>
            Payments go through Stripe Managed Payments, which acts as merchant of record: Stripe,
            through its affiliate Lemon Squeezy LLC, charges your card, collects taxes, issues
            invoices and handles refunds, and its own terms apply to the purchase.
          </li>
          <li>
            We may change prices. We tell you at least 30 days before a new price applies to your
            subscription, and you can cancel before then.
          </li>
          <li>
            If a payment fails, tracking pauses until it succeeds. Your data stays in your account.
          </li>
        </List>
      </Section>

      <Section id="refunds" title="5. Cancellation and refunds">
        <List>
          <li>
            You can cancel at any time from billing. Your plan keeps running until the end of the
            period you paid for, and is not renewed.
          </li>
          <li>
            If you are not satisfied, ask for a refund within 30 days of your first paid charge and
            we refund that charge in full. This applies once per organization.
          </li>
          <li>
            Other charges, including renewals and partial months after you cancel, are not
            refundable, unless the law or the merchant of record requires a refund.
          </li>
          <li>
            Ask for refunds at{" "}
            <TextLink href={`mailto:${business.email}`}>{business.email}</TextLink> from the email
            on your account.
          </li>
        </List>
      </Section>

      <Section id="use" title="6. Acceptable use">
        <p>You agree not to:</p>
        <List>
          <li>use Koi for anything illegal, or to harass, defame or mislead anyone;</li>
          <li>
            get around limits, such as creating extra organizations for more free checks or trials;
          </li>
          <li>
            overload, probe or attack the service, or access it in ways other than the app, the
            documented API and MCP;
          </li>
          <li>resell the service or give access to people outside your organization;</li>
          <li>
            put other people's personal data or confidential information in prompts or brand
            details.
          </li>
        </List>
      </Section>

      <Section id="content" title="7. Your content">
        <p>
          Your brands, prompts, competitors and results remain yours. You give us the rights we need
          to store and process them to run the service for you, and nothing more. If you send us
          feedback or ideas, we may use them freely.
        </p>
        <p>
          How we handle personal data is described in the{" "}
          <TextLink href="/privacy">Privacy Policy</TextLink>.
        </p>
      </Section>

      <Section id="availability" title="8. Availability and changes">
        <p>
          We work to keep Koi running but do not promise any level of availability. Engines and
          providers Koi relies on can change or stop answering, which can interrupt collection. We
          may change or remove features; if a change materially reduces what you pay for, you may
          cancel and ask for a pro rata refund of the unused period.
        </p>
      </Section>

      <Section id="suspension" title="9. Suspension and termination">
        <p>
          We may suspend or close an account that breaks these terms, puts the service or others at
          risk, or is not paid, and we tell you why unless the law prevents it. You can close your
          account at any time by writing to us; we then delete its data as described in the Privacy
          Policy. Sections 3, 7, 10, 11 and 13 survive termination.
        </p>
      </Section>

      <Section id="liability" title="10. Warranties and liability">
        <p>
          The service is provided as is and as available. To the extent the law allows, we make no
          other warranties, express or implied.
        </p>
        <p>
          To the extent the law allows, we are not liable for indirect or consequential losses, such
          as lost profits, revenue, data or business opportunities. Our total liability for any
          claim is limited to what you paid us in the 12 months before the claim arose.
        </p>
        <p>
          Nothing in these terms limits liability for fraud, wilful misconduct or gross negligence,
          or any liability that cannot be limited under Italian law (including Art. 1229 of the
          Italian Civil Code).
        </p>
      </Section>

      <Section id="indemnity" title="11. Your responsibility to us">
        <p>
          You will cover our reasonable costs if a third party brings a claim against us because of
          content you put into Koi or your breach of these terms.
        </p>
      </Section>

      <Section id="changes" title="12. Changes to these terms">
        <p>
          We may update these terms. For material changes we email account holders at least 30 days
          before they take effect. If you keep using Koi after that, the new terms apply; if you do
          not accept them, cancel before they take effect.
        </p>
      </Section>

      <Section id="law" title="13. Governing law">
        <p>
          These terms are governed by Italian law. The courts of Milan, Italy, have exclusive
          jurisdiction over any dispute about them.
        </p>
      </Section>

      <Section id="contact" title="14. Contact">
        <p>
          {business.name}, {business.address}. VAT {business.vat}.{" "}
          <TextLink href={`mailto:${business.email}`}>{business.email}</TextLink>
        </p>
      </Section>
    </LegalPage>
  );
}
