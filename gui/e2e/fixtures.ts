import { expect, Page } from "@playwright/test";

export const apiURL = process.env.E2E_API_URL ?? "http://localhost:8080";
export const admin = { username: "e2e-admin", email: "e2e-admin@example.com", password: "e2e-password-123" };

/** Logs in through the login form and waits for the app shell. */
export async function login(page: Page) {
  await page.goto("/login");
  await page.getByPlaceholder("Username").fill(admin.username);
  await page.getByPlaceholder("Password").fill(admin.password);
  await page.getByRole("button", { name: "Login" }).click();
  await expect(page).toHaveURL(/\/workspace$/);
}

/** Collects uncaught page errors, so a test can assert there were none. */
export function trackErrors(page: Page): string[] {
  const errors: string[] = [];
  page.on("pageerror", (err) => errors.push(err.message));
  return errors;
}
