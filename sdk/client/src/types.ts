/**
 * Content types known to the client, filled in by generated types
 * (`npx @gohead-cms/client types`), which augment these interfaces:
 *
 * ```ts
 * declare module "@gohead-cms/client" {
 *   interface Collections { articles: { attributes: Article; input: ArticleInput } }
 *   interface Singletons { homepage: { attributes: Homepage; input: HomepageInput } }
 * }
 * ```
 *
 * Without generated types, every collection and singleton is untyped
 * (`Record<string, unknown>`).
 */
// eslint-disable-next-line @typescript-eslint/no-empty-object-type
export interface Collections {}
// eslint-disable-next-line @typescript-eslint/no-empty-object-type
export interface Singletons {}

/** A content type's shape: its attributes as read, and as written. */
export interface TypeShape {
  attributes: object;
  input: object;
}

export type CollectionName = keyof Collections & string;
export type SingletonName = keyof Singletons & string;

/** Any JSON value (json attributes). */
export type JsonValue = string | number | boolean | null | JsonValue[] | { [key: string]: JsonValue };

/** Fields every entry has. */
export interface SystemFields {
  id: number;
  /** ISO 8601 (UTC). */
  createdAt: string;
  updatedAt: string;
}

/** Fields of entries in draft-and-publish collections. */
export interface PublishFields {
  status?: "draft" | "published";
  publishedAt?: string | null;
}

/** Fields of entries read in a locale. */
export interface LocaleFields {
  /** The locale the localized values are in. */
  locale?: string;
  /** The locales the entry has values in. */
  locales?: string[];
}

/** An entry as the API returns it: its attributes and system fields. */
export type Entry<T = Record<string, unknown>> = T & SystemFields & PublishFields & LocaleFields;

/** A file of the media library. */
export interface Media {
  id: number;
  /** Absolute with an S3/CDN store, or a path such as /uploads/… with the local store: see `client.media.url`. */
  url: string;
  name: string;
  mime: string;
  size: number;
  width?: number | null;
  height?: number | null;
  hash?: string;
  alt?: string;
  caption?: string;
  uploadedBy?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface Pagination {
  page: number;
  pageSize: number;
  pageCount: number;
  total: number;
}

/** A page of entries. */
export interface ListResponse<T> {
  data: T[];
  meta: { pagination: Pagination };
}

export interface Locale {
  id: number;
  code: string;
  name: string;
  isDefault: boolean;
  createdAt?: string;
}

/** The signed-in account or token (`GET /auth/me`). */
export interface Me {
  id?: number;
  username: string;
  email?: string;
  role: string;
  kind: "user" | "token";
  permissions: {
    content: Record<string, string[]>;
    media: string[];
  };
}

// ---- Queries ----

type Scalar = string | number | boolean;

/** Filter operators on one field. */
export interface Operators<V = Scalar> {
  $eq?: V;
  $ne?: V;
  $lt?: V;
  $lte?: V;
  $gt?: V;
  $gte?: V;
  $contains?: string;
}

type FilterValue<V> = V extends Scalar ? V | Operators<V> : never;

/** Fields that can be filtered on: scalar attributes and system fields. */
type FilterableKeys<T> = {
  [K in keyof T & string]-?: NonNullable<T[K]> extends Scalar ? K : never;
}[keyof T & string];

/**
 * Conditions combined with AND, e.g. `{ category: "tech", views: { $gte: 10 } }`.
 * Relations, components, media and JSON attributes cannot be filtered on.
 */
export type Filters<T = Record<string, unknown>> = Record<string, unknown> extends T
  ? Record<string, Scalar | Operators>
  : { [K in FilterableKeys<T>]?: FilterValue<NonNullable<T[K]>> } & {
      id?: FilterValue<number>;
      createdAt?: FilterValue<string>;
      updatedAt?: FilterValue<string>;
      publishedAt?: FilterValue<string>;
    };

type SortKey<T> = Record<string, unknown> extends T
  ? string
  : FilterableKeys<T> | "id" | "createdAt" | "updatedAt" | "publishedAt";

/** `"views"`, `"views:desc"`, or several keys. Ties are broken by id. */
export type Sort<T = Record<string, unknown>> =
  | SortKey<T>
  | `${SortKey<T>}:${"asc" | "desc"}`
  | Array<SortKey<T> | `${SortKey<T>}:${"asc" | "desc"}`>;

/** Options of reads that return entries. */
export interface ReadOptions {
  /** How deep relations are populated: 0 (IDs only) to 5. Server default: 1. */
  level?: 0 | 1 | 2 | 3 | 4 | 5;
  /** Locale of localized values (defaults to the client's `locale`, then the server's default). */
  locale?: string;
  /** Fall back to the default locale for missing values (server default: true). */
  fallback?: boolean;
}

export interface FindOptions<T = Record<string, unknown>> extends ReadOptions {
  filters?: Filters<T>;
  sort?: Sort<T>;
  /** 1-based. */
  page?: number;
  /** 1 to 100 (server default: 10). */
  pageSize?: number;
  /** Draft-and-publish collections: `published` (default), `draft` or `all`. Drafts need a token that may edit. */
  status?: "published" | "draft" | "all";
}

export interface WriteOptions {
  /** Draft-and-publish collections: publish or unpublish with the write. */
  status?: "draft" | "published";
  /** Depth of relations in the returned entry. */
  level?: ReadOptions["level"];
  /** Write the values of this locale (a translation) instead of the default locale. */
  locale?: string;
}

/**
 * Values to write. Relations take IDs (or objects to create related
 * entries), media take file IDs, components take objects.
 */
export type Input<T> = Partial<T>;
