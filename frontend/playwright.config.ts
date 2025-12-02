import { defineConfig } from "@playwright/test";

const port = process.env.FRONTEND_PORT ?? "5173";
const baseURL = process.env.PLAYWRIGHT_BASE_URL ?? `http://localhost:${port}`;
const useExternalServer = process.env.PLAYWRIGHT_USE_EXTERNAL_SERVER === "true";

export default defineConfig({
  testDir: "./tests/e2e",
  fullyParallel: true,
  retries: process.env.CI ? 2 : 0,
  use: {
    baseURL,
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "retain-on-failure"
  },
  webServer: useExternalServer
    ? undefined
    : {
        command: `npm run dev -- --host 0.0.0.0 --port ${port}`,
        url: baseURL,
        reuseExistingServer: !process.env.CI,
        stdout: "pipe",
        stderr: "pipe",
        timeout: 120_000
      }
});
