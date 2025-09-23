import { useCallback } from "react";
import { Edge } from "@xyflow/react";
import dagre from "@dagrejs/dagre";
import type { AppNode } from "../../../shared/types/workspace";

const dagreGraph = new dagre.graphlib.Graph();
dagreGraph.setDefaultEdgeLabel(() => ({}));

const nodeWidth = 200;
const nodeHeight = 60;

export function useLayout() {
  const getLayoutedElements = useCallback((nodes: AppNode[], edges: Edge[], direction = "TB") => {
    dagreGraph.setGraph({
      rankdir: direction,
      nodesep: 300,
      ranksep: 350,
    });

    nodes.forEach((node) => {
      dagreGraph.setNode(node.id, { width: nodeWidth, height: nodeHeight });
    });

    edges.forEach((edge) => {
      dagreGraph.setEdge(edge.source, edge.target);
    });

    dagre.layout(dagreGraph);

    const layoutedNodes = nodes.map((node) => {
      const nodeWithPosition = dagreGraph.node(node.id);

      node.position = {
        x: nodeWithPosition.x - nodeWidth / 2,
        y: nodeWithPosition.y - nodeHeight / 2,
      };

      return node;
    });

    return { nodes: layoutedNodes, edges };
  }, []);

  return { getLayoutedElements };
}