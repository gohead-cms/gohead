import { ApiError } from "../../services/api";
import { getJSON } from "../../shared/hooks/useApiQuery";
import { errorMessage } from "../../shared/utils/errors";
import type { ApiCollection, ApiNamedType, ModelKind, TypeForm } from "./schema";
import {
  formFromCollection,
  formFromNamedType,
  listPath,
} from "./schema";

/** One row of the content model lists. */
export interface TypeSummary {
  name: string;
  description: string;
  attributeCount: number;
  /** Attributes with a value per locale. */
  localizedCount: number;
  /** Collections only. */
  draftAndPublish?: boolean;
}

export function summarizeCollections(list: ApiCollection[]): TypeSummary[] {
  return list.map((c) => ({
    name: c.schema.collectionName,
    description: c.schema.description ?? "",
    attributeCount: Object.keys(c.schema.attributes ?? {}).length,
    localizedCount: Object.values(c.schema.attributes ?? {}).filter((a) => a.localized).length,
    draftAndPublish: !!c.schema.options?.draftAndPublish,
  }));
}

export function summarizeNamed(list: ApiNamedType[]): TypeSummary[] {
  return list.map((t) => ({
    name: t.name,
    description: t.description ?? "",
    attributeCount: t.attributes?.length ?? 0,
    localizedCount: (t.attributes ?? []).filter((a) => a.localized).length,
  }));
}

/** Loads the types of one kind as list rows. */
export async function loadSummaries(kind: ModelKind): Promise<TypeSummary[]> {
  if (kind === "collections") {
    const res = await getJSON<ApiCollection[]>(listPath(kind), "Failed to load collections");
    return summarizeCollections(res.data ?? []);
  }
  const res = await getJSON<ApiNamedType[]>(listPath(kind), `Failed to load ${kind}`);
  return summarizeNamed(res.data ?? []);
}

export interface LoadedType {
  form: TypeForm;
}

/** Loads one type into form state, or null when it does not exist. */
export async function loadType(kind: ModelKind, name: string): Promise<LoadedType | null> {
  if (kind === "collections") {
    const res = await getJSON<ApiCollection>(`/admin/collections/${encodeURIComponent(name)}`, "Failed to load the collection");
    return { form: formFromCollection(res.data) };
  }
  // The list carries the description, which GET /admin/singleton/{name} leaves out.
  const res = await getJSON<ApiNamedType[]>(listPath(kind), `Failed to load ${kind}`);
  const found = (res.data ?? []).find((t) => t.name === name);
  return found ? { form: formFromNamedType(found) } : null;
}

/**
 * The server's message for a failed request. Some singleton endpoints send
 * an object as the message and the real reason in "details".
 */
export function serverMessage(err: unknown): string {
  if (err instanceof ApiError) {
    if (typeof err.details === "string" && err.details) return err.details;
    if (err.message.startsWith("[object Object]")) return `Request failed (${err.status})`;
  }
  return errorMessage(err);
}
