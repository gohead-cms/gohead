import type { Node } from '@xyflow/react';

/**
 * Represents a single function tool available to the agent.
 */
export interface AgentFunction {
  name: string;
  description: string;
}

/**
 * Defines the structure of the custom data payload for an agent node.
 * This is the data property within a Node, not the Node itself.
 * Must extend Record<string, unknown> for React Flow compatibility.
 */
export interface AgentNodeData {
  label: string;
  name: string;
  schema: {
    functions: AgentFunction[];
    llmConfig: {
      provider: string;
      model: string;
    };
    trigger: {
      type: string;
      event_trigger?: {
        collection: string;
        events: string[];
      };
    };
    systemPrompt?: string;
  };
  [key: string]: unknown;
}

/**
 * Represents a single function tool available to the agent.
 */
export interface AgentFunction {
  name: string;
  description: string;
}

/** 
 * Typed React Flow Node for this agent.
 * This represents the complete node with id, position, data, etc.
 */
export type AgentNodeType = Node<AgentNodeData, 'agentNode'>;