import { expect, type Page, test } from "@playwright/test";
import { adminHeaders, apiURL, editor, login, trackErrors } from "./fixtures";

// Agents v2 end to end: templates, approvals and test runs. Proposals need
// an LLM to be created, so the Approvals page is covered through its empty
// states and who can see it.

async function signOut(page: Page) {
  await page.getByRole("button", { name: "Account menu" }).click();
  await page.getByRole("menuitem", { name: "Sign out" }).click();
  await expect(page).toHaveURL(/\/login$/);
}

/** The Approvals entry of the sidebar (menu items are links with role group). */
const approvalsMenu = (page: Page) => page.locator('a[href="/approvals"]');

/** Chakra hides native checkboxes, switches and radios under a styled control: click the input itself. */
async function clickInput(page: Page, label: string | RegExp, scope = page.locator("body")) {
  await scope.getByLabel(label).evaluate((el) => (el as HTMLInputElement).click());
}

test("approvals: reviewers get the page and its badge, viewers do not", async ({ page, request }) => {
  const errors = trackErrors(page);
  const headers = await adminHeaders(request);
  const suffix = `${Date.now().toString(36)}${test.info().retry}`;
  const viewer = { username: `v${suffix}`, password: "viewer-password-123" };
  const created = await request.post(`${apiURL}/admin/users`, {
    headers,
    data: { username: viewer.username, email: `${viewer.username}@example.com`, password: viewer.password, role_name: "viewer" },
  });
  expect(created.ok(), await created.text()).toBeTruthy();

  try {
    await login(page);
    await approvalsMenu(page).first().click();
    await expect(page).toHaveURL(/\/approvals$/);
    await expect(page.getByRole("heading", { name: "Approvals" })).toBeVisible();
    await expect(page.getByRole("tab", { name: "Pending" })).toHaveAttribute("aria-selected", "true");
    await expect(page.getByText("Nothing to review")).toBeVisible();
    // No proposal is pending: the menu shows no count.
    await expect(page.getByTestId("approvals-count")).toHaveCount(0);

    await page.getByRole("tab", { name: "All" }).click();
    await expect(page).toHaveURL(/status=all/);
    await expect(page.getByText("No proposals", { exact: true })).toBeVisible();

    // Filtered by a run (the link from a run's details).
    await page.goto("/approvals?status=all&agentRun=999999");
    await expect(page.getByText("Run #999999")).toBeVisible();
    await expect(page.getByText("No proposals match these filters.")).toBeVisible();
    await page.getByRole("button", { name: "Clear filters" }).click();
    await expect(page).not.toHaveURL(/agentRun/);
    await expect(page.getByText("No proposals", { exact: true })).toBeVisible();

    // Editors review too.
    await signOut(page);
    await login(page, editor);
    await expect(approvalsMenu(page)).toBeVisible();

    // Viewers change nothing, so they review nothing.
    await signOut(page);
    await login(page, viewer);
    await expect(page.locator('a[href="/contrib"]').first()).toBeVisible();
    await expect(approvalsMenu(page)).toHaveCount(0);
    await page.goto("/approvals");
    await expect(page).toHaveURL(/\/dashboard$/);
    expect(errors).toEqual([]);
  } finally {
    const users = await request.get(`${apiURL}/admin/users`, { headers });
    const list = ((await users.json()).data?.users ?? []) as { ID: number; username: string }[];
    const id = list.find((u) => u.username === viewer.username)?.ID;
    if (id) await request.delete(`${apiURL}/admin/users/${id}`, { headers });
  }
});

