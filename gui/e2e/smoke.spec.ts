import { expect, test } from "@playwright/test";
import { login, trackErrors } from "./fixtures";

test("redirects to the login page when signed out", async ({ page }) => {
  await page.goto("/dashboard");
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByText("Welcome back!")).toBeVisible();
});

test("rejects a wrong password", async ({ page }) => {
  await page.goto("/login");
  await page.getByPlaceholder("Username").fill("e2e-admin");
  await page.getByPlaceholder("Password").fill("wrong-password");
  await page.getByRole("button", { name: "Login" }).click();
  await expect(page.getByRole("alert")).toBeVisible();
  await expect(page).toHaveURL(/\/login$/);
});

test.describe("signed in", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("schema designer shows the collections", async ({ page }) => {
    const errors = trackErrors(page);
    await page.goto("/workspace");
    await expect(page.locator(".react-flow")).toBeVisible();
    await expect(page.locator(".react-flow__node").filter({ hasText: "articles" }).first()).toBeVisible();
    expect(errors).toEqual([]);
  });

  test("content editor lists and creates entries", async ({ page }) => {
    const errors = trackErrors(page);
    // Unique per attempt: a retry must not find the first attempt's entry.
    const title = `Written by Playwright #${test.info().retry}`;
    await page.goto("/contrib");
    await expect(page.getByRole("heading", { name: /articles/i })).toBeVisible();
    await expect(page.getByRole("cell", { name: "Seeded article" })).toBeVisible();

    await page.getByRole("button", { name: "Quick Add" }).click();
    const drawer = page.getByRole("dialog");
    await drawer.getByLabel("title").fill(title);
    await drawer.getByLabel("body").fill("Smoke test entry");
    await drawer.getByRole("button", { name: "Save Changes" }).click();

    await expect(page.getByRole("cell", { name: title })).toBeVisible();
    expect(errors).toEqual([]);
  });

  test("full-page editor creates an entry", async ({ page }) => {
    const errors = trackErrors(page);
    const title = `From the full editor #${test.info().retry}`;
    await page.goto("/contrib");
    await page.getByRole("button", { name: "Contrib" }).click();
    await expect(page).toHaveURL(/\/contrib\/edit\/articles$/);
    await page.getByPlaceholder("Your post title...").fill(title);
    await page.getByPlaceholder("Begin writing your content...").fill("Long form");
    await page.getByRole("button", { name: "Save Entry" }).click();

    await expect(page).toHaveURL(/\/contrib$/);
    await expect(page.getByRole("cell", { name: title })).toBeVisible();
    expect(errors).toEqual([]);
  });

  test("agents page shows the agent, its limits and runs", async ({ page }) => {
    const errors = trackErrors(page);
    await page.goto("/agents");
    await expect(page.getByRole("heading", { name: "e2e-tagger" })).toBeVisible();
    await expect(page.getByText("10 runs / hour")).toBeVisible();
    await expect(page.getByText("(only articles)")).toBeVisible();

    await page.getByRole("tab", { name: "Runs" }).click();
    await expect(page.getByText("Runs, last hour")).toBeVisible();
    await expect(page.getByText(/No runs yet/)).toBeVisible();
    expect(errors).toEqual([]);
  });

  test("dashboard renders", async ({ page }) => {
    const errors = trackErrors(page);
    await page.goto("/dashboard");
    await expect(page.getByText("Overview of your GoHead CMS system")).toBeVisible();
    expect(errors).toEqual([]);
  });

  test("logging out returns to the login page", async ({ page }) => {
    await page.evaluate(() => localStorage.removeItem("token"));
    await page.goto("/dashboard");
    await expect(page).toHaveURL(/\/login$/);
  });
});
