import { expect, test } from "@playwright/test";
import { trackErrors } from "./fixtures";

// Demo mode (GOHEAD_DEMO_ENABLED=true in CI): visitors sign in from the
// login page and see the admin read-only.

test("demo: visitors sign in from the login page and look around", async ({ page }) => {
  const errors = trackErrors(page);
  await page.goto("/login");
  const panel = page.getByTestId("demo-login");
  test.skip(!(await panel.isVisible({ timeout: 5_000 }).catch(() => false)), "the API is not in demo mode");
  await expect(panel.getByText("try-gohead-demo")).toBeVisible();
  await panel.getByRole("button", { name: "Sign in to the demo" }).click();
  await expect(page).toHaveURL(/\/dashboard$/);
  await expect(page.getByRole("status").filter({ hasText: "GoHead demo." })).toBeVisible();

  // The admin pages are there, without the buttons that change settings.
  await page.locator('a[href="/agents"]').first().click();
  await expect(page.getByRole("link", { name: "New agent" })).toHaveCount(0);
  await page.locator('a[href="/connections"]').first().click();
  await expect(page.getByRole("heading", { name: "Connected apps" })).toBeVisible();
  expect(errors).toEqual([]);
});
