import "vitest/browser";

declare module "vitest/browser" {
  interface BrowserCommands {
    emulateMedia(options: { reducedMotion: "reduce" | "no-preference" }): Promise<void>;
  }
}
