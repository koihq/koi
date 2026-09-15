import "i18next";
import type { defaultNS, resources } from "@/config/i18n";

declare module "i18next" {
  interface CustomTypeOptions {
    defaultNS: typeof defaultNS;
    resources: (typeof resources)["en"];
    interpolationPrefix: "{";
    interpolationSuffix: "}";
    enableSelector: false;
  }
}
