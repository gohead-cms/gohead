import { useMe } from "../../auth/me";

/** What the signed-in role may do with the entries of one collection. */
export interface EntryPermissions {
  create: boolean;
  update: boolean;
  delete: boolean;
  publish: boolean;
}

export function useEntryPermissions(collection: string): EntryPermissions {
  const { can } = useMe();
  return {
    create: can("create", collection),
    update: can("update", collection),
    delete: can("delete", collection),
    publish: can("publish", collection),
  };
}
