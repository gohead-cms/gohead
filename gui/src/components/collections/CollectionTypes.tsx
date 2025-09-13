// src/components/collections/types.ts

export interface Attribute {
  type: string;
  required?: boolean;
  enum?: string[];
  target?: string;
}

export interface Schema {
  attributes: Record<string, Attribute>;
  collectionName?: string;
  info?: {
    displayName?: string;
    pluralName?: string;
    singularName?: string;
  };
}
