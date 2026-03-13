import { test, expect } from "@playwright/test";

// These tests use the authenticated storage state from auth.setup.ts.
// They assume the "default" project with "production" stage exists (seeded by the backend).

const PROJECT = "default";
const STAGE = "production";
const FLAG_KEY = `e2e-flag-${Date.now()}`;

test.describe("Flags page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(`/projects/${PROJECT}/stages/${STAGE}/flags`);
    await expect(page.getByRole("heading", { name: "Feature Flags" })).toBeVisible();
  });

  test("shows flags list page with breadcrumb", async ({ page }) => {
    await expect(page.getByRole("link", { name: PROJECT, exact: true })).toBeVisible();
    await expect(page.getByText(STAGE, { exact: true })).toBeVisible();
  });

  test("creates a new flag", async ({ page }) => {
    await page.getByRole("button", { name: "New Flag" }).click();

    // Fill in modal
    await page.getByLabel("Key").fill(FLAG_KEY);
    await page.getByLabel("Name").fill("E2E Test Flag");
    await page.getByLabel("Description").fill("Created by Playwright");

    await page.getByRole("button", { name: "Create" }).click();

    // Should see success toast and flag in list
    await expect(page.getByText("Flag created")).toBeVisible();
    await expect(page.getByText(FLAG_KEY)).toBeVisible();
  });

  test("searches for flag", async ({ page }) => {
    const searchInput = page.getByPlaceholder("Search flags...");
    await searchInput.fill(FLAG_KEY);

    await expect(page.getByText(FLAG_KEY)).toBeVisible();

    // Search for nonexistent flag
    await searchInput.fill("nonexistent-flag-xyz");
    await expect(page.getByText("No matches")).toBeVisible();
  });

  test("toggles flag enabled state", async ({ page }) => {
    // Find the row with our flag and its switch
    const flagRow = page.locator(`a:has-text("${FLAG_KEY}")`);
    await expect(flagRow).toBeVisible();

    const toggle = flagRow.locator("button[role='switch']");
    const initialState = await toggle.getAttribute("data-state");

    await toggle.click();

    // State should have changed
    const newState = await toggle.getAttribute("data-state");
    expect(newState).not.toBe(initialState);
  });

  test("navigates to flag detail", async ({ page }) => {
    await page.getByText(FLAG_KEY).click();

    await expect(page).toHaveURL(new RegExp(`/flags/${FLAG_KEY}$`));
    await expect(page.getByRole("heading", { name: FLAG_KEY })).toBeVisible();
  });
});
