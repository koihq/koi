import { Fragment } from "react";
import { BrandPill } from "@/components/blocks/brand-pill";
import { ENGINES } from "../-lib/data";
import { HOME_LOGOS } from "../-lib/home-evidence";

const brands = [
  ...Object.entries(HOME_LOGOS).map(([name, logo]) => ({ name, ...logo })),
  ...ENGINES,
];
const brandsByName = new Map(brands.map((brand) => [brand.name.toLowerCase(), brand]));
const names = brands
  .map((brand) => brand.name)
  .sort((a, b) => b.length - a.length)
  .map((name) => name.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"));
const brandPattern = new RegExp(
  `(?<![\\p{L}\\p{N}_])(${names.join("|")})(?![\\p{L}\\p{N}_])`,
  "giu",
);

export function splitHomeBrandText(text: string) {
  let offset = 0;

  return text
    .split(brandPattern)
    .filter(Boolean)
    .map((part) => {
      const start = offset;
      offset += part.length;

      return { text: part, offset: start, brand: brandsByName.get(part.toLowerCase()) };
    });
}

export function HomeBrandText({ text }: { text: string }) {
  return splitHomeBrandText(text).map((part) =>
    part.brand ? (
      <BrandPill key={part.offset} {...part.brand} name={part.text} className="hc-inline-brand" />
    ) : (
      <Fragment key={part.offset}>{part.text}</Fragment>
    ),
  );
}
