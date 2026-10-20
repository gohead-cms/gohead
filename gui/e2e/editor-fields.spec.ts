import { expect, test } from "@playwright/test";
import { adminHeaders, apiURL, login, trackErrors } from "./fixtures";

// The entry editor sets relations, components (single and repeatable),
// json and media picked from the library.

// A 1×1 PNG.
const PNG = Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==", "base64");

test.describe("entry fields", () => {
  let headers: Record<string, string>;

  test.beforeAll(async ({ request }) => {
    headers = await adminHeaders(request);
    // Idempotent: a retry finds what the first attempt created.
    const ensure = async (path: string, data: unknown) => {
      const res = await request.post(`${apiURL}${path}`, { data, headers });
      const text = await res.text();
      expect(res.ok() || res.status() === 409 || /already exist/i.test(text), `${path}: ${text}`).toBeTruthy();
      return res.ok();
    };
    await ensure("/admin/components", {
      name: "e2e_faq",
      attributes: { question: { type: "string", required: true }, answer: { type: "text" } },
    });
    await ensure("/admin/components", { name: "e2e_seo", attributes: { meta_title: { type: "string" } } });
    const newPeople = await ensure("/admin/collections", {
      name: "e2e_people",
      kind: "collection",
      attributes: { name: { type: "string", required: true } },
    });
    await ensure("/admin/collections", {
      name: "e2e_pages",
      kind: "collection",
      attributes: {
        title: { type: "string", required: true },
        owner: { type: "relation", relation: "oneToOne", target: "e2e_people" },
        people: { type: "relation", relation: "manyToMany", target: "e2e_people" },
        faqs: { type: "component", component: "e2e_faq", repeatable: true },
        seo: { type: "component", component: "e2e_seo" },
        settings: { type: "json" },
        gallery: { type: "media", multiple: true, allowed_types: ["image/*"] },
      },
    });
    if (!newPeople) return;
    for (const name of ["Ada Lovelace", "Grace Hopper", "Alan Turing"]) {
      await ensure("/api/collections/e2e_people", { data: { name } });
    }
    for (const name of ["first.png", "second.png"]) {
      const res = await request.post(`${apiURL}/api/media`, {
        headers,
        multipart: { files: { name, mimeType: "image/png", buffer: PNG } },
      });
      expect(res.ok(), await res.text()).toBeTruthy();
    }
  });

  test("relations, components, json and library media", async ({ page, request }) => {
    const errors = trackErrors(page);
    const title = `Editor fields #${test.info().retry}`;
    await login(page);
    await page.goto("/contrib");
    await page.getByRole("button", { name: "e2e_pages" }).click();
    await page.getByRole("button", { name: "Quick Add" }).click();
    const drawer = page.getByRole("dialog");
    await drawer.getByLabel("title").fill(title);

    // One-to-one: pick from a searchable list.
    await drawer.getByRole("button", { name: "Choose e2e_people" }).click();
    let picker = page.getByRole("dialog", { name: "Choose e2e_people" });
    await picker.getByPlaceholder("Search by name").fill("grace");
    await expect(picker.getByRole("button", { name: /Ada Lovelace/ })).toHaveCount(0);
    await picker.getByRole("button", { name: /Grace Hopper/ }).click();
    await expect(drawer.getByText("Grace Hopper")).toBeVisible();

    // Many-to-many: several at once.
    await drawer.getByRole("button", { name: "Add e2e_people" }).click();
    picker = page.getByRole("dialog", { name: "Add e2e_people" });
    await picker.getByRole("button", { name: /Ada Lovelace/ }).click();
    await picker.getByRole("button", { name: /Alan Turing/ }).click();
    await picker.getByRole("button", { name: "Add 2" }).click();
    await expect(drawer.getByText("Alan Turing")).toBeVisible();

    // Repeatable component: two instances; single component: one.
    await drawer.getByRole("button", { name: "Add e2e_faq" }).click();
    await drawer.getByRole("button", { name: "Add e2e_faq" }).click();
    await drawer.getByLabel("question").nth(0).fill("Why?");
    await drawer.getByLabel("question").nth(1).fill("How?");
    await drawer.getByLabel("answer").nth(1).fill("Like this.");
    await drawer.getByRole("button", { name: "Add e2e_seo" }).click();
    await drawer.getByLabel("meta_title").fill("SEO title");

    // JSON: invalid text blocks saving.
    await drawer.getByLabel("settings").fill('{"theme": ');
    await expect(drawer.getByText(/Not valid JSON/)).toBeVisible();
    await drawer.getByRole("button", { name: "Save Changes" }).click();
    await expect(page.getByText("Fix the entry before saving")).toBeVisible();
    await drawer.getByLabel("settings").fill('{"theme": "dark", "columns": 2}');

    // Media from the library.
    await drawer.getByRole("button", { name: "Choose from library" }).click();
    const library = page.getByRole("dialog", { name: "Add files from the library" });
    await library.getByRole("button", { name: "Choose first.png" }).click();
    await library.getByRole("button", { name: "Choose second.png" }).click();
    await library.getByRole("button", { name: "Add 2" }).click();
    await expect(drawer.getByText("second.png")).toBeVisible();

    await drawer.getByRole("button", { name: "Save Changes" }).click();
    await expect(page.getByRole("cell", { name: title })).toBeVisible();

    // Stored as the API expects.
    const res = await request.get(`${apiURL}/api/collections/e2e_pages?level=0&filters[title]=${encodeURIComponent(title)}`, { headers });
    const [entry] = (await res.json()).data;
    const people = await (await request.get(`${apiURL}/api/collections/e2e_people?sort=id`, { headers })).json();
    const id = (name: string) => people.data.find((p: { name: string }) => p.name === name).id;
    expect(entry.owner).toBe(id("Grace Hopper"));
    expect(entry.people).toEqual([id("Ada Lovelace"), id("Alan Turing")]);
    expect(entry.faqs).toEqual([{ question: "Why?" }, { question: "How?", answer: "Like this." }]);
    expect(entry.seo).toEqual({ meta_title: "SEO title" });
    expect(entry.settings).toEqual({ theme: "dark", columns: 2 });
    expect(entry.gallery).toHaveLength(2);

    // Editing again shows the labels, and keeps everything when saved.
    await page.getByRole("row", { name: new RegExp(title) }).getByRole("button", { name: /edit/i }).click();
    const edit = page.getByRole("dialog");
    await expect(edit.getByText("Grace Hopper")).toBeVisible();
    await expect(edit.getByText("Alan Turing")).toBeVisible();
    await expect(edit.getByLabel("question").nth(1)).toHaveValue("How?");
    await edit.getByRole("button", { name: "Remove e2e_faq 1" }).click();
    await edit.getByRole("button", { name: "Save Changes" }).click();
    await expect(edit).toBeHidden();
    const after = (await (await request.get(`${apiURL}/api/collections/e2e_pages/${entry.id}?level=0`, { headers })).json()).data;
    expect(after.faqs).toEqual([{ question: "How?", answer: "Like this." }]);
    expect(after.people).toEqual(entry.people);
    expect(after.settings).toEqual(entry.settings);
    expect(after.gallery).toEqual(entry.gallery);
    expect(errors).toEqual([]);
  });
});
