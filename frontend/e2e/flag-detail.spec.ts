import { test, expect } from "@playwright/test";

// Tests for the three-entity model: flag identity → ranges → versions.
// Uses authenticated state from auth.setup.ts.

const PROJECT = "default";
const STAGE = "production";
const FLAG_KEY = `e2e-detail-${Date.now()}`;

test.describe("Flag detail page — three-entity model", () => {
  // Create a flag via API before tests, so we have a known flag to work with.
  test.beforeAll(async ({ request }) => {
    // Login to get token
    const loginResp = await request.post("/api/v1/auth/login", {
      data: { username: "admin", password: "admin123" },
    });
    const { accessToken } = await loginResp.json();

    // Create flag
    await request.post(`/api/v1/admin/${PROJECT}/${STAGE}/flags`, {
      headers: { Authorization: `Bearer ${accessToken}` },
      data: {
        key: FLAG_KEY,
        name: "E2E Detail Flag",
        description: "For detail page testing",
        enabled: true,
        defaultKey: "off",
        variations: [
          { key: "off", type: "boolean", value: false },
          { key: "on", type: "boolean", value: true },
        ],
      },
    });
  });

  test("shows flag identity details", async ({ page }) => {
    await page.goto(`/projects/${PROJECT}/stages/${STAGE}/flags/${FLAG_KEY}`);

    await expect(page.getByRole("heading", { name: FLAG_KEY })).toBeVisible();
    await expect(page.getByText("E2E Detail Flag")).toBeVisible();
    await expect(page.getByText("For detail page testing")).toBeVisible();
    await expect(page.getByText("Variations")).toBeVisible();
    await expect(page.getByText("off", { exact: true }).first()).toBeVisible();
    await expect(page.getByText("on", { exact: true }).first()).toBeVisible();
  });

  test("edits flag details inline", async ({ page }) => {
    await page.goto(`/projects/${PROJECT}/stages/${STAGE}/flags/${FLAG_KEY}`);

    await page.getByRole("button", { name: "Edit" }).click();

    // After clicking Edit, the Details card shows input fields.
    // The first input in the Details card is the Name field.
    const detailsCard = page.locator("[data-slot='card']").filter({ hasText: "Details" });
    const nameInput = detailsCard.locator("input").first();
    await nameInput.fill("Updated E2E Flag");

    await page.getByRole("button", { name: "Save" }).click();
    await expect(page.getByText("Details updated")).toBeVisible();
    await expect(page.getByText("Updated E2E Flag")).toBeVisible();
  });

  test("shows empty ranges state", async ({ page }) => {
    await page.goto(`/projects/${PROJECT}/stages/${STAGE}/flags/${FLAG_KEY}`);

    await expect(page.getByText("Temporal Ranges")).toBeVisible();
    await expect(page.getByText("No ranges configured")).toBeVisible();
  });

  test("creates a temporal range", async ({ page }) => {
    await page.goto(`/projects/${PROJECT}/stages/${STAGE}/flags/${FLAG_KEY}`);

    await page.getByRole("button", { name: "Add Range" }).click();

    // Fill in valid from (yesterday)
    const yesterday = new Date(Date.now() - 86400000);
    const dateStr = yesterday.toISOString().slice(0, 16); // YYYY-MM-DDTHH:mm
    await page.getByLabel("Valid From").fill(dateStr);

    await page.getByRole("button", { name: "Create" }).click();
    await expect(page.getByText("Range created")).toBeVisible();

    // Should now see the range card
    await expect(page.getByText("Inactive")).toBeVisible();
  });

  test("publishes a version and activates the range", async ({ page }) => {
    await page.goto(`/projects/${PROJECT}/stages/${STAGE}/flags/${FLAG_KEY}`);

    // The range card should already be expanded (first range auto-expands)
    // Wait for the draft section to be visible
    await expect(page.getByText(/Draft \(v/)).toBeVisible({ timeout: 5000 });

    await page.getByRole("button", { name: "Publish" }).click();
    await expect(page.getByText("Version published")).toBeVisible();

    // Now activate the range
    await page.getByRole("button", { name: "Activate" }).click();
    await expect(page.getByText("Range activated")).toBeVisible();
  });

  test("deactivates a range", async ({ page }) => {
    await page.goto(`/projects/${PROJECT}/stages/${STAGE}/flags/${FLAG_KEY}`);

    // The first range auto-expands
    await page.getByRole("button", { name: "Deactivate" }).click();
    await expect(page.getByText("Range deactivated")).toBeVisible();
  });

  test("creates rollback draft from published version", async ({ page }) => {
    await page.goto(`/projects/${PROJECT}/stages/${STAGE}/flags/${FLAG_KEY}`);

    // Should see Published version with rollback button (first range auto-expands)
    const rollbackBtn = page.getByRole("button", { name: "Create rollback draft" });
    await expect(rollbackBtn).toBeVisible({ timeout: 5000 });
    await rollbackBtn.click();
    await expect(page.getByText("Rollback draft created")).toBeVisible();
  });

  test("toggles flag enabled/disabled", async ({ page }) => {
    await page.goto(`/projects/${PROJECT}/stages/${STAGE}/flags/${FLAG_KEY}`);

    const toggle = page.locator("button[role='switch']").first();
    const initialState = await toggle.getAttribute("data-state");

    await toggle.click();
    // Wait for optimistic update
    await page.waitForTimeout(500);

    const newState = await toggle.getAttribute("data-state");
    expect(newState).not.toBe(initialState);
  });

  test("deletes the flag", async ({ page }) => {
    await page.goto(`/projects/${PROJECT}/stages/${STAGE}/flags/${FLAG_KEY}`);

    await page.getByRole("button", { name: "Delete", exact: true }).first().click();

    // Confirm dialog
    const dialog = page.locator("[role='alertdialog'], [role='dialog']").filter({ hasText: "Delete flag" });
    await expect(dialog).toBeVisible();
    await dialog.getByRole("button", { name: "Delete" }).click();

    await expect(page.getByText("Flag deleted")).toBeVisible();
    // Should redirect back to flags list
    await expect(page).toHaveURL(new RegExp(`/flags$`));
  });
});
