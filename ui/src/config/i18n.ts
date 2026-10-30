import i18n from "i18next";
import { initReactI18next } from "react-i18next";

import auth from "@/locales/en/auth.json";
import billing from "@/locales/en/billing.json";
import brand from "@/locales/en/brand.json";
import common from "@/locales/en/common.json";
import connections from "@/locales/en/connections.json";
import dev from "@/locales/en/dev.json";
import errors from "@/locales/en/errors.json";
import home from "@/locales/en/home.json";
import management from "@/locales/en/management.json";
import visibility from "@/locales/en/visibility.json";

export const defaultNS = "common";

export const resources = {
  en: { common, auth, billing, brand, connections, errors, home, dev, management, visibility },
} as const;

i18n.use(initReactI18next).init({
  resources,
  lng: "en",
  fallbackLng: "en",
  defaultNS,
  ns: [
    "common",
    "auth",
    "billing",
    "brand",
    "connections",
    "errors",
    "home",
    "dev",
    "management",
    "visibility",
  ],
  interpolation: {
    escapeValue: false,
    prefix: "{",
    suffix: "}",
  },
});

export default i18n;
