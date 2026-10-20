import type { SetupIssue } from "@/lib/gohead";

/**
 * Shown instead of the site while GoHead cannot serve it: not running, model
 * not pushed yet, or content not readable. Meant for the developer, so it is
 * in English and says exactly what to run.
 */
export function SetupScreen({ issue }: { issue: SetupIssue }) {
  const { title, intro, steps } = explain(issue);
  return (
    <main id="main" className="wrap py-16 sm:py-24">
      <p className="text-sm font-semibold text-link">GoHead + Next.js starter</p>
      <h1 className="mt-4 max-w-[16ch] text-5xl leading-[0.95] font-extrabold tracking-[-0.045em] text-balance sm:text-7xl">
        {title}
      </h1>
      <p className="mt-6 max-w-[56ch] font-serif text-xl leading-relaxed text-muted">{intro}</p>

      <ol className="mt-12 grid max-w-3xl gap-8">
        {steps.map((step, i) => (
          <li key={i} className="grid grid-cols-[2.5rem_1fr] gap-x-4">
            <span
              aria-hidden
              className="grid size-10 place-items-center rounded-full bg-marker text-lg font-bold text-on-marker"
            >
              {i + 1}
            </span>
            <div className="min-w-0 pt-1.5">
              <p className="text-lg font-semibold">{step.text}</p>
              {step.code && (
                <pre className="mt-3 overflow-x-auto rounded-md bg-paper-raised px-4 py-3 text-[0.9rem] leading-relaxed shadow-[inset_0_0_0_1px_var(--rule)]">
                  <code>{step.code}</code>
                </pre>
              )}
              {step.note && <p className="mt-2 text-muted">{step.note}</p>}
            </div>
          </li>
        ))}
      </ol>

      <details className="mt-14 max-w-3xl text-sm text-muted">
        <summary className="cursor-pointer font-medium text-ink">What GoHead answered</summary>
        <p className="mt-2 break-words">
          <code>{issue.detail}</code>
        </p>
        <p className="mt-2">
          GOHEAD_URL is <code>{issue.url}</code>. See the README for every setting.
        </p>
      </details>
    </main>
  );
}

interface Step {
  text: string;
  code?: string;
  note?: string;
}

function explain(issue: SetupIssue): { title: string; intro: string; steps: Step[] } {
  const setup: Step = {
    text: "Create the content model and the sample articles.",
    code: "npm run gohead:setup",
    note: "It signs in with GOHEAD_USERNAME and GOHEAD_PASSWORD from .env.local (an admin account).",
  };
  const reload: Step = { text: "Reload this page." };
  switch (issue.kind) {
    case "unreachable":
      return {
        title: "Connect your GoHead",
        intro: `This blog reads its content from GoHead, and nothing answered at ${issue.url}.`,
        steps: [
          {
            text: "Start GoHead. The docker-compose.yml in this project runs it on port 8080.",
            code: "docker compose up -d",
            note: "Running GoHead somewhere else? Set GOHEAD_URL in .env.local and restart npm run dev.",
          },
          setup,
          reload,
        ],
      };
    case "model":
      return {
        title: "Almost there",
        intro: `GoHead is running at ${issue.url}, but it does not have this blog’s content types yet.`,
        steps: [setup, reload],
      };
    case "access":
      return {
        title: "Let the site read your content",
        intro: `GoHead at ${issue.url} answered, but did not let this site read articles.`,
        steps: [
          {
            text: "Allow anonymous reads of the blog’s content types (gohead/model.json lists them under public).",
            code: "npm run gohead:push",
          },
          {
            text: "Or keep your content private and give the site a read-only API token.",
            code: "npx gohead-client token create --name website --role viewer",
            note: "Put the token in .env.local as GOHEAD_TOKEN and restart the server.",
          },
          reload,
        ],
      };
  }
}
