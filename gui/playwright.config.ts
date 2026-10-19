import { defineConfig, devices } from "@playwright/test";

// Smoke tests for the admin UI against a real API.
//
// They expect a GoHead API on E2E_API_URL (default http://localhost:8080)
// with an empty database, started with the administrator of e2e/fixtures.ts
// (GOHEAD_ADMIN_USERNAME / GOHEAD_ADMIN_PASSWORD): the global setup logs in as
// that user and creates the content the tests use. The UI is built and
// served by `vite preview`.
const apiURL = process.env.E2E_API_URL ?? "http://localhost:8080";

export default defineConfig({
  testDir: "./e2e",
  globalSetup: "./e2e/global-setup.ts",
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["github"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL: "http://localhost:4173",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [
    {
      name: "chromium",
      use: {
        ...devices["Desktop Chrome"],
        // Use an installed Chromium instead of Playwright's download.
        launchOptions: process.env.E2E_CHROMIUM ? { executablePath: process.env.E2E_CHROMIUM } : {},
      },
    },
  ],
  webServer: {
    command: "npm run build && npx vite preview --port 4173 --strictPort",
    url: "http://localhost:4173",
    reuseExistingServer: !process.env.CI,
    timeout: 180_000,
    env: { VITE_API_BASE_URL: apiURL },
  },
});
