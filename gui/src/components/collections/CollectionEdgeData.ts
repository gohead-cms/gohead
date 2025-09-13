import { Edge } from "@xyflow/react";

export interface CollectionEdgeData {
  label: string;
  relationType: string;
  attributes: string[];
  [key: string]: unknown;
}

export type CollectionEdgeType = Edge<CollectionEdgeData, 'collection'>;