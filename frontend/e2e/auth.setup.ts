import { test as setup, expect } from "@playwright/test";

const authFile = "e2e/.auth/user.json";

setup("authenticate as admin", async ({ page }) => {
  await page.goto("/login");

  await page.getByLabel("Username").fill("admin");
  await page.getByLabel("Password").fill("admin123");
  await page.getByRole("button", { name: "Sign in" }).click();

  // Wait for redirect away from login page
  await expect(page).not.toHaveURL(/\/login/);

  // Save signed-in state
  await page.context().storageState({ path: authFile });
});