test("agents: create an agent from a template that needs approval", async ({ page, request }) => {
  const errors = trackErrors(page);
  const headers = await adminHeaders(request);
  const suffix = `${Date.now().toString(36)}${test.info().retry}`;
  const collection = `posts${suffix}`;
  const agentName = `summarizer-${collection}`;
  const created = await request.post(`${apiURL}/admin/collections`, {
    headers,
    data: {
      name: collection,
      kind: "collection",
      attributes: { title: { type: "string" }, body: { type: "text" }, summary: { type: "text" }, views: { type: "int" } },
    },
  });
  expect(created.ok(), await created.text()).toBeTruthy();

  try {
    await login(page);
    await page.goto("/agents/new");
    await expect(page.getByRole("heading", { name: "Start from a template" })).toBeVisible();
    await expect(page.getByRole("article", { name: "Blank agent" })).toBeVisible();
    await page.getByRole("article", { name: "Summarizer" }).getByRole("button", { name: "Use template" }).click();

    const dialog = page.getByRole("dialog", { name: "Summarizer" });
    await expect(dialog).toBeVisible();
    // The required fields are checked before the template is rendered.
    await dialog.getByRole("button", { name: "Continue" }).click();
    await expect(dialog.getByText("Choose a collection.")).toBeVisible();

    await dialog.getByRole("combobox", { name: "Collection", exact: true }).selectOption(collection);
    // Only text fields can be summarized: views (an int) is not offered.
    const source = dialog.getByRole("combobox", { name: "Text to summarize" });
    await expect(source.locator("option", { hasText: "body (text)" })).toHaveCount(1);
    await expect(source.locator("option", { hasText: "views" })).toHaveCount(0);
    await source.selectOption("body");
    await dialog.getByRole("combobox", { name: "Field for the summary" }).selectOption("summary");
    await expect(dialog.getByLabel("Agent name")).toHaveAttribute("placeholder", agentName);
    const approval = dialog.getByLabel("Changes need an editor's approval");
    await expect(approval).not.toBeChecked();
    await clickInput(page, "Changes need an editor's approval", dialog);
    await expect(approval).toBeChecked();
    await dialog.getByRole("button", { name: "Continue" }).click();
    await expect(page.getByRole("dialog")).toHaveCount(0);

    // The form opens with the rendered definition, to review.
    await expect(page.getByText("Prepared from the Summarizer template")).toBeVisible();
    await expect(page.getByRole("textbox", { name: /^Name/ })).toHaveValue(agentName);
    await expect(page.getByRole("textbox", { name: /^System prompt/ })).toHaveValue(new RegExp(`"${collection}" collection`));
    const requireApproval = page.getByLabel("Require approval");
    await expect(requireApproval).toBeChecked();
    // Toggled off and on again.
    await clickInput(page, "Require approval");
    await expect(requireApproval).not.toBeChecked();
    await clickInput(page, "Require approval");
    await expect(requireApproval).toBeChecked();

    const saved = page.waitForRequest((r) => r.method() === "POST" && new URL(r.url()).pathname === "/admin/agents");
    await page.getByRole("button", { name: "Create agent" }).click();
    const body = (await saved).postDataJSON();
    expect(body.require_approval).toBe(true);
    expect(body.trigger).toEqual({ type: "collection_event", event_trigger: { collection, events: ["item:created", "item:updated"] } });
    expect(body.functions.map((f: { impl_key: string }) => f.impl_key)).toEqual(["collections.patch_item"]);
    expect(body.functions[0].collections).toEqual([collection]);

    await expect(page).toHaveURL(new RegExp(`/agents\\?name=${agentName}$`));
    await expect(page.getByRole("heading", { name: agentName })).toBeVisible();
    const stored = await request.get(`${apiURL}/admin/agents/${agentName}`, { headers });
    expect(stored.ok(), await stored.text()).toBeTruthy();
    expect(errors).toEqual([]);
  } finally {
    await request.delete(`${apiURL}/admin/agents/${agentName}`, { headers });
    await request.delete(`${apiURL}/admin/collections/${collection}`, { headers });
  }
});

