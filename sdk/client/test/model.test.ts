import { describe, expect, it } from "vitest";

import { normalizeAttribute, validateModel, type Model } from "../src/cli/model.js";
import { generateTypes, singularize, typeName } from "../src/cli/typegen.js";

const model: Model = {
  components: { seo: { attributes: { meta_title: { type: "string" }, image: { type: "media" } } } },
  collections: {
    authors: { attributes: { name: { type: "string", required: true }, secret: { type: "password" } } },
    articles: {
      draftAndPublish: true,
      attributes: {
        title: { type: "string", required: true, localized: true },
        category: { type: "enum", options: ["tech", "life"] },
        views: { type: "int" },
        published: { type: "bool" },
        author: { type: "relation", relation: "oneToOne", target: "authors" },
        related: { type: "relation", relation: "manyToMany", target: "articles" },
        seo: { type: "component", component: "seo" },
        blocks: { type: "component", component: "seo", repeatable: true },
        gallery: { type: "media", multiple: true },
        data: { type: "json" },
        "odd-name": { type: "string" },
      },
    },
  },
  singletons: { homepage: { attributes: { featured: { type: "relation", relation: "manyToMany", target: "articles" } } } },
};

describe("type generation", () => {
  const ts = generateTypes(model, { source: "test" });

  it("maps attribute types", () => {
    expect(ts).toContain("export interface Article {");
    expect(ts).toContain("  title: string;");
    expect(ts).toContain('  category?: "tech" | "life" | null;');
    expect(ts).toContain("  views?: number | null;");
    expect(ts).toContain("  published?: boolean | null;");
    expect(ts).toContain("  author?: Entry<Author> | number | null;");
    expect(ts).toContain("  related?: Array<Entry<Article> | number>;");
    expect(ts).toContain("  seo?: Seo | null;");
    expect(ts).toContain("  blocks?: Seo[];");
    expect(ts).toContain("  gallery?: Array<Media | number>;");
    expect(ts).toContain("  data?: JsonValue | null;");
    expect(ts).toContain('  "odd-name"?: string | null;');
  });

  it("writes inputs with IDs for relations and media", () => {
    expect(ts).toContain("export interface ArticleInput {");
    expect(ts).toContain("  author?: number | AuthorInput | null;");
    expect(ts).toContain("  related?: Array<number | ArticleInput>;");
    expect(ts).toContain("  gallery?: number[];");
    expect(ts).toContain("  seo?: SeoInput | null;");
  });

  it("never reads passwords but can write them", () => {
    const author = ts.slice(ts.indexOf("export interface Author {"), ts.indexOf("export interface AuthorInput"));
    expect(author).not.toContain("secret");
    expect(ts).toMatch(/export interface AuthorInput \{[^}]*secret\?: string \| null;/);
  });

  it("augments the client module", () => {
    expect(ts).toContain('declare module "@gohead-cms/client" {');
    expect(ts).toContain("    articles: { attributes: Article; input: ArticleInput };");
    expect(ts).toContain("    homepage: { attributes: Homepage; input: HomepageInput };");
  });

  it("names types", () => {
    expect(typeName("articles")).toBe("Article");
    expect(typeName("blog_posts")).toBe("BlogPost");
    expect(typeName("categories")).toBe("Category");
    expect(typeName("site-settings", false)).toBe("SiteSettings");
    expect(singularize("status")).toBe("status");
    expect(singularize("boxes")).toBe("box");
    // A collection and a singleton with the same name get distinct types.
    const clash = generateTypes({ collections: { page: { attributes: {} } }, singletons: { page: { attributes: {} } } });
    expect(clash).toContain("export interface Page {");
    expect(clash).toContain("export interface PageSingle {");
  });
});

describe("models", () => {
  it("normalizes server attributes", () => {
    expect(normalizeAttribute({ type: "relation", target: "api::authors.authors", relation: "oneToOne", required: false, unique: false, options: [] })).toEqual({
      type: "relation",
      target: "authors",
      relation: "oneToOne",
    });
  });

  it("validates references", () => {
    expect(() => validateModel(model)).not.toThrow();
    expect(() =>
      validateModel({
        collections: {
          a: { attributes: { b: { type: "relation", target: "missing" }, c: { type: "component", component: "nope" }, d: { type: "enum" }, e: {} as never } },
        },
        locales: [{ code: "en", default: true }, { code: "fr", default: true }],
      }),
    ).toThrow(/target "missing"[\s\S]*component "nope"[\s\S]*enums need[\s\S]*"type" is required[\s\S]*only one locale/);
  });
});
