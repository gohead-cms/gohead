import { expect, test } from "@playwright/test";
import { editor, login, trackErrors } from "./fixtures";

// The admin sections, end to end against the real API: each test changes
// something through the UI and checks the result the API sends back.
test.describe("admin", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("content model: create a collection and write to it", async ({ page }) => {
    const errors = trackErrors(page);
    const name = `notes${test.info().retry}`;
    await page.goto("/model");
    await expect(page.getByRole("cell", { name: "articles", exact: true })).toBeVisible();

    await page.getByRole("link", { name: "New collection" }).click();
    await expect(page).toHaveURL(/\/model\/collections\/new$/);
    await page.getByRole("textbox", { name: /^Name/ }).fill(name);
    await page.getByRole("textbox", { name: /^Attribute name/ }).first().fill("headline");
    await page.getByRole("button", { name: "Save", exact: true }).click();

    await expect(page).toHaveURL(/\/model(\?.*)?$/);
    await expect(page.getByRole("cell", { name, exact: true })).toBeVisible();

    // The new type is immediately available in the content editor.
    await page.goto(`/contrib?collection=${name}`);
    await expect(page.getByRole("heading", { name: new RegExp(name, "i") })).toBeVisible();
    expect(errors).toEqual([]);
  });

  test("users: create an account", async ({ page }) => {
    const username = `writer${test.info().retry}`;
    await page.goto("/settings/users");
    await page.getByRole("button", { name: "New user" }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByRole("textbox", { name: /^Username/ }).fill(username);
    await dialog.getByRole("textbox", { name: /^Email/ }).fill(`${username}@example.com`);
    await dialog.getByLabel(/^Password/).fill("writer-password-123");
    await dialog.getByRole("button", { name: "Create user" }).click();
    await expect(page.getByRole("cell", { name: username, exact: true })).toBeVisible();
  });

  test("API tokens: the secret is shown once", async ({ page }) => {
    await page.goto("/settings/tokens");
    await page.getByRole("button", { name: "New token" }).click();
    await page.getByRole("dialog").getByRole("textbox", { name: /^Name/ }).fill(`ci${test.info().retry}`);
    await page.getByRole("button", { name: "Create token" }).click();
    await expect(page.getByText(/ght_[A-Za-z0-9_-]{10,}/)).toBeVisible();
    await page.getByRole("button", { name: "I have stored it" }).click();
    await expect(page.getByRole("table").getByText(`ci${test.info().retry}`, { exact: true })).toBeVisible();
  });

  test("media: upload a file", async ({ page }) => {
    await page.goto("/media");
    // A 1×1 transparent PNG.
    const png = Buffer.from(
      "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==",
      "base64",
    );
    await page.getByLabel("Choose files to upload").setInputFiles({ name: "pixel.png", mimeType: "image/png", buffer: png });
    await expect(page.getByText("pixel.png").first()).toBeVisible();
  });

  test("agents: the editor and the activity page load", async ({ page }) => {
    const errors = trackErrors(page);
    await page.goto("/agents/new");
    await expect(page.getByRole("heading", { name: /agent/i }).first()).toBeVisible();
    await page.goto("/agents/e2e-tagger/edit");
    await expect(page.getByRole("textbox", { name: /^Name/ })).toHaveValue("e2e-tagger");
    await page.goto("/activity");
    await expect(page.getByRole("heading", { name: "Activity" })).toBeVisible();
    await page.goto("/settings/webhooks");
    await expect(page.getByRole("heading", { name: "Webhooks" })).toBeVisible();
    expect(errors).toEqual([]);
  });
});

test("an editor edits content but sees no admin sections", async ({ page }) => {
  const errors = trackErrors(page);
  await login(page, editor);
  await page.goto("/contrib");
  await expect(page.getByRole("cell", { name: "Seeded article" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Agents" })).toHaveCount(0);
  await page.goto("/settings/users");
  await expect(page).toHaveURL(/\/dashboard$/);
  expect(errors).toEqual([]);
});

test("content history: an editor restores an earlier version", async ({ page }) => {
  const errors = trackErrors(page);
  await login(page, editor);
  // Unique per attempt: a retry must not find the first attempt's entry.
  const original = `Versioned article #${test.info().retry}`;
  const edited = `${original} (edited)`;
  await page.goto("/contrib?collection=articles");
  await expect(page.getByRole("heading", { name: /articles/i })).toBeVisible();

  // Version 1: create the entry.
  await page.getByRole("button", { name: "Quick Add" }).click();
  const form = page.getByRole("dialog");
  await form.getByLabel("title").fill(original);
  await form.getByRole("button", { name: "Save Changes" }).click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  const row = () => page.getByRole("row").filter({ hasText: original });
  await expect(row()).toHaveCount(1);

  // Version 2: edit its title.
  await row().getByRole("button", { name: "Edit item" }).click();
  await expect(form.getByLabel("title")).toHaveValue(original);
  await form.getByLabel("title").fill(edited);
  await form.getByRole("button", { name: "Save Changes" }).click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(page.getByRole("cell", { name: edited, exact: true })).toBeVisible();

  // The history lists both versions; version 1 shows the original title.
  await row().getByRole("button", { name: "Item history" }).click();
  const history = page.getByRole("dialog");
  await expect(history.getByText("History", { exact: true })).toBeVisible();
  const versions = history.getByRole("button", { name: /^Version \d+,/ });
  await expect(versions).toHaveCount(2);
  await history.getByRole("button", { name: /^Version 1,/ }).click();
  await expect(history.getByRole("heading", { name: "Version 1" })).toBeVisible();
  await expect(history.getByTestId("change-title")).toContainText(original);

  // Restoring version 1 records version 3 with the original title.
  await history.getByRole("button", { name: "Restore this version" }).click();
  await page.getByRole("alertdialog").getByRole("button", { name: "Restore", exact: true }).click();
  await expect(page.getByRole("alertdialog")).toHaveCount(0);
  await expect(versions).toHaveCount(3);
  await expect(history.getByRole("button", { name: /^Version 3, restore/ })).toBeVisible();
  await history.getByRole("button", { name: "Close" }).click();
  await expect(page.getByRole("dialog")).toHaveCount(0);

  await expect(page.getByRole("cell", { name: original, exact: true })).toBeVisible();
  await expect(page.getByRole("cell", { name: edited, exact: true })).toHaveCount(0);
  expect(errors).toEqual([]);
});
