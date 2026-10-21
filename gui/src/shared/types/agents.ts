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
  /** JSON schema of the arguments, sent to the model. */
  parameters?: Record<string, unknown>;
}

/**
 * A tool of an agent, in the short form the API reads and returns: which
 * tool ("content.update"), on which collections. Name, description and
 * parameters are overrides; left out, they come from the catalog.
 */
export interface AgentTool {
  use: string;
  on?: string[];
  name?: string;
  description?: string;
  parameters?: Record<string, unknown>;
}

/** A built-in tool from GET /admin/agent-functions. */
export interface AgentFunctionInfo {
  impl_key: string;
  /** The short name agents use ("content.update"). */
  use: string;
  /** The name the model sees by default. */
  name: string;
  label: string;
  group: "content" | "ai" | "system" | string;
  description: string;
  parameters: Record<string, unknown>;
  /** The tool works on collections and accepts an "on" scope. */
  scoped: boolean;
  /** The tool changes content. */
  writes: boolean;
}

export interface AgentLimits {
  max_runs_per_hour?: number;
  max_runs_per_day?: number;
  max_tokens_per_run?: number;
  max_tokens_per_day?: number;
  max_cost_per_day?: number;
}

/** queued: requested (run now, test run) and waiting for a worker. */
export type RunStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'limited';

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
  agentId?: number;
  agentName?: string;
  status: RunStatus;
  /** A test run: content functions only reported what they would change. */
  dryRun?: boolean;
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
  /** Numeric ID (used by the webhook trigger URL). */
  id?: number;
  name: string;
  schema: {
    maxTurns?: number;
    memory?: { type: string; session_scope: string };
    /** Every tool as the model gets it (names and schemas filled in). */
    functions: AgentFunction[];
    /** The tools as written (short form). */
    tools?: AgentTool[];
    llmConfig: {
      provider: string;
      model: string;
      /** "********" when a key is stored, "" otherwise. */
      api_key?: string;
      api_secret?: string;
      base_url?: string;
      input_cost_per_million?: number;
      output_cost_per_million?: number;
    };
    trigger: {
      type: string;
      cron?: string;
      /** "********" when a token is stored. */
      webhook_token?: string;
      event_trigger?: {
        collection: string;
        events: string[];
      };
    };
    systemPrompt?: string;
    limits?: AgentLimits;
    /** Content writes become proposals an editor approves (when the API sends it). */
    requireApproval?: boolean;
    require_approval?: boolean;
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

/** A parameter of an agent template (GET /admin/agent-templates). */
export interface AgentTemplateParam {
  key: string;
  label: string;
  type: 'collection' | 'field' | 'fields' | 'text' | 'number' | 'bool';
  required: boolean;
  default?: unknown;
  help?: string;
  /** field / fields: the attribute types allowed. */
  types?: string[];
  /** field / fields: only localized attributes. */
  localized?: boolean;
}

/** A ready-made agent. */
export interface AgentTemplate {
  id: string;
  name: string;
  description: string;
  params: AgentTemplateParam[];
}

/** An agent definition: the body of POST /admin/agents (what a template renders). */
export interface AgentDefinition {
  name: string;
  system_prompt: string;
  max_turns?: number;
  trigger?: {
    type: string;
    cron?: string;
    webhook_token?: string;
    event_trigger?: { collection: string; events: string[] };
  };
  functions?: (AgentFunction & { collections?: string[]; parameters?: Record<string, unknown> })[];
  tools?: AgentTool[];
  llm_config?: AgentNodeData['schema']['llmConfig'];
  memory?: { type: string; session_scope: string };
  require_approval?: boolean;
  limits?: AgentLimits;
}

/** 
 * Typed React Flow Node for this agent.
 * This represents the complete node with id, position, data, etc.
 */
export type AgentNodeType = Node<AgentNodeData, 'agentNode'>;