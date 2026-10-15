import { useRef, useState } from "react";
import type { CreateBrand } from "@/api/brands.api";
import type { CreateCompetitor } from "@/api/competitors.api";
import {
  confirmItem,
  createPreviewBrand,
  type ExplorationState,
  finishReview,
  type PreviewBrand,
  SUGGESTIONS,
} from "../-lib/exploration-fixtures";
import { HOME_BRANDS, type HomeBrand } from "../-lib/home-evidence";
import C from "../-locales/mvp.en.json";

export function useExplorationState(scenario?: ExplorationState) {
  const [brands, setBrands] = useState(
    () =>
      Object.fromEntries(
        (Object.keys(HOME_BRANDS) as HomeBrand[]).map((id) => [
          id,
          createPreviewBrand(id, scenario),
        ]),
      ) as Record<HomeBrand, PreviewBrand>,
  );
  const failed = useRef(false);

  function update(brand: HomeBrand, change: (data: PreviewBrand) => PreviewBrand) {
    setBrands((previous) => ({ ...previous, [brand]: change(previous[brand]) }));
  }

  function checkSave() {
    if (scenario === "save-error" && !failed.current) {
      failed.current = true;
      throw new Error(C.saveError);
    }
  }

  function savePrompt(brand: HomeBrand, text: string, id?: string) {
    checkSave();
    const prompt = {
      id: crypto.randomUUID(),
      brandId: brand,
      text,
      source: "manual" as const,
      confirmed: true,
    };
    update(brand, (data) => ({
      ...data,
      prompts: id
        ? data.prompts.map((item) => (item.id === id ? { ...item, text } : item))
        : [...data.prompts, prompt],
    }));
  }

  function saveCompetitor(brand: HomeBrand, request: CreateCompetitor, id?: string) {
    checkSave();
    const identity = (value: string) => value.trim().toLowerCase().replace(/\/$/, "");
    if (
      brands[brand].competitors.some(
        (item) =>
          item.id !== id &&
          ((request.domain && identity(item.domain) === identity(request.domain)) ||
            (request.githubRepo && identity(item.githubRepo) === identity(request.githubRepo)) ||
            item.name.toLowerCase() === request.name.toLowerCase()),
      )
    )
      throw new Error(C.duplicate);

    const competitor = {
      ...request,
      domain: request.domain ?? "",
      githubRepo: request.githubRepo ?? "",
      aliases: request.aliases ?? [],
      additionalUrls: request.additionalUrls ?? [],
      id: crypto.randomUUID(),
      brandId: brand,
      source: "manual" as const,
      confirmed: true,
    };
    update(brand, (data) => ({
      ...data,
      competitors: id
        ? data.competitors.map((item) =>
            item.id === id
              ? { ...competitor, id: item.id, source: item.source, confirmed: item.confirmed }
              : item,
          )
        : [...data.competitors, competitor],
    }));
  }

  function draft(brand: HomeBrand, kind: "prompts" | "competitors") {
    const data = brands[brand];
    const suggestion = SUGGESTIONS[brand];
    const text = `When should I choose ${data.settings.name} over its alternatives?`;
    const exists =
      kind === "prompts"
        ? data.prompts.some((item) => item.text === text)
        : data.competitors.some((item) => item.domain === suggestion.domain);
    if (exists || scenario === "draft-empty") return 0;

    const id = crypto.randomUUID();

    update(brand, (current) =>
      kind === "prompts"
        ? {
            ...current,
            prompts: [
              ...current.prompts,
              {
                id,
                brandId: brand,
                text,
                source: "drafted",
                confirmed: false,
              },
            ],
          }
        : {
            ...current,
            competitors: [
              ...current.competitors,
              {
                id,
                brandId: brand,
                ...suggestion,
                githubRepo: "",
                aliases: [],
                additionalUrls: [],
                source: "drafted",
                confirmed: false,
              },
            ],
          },
    );

    return 1;
  }

  return {
    brands,
    savePrompt,
    saveCompetitor,
    draft,
    saveBrand: (brand: HomeBrand, request: CreateBrand) => {
      checkSave();
      update(brand, (data) => ({ ...data, settings: { ...data.settings, ...request } }));
    },
    review: (brand: HomeBrand, kind: "prompts" | "competitors") =>
      update(brand, (data) => finishReview(data, kind)),
    confirm: (brand: HomeBrand, kind: "prompts" | "competitors", id: string) =>
      update(brand, (data) => confirmItem(data, kind, id)),
    remove: (brand: HomeBrand, kind: "prompts" | "competitors", id: string) => {
      checkSave();
      update(brand, (data) => ({ ...data, [kind]: data[kind].filter((item) => item.id !== id) }));
    },
  };
}
