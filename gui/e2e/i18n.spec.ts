import { type APIRequestContext, expect, test } from "@playwright/test";
import { admin, apiURL, login, trackErrors } from "./fixtures";

const LOCALE = "fr";

async function adminHeaders(request: APIRequestContext) {
  const res = await request.post(`${apiURL}/auth/login`, { data: admin });
  expect(res.ok(), await res.text()).toBeTruthy();
  return { Authorization: `Bearer ${(await res.json()).data.token as string}` };
}

// Field-level localization end to end: an admin adds a locale on the
// Locales page, then translates the localized title of an entry in the
// content editor; the API keeps the default title and stores the French one.
test("locales: translate an entry's localized attribute", async ({ page, request }) => {
  const errors = trackErrors(page);
  const headers = await adminHeaders(request);
  // Unique per run and attempt, so re-runs against the same database work.
  const collection = `books${Date.now().toString(36)}${test.info().retry}`;
  const title = `The Book ${collection}`;
  const french = `Le Livre ${collection}`;
  const entries = `${apiURL}/api/collections/${collection}`;

  // A locale left behind by an interrupted run would make "Add locale" fail.
  await request.delete(`${apiURL}/admin/locales/${LOCALE}?purge=true`, { headers });

  const created = await request.post(`${apiURL}/admin/collections`, {
    headers,
    data: {
      name: collection,
      kind: "collection",
      attributes: {
        title: { type: "string", required: true, localized: true },
        isbn: { type: "string" },
      },
    },
  });
  expect(created.ok(), await created.text()).toBeTruthy();

  try {
    const entry = await request.post(entries, { headers, data: { data: { title, isbn: "978-0" } } });
    expect(entry.ok(), await entry.text()).toBeTruthy();
    const id = (await entry.json()).data.id as number;

    // The schema says which attribute is localized.
    const schema = await request.get(`${apiURL}/admin/collections/${collection}`, { headers });
    expect((await schema.json()).data.schema.attributes.title.localized).toBe(true);

    await login(page);
    await page.goto("/settings/locales");
    await expect(page.getByRole("heading", { name: "Locales" })).toBeVisible();
    await page.getByRole("button", { name: "Add locale" }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel(/^Code/).fill(LOCALE);
    await dialog.getByRole("textbox", { name: /^Name/ }).fill("Français");
    await dialog.getByRole("button", { name: "Add locale" }).click();
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(page.getByRole("row").filter({ hasText: "Français" })).toBeVisible();

    // Translate the entry in the content editor.
    await page.goto(`/contrib?collection=${collection}`);
    const row = page.getByRole("row").filter({ hasText: title });
    await expect(row).toHaveCount(1);
    await row.getByRole("button", { name: "Edit item" }).click();
    const form = page.getByRole("dialog");
    await expect(form.getByLabel("title")).toHaveValue(title);
    await form.getByLabel("Locale").selectOption(LOCALE);
    // Untranslated: the title falls back to the default locale's.
    await expect(form.getByText(`Falls back to the default (en): ${title}`)).toBeVisible();
    // Shared attributes are only edited in the default locale.
    await expect(form.getByLabel("isbn")).toHaveCount(0);
    await expect(form.getByLabel("title")).toHaveValue("");
    await form.getByLabel("title").fill(french);
    await expect(form.getByRole("button", { name: "Use default" })).toBeVisible();

    // Switching locale with unsaved edits asks first; keeping them stays in fr.
    await form.getByLabel("Locale").selectOption("en");
    const discard = page.getByRole("alertdialog");
    await expect(discard.getByText("Discard unsaved changes?")).toBeVisible();
    await discard.getByRole("button", { name: "Keep editing" }).click();
    await expect(page.getByRole("alertdialog")).toHaveCount(0);
    await expect(form.getByLabel("Locale")).toHaveValue(LOCALE);
    await expect(form.getByLabel("title")).toHaveValue(french);
    await form.getByRole("button", { name: "Save translation" }).click();
    await expect(page.getByRole("dialog")).toHaveCount(0);

    // The French title is stored; the default one is unchanged.
    const fr = await request.get(`${entries}/${id}?locale=${LOCALE}&fallback=false`, { headers });
    expect(fr.ok(), await fr.text()).toBeTruthy();
    const frData = (await fr.json()).data;
    expect(frData.title).toBe(french);
    expect(frData.isbn).toBe("978-0");
    const def = await request.get(`${entries}/${id}`, { headers });
    const defData = (await def.json()).data;
    expect(defData.title).toBe(title);
    expect(defData.locales).toContain(LOCALE);

    // The list can be read in French.
    await page.getByLabel("View in locale").selectOption(LOCALE);
    await expect(page.getByRole("cell", { name: french, exact: true })).toBeVisible();
    expect(errors).toEqual([]);
  } finally {
    await request.delete(`${apiURL}/admin/collections/${collection}`, { headers });
    // Other tests expect a single locale (no locale switchers).
    await request.delete(`${apiURL}/admin/locales/${LOCALE}?purge=true`, { headers });
  }
});
