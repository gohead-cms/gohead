import { expect, test } from "@playwright/test";
import { adminHeaders, apiURL, login, trackErrors } from "./fixtures";

// Custom roles end to end: an admin creates a role on the Roles page that
// may write drafts in one collection (no publish), gives it to a new user on
// the Users page, and that user only sees and writes what the role allows.
test("roles: a custom role writes drafts in one collection only", async ({ page, request }) => {
  const errors = trackErrors(page);
  const headers = await adminHeaders(request);
  // Unique per run and attempt, so re-runs against the same database work.
  const suffix = `${Date.now().toString(36)}${test.info().retry}`;
  const collection = `memos${suffix}`;
  const roleName = `writer-${suffix}`;
  const user = { username: `w${suffix}`, password: "writer-password-123" };
  const draft = `Draft memo ${suffix}`;

  const created = await request.post(`${apiURL}/admin/collections`, {
    headers,
    data: { name: collection, kind: "collection", draft_and_publish: true, attributes: { title: { type: "string", required: true } } },
  });
  expect(created.ok(), await created.text()).toBeTruthy();

  try {
    // The admin creates the role with the permission matrix.
    await login(page);
    await page.goto("/settings/roles");
    await expect(page.getByRole("heading", { name: "Roles" })).toBeVisible();
    await expect(page.getByRole("row").filter({ hasText: "editor" }).getByText("built-in")).toBeVisible();
    await page.getByRole("button", { name: "New role" }).first().click();
    const drawer = page.getByRole("dialog");
    await drawer.getByRole("textbox", { name: /^Name/ }).fill(roleName);
    await drawer.getByRole("textbox", { name: /^Description/ }).fill("Writes memos, an editor publishes them");
    const box = (action: string) => drawer.getByRole("checkbox", { name: `${action} ${collection}`, exact: true });
    // Chakra hides the native input under a styled control, and the matrix
    // can be long (other tests leave collections behind): click the input
    // itself instead of a screen position.
    const tick = async (action: string) => {
      await box(action).evaluate((el) => (el as HTMLInputElement).click());
      await expect(box(action)).toBeChecked();
    };
    await expect(box("Create")).not.toBeChecked();
    // Any action implies read.
    await tick("Create");
    await expect(box("Read")).toBeChecked();
    await tick("Update");
    await expect(box("Publish")).not.toBeChecked();
    await drawer.getByRole("button", { name: "Create role" }).click();
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(page.getByRole("row").filter({ hasText: roleName })).toContainText(`${collection}: read, create, update`);

    const stored = await request.get(`${apiURL}/admin/roles/${roleName}`, { headers });
    expect(stored.ok(), await stored.text()).toBeTruthy();
    const perms = (await stored.json()).data.permissions;
    expect(perms.content[collection]).toEqual(expect.arrayContaining(["read", "create", "update"]));
    expect(perms.content[collection]).not.toContain("publish");
    expect(perms.content["*"] ?? []).toEqual([]);

    // The Users page offers the custom role.
    await page.goto("/settings/users");
    await page.getByRole("button", { name: "New user" }).first().click();
    const form = page.getByRole("dialog");
    await form.getByRole("textbox", { name: /^Username/ }).fill(user.username);
    await form.getByRole("textbox", { name: /^Email/ }).fill(`${user.username}@example.com`);
    await form.getByLabel(/^Password/).fill(user.password);
    await form.getByLabel("Role").selectOption(roleName);
    await form.getByRole("button", { name: "Create user" }).click();
    await expect(page.getByRole("row").filter({ hasText: user.username })).toContainText(roleName);

    // Signed in with the role: only its collection, no media, no publishing.
    await page.getByRole("button", { name: "Account menu" }).click();
    await page.getByRole("menuitem", { name: "Sign out" }).click();
    await login(page, user);
    await expect(page.getByRole("link", { name: "Content Editor" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Media Library" })).toHaveCount(0);
    await expect(page.getByRole("link", { name: "Users" })).toHaveCount(0);

    await page.goto("/contrib");
    // Types are listed by display name (capitalized).
    await expect(page.getByRole("button", { name: new RegExp(`^${collection}$`, "i") })).toBeVisible();
    await expect(page.getByRole("button", { name: /^articles$/i })).toHaveCount(0);
    await expect(page.getByRole("heading", { name: new RegExp(`^${collection}$`, "i") })).toBeVisible();

    await page.getByRole("button", { name: "Quick Add" }).click();
    const entry = page.getByRole("dialog");
    await entry.getByLabel("title").fill(draft);
    await entry.getByRole("button", { name: "Save draft" }).click();
    await expect(page.getByRole("dialog")).toHaveCount(0);
    const row = page.getByRole("row").filter({ hasText: draft });
    await expect(row).toHaveCount(1);
    await expect(row.getByText("draft", { exact: true })).toBeVisible();
    await expect(row.getByRole("button", { name: "Edit item" })).toBeVisible();
    await expect(row.getByRole("button", { name: /^(Publish|Unpublish) item$/ })).toHaveCount(0);
    await expect(row.getByRole("button", { name: "Delete item" })).toHaveCount(0);

    // Once someone who can publish does, the entry is live: the writer can
    // no longer change it, nor restore an earlier version.
    const listed = await request.get(`${apiURL}/api/collections/${collection}?status=all`, { headers });
    const item = ((await listed.json()).data as { id: number; title: string }[]).find((i) => i.title === draft);
    expect(item).toBeTruthy();
    const published = await request.post(`${apiURL}/api/collections/${collection}/${item!.id}/publish`, { headers });
    expect(published.ok(), await published.text()).toBeTruthy();
    await page.reload();
    await expect(row.getByText("published", { exact: true })).toBeVisible();
    await expect(row.getByRole("button", { name: "Edit item" })).toBeDisabled();
    await row.getByRole("button", { name: "Item history" }).click();
    const history = page.getByRole("dialog");
    await history.getByRole("button", { name: /^Version 1,/ }).click();
    await expect(history.getByRole("heading", { name: "Version 1" })).toBeVisible();
    await expect(history.getByRole("button", { name: "Restore this version" })).toBeDisabled();
    await history.getByRole("button", { name: "Close" }).click();
    await expect(page.getByRole("dialog")).toHaveCount(0);

    // Other collections are out of reach.
    await page.goto("/contrib?collection=articles");
    await expect(page.getByText("“articles” was not found.")).toBeVisible();
    expect(errors).toEqual([]);
  } finally {
    // Delete the user first: a role in use cannot be deleted.
    const users = await request.get(`${apiURL}/admin/users`, { headers });
    const list = ((await users.json()).data?.users ?? []) as { ID: number; username: string }[];
    const id = list.find((u) => u.username === user.username)?.ID;
    if (id) await request.delete(`${apiURL}/admin/users/${id}`, { headers });
    await request.delete(`${apiURL}/admin/roles/${roleName}`, { headers });
    await request.delete(`${apiURL}/admin/collections/${collection}`, { headers });
  }
});

test("roles: built-in roles are read-only and a role in use cannot be deleted", async ({ page, request }) => {
  const errors = trackErrors(page);
  const headers = await adminHeaders(request);
  const suffix = `${Date.now().toString(36)}${test.info().retry}`;
  const roleName = `reader-${suffix}`;
  const role = await request.post(`${apiURL}/admin/roles`, {
    headers,
    data: { name: roleName, description: "Reads everything", permissions: { content: { "*": ["read"] }, media: ["read"] } },
  });
  expect(role.ok(), await role.text()).toBeTruthy();
  const token = await request.post(`${apiURL}/admin/tokens`, { headers, data: { name: `t-${suffix}`, role: roleName } });
  expect(token.ok(), await token.text()).toBeTruthy();
  const tokenId = (await token.json()).data.id as number;

  try {
    await login(page);
    await page.goto("/settings/roles");
    // A built-in role opens read-only.
    await page.getByRole("button", { name: "View viewer" }).click();
    const drawer = page.getByRole("dialog");
    await expect(drawer.getByRole("checkbox", { name: "Read All other types", exact: true })).toBeChecked();
    await expect(drawer.getByRole("button", { name: "Save changes" })).toHaveCount(0);
    await drawer.getByRole("button", { name: "Close" }).first().click();
    await expect(page.getByRole("dialog")).toHaveCount(0);

    // The token's role cannot be deleted while the token has it.
    await page.getByRole("button", { name: `Delete ${roleName}` }).click();
    await page.getByRole("alertdialog").getByRole("button", { name: "Delete", exact: true }).click();
    await expect(page.getByText(`Role ${roleName} is still in use`)).toBeVisible();
    await expect(page.getByText("It is given to 1 API token.")).toBeVisible();

    // Revoked, it is no longer counted.
    await request.delete(`${apiURL}/admin/tokens/${tokenId}`, { headers });
    await page.getByRole("button", { name: `Delete ${roleName}` }).click();
    await page.getByRole("alertdialog").getByRole("button", { name: "Delete", exact: true }).click();
    await expect(page.getByText(`Role ${roleName} deleted`)).toBeVisible();
    await expect(page.getByRole("row").filter({ hasText: roleName })).toHaveCount(0);
    expect(errors).toEqual([]);
  } finally {
    await request.delete(`${apiURL}/admin/tokens/${tokenId}`, { headers });
    await request.delete(`${apiURL}/admin/roles/${roleName}`, { headers });
  }
});
