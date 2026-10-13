import { useState } from "react";
import { BrandIcon } from "@/components/blocks/brand-icon";
import { C, Choice } from "../-components/common";
import { dateLabel, metrics, percent } from "../-lib/data";
import { useMock } from "../-lib/state";
import { visibilitySeries } from "./-visibility-data";

const COLORS = ["var(--exp-signal)", "#82aaff", "#c59aff", "#ffbc78", "#a9a69f"];

export function VisibilityChart({
  selectedPrompt,
  onSelectPrompt,
}: {
  selectedPrompt: string | null;
  onSelectPrompt: (id: string | null) => void;
}) {
  const { results, filters, prompts, competitors } = useMock();
  const [active, setActive] = useState<number | null>(null);
  const selection =
    selectedPrompt && prompts.some((prompt) => prompt.id === selectedPrompt)
      ? selectedPrompt
      : null;
  const answers = selection ? results.filter((answer) => answer.promptId === selection) : results;
  const brands = [
    { id: "tally", name: C.tally, domain: "tally.so" },
    ...competitors.filter((brand) => brand.confirmed),
  ];
  const days = Array.from({ length: filters.days }, (_, index) => 180 - filters.days + index);
  const comparison = visibilitySeries(
    results,
    filters,
    brands.map((brand) => brand.id),
    selection,
  );
  const series = brands.map((brand, index) => ({
    ...brand,
    color: COLORS[index % COLORS.length],
    points: comparison[index].points.map((point) => point.value),
  }));
  const index = Math.min(active ?? days.length - 1, days.length - 1);
  const width = 1000;
  const height = 240;
  const x = (index: number) => (index / (days.length - 1)) * width;
  const y = (value: number) => 8 + ((100 - value) / 100) * (height - 16);
  const path = (points: (number | null)[]) => {
    let connected = false;
    return points
      .map((value, index) => {
        if (value === null) {
          connected = false;
          return "";
        }
        const segment = `${connected ? "L" : "M"}${x(index)},${y(value)}`;
        connected = true;
        return segment;
      })
      .join(" ");
  };

  return (
    <figure className="tr-visibility">
      <Choice
        label={C.visibilityScope}
        value={selection ?? "all"}
        options={[
          { value: "all", label: C.overallVisibility },
          ...prompts.map((prompt) => ({ value: prompt.id, label: prompt.text })),
        ]}
        onChange={(value) => {
          onSelectPrompt(value === "all" ? null : value);
          setActive(null);
        }}
      />
      {metrics(answers).completed ? (
        <>
          <div className="tr-visibility-plot">
            <div className="tr-visibility-axis" aria-hidden="true">
              {[100, 75, 50, 25, 0].map((value) => (
                <span key={value}>{`${value}%`}</span>
              ))}
            </div>
            <svg
              viewBox={`0 0 ${width} ${height}`}
              preserveAspectRatio="none"
              role="img"
              aria-label={C.visibilityDescription}
              onPointerMove={(event) => {
                const box = event.currentTarget.getBoundingClientRect();
                setActive(
                  Math.max(
                    0,
                    Math.min(
                      days.length - 1,
                      Math.round(((event.clientX - box.left) / box.width) * (days.length - 1)),
                    ),
                  ),
                );
              }}
              onPointerLeave={() => setActive(null)}
            >
              {[0, 25, 50, 75, 100].map((value) => (
                <line
                  key={value}
                  x1={0}
                  x2={width}
                  y1={y(value)}
                  y2={y(value)}
                  className="exp-gridline"
                />
              ))}
              {series.map((brand) => (
                <path
                  key={brand.id}
                  d={path(brand.points)}
                  stroke={brand.color}
                  className="tr-visibility-line"
                />
              ))}
              {active !== null ? (
                <line
                  x1={x(index)}
                  x2={x(index)}
                  y1={0}
                  y2={height}
                  className="exp-chart-crosshair"
                />
              ) : null}
            </svg>
          </div>
          <div className="exp-chart-dates">
            <span>{dateLabel(days[0])}</span>
            <span>{dateLabel(days[Math.floor(days.length / 2)])}</span>
            <span>{dateLabel(days[days.length - 1])}</span>
          </div>
          <input
            className="tr-visibility-slider"
            type="range"
            min={0}
            max={days.length - 1}
            value={index}
            aria-label={C.inspectDate}
            aria-valuetext={series
              .map(
                (brand) =>
                  `${brand.name}: ${brand.points[index] === null ? C.notCollected : percent(brand.points[index])}`,
              )
              .join(", ")}
            onChange={(event) => setActive(Number(event.target.value))}
          />
          <output className="tr-visibility-date">{dateLabel(days[index])}</output>
          <ul className="tr-visibility-brands" aria-label={C.visibilityBrands}>
            {series.map((brand) => (
              <li key={brand.id}>
                <i style={{ background: brand.color }} aria-hidden="true" />
                <BrandIcon name={brand.name} domain={new URL(`https://${brand.domain}`).hostname} />
                <span>{brand.name}</span>
                <strong>{brand.points[index] === null ? "—" : percent(brand.points[index])}</strong>
              </li>
            ))}
          </ul>
        </>
      ) : (
        <p className="exp-muted">{C.visibilityNoData}</p>
      )}
      <figcaption>{C.visibilityDescription}</figcaption>
    </figure>
  );
}
