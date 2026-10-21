# @gohead-cms/create

Create a website on [GoHead](https://github.com/gohead-cms/gohead) from a starter template.

```bash
npm create @gohead-cms@latest my-site
# or: pnpm create @gohead-cms my-site · yarn create @gohead-cms my-site · bun create @gohead-cms my-site
```

Options (after `--` with npm): `--template <name>`, `--url <your GoHead>`, `--no-install`, `--yes`.

| Template | |
|----------|-|
| `nextjs-blog` | Next.js (App Router) blog: articles, authors, categories, English and French, revalidation on publish |
| `nextjs-conference` | Next.js conference site that uses every GoHead feature: forms through a scoped token, draft preview, GraphQL, seven agents ([feature tour](https://gohead-cms.github.io/gohead/frontend/feature-tour/)) |

Each template ships its content model (`gohead/model.json`), sample content (`gohead/seed.json`) and an `npm run gohead:setup` script that applies them with [`@gohead-cms/client`](https://www.npmjs.com/package/@gohead-cms/client).

MIT license
