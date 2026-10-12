import { createFileRoute, Link } from "@tanstack/react-router";
import { KoiMark } from "@/components/blocks/koi-mark";
import { DIRECTIONS } from "./-lib/data";
import C from "./-locales/en.json";
import "./-styles.css";

export const Route = createFileRoute("/dev/explorations/")({ component: Gallery });

const DESCRIPTIONS = {
  visitors: "A quiet signal. Expansive charts, minimal chrome, and a floating dock.",
  cloudflare: "A clear control plane. Labeled navigation, operational context, and guided action.",
  fey: "An editorial perspective. A dark canvas, competitive landscape, and an evidence-led feed.",
  "trade-republic":
    "Your own research workspace. Linked panels, dense evidence, and layouts you can rearrange.",
};

function Gallery() {
  return (
    <main className="exp-gallery">
      <header>
        <KoiMark className="size-8" />
        <span>{C.explorations}</span>
      </header>
      <h1>{C.galleryTitle}</h1>
      <p>{C.galleryLead}</p>
      <div className="exp-gallery-grid">
        {DIRECTIONS.map((direction, index) => (
          <Link
            key={direction}
            to="/dev/explorations/$direction/$section"
            params={{ direction, section: "overview" }}
            search={{ state: "populated" }}
            className={`exp-gallery-item ${direction}`}
          >
            <div className="exp-gallery-preview" aria-hidden={true}>
              <div className="exp-preview-nav" />
              <div className="exp-preview-content">
                <span />
                <span />
                <svg viewBox="0 0 200 70">
                  <title>{C[direction]}</title>
                  <path d="M0 62 L20 55 L40 60 L60 34 L80 46 L100 26 L120 33 L140 12 L160 25 L180 14 L200 4" />
                </svg>
                <div>
                  <i />
                  <i />
                  <i />
                </div>
              </div>
            </div>
            <div className="exp-gallery-label">
              <span>{`0${index + 1}`}</span>
              <h2>{C[direction]}</h2>
              <span>{C.open}</span>
            </div>
            <p>{DESCRIPTIONS[direction]}</p>
          </Link>
        ))}
      </div>
      <footer>{C.sampleEvidence}</footer>
    </main>
  );
}
