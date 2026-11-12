import { defineConfig, devices } from "@playwright/test";

/**
 * Full-stack journeys against the real binary, embedded UI and Postgres, with no external
 * services. Browser-level UI behaviour with a mocked API lives in Vitest Browser Mode instead.
 */
export default defineConfig({
  testDir: "./e2e",
  testMatch: "**/*.e2e.ts",
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  workers: process.env.CI ? 2 : undefined,
  use: {
    baseURL: "http://127.0.0.1:8090",
    trace: "retain-on-failure",
  },
  projects: [
    {
      name: "chromium",
      use: {
        ...devices["Desktop Chrome"],
        permissions: ["clipboard-read", "clipboard-write"],
      },
    },
  ],
  webServer: {
    // `exec` hands the binary SIGTERM directly, so it drops its database schema on the way out.
    command:
      "cd .. && make ui-build && go build -o ui/.e2e/koi ./cmd/koi && go build -o ui/.e2e/server ./internal/e2e/server && exec ui/.e2e/server",
    url: "http://127.0.0.1:8090/health",
    reuseExistingServer: !process.env.CI,
    timeout: 180_000,
    gracefulShutdown: { signal: "SIGTERM", timeout: 10_000 },
  },
});
