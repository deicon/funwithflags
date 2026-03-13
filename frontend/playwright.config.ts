import { defineConfig, devices } from "@playwright/test";

/**
 * Playwright E2E tests for funwithflags.
 *
 * Default setup (in-memory backend):
 *   cd frontend && npx playwright test
 *   → Starts Go backend (STORAGE_TYPE=memory, port 9090) and Vite dev server automatically.
 *
 * Against docker-compose:
 *   docker-compose up -d
 *   API_URL=http://localhost:8080 npx playwright test --config playwright.config.ts
 *
 * The API_URL env var controls which backend the Vite proxy targets.
 */

const apiUrl = process.env.API_URL || "http://localhost:9090";

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: process.env.CI ? "github" : "list",
  timeout: 30_000,

  use: {
    baseURL: "http://localhost:5173",
    trace: "on-first-retry",
    screenshot: "only-on-failure",
  },

  projects: [
    { name: "setup", testMatch: /.*\.setup\.ts/, teardown: "teardown" },
    { name: "teardown", testMatch: /.*\.teardown\.ts/ },
    {
      name: "chromium",
      use: {
        ...devices["Desktop Chrome"],
        storageState: "./e2e/.auth/user.json",
      },
      dependencies: ["setup"],
    },
  ],

  webServer: [
    // Go backend with in-memory storage (skipped if API_URL is set externally)
    ...(!process.env.API_URL
      ? [
          {
            command:
              "STORAGE_TYPE=memory PORT=9090 CORS_ALLOWED_ORIGINS=http://localhost:5173 go run ../cmd/server/main.go",
            url: "http://localhost:9090/healthz",
            reuseExistingServer: true,
            timeout: 30_000,
          },
        ]
      : []),
    // Vite dev server
    {
      command: `API_URL=${apiUrl} npm run dev`,
      url: "http://localhost:5173",
      reuseExistingServer: !process.env.CI,
      timeout: 10_000,
    },
  ],
});
