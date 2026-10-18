import type { Node } from '@xyflow/react';

/**
 * Represents a single function tool available to the agent.
 */
export interface AgentFunction {
  name: string;
  description: string;
  impl_key?: string;
  /** collections.* functions: the collections they may use (empty: all). */
  collections?: string[];
}

export interface AgentLimits {
  max_runs_per_hour?: number;
  max_runs_per_day?: number;
  max_tokens_per_run?: number;
  max_tokens_per_day?: number;
  max_cost_per_day?: number;
}

export type RunStatus = 'running' | 'succeeded' | 'failed' | 'limited';

export interface AgentRunStep {
  id: number;
  seq: number;
  type: 'llm' | 'tool';
  name: string;
  implKey?: string;
  arguments: string;
  result: string;
  error: string;
  inputTokens: number;
  outputTokens: number;
  costUsd: number;
  durationMs: number;
  createdAt: string;
}

export interface AgentRun {
  id: number;
  status: RunStatus;
  triggerType: string;
  trigger: string;
  input: string;
  output: string;
  error: string;
  attempt: number;
  turns: number;
  toolCalls: number;
  provider: string;
  model: string;
  inputTokens: number;
  outputTokens: number;
  costUsd: number;
  startedAt: string;
  finishedAt: string | null;
  durationMs: number;
  steps?: AgentRunStep[];
}

export interface AgentUsageWindow {
  runs: number;
  inputTokens: number;
  outputTokens: number;
  costUsd: number;
}

export interface AgentUsage {
  lastHour: AgentUsageWindow;
  lastDay: AgentUsageWindow;
  limits: AgentLimits;
  costKnown: boolean;
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
      input_cost_per_million?: number;
      output_cost_per_million?: number;
    };
    trigger: {
      type: string;
      event_trigger?: {
        collection: string;
        events: string[];
      };
    };
    systemPrompt?: string;
    limits?: AgentLimits;
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