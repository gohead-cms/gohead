// Compile-time checks of the public types (npm run typecheck); not executed.
import { createClient, type Entry, type Media } from "@gohead-cms/client";

interface Article {
  title: string;
  slug?: string | null;
  views?: number | null;
  category?: "tech" | "life" | null;
  cover?: Media | null;
  author?: Entry<{ name: string }> | null;
}
interface ArticleInput {
  title: string;
  slug?: string | null;
  views?: number | null;
  category?: "tech" | "life" | null;
  cover?: number | null;
  author?: number | null;
}

declare module "@gohead-cms/client" {
  interface Collections {
    articles: { attributes: Article; input: ArticleInput };
  }
  interface Singletons {
    homepage: { attributes: { headline?: string | null }; input: { headline?: string | null } };
  }
}

const gohead = createClient({ url: "http://localhost:8080" });

export async function checks(): Promise<void> {
  const articles = gohead.collection("articles");
  const { data } = await articles.find({ filters: { views: { $gte: 10 }, category: "tech", slug: { $contains: "go" } }, sort: ["views:desc", "title"] });
  const first = data[0];
  if (first) {
    const title: string = first.title;
    const id: number = first.id;
    const name: string | undefined = first.author?.name;
    void [title, id, name];
  }
  await articles.create({ title: "x", author: 1 });
  // @ts-expect-error title is required
  await articles.create({ slug: "x" });
  // @ts-expect-error unknown field
  await articles.find({ filters: { nope: 1 } });
  // @ts-expect-error relations cannot be filtered
  await articles.find({ filters: { author: 1 } });
  // @ts-expect-error not a sortable field
  await articles.find({ sort: "cover:desc" });
  // @ts-expect-error wrong enum value
  await articles.find({ filters: { category: "food" } });

  const home = await gohead.singleton("homepage").find();
  const headline: string | null | undefined = home?.headline;
  void headline;

  // Unknown names stay untyped.
  const other = await gohead.collection("unknown").findOne(1);
  const anything: unknown = other.whatever;
  void anything;
}
