import { Node, Edge } from "@xyflow/react";

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