import { expect, test } from "@playwright/test";
import { adminHeaders, apiURL, login, trackErrors } from "./fixtures";
import { MCP_TOKEN, startMcpServer } from "./mcp-server";

// Connected apps: connecting an MCP server, its tools in the agent's tool
// picker, approval settings. The API must allow private networks for
// connections (GOHEAD_CONNECTIONS_ALLOW_PRIVATE_NETWORKS=true): the test
// server runs on 127.0.0.1.

test("connected apps: connect an MCP server and give its tools to an agent", async ({ page, request }) => {
  const errors = trackErrors(page);
  const headers = await adminHeaders(request);
  const suffix = `${Date.now().toString(36)}${test.info().retry}`;
  const name = `tracker-${suffix}`;
  const agentName = `triager-${suffix}`;
  const server = await startMcpServer();

  try {
    await login(page);
    await page.locator('a[href="/connections"]').first().click();
    await expect(page.getByRole("heading", { name: "Connected apps" })).toBeVisible();

    // Connect a custom server.
    await page.getByRole("button", { name: "Connect an app" }).first().click();
    const dialog = page.getByRole("dialog", { name: "Connect an app" });
    await expect(dialog.getByRole("button", { name: "Connect GitHub" })).toBeVisible();
    await dialog.getByRole("button", { name: "Connect Other MCP server" }).click();
    const form = page.getByRole("dialog", { name: "Connect an MCP server" });
    await form.getByRole("textbox", { name: /^Name/ }).fill(`Tracker ${suffix}`);
    await form.getByRole("textbox", { name: /^Prefix of its tools/ }).fill(name);
    await form.getByRole("textbox", { name: /^Server URL/ }).fill(server.url);
    await form.getByLabel(/^Token/).fill(MCP_TOKEN);
    await form.getByRole("button", { name: "Connect", exact: true }).click();
    await expect(form).toBeHidden();

    const card = page.getByTestId(`connection-${name}`);
    await expect(card.getByText("2 tools")).toBeVisible();
    await card.getByRole("button", { name: "Show the tools" }).click();
    await expect(card.getByText(`${name}.create_issue`)).toBeVisible();
    await expect(card.getByText("Writes · approval")).toBeVisible();

    // The token is never sent back.
    const res = await request.get(`${apiURL}/admin/connections/${name}`, { headers });
    const conn = (await res.json()).data;
    expect(conn.hasToken).toBe(true);
    expect(JSON.stringify(conn)).not.toContain(MCP_TOKEN);

    // Its tools are in the agent's tool picker, by app.
    await page.goto("/agents/new");
    await page.getByRole("article", { name: "Blank agent" }).getByRole("button", { name: "Start blank" }).click();
    await page.getByRole("textbox", { name: /^Name/ }).fill(agentName);
    await page.getByRole("textbox", { name: /^System prompt/ }).fill("File an issue for each bug report.");
    await page.getByRole("button", { name: "Add a tool" }).click();
    const picker = page.getByRole("dialog", { name: "Add a tool" });
    await picker.getByRole("textbox", { name: "Search tools" }).fill("create issue");
    const apps = picker.getByRole("region", { name: "Connected apps" });
    await expect(apps.getByRole("group", { name: `Tracker ${suffix}` })).toBeVisible();
    await expect(picker.getByRole("region", { name: "Content" })).toBeHidden();
    await apps.getByRole("button", { name: /^Create an issue/ }).click();

    // Tools that may change something wait for approval by default.
    const tool = page.getByTestId(`tool-${name}.create_issue`);
    await expect(tool.getByText(`called ${name}_create_issue by the model`)).toBeVisible();
    const approval = tool.getByLabel("Wait for an admin's approval before each call");
    await expect(approval).toBeChecked();

    const saved = page.waitForRequest((r) => r.method() === "POST" && new URL(r.url()).pathname === "/admin/agents");
    await page.getByRole("button", { name: "Create agent" }).click();
    expect((await saved).postDataJSON().tools).toEqual([{ use: `${name}.create_issue` }]);
    await expect(page).toHaveURL(new RegExp(`/agents\\?name=${agentName}$`));
    expect(errors).toEqual([]);
  } finally {
    await request.delete(`${apiURL}/admin/agents/${agentName}`, { headers });
    await request.delete(`${apiURL}/admin/connections/${name}`, { headers });
    await server.close();
  }
});
