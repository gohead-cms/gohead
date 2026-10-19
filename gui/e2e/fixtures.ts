import { expect, Page } from "@playwright/test";

export const apiURL = process.env.E2E_API_URL ?? "http://localhost:8080";
// The API must be started with this administrator:
// GOHEAD_ADMIN_USERNAME=e2e-admin GOHEAD_ADMIN_PASSWORD=e2e-password-123
export const admin = {
  username: process.env.E2E_ADMIN_USERNAME ?? "e2e-admin",
  password: process.env.E2E_ADMIN_PASSWORD ?? "e2e-password-123",
};

// An editor created by the global setup.
export const editor = { username: "e2e-editor", password: "editor-password-123" };

/** Logs in through the login form and waits for the app shell. */
export async function login(page: Page, as: { username: string; password: string } = admin) {
  await page.goto("/login");
  await page.getByPlaceholder("Username").fill(as.username);
  await page.getByPlaceholder("Password").fill(as.password);
  await page.getByRole("button", { name: "Login" }).click();
  await expect(page).toHaveURL(/\/dashboard$/);
}

/** Collects uncaught page errors, so a test can assert there were none. */
export function trackErrors(page: Page): string[] {
  const errors: string[] = [];
  page.on("pageerror", (err) => errors.push(err.message));
  return errors;
}
