import React from 'react';
import { Handle, NodeProps, Position, Node } from '@xyflow/react';

export interface Attribute {
  name: string;
  type: string;
}

// Define the structure of your node's data
export interface CollectionNodeData {
  label: string;
  attributes?: Attribute[];
  [key: string]: unknown;
}

export type CollectionNodeType = Node<CollectionNodeData, 'collectionNode'>;