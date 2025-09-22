import { Node } from '@xyflow/react';
import { AgentNodeData, AgentNodeType } from './agents';
/**
 * Represents the data structure for a single attribute within a collection.
 */
export interface AttributeItem {
  name: string;
  type: string;
}

/**
 * Defines a React Flow node specifically for a 'Collection'.
 * The `data` property contains the label and attributes.
 */
export type CollectionNode = Node<{
  label: string;
  attributes: AttributeItem[];
}, 'collectionNode'>;

/**
 * Defines a React Flow node specifically for an 'Agent'.
 * The `data` property contains the full agent data structure.
 */
export type AgentNode = Node<AgentNodeData, 'agentNode'>;

/**
 * A union type that represents any possible node in your workspace.
 * This is useful for state management and function parameters.
 */
export type AppNode = CollectionNode | AgentNode;