test("agents: a test run of a collection-event agent", async ({ page, request }) => {
  const errors = trackErrors(page);
  const headers = await adminHeaders(request);
  // An agent of its own: with Redis the run is recorded, and other tests
  // expect e2e-tagger to have none. Deleting it deletes its runs.
  const agent = `tester-${Date.now().toString(36)}${test.info().retry}`;
  const created = await request.post(`${apiURL}/admin/agents`, {
    headers,
    data: {
      name: agent,
      system_prompt: "Tag new articles.",
      max_turns: 2,
      llm_config: { provider: "openai", model: "gpt-4o-mini", api_key: "sk-e2e" },
      memory: { type: "in-memory", session_scope: "agent" },
      trigger: { type: "collection_event", event_trigger: { collection: "articles", events: ["item:created", "item:updated"] } },
      require_approval: true,
      limits: { max_runs_per_hour: 10 },
      functions: [],
    },
  });
  expect(created.ok(), await created.text()).toBeTruthy();

  try {
    await login(page);
    await page.goto(`/agents?name=${agent}`);
    await expect(page.getByRole("heading", { name: agent })).toBeVisible();
    await page.getByRole("button", { name: "Test run" }).click();

    const dialog = page.getByRole("dialog", { name: `Test ${agent}` });
    await expect(dialog).toBeVisible();
    // A collection-event agent is tried on an entry of its collection.
    await expect(dialog.getByLabel("On an entry of articles")).toBeChecked();
    const entry = dialog.getByRole("combobox", { name: "Entry", exact: true });
    const seeded = entry.locator("option", { hasText: "Seeded article" }).first();
    await expect(seeded).toBeAttached();
    await entry.selectOption((await seeded.getAttribute("value")) ?? "");
    await expect(dialog.getByRole("combobox", { name: "Event", exact: true })).toHaveValue("item:created");

    const queued = page.waitForResponse((r) => r.request().method() === "POST" && new URL(r.url()).pathname === `/admin/agents/${agent}/test`);
    await dialog.getByRole("button", { name: "Start test run" }).click();
    const res = await queued;
    const sent = res.request().postDataJSON();
    expect(sent.event).toBe("item:created");
    expect(Number.isInteger(sent.item_id)).toBeTruthy();

    if (!res.ok()) {
      // Without Redis runs cannot be queued (503 when it is not configured,
      // 500 when it is unreachable): the modal shows why and stays open.
      const message = (await res.json()).error.message as string;
      expect(message).toMatch(/Redis|queue/i);
      await expect(dialog.getByRole("alert")).toContainText(message);
      await expect(dialog).toBeVisible();
      // The same with a message instead of an entry.
      await clickInput(page, "With a message", dialog);
      await dialog.getByLabel("Input (optional)").fill("Tag the latest article");
      const again = page.waitForResponse((r) => r.request().method() === "POST" && new URL(r.url()).pathname === `/admin/agents/${agent}/test`);
      await dialog.getByRole("button", { name: "Start test run" }).click();
      expect((await again).request().postDataJSON()).toEqual({ input: "Tag the latest article" });
      await expect(dialog.getByRole("alert")).toContainText(message);
      await dialog.getByRole("button", { name: "Cancel" }).click();
      await expect(page.getByRole("dialog")).toHaveCount(0);
    } else {
      // Queued: the run opens in the Runs tab and is followed until it ends.
      expect(res.status()).toBe(202);
      const { runId, dryRun } = (await res.json()).data;
      expect(dryRun).toBe(true);
      const drawer = page.getByRole("dialog");
      await expect(drawer).toContainText(`Run #${runId}`);
      await expect(drawer.getByText("test run", { exact: true })).toBeVisible();
      await expect(drawer.getByText(/Test run: the agent ran for real/)).toBeVisible();
      await drawer.getByRole("button", { name: "Close" }).click();
      await expect(page.getByRole("row").filter({ hasText: "test run" }).first()).toBeVisible();
    }
    expect(errors).toEqual([]);
  } finally {
    await request.delete(`${apiURL}/admin/agents/${agent}`, { headers });
  }
});
