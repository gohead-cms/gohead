# Public demo

A public demo lets people try GoHead without installing it: they open the admin UI, sign in
with an account shown on the login page, and look at everything. GoHead's **demo mode**
keeps them from breaking it, and puts the content back every night.

## Demo mode

```yaml
demo:
  enabled: true            # GOHEAD_DEMO_ENABLED
  username: demo           # GOHEAD_DEMO_USERNAME
  password: try-gohead-demo  # GOHEAD_DEMO_PASSWORD
  reset_at: "03:00"        # GOHEAD_DEMO_RESET_AT: daily, UTC ("" never)
  snapshot: ""             # GOHEAD_DEMO_SNAPSHOT: default <database file>.demo-snapshot
```

At start, GoHead creates the visitors' account with the `demo` role, and the login page
offers **Sign in to the demo**. Visitors:

- see the whole admin: content, the content model, agents and their runs, connected apps,
  approvals (including calls to connected apps), webhooks, users and API tokens (never their
  secrets);
- create and edit entries: drafts in draft-and-publish collections, since they cannot
  publish, and nothing they can delete;
- try agents with **Test run**: runs read content and call the model, and report the
  changes they would make;
- cannot change anything else: settings, the content model, agents, users, tokens,
  connections, webhooks; they cannot upload files, and calls to connected apps waiting for
  approval are yours to approve.

A banner tells them, and the server refuses the rest with `403`.

Every day at `reset_at`, GoHead puts every table back as it was in the **snapshot**, in one
transaction: content, agents and their runs, proposals, users. Take the snapshot once your
demo is set up, as an admin:

```bash
curl -X POST $GOHEAD/admin/demo/snapshot -H "$AUTH"   # save the database as it is now
curl -X POST $GOHEAD/admin/demo/reset -H "$AUTH"      # go back to it now
curl $GOHEAD/admin/demo -H "$AUTH"                    # settings, snapshot time, next reset
```

Demo mode needs SQLite (`gohead standalone`): the reset copies SQLite tables. Uploaded files
are not part of the snapshot, which is fine since visitors cannot upload.

!!! warning "Visitors use your model"
    Agents run on your LLM key whenever visitors create entries or start test runs. Use a
    small model, give every agent `limits`, and set a monthly budget with your provider.

## The GoHead demo on Render

[`deploy/demo/render.yaml`](https://github.com/gohead-cms/gohead/blob/master/deploy/demo/render.yaml)
deploys a complete demo in your Render account, for about $14 a month:

- **gohead-demo**: GoHead in demo mode (`gohead standalone`, SQLite on a 1 GB disk), built
  from the latest commit of `master` (`GOHEAD_VERSION=edge`), with `gpt-4o-mini` for agents;
- **gohead-demo-site**: the [conference starter](../frontend/feature-tour.md), which reads
  from GoHead over Render's private network.

1. In the [Render dashboard](https://dashboard.render.com), choose **New → Blueprint**, connect
   `gohead-cms/gohead` (or your fork), and set the **Blueprint path** to
   `deploy/demo/render.yaml`.
2. Fill in your admin account (`GOHEAD_ADMIN_EMAIL`, `GOHEAD_ADMIN_PASSWORD`, 12 characters or
   more) and your OpenAI key (`GOHEAD_LLM_API_KEY`). Leave the site's tokens empty for now.
   Click **Deploy Blueprint**.
3. Once **gohead-demo** is live, set up its content from your machine, in a clone of the
   repository. In `templates/nextjs-conference`, create `.env.local`:

    ```bash
    GOHEAD_URL=https://gohead-demo.onrender.com      # gohead-demo's URL
    GOHEAD_USERNAME=admin
    GOHEAD_PASSWORD=<your admin password>
    GOHEAD_REVALIDATE_URL=https://gohead-demo-site.onrender.com/api/revalidate
    # Optional: an agent that opens a GitHub issue for each talk proposal.
    GOHEAD_DEMO_GITHUB_TOKEN=github_pat_…   # fine-grained, Issues: read and write, one repository
    GOHEAD_DEMO_GITHUB_REPO=you/gohead-demo-issues
    ```

    Then run:

    ```bash
    npm install
    npm run gohead:setup && npm run gohead:agents && node ../../deploy/demo/setup.mjs
    ```

    The first two load the starter's content model, sample content, roles, tokens, webhook
    and agents. The last one adds the GitHub connection and its agent, takes the snapshot, and
    prints six settings.
4. Paste them into **gohead-demo-site → Environment**, and deploy the site again.

The site is at `https://gohead-demo-site.onrender.com`, and the admin UI at
`https://gohead-demo.onrender.com`. Try it as a visitor would: send a talk proposal from the
site's call for papers, then sign in to the demo. **proposal-triage** has scored it, and, with
GitHub connected, **cfp-to-github** has a call waiting on the Approvals page. Sign in as
yourself to approve it, and the issue appears in your repository.

### Keeping it up to date

The demo is not rebuilt when GoHead changes. To show the latest commit, deploy both services
again (**Manual Deploy → Clear build cache & deploy** for GoHead). When the new version
changes the starter's content model or agents, run step 3 again: it keeps what exists, adds
what is new and takes a new snapshot.
