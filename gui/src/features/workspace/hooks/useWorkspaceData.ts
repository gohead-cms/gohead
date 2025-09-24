import { useState, useCallback } from "react";
import { Edge, MarkerType } from "@xyflow/react";
import { apiFetchWithAuth } from "../../../services/api";
import type { Schema, AgentNodeData } from "../../../shared/types";
import type { CollectionEdgeType } from "../../../shared/types";
import {
  TriggerEdgeType }
from '../../agents';
import type { AppNode, CollectionNode, AgentNode } from "../../../shared/types/workspace";
import { useLayout } from "./useLayout";


export function useWorkspaceData() {
  const [nodes, setNodes] = useState<AppNode[]>([]);
  const [edges, setEdges] = useState<Edge[]>([]);
  const [loading, setLoading] = useState(true);
  const { getLayoutedElements } = useLayout();

  const fetchDataAndLayout = useCallback((direction: string) => {
    setLoading(true);
    Promise.all([
      apiFetchWithAuth("/admin/collections"),
      apiFetchWithAuth("/admin/agents"),
    ]).then(async ([collectionsRes, agentsRes]) => {
      // Process Collections
      const collectionsJson = await collectionsRes.json();
      const collections: { schema: Schema }[] = collectionsJson.data || [];
      const collectionNodes: CollectionNode[] = collections.map((col) => ({
        id: col.schema.collectionName!,
        type: "collectionNode" as const,
        position: { x: 0, y: 0 },
        data: {
          label: col.schema.info?.displayName || col.schema.collectionName || '',
          attributes: Object.entries(col.schema.attributes).map(([name, attr]: [string, { type: string }]) => ({ name, type: attr.type })),
        },
      }));

      const collectionEdges: CollectionEdgeType[] = collections.flatMap((col) => {
        const collectionName = col.schema.collectionName;
        if (!collectionName) return [];
        return Object.entries(col.schema.attributes || {}).map(([attrName, attr]: [string, any]) => {
          if (attr.type.includes("relation") && attr.target) {
            const targetName = attr.target.split(".").pop();
            return {
              id: `${collectionName}-${targetName!}-${attrName}`,
              source: collectionName, target: targetName!, animated: true, style: { stroke: "#52b4ca" }, type: "collection",
              markerEnd: { type: MarkerType.ArrowClosed }, data: { label: attrName, relationType: attr.relation }
            };
          }
          return null;
        }).filter(Boolean) as CollectionEdgeType[];
      });

      // Process Agents
      const agentsJson = await agentsRes.json();
      const agents: AgentNodeData[] = agentsJson.data || [];
      const agentNodes: AgentNode[] = agents.map((agent) => ({
        id: agent.name,
        type: "agentNode" as const,
        position: { x: 0, y: 0 },
        data: agent,
      }));

        // --- UPDATED LOGIC HERE ---
        const agentTriggerEdges: TriggerEdgeType[] = agents.flatMap((agent) => {
          const trigger = agent.schema?.trigger;
          if (trigger?.type === "collection_event" && trigger.event_trigger?.collection) {
            return {
              id: `trigger-${trigger.event_trigger.collection}-to-${agent.name}`,
              source: trigger.event_trigger.collection,
              target: agent.name,
              type: "triggerEdge", 
              data: {
                events: trigger.event_trigger.events,
              },
            };
          }
          return [];
        });

      // Combine and Layout
      const allNodes: AppNode[] = [...collectionNodes, ...agentNodes];
      const allEdges: Edge[] = [...collectionEdges, ...agentTriggerEdges];
      const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(allNodes, allEdges, direction);

      setNodes(layoutedNodes);
      setEdges(layoutedEdges);
    }).finally(() => setLoading(false));
  }, [getLayoutedElements]);

  return {
    nodes,
    edges,
    loading,
    setNodes,
    setEdges,
    fetchDataAndLayout,
  };
}