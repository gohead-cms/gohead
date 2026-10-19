/** What the content editor shows: a collection's entries or a singleton. */
export interface Selection {
  kind: "collection" | "singleton";
  name: string;
}

/** Query parameter of each kind, e.g. /contrib?singleton=homepage. */
const PARAMS: Record<Selection["kind"], string> = { collection: "collection", singleton: "singleton" };

/** The selection named in the URL, if any. */
export function selectionFromParams(params: URLSearchParams): Selection | null {
  const singleton = params.get(PARAMS.singleton);
  if (singleton) return { kind: "singleton", name: singleton };
  const collection = params.get(PARAMS.collection);
  if (collection) return { kind: "collection", name: collection };
  return null;
}

/** The URL query for a selection. */
export function selectionParams(selection: Selection): Record<string, string> {
  return { [PARAMS[selection.kind]]: selection.name };
}

/** A link to a selection in the content editor. */
export function contribLink(selection: Selection): string {
  return `/contrib?${new URLSearchParams(selectionParams(selection))}`;
}
