import { type Node, type Edge } from "@xyflow/react";

export interface Attribute {
  type: string;
  name: string;
  required?: boolean;
  enum?: string[];
  target?: string;
  relation?: string;
  options?: string[];
  component?: string;
  repeatable?: boolean;
  multiple?: boolean;
  allowed_types?: string[];
  /** A value per locale (see GET /api/locales). */
  localized?: boolean;
}

/** A value of a simple form field (text, number, date, boolean). */
export type FieldValue = string | number | boolean;

export interface Schema {
  attributes: Record<string, Attribute>;
  collectionName?: string;
  options?: {
    draftAndPublish?: boolean;
  };
  info?: {
    displayName?: string;
    pluralName?: string;
    singularName?: string;
  };
}

// Define the structure of node's data
export interface CollectionNodeData {
  label: string;
  attributes?: Attribute[];
  [key: string]: unknown;
};

export interface CollectionEdgeData {
  label: string;
  relationType: string;
  attributes: string[];
  [key: string]: unknown;
}

export type CollectionNodeType = Node<CollectionNodeData, 'collectionNode'>
export type CollectionEdgeType = Edge<CollectionEdgeData, 'collection'>;