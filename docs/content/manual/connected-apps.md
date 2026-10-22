# Connected apps

Connected apps let [agents](agents.md) work with other services: open a GitHub issue for a
bug report, email a summary with Gmail, add a row to a Google Sheet. Only admins connect
apps, under **AI → Connected apps**.

## Connect an app

1. Open **AI → Connected apps** and click **Connect an app**.
2. Choose the service:
    - **GitHub**, for issues, pull requests and repositories;
    - **Zapier**, for Gmail, Google Drive, Sheets, Slack, Notion and the thousands of other
      apps Zapier connects to;
    - **Other MCP server**, for any other service or your own server that speaks MCP (the
      Model Context Protocol). Enter its **Server URL** and how it expects the token.
3. Paste the token. The help under the field says where to get one, with a link:
    - for GitHub, create a *fine-grained personal access token* that only reaches the
      repositories, and only has the permissions, the agents need;
    - for Zapier, add an MCP server at mcp.zapier.com, choose **Other** as the AI agent, add
      the actions agents may use (for example *Gmail: Send Email*), then generate a token
      in its **Settings**.
4. Click **Connect**.

The app appears with the number of its tools. **Show the tools** lists them: what each one
does and whether it only **reads** or may **write** (send, create, delete…). If the app
could not be reached or refused the token, it shows the error: click **Edit** to fix the
token, or **Refresh** to try again.

!!! note "Tokens"
    A token is never shown again once saved. To replace it, click **Edit** and paste the new
    one; leave the field empty to keep the current one. If the form says tokens are stored
    unencrypted, ask whoever runs GoHead to set `jwt_secret`.

When you change what a service offers (for Zapier, after adding or removing actions), click
**Refresh** so agents see the new tools.

## Use an app's tools in an agent

1. Open the agent and click **Edit**.
2. Under **Tools**, click **Add a tool**. The **Connected apps** section lists each app's
   tools; type in the search box to find one, for example *create issue*.
3. Click the tool to add it. Its card shows the app it belongs to and the name the model
   knows it by, such as `github_create_issue`: use that name in the system prompt to tell
   the agent when to use it.
4. Check **Wait for an admin's approval before each call**. It is on for tools that may
   write, so you see each email or issue before it is sent; turn it off only for agents you
   trust with that tool. If the agent has **Require approval** on, its calls that may write
   wait for approval anyway.
5. Click **Save**.

Try the agent with a [test run](agents.md#try-an-agent-with-a-test-run) first: tools that
may write are not called in a test run, and the run shows the calls the agent would make.

## Approve a call

Calls that wait for approval appear on the [Approvals](approvals.md) page, marked **call**,
with the tool's name (`github.create_issue`). Only admins see them. Open one to see which app
and tool the agent wants to call, with which arguments, then:

- **Approve and call** makes the call. The proposal then shows what the app answered.
- **Edit and call** lets you correct the arguments (as JSON) before calling.
- **Reject** discards it, with an optional note. Nothing is sent.

If the call fails (the app refused it, or the connection was removed), the proposal is
marked **failed** with the reason.

## Disconnect an app

Click the bin icon on the app and confirm. Its token is deleted, its calls waiting for
approval are rejected, and agents that use its tools get an error when they call them:
remove those tools from the agents.

More details, and the API: [Connected apps](../agents/connected-apps.md).
