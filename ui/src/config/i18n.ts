import i18n from "i18next";
import { initReactI18next } from "react-i18next";

import auth from "@/locales/en/auth.json";
import common from "@/locales/en/common.json";
import dev from "@/locales/en/dev.json";
import errors from "@/locales/en/errors.json";
import home from "@/locales/en/home.json";

export const defaultNS = "common";
export const resources = {
  en: { common, auth, errors, home, dev },
} as const;

i18n.use(initReactI18next).init({
  resources,
  lng: "en",
  fallbackLng: "en",
  defaultNS,
  ns: ["common", "auth", "errors", "home", "dev"],
  interpolation: {
    escapeValue: false,
    prefix: "{",
    suffix: "}",
  },
});

export default i18n;
