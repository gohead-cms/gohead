import { Node } from '@xyflow/react';
import { AgentNodeData, AgentNodeType } from './agents';

export interface AttributeItem {
  name: string;
  type: string;
}

export type CollectionNode = Node<{
  label: string;
  attributes: AttributeItem[];
}, 'collectionNode'>;


export type CollectionNodeData = {
  label: string;
  attributes: AttributeItem[];
};

export type AgentNode = Node<AgentNodeData, 'agentNode'>;

export type AppNode = CollectionNode | AgentNode;

export type ContextMenuState = {
  x: number;
  y: number;
  node: AppNode;
} | null;

export type NewCollectionData = {
  displayName: string;
  singularId: string;
  pluralId: string;
};

export type ValidationErrors = {
  displayName: string;
};
