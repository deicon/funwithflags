import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

const useLiveBackend = process.env.E2E_USE_LIVE_BACKEND === "true";
const apiBase = process.env.E2E_API_BASE ?? "http://localhost:8080";
const adminUsername = process.env.E2E_ADMIN_USER ?? "admin";
const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? "admin123";

const now = new Date("2024-05-20T12:00:00Z");
const isoNow = now.toISOString();

const demoProject = {
  key: "demo-shop",
  name: "Demo Shop",
  description: "Sample storefront",
  createdAt: isoNow,
  updatedAt: isoNow
};

const demoStage = {
  projectKey: demoProject.key,
  key: "dev",
  name: "Development",
  description: "Unreleased changes",
  createdAt: isoNow,
  updatedAt: isoNow
};

const demoFlag = {
  id: 101,
  project: demoProject.key,
  stage: demoStage.key,
  key: "checkout-flow",
  name: "Checkout flow",
  description: "Enable revamped checkout",
  enabled: true,
  active: true,
  validFrom: isoNow,
  validTo: undefined,
  defaultKey: "control",
  variations: [
    { key: "control", type: "boolean", value: false, description: "Old flow" },
    { key: "treatment", type: "boolean", value: true, description: "New flow" }
  ],
  rules: [],
  createdAt: isoNow,
  updatedAt: isoNow
};

const loginResponse = {
  tokenType: "Bearer",
  accessToken: "fake-access-token",
  accessTokenExpiresAt: new Date(now.getTime() + 15 * 60_000).toISOString(),
  refreshToken: "fake-refresh-token",
  refreshTokenExpiresAt: new Date(now.getTime() + 60 * 60_000).toISOString(),
  user: { username: "admin", role: "admin" }
};

type LiveFixture = { project: typeof demoProject; stage: typeof demoStage; flag: typeof demoFlag };

async function mockSuccessfulAdminSession(page: Page) {
  await page.route("**/api/v1/auth/login", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(loginResponse)
    });
  });

  await page.route("**/api/v1/admin/projects", async (route) => {
    await expect(route.request().headers()["authorization"]).toContain(loginResponse.accessToken);
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ projects: [demoProject] })
    });
  });

  await page.route(`**/api/v1/admin/projects/${demoProject.key}/stages`, async (route) => {
    await expect(route.request().headers()["authorization"]).toContain(loginResponse.accessToken);
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ stages: [demoStage] })
    });
  });

  await page.route(`**/api/v1/admin/${demoProject.key}/${demoStage.key}/flags`, async (route) => {
    await expect(route.request().headers()["authorization"]).toContain(loginResponse.accessToken);
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ flags: [demoFlag] })
    });
  });
}

async function seedLiveBackend(apiRequest: APIRequestContext): Promise<LiveFixture> {
  const loginResponse = await apiRequest.post("/api/v1/auth/login", {
    data: { username: adminUsername, password: adminPassword }
  });

  if (!loginResponse.ok()) {
    throw new Error(`failed to log in to API: ${loginResponse.status()} ${loginResponse.statusText()}`);
  }

  const { tokenType, accessToken } = (await loginResponse.json()) as { tokenType: string; accessToken: string };
  const authHeaders = { Authorization: `${tokenType} ${accessToken}` };

  const suffix = `${Date.now()}`;
  const project = { key: `e2e-${suffix}`, name: `E2E Project ${suffix}`, description: "Playwright smoke", createdAt: isoNow, updatedAt: isoNow };
  const stage = { projectKey: project.key, key: `stage-${suffix}`, name: "QA", description: "Playwright stage", createdAt: isoNow, updatedAt: isoNow };
  const flag = {
    id: Math.floor(Math.random() * 100_000),
    project: project.key,
    stage: stage.key,
    key: `flag-${suffix}`,
    name: "Live flag",
    description: "Playwright live backend flag",
    enabled: true,
    active: true,
    validFrom: new Date().toISOString(),
    validTo: undefined,
    defaultKey: "control",
    variations: [
      { key: "control", type: "boolean", value: false, description: "Feature off" },
      { key: "enabled", type: "boolean", value: true, description: "Feature on" }
    ],
    rules: [],
    createdAt: isoNow,
    updatedAt: isoNow
  };

  const projectResponse = await apiRequest.post("/api/v1/admin/projects", { data: project, headers: authHeaders });
  if (!projectResponse.ok()) {
    throw new Error(`failed to seed project: ${projectResponse.status()} ${projectResponse.statusText()}`);
  }

  const stageResponse = await apiRequest.post(`/api/v1/admin/projects/${project.key}/stages`, { data: stage, headers: authHeaders });
  if (!stageResponse.ok()) {
    throw new Error(`failed to seed stage: ${stageResponse.status()} ${stageResponse.statusText()}`);
  }

  const flagResponse = await apiRequest.post(`/api/v1/admin/${project.key}/${stage.key}/flags`, {
    data: {
      key: flag.key,
      name: flag.name,
      description: flag.description,
      enabled: flag.enabled,
      active: flag.active,
      defaultKey: flag.defaultKey,
      validFrom: flag.validFrom,
      variations: flag.variations,
      rules: flag.rules
    },
    headers: authHeaders
  });
  if (!flagResponse.ok()) {
    throw new Error(`failed to seed flag: ${flagResponse.status()} ${flagResponse.statusText()}`);
  }

  return { project, stage, flag };
}

let liveFixture: LiveFixture | null = null;

test.beforeAll(async ({ playwright }) => {
  if (!useLiveBackend) {
    return;
  }

  const apiRequest = await playwright.request.newContext({ baseURL: apiBase });
  liveFixture = await seedLiveBackend(apiRequest);
  await apiRequest.dispose();
});

test("admin can sign in and view projects, stages, and flags", async ({ page }) => {
  if (!useLiveBackend) {
    await mockSuccessfulAdminSession(page);
  }

  if (useLiveBackend && !liveFixture) {
    throw new Error("live backend fixtures were not prepared");
  }

  await page.goto("/");
  await page.getByLabel("Username").fill(useLiveBackend ? adminUsername : "admin");
  await page.getByLabel("Password").fill(useLiveBackend ? adminPassword : "correct horse battery staple");
  await page.getByRole("button", { name: "Sign in" }).click();

  const fixture = liveFixture ?? { project: demoProject, stage: demoStage, flag: demoFlag };

  await expect(page.getByRole("heading", { name: "Projects" })).toBeVisible();
  await expect(page.getByRole("cell", { name: fixture.project.name })).toBeVisible();

  await expect(page.getByRole("heading", { name: "Stages" })).toBeVisible();
  await expect(page.getByRole("cell", { name: fixture.stage.name })).toBeVisible();

  await expect(page.getByRole("heading", { name: "Flags" })).toBeVisible();
  await expect(page.getByRole("cell", { name: fixture.flag.key })).toBeVisible();
  await expect(page.getByRole("cell", { name: fixture.flag.defaultKey })).toBeVisible();
});

test("shows a helpful error when credentials are rejected", async ({ page }) => {
  await page.route("**/api/v1/auth/login", async (route) => {
    await route.fulfill({
      status: 401,
      contentType: "application/json",
      body: JSON.stringify({ message: "invalid credentials" })
    });
  });

  await page.goto("/");
  await page.getByRole("button", { name: "Sign in" }).click();

  await expect(page.getByRole("alert")).toContainText("invalid credentials");
});
