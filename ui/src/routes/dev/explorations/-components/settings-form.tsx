import { useState } from "react";
import type { Brand, CreateBrand } from "@/api/brands.api";
import { BrandForm } from "@/components/blocks/brand-form";
import codes from "../-lib/country-codes.json";
import C from "../-locales/mvp.en.json";

const names = new Intl.DisplayNames(["en"], { type: "region" });
const countryOptions = codes.map((code) => ({ code, name: names.of(code) ?? code }));

export function SettingsForm({
  brand,
  onSave,
}: {
  brand: Brand;
  onSave: (body: CreateBrand) => void;
}) {
  const [saved, setSaved] = useState(false);

  return (
    <div className="hc-settings-form">
      <BrandForm
        brand={brand}
        countryOptions={countryOptions}
        submitLabel={C.save}
        onSubmit={async (request) => {
          setSaved(false);
          onSave(request);
          setSaved(true);
        }}
      />
      <output className="hc-caption">{saved ? C.saved : C.previewNote}</output>
    </div>
  );
}
