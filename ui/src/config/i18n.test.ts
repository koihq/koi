import { expect, it } from "vitest";
import i18n from "@/config/i18n";

it("interpolates the brand page count", () => {
  expect(i18n.t("settings.page", { ns: "brand", current: 2, total: 3 })).toBe("Page 2 of 3");
});
