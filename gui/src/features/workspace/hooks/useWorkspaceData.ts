import { useState, useEffect, useCallback } from "react";
import { Edge } from "@xyflow/react";
import dagre from "@dagrejs/dagre";
import { apiFetchWithAuth } from "../../../services/api";
import { AgentNodeType } from "../../../shared/types/agents";
import { AppNode, CollectionNode } from "../../../shared/types";
import { Schema as CollectionSchema } from "../../../shared/types/collections";
import { AgentNodeData } from "../../../shared/types/agents";
import { MarkerType } from "@xyflow/react";

const dagreGraph = new dagre.graphlib.Graph(); 
dagreGraph.setDefaultEdgeLabel(() => ({}));

const nodeWidth = 200;
const nodeHeight = 60;

const getLayoutedElements = (nodes: AppNode[], edges: Edge[], direction: string) => {
  dagreGraph.setGraph({ rankdir: direction, nodesep: 300, ranksep: 350 });
  nodes.forEach((node) => {
    dagreGraph.setNode(node.id, { width: nodeWidth, height: nodeHeight });
  });
  edges.forEach((edge) => dagreGraph.setEdge(edge.source, edge.target));
  dagre.layout(dagreGraph);

  nodes.forEach((node) => {
    const nodeWithPosition = dagreGraph.node(node.id);
    node.position = {
      x: nodeWithPosition.x - nodeWidth / 2,
      y: nodeWithPosition.y - nodeHeight / 2,
    };
  });
  return { nodes, edges };
};

export function useWorkspaceData(direction: string) {
  const [nodes, setNodes] = useState<AppNode[]>([]);
  const [edges, setEdges] = useState<Edge[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchDataAndLayout = useCallback(() => {
    setLoading(true);
    Promise.all([
      apiFetchWithAuth("/admin/collections"),
      apiFetchWithAuth("/admin/agents"),
    ])
      .then(async ([collectionsRes, agentsRes]) => {
        // Process Collections
        const collectionsJson = await collectionsRes.json();
        const collections: { schema: CollectionSchema }[] = collectionsJson.data || [];
        const collectionNodes: CollectionNode[] = collections.map((col) => ({
          id: col.schema.collectionName!,
          type: "collectionNode",
          position: { x: 0, y: 0 },
          data: {
            label: col.schema.info?.displayName || col.schema.collectionName!,
            attributes: Object.entries(col.schema.attributes).map(
              ([name, attr]: any) => ({ name, type: attr.type })
            ),
          },
        }));

        const collectionEdges: Edge[] = collections.flatMap((col) => {
            const collectionName = col.schema.collectionName;
            if (!collectionName) return [];
            return Object.entries(col.schema.attributes || {}).map(([attrName, attr]: [string, any]) => {
                if (attr.type.includes("relation") && attr.target) {
                    const targetName = attr.target.split(".").pop();
                    return {
                        id: `${collectionName}-${targetName}-${attrName}`,
                        source: collectionName, target: targetName, animated: true, style: { stroke: "#52b4ca" }, type: "collection",
                        markerEnd: { type: MarkerType.ArrowClosed },
                        data: { label: attrName, relationType: attr.relation }
                    };
                }
                return null;
            }).filter(Boolean) as Edge[];
        });

        // Process Agents
        const agentsJson = await agentsRes.json();
        const agents: AgentNodeData[] = agentsJson.data || [];
        const agentNodes: AgentNodeType[] = agents.map((agent) => ({
            id: agent.name,
            type: 'agentNode',
            position: { x: 0, y: 0 },
            data: agent
        }));
        
        const agentTriggerEdges: Edge[] = agents.flatMap((agent) => {
            const trigger = agent.schema?.trigger;
            if (trigger?.type === 'collection_event' && trigger.event_trigger?.collection) {
                return {
                    id: `trigger-${trigger.event_trigger.collection}-to-${agent.name}`,
                    source: trigger.event_trigger.collection,
                    target: agent.name,
                    type: 'smoothstep',
                    animated: true,
                    style: { stroke: '#8a52ca', strokeDasharray: '5,5' },
                    markerEnd: { type: MarkerType.ArrowClosed, color: '#8a52ca' },
                };
            }
            return [];
        });

        // Combine and Layout
        const allNodes = [...collectionNodes, ...agentNodes];
        const allEdges = [...collectionEdges, ...agentTriggerEdges];
        const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(
          allNodes,
          allEdges,
          direction
        );

        setNodes(layoutedNodes);
        setEdges(layoutedEdges);
      })
      .finally(() => setLoading(false));
  }, [direction]);

  useEffect(() => {
    fetchDataAndLayout();
  }, [fetchDataAndLayout]);

  return { nodes, setNodes, edges, setEdges, loading };
}
