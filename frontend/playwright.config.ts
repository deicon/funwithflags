import { defineConfig, devices } from "@playwright/test";

/**
 * Playwright E2E tests for funwithflags.
 *
 * Prerequisites:
 *   docker-compose up -d postgres    # Start PostgreSQL
 *
 * Run:
 *   cd frontend && npx playwright test
 *   → Starts Go backend (STORAGE_TYPE=postgres, port 9090) and Vite dev server automatically.
 *
 * Against full docker-compose (app + postgres):
 *   docker-compose up -d
 *   API_URL=http://localhost:8080 npx playwright test
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
    // Go backend with PostgreSQL (skipped if API_URL is set externally)
    ...(!process.env.API_URL
      ? [
          {
            command: [
              "STORAGE_TYPE=postgres",
              "DB_HOST=localhost",
              "DB_PORT=5432",
              "DB_USER=postgres",
              "DB_PASSWORD=postgres",
              "DB_NAME=funwithflags",
              "MIGRATIONS_PATH=../migrations",
              "PORT=9090",
              "CORS_ALLOWED_ORIGINS=http://localhost:5173",
              "go run ../cmd/server/main.go",
            ].join(" "),
            url: "http://localhost:9090/healthz",
            reuseExistingServer: !process.env.CI,
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
