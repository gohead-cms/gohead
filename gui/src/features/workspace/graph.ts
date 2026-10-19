// Builds the read-only Schema Designer graph from the saved model. Pure functions.
import { MarkerType } from "@xyflow/react";
import type { Edge } from "@xyflow/react";
import dagre from "@dagrejs/dagre";
import type { AgentNodeData, CollectionEdgeType } from "../../shared/types";
import type { AppNode, AttributeItem, CollectionNode, AgentNode } from "../../shared/types/workspace";
import type { TriggerEdgeType } from "../agents";
import { relationTarget } from "../model/schema";
import type { ApiAttribute, ApiCollection } from "../model/schema";

export type LayoutDirection = "LR" | "TB";

export type NodeRef = { kind: "collection" | "agent"; name: string };

const COLLECTION_PREFIX = "collection:";
const AGENT_PREFIX = "agent:";

/** Node ids are prefixed so a collection and an agent can share a name. */
export const collectionNodeId = (name: string) => COLLECTION_PREFIX + name;
export const agentNodeId = (name: string) => AGENT_PREFIX + name;

export function parseNodeId(id: string): NodeRef | null {
  if (id.startsWith(COLLECTION_PREFIX)) return { kind: "collection", name: id.slice(COLLECTION_PREFIX.length) };
  if (id.startsWith(AGENT_PREFIX)) return { kind: "agent", name: id.slice(AGENT_PREFIX.length) };
  return null;
}

function attributeSummary(name: string, a: ApiAttribute): AttributeItem {
  if (a.type === "relation") return { name, type: `→ ${relationTarget(a.target)}` };
  if (a.type === "component") return { name, type: `${a.component ?? "component"}${a.repeatable ? "[]" : ""}` };
  return { name, type: a.type };
}

export function buildGraph(collections: ApiCollection[], agents: AgentNodeData[]): { nodes: AppNode[]; edges: Edge[] } {
  const names = new Set(collections.map((c) => c.schema.collectionName));

  const collectionNodes: CollectionNode[] = collections.map((c) => ({
    id: collectionNodeId(c.schema.collectionName),
    type: "collectionNode",
    position: { x: 0, y: 0 },
    data: {
      label: c.schema.collectionName,
      attributes: Object.entries(c.schema.attributes ?? {}).map(([n, a]) => attributeSummary(n, a)),
    },
  }));

  const relationEdges: CollectionEdgeType[] = collections.flatMap((c) => {
    const source = c.schema.collectionName;
    return Object.entries(c.schema.attributes ?? {}).flatMap(([attrName, a]) => {
      const target = relationTarget(a.target);
      if (a.type !== "relation" || !names.has(target)) return [];
      const edge: CollectionEdgeType = {
        id: `relation:${source}:${attrName}`,
        source: collectionNodeId(source),
        target: collectionNodeId(target),
        type: "collection",
        markerEnd: { type: MarkerType.ArrowClosed },
        data: { label: attrName, relationType: a.relation ?? "", attributes: [] },
      };
      return [edge];
    });
  });

  const agentNodes: AgentNode[] = agents.map((agent) => ({
    id: agentNodeId(agent.name),
    type: "agentNode",
    position: { x: 0, y: 0 },
    data: agent,
  }));

  const triggerEdges: TriggerEdgeType[] = agents.flatMap((agent) => {
    const trigger = agent.schema?.trigger;
    const collection = trigger?.event_trigger?.collection;
    if (trigger?.type !== "collection_event" || !collection || !names.has(collection)) return [];
    const edge: TriggerEdgeType = {
      id: `trigger:${collection}:${agent.name}`,
      source: collectionNodeId(collection),
      target: agentNodeId(agent.name),
      type: "triggerEdge",
      data: { events: trigger.event_trigger?.events ?? [] },
    };
    return [edge];
  });

  return { nodes: [...collectionNodes, ...agentNodes], edges: [...relationEdges, ...triggerEdges] };
}

const NODE_WIDTH = 260;
const NODE_HEIGHT = 220;

/** Positions nodes with dagre; returns new node objects. */
export function layoutGraph(nodes: AppNode[], edges: Edge[], direction: LayoutDirection): AppNode[] {
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir: direction, nodesep: 80, ranksep: 160 });
  nodes.forEach((n) => g.setNode(n.id, { width: NODE_WIDTH, height: NODE_HEIGHT }));
  edges.forEach((e) => g.setEdge(e.source, e.target));
  dagre.layout(g);
  return nodes.map((n) => {
    const p = g.node(n.id);
    return { ...n, position: { x: p.x - NODE_WIDTH / 2, y: p.y - NODE_HEIGHT / 2 } };
  });
}

/** A short hash of what the graph shows, used to reset the view when the model changes. */
export function graphSignature(nodes: AppNode[], edges: Edge[]): string {
  const text = JSON.stringify([nodes.map((n) => [n.id, n.data]), edges.map((e) => [e.id, e.data])]);
  let h = 0;
  for (let i = 0; i < text.length; i++) h = (Math.imul(31, h) + text.charCodeAt(i)) | 0;
  return (h >>> 0).toString(36);
}
